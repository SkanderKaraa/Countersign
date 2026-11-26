package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Action is one entry from a closed list. An unknown action is refused.
type Action string

const (
	ActionRead   Action = "read"
	ActionPush   Action = "push"
	ActionDeploy Action = "deploy"
	ActionMerge  Action = "merge"
	ActionDelete Action = "delete"
	ActionPay    Action = "pay"
)

var knownActions = map[Action]bool{
	ActionRead: true, ActionPush: true, ActionDeploy: true,
	ActionMerge: true, ActionDelete: true, ActionPay: true,
}

var ErrUnknownAction = errors.New("unknown action")

func ParseAction(s string) (Action, error) {
	a := Action(s)
	if !knownActions[a] {
		return "", fmt.Errorf("%w: %q", ErrUnknownAction, s)
	}
	return a, nil
}

// Class is an action class (PRD §3). The floor of four is not a class of its
// own: an action in the floor is always ClassHumanOnly.
type Class string

const (
	ClassContained Class = "contained"
	ClassHumanOnly Class = "human-only"
)

// ValidClass reports whether c is one of the two known classes.
func ValidClass(c Class) bool { return c == ClassContained || c == ClassHumanOnly }

// GrantKind is "task" or "service" (FR-3).
type GrantKind string

const (
	KindTask    GrantKind = "task"
	KindService GrantKind = "service"
)

// Approver is either a person or the approval rule (FR-10). It can't hold an
// agent: there is no constructor that takes an AgentID.
type Approver struct {
	person PersonID
	rule   string
}

func ApprovedByPerson(p PersonID) Approver { return Approver{person: p} }
func ApprovedByRule(rule string) Approver  { return Approver{rule: rule} }

func (a Approver) IsZero() bool     { return a.person.IsZero() && a.rule == "" }
func (a Approver) Person() PersonID { return a.person }
func (a Approver) String() string {
	if a.rule != "" {
		return "rule:" + a.rule
	}
	if a.person.IsZero() {
		return ""
	}
	return a.person.String()
}

// Grant is immutable once created. Withdrawal, narrowing and expiry are
// events or computed facts, never fields (grant-schema.md).
type Grant struct {
	ID          string
	Principal   Identity
	Action      Action
	Resource    Resource
	Kind        GrantKind
	Task        string    // only for KindTask
	Approver    Approver  // empty on a child grant
	IssuedAt    time.Time // set by the store
	EndsAt      time.Time
	Parent      string   // empty for a root grant
	HandedBy    Identity // set when Parent is set
	Depth       int      // human-only grants only; 0 means no sharing
	Owner       string   // only for KindService
	CreatedBy   Identity // only for KindService
	ReviewDate  time.Time
	LogPosition uint64
}

// Request has no approver field: a request can't name its own approver
// (FR-1, NFR-5 rule 2). It has no ID either: the store assigns one, so one
// caller can't replace another's pending request. ParseRequest refuses any
// extra field.
type Request struct {
	Principal string   `json:"principal"`
	Action    string   `json:"action"`
	Resource  string   `json:"resource"`
	Task      string   `json:"task"`
	Duration  Duration `json:"duration"`
}

// Duration reads a Go duration string such as "2h".
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

var ErrBadRequest = errors.New("malformed request")

// ParseRequest parses strictly: unknown fields such as "approver" are an
// error, so a crafted request is refused before any rule runs.
func ParseRequest(b []byte) (Request, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var r Request
	if err := dec.Decode(&r); err != nil {
		return Request{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	if r.Principal == "" || r.Action == "" || r.Resource == "" || r.Task == "" || r.Duration <= 0 {
		return Request{}, fmt.Errorf("%w: missing field", ErrBadRequest)
	}
	return r, nil
}

// Marking sets an action's class on a resource (FR-6).
type Marking struct {
	Resource    Resource
	Action      Action
	Class       Class
	SetBy       PersonID
	LogPosition uint64
}

// ResourceInfo classifies a resource for the floor of four. It is set by a
// person with authority; an unclassified resource is treated as production
// (fail closed).
type ResourceInfo struct {
	Resource   Resource
	Production bool
	RealSecret bool
	Money      bool
	SetBy      PersonID
}

// ApprovalAuthority lets a person approve given actions on a resource (FR-7).
type ApprovalAuthority struct {
	Holder      PersonID
	Actions     []Action
	Resource    Resource
	GivenBy     PersonID
	EndsAt      time.Time
	LogPosition uint64
}

// SharingLimit is the standing sharing limit for a contained action
// (FR-8, [ASSUMPTION]).
type SharingLimit struct {
	Resource    Resource
	Action      Action
	Depth       int
	SetBy       PersonID
	LogPosition uint64
}

// ServiceAuthorisation lets an agent create service grants (FR-16).
type ServiceAuthorisation struct {
	Agent       AgentID
	Actions     []Action
	Resource    Resource
	GivenBy     PersonID
	EndsAt      time.Time
	LogPosition uint64
}

// Administrator is one of the installation's root authorities (FR-9).
type Administrator struct {
	Person      PersonID
	AddedBy     string // a person ID, or "installation" for the first ones
	EndsAt      time.Time
	LogPosition uint64
}
