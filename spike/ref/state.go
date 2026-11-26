// Package ref is the own-Go reference: Countersign's rules written directly
// in Go. In the spike it is a reference, not a candidate
// (spike-preregistration.md).
//
// State is the permission store's current state. Every write goes through
// a method that checks the write-time rules, then appends one event to the
// log. Records are never changed in place: withdrawal, narrowing and
// signals are kept beside the grants they affect.
package ref

import (
	"errors"
	"sync"
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
)

// Limits holds the tunable values. Values marked [tentative] in the PRD
// are targets, not decisions.
type Limits struct {
	TaskMax             time.Duration // FR-3: under 24 h [tentative]
	ContainedMax        time.Duration // FR-10: 4 h [tentative]
	ServiceMax          time.Duration // FR-3: [OPEN]
	AuthorityMax        time.Duration // FR-7: 90 days [tentative]
	SystemMaxDepth      int           // FR-18: 3 [tentative]
	AdminMin            int           // FR-9: 2
	AdminStagger        time.Duration // FR-9: 14 days [tentative]
	PendingMax          time.Duration // how long a request waits for approval [tentative]
	PendingPerPrincipal int           // open requests one principal may have at once [tentative]
}

func DefaultLimits() Limits {
	return Limits{
		TaskMax:             24 * time.Hour,
		ContainedMax:        4 * time.Hour,
		ServiceMax:          30 * 24 * time.Hour, // placeholder: the PRD leaves it open
		AuthorityMax:        90 * 24 * time.Hour,
		SystemMaxDepth:      3,
		AdminMin:            2,
		AdminStagger:        14 * 24 * time.Hour,
		PendingMax:          24 * time.Hour,
		PendingPerPrincipal: 20,
	}
}

// Event is one entry in the append-only log.
type Event struct {
	Position uint64
	Type     string
	At       time.Time
	By       string
	Subject  string
}

type narrowing struct {
	resource model.Resource
	endsAt   time.Time
}

type withdrawal struct {
	by model.Identity
	at time.Time
}

// pending is a request waiting for a person's approval. It expires after
// Limits.PendingMax.
type pending struct {
	grant     model.Grant // the grant to create, without approver or position
	requester model.Identity
	at        time.Time
}

// State is safe for concurrent use: checks take a read lock and writes take
// the write lock. This is the prototype's simplification. The replica's
// design is an immutable state swapped atomically, so checks never wait;
// the spike measures the cost of the lock instead (part 4).
type State struct {
	mu        sync.RWMutex
	limits    Limits
	engine    engine.Engine
	prefilter bool
	log       []Event

	grants      map[string]*model.Grant
	byPrincipal map[model.Identity][]*model.Grant
	children    map[string][]string
	narrowed    map[string]narrowing
	withdrawn   map[string]withdrawal

	openTasks   map[string]bool
	closedTasks map[string]uint64         // log position of the latest "task closed"
	endedAgents map[model.Identity]uint64 // log position of the signal

	resourceInfo []model.ResourceInfo
	markings     []model.Marking
	authorities  []model.ApprovalAuthority
	sharing      []model.SharingLimit
	serviceAuth  []model.ServiceAuthorisation
	admins       map[model.PersonID]model.Administrator

	pending   map[string]pending
	proposals map[string]proposal
	nextID    int
}

var (
	ErrRefused      = errors.New("refused")
	ErrNotPermitted = errors.New("not permitted")
	ErrNeedsSecond  = errors.New("needs a second administrator")
)

// proposal is a classification waiting for a second administrator.
type proposal struct {
	info model.ResourceInfo
	by   model.PersonID
	at   time.Time
}

// NewState creates an installation with its first administrators, set once
// at install time (FR-9). At least Limits.AdminMin are required.
//
// The engine decides; State gathers the facts it needs from the log.
func NewState(l Limits, e engine.Engine, now time.Time, firstAdmins ...model.PersonID) (*State, error) {
	if len(firstAdmins) < l.AdminMin {
		return nil, errors.New("an installation needs at least two administrators")
	}
	s := &State{
		limits:      l,
		engine:      engine.Guard(e),
		prefilter:   true,
		grants:      map[string]*model.Grant{},
		byPrincipal: map[model.Identity][]*model.Grant{},
		children:    map[string][]string{},
		narrowed:    map[string]narrowing{},
		withdrawn:   map[string]withdrawal{},
		openTasks:   map[string]bool{},
		closedTasks: map[string]uint64{},
		endedAgents: map[model.Identity]uint64{},
		admins:      map[model.PersonID]model.Administrator{},
		pending:     map[string]pending{},
		proposals:   map[string]proposal{},
	}
	for i, p := range firstAdmins {
		// Staggered expiries, so the first administrators don't lapse together.
		ends := now.Add(l.AuthorityMax - time.Duration(i)*l.AdminStagger)
		pos := s.append("administrator_added", now, "installation", p.String())
		s.admins[p] = model.Administrator{Person: p, AddedBy: "installation", EndsAt: ends, LogPosition: pos}
	}
	return s, nil
}

func (s *State) append(typ string, at time.Time, by, subject string) uint64 {
	pos := uint64(len(s.log) + 1)
	s.log = append(s.log, Event{Position: pos, Type: typ, At: at, By: by, Subject: subject})
	return pos
}

// SetPrefilter turns the candidate prefilter on or off (it is on by
// default). With it on, the layer skips grants whose own scope can't cover
// the resource before calling the engine: the engine would refuse them
// anyway, so no decision changes. Off is the "full path" the spike also
// reports (spike-preregistration.md, changes after measurement, 3).
func (s *State) SetPrefilter(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefilter = on
}

// Log returns a copy of the event log.
func (s *State) Log() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Event(nil), s.log...)
}

func (s *State) newID(prefix string) string {
	s.nextID++
	return prefix + "-" + itoa(s.nextID)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// OpenTask registers a task as real and open, or opens a closed one again.
// In the product, tasks come from the calling product (FR-3).
func (s *State) OpenTask(task string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.openTasks[task] = true
}

// Grant returns a copy of a grant, if it exists.
func (s *State) Grant(id string) (model.Grant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.grants[id]
	if !ok {
		return model.Grant{}, false
	}
	return *g, true
}

func (s *State) addGrant(g model.Grant, at time.Time, eventType, by string) *model.Grant {
	g.IssuedAt = g.IssuedAt.Truncate(time.Millisecond)
	g.EndsAt = g.EndsAt.Truncate(time.Millisecond)
	g.ReviewDate = g.ReviewDate.Truncate(time.Millisecond)
	g.LogPosition = s.append(eventType, at, by, g.ID)
	p := &g
	s.grants[g.ID] = p
	s.byPrincipal[g.Principal] = append(s.byPrincipal[g.Principal], p)
	if g.Parent != "" {
		s.children[g.Parent] = append(s.children[g.Parent], g.ID)
	}
	return p
}
