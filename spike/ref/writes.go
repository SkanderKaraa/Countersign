package ref

import (
	"fmt"
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
)

// Outcome is the answer to a request: approved, ask, or refuse (glossary:
// Request outcome).
type Outcome struct {
	Approved  bool
	Ask       bool
	GrantID   string
	RequestID string
	Approvers []model.PersonID
	Reason    string
}

func refused(format string, args ...any) Outcome {
	return Outcome{Reason: fmt.Sprintf(format, args...)}
}

// SubmitRequest takes the raw request, so a request that names an approver
// is refused by the strict parser before any rule runs (FR-1).
func (s *State) SubmitRequest(raw []byte, now time.Time) Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, err := model.ParseRequest(raw)
	if err != nil {
		return refused("%v", err)
	}
	p, err := model.ParseIdentity(req.Principal)
	if err != nil {
		return refused("%v", err)
	}
	a, err := model.ParseAction(req.Action)
	if err != nil {
		return refused("%v", err)
	}
	r, err := model.ParseResource(req.Resource)
	if err != nil {
		return refused("%v", err)
	}
	if !s.openTasks[req.Task] {
		return refused("task %q is not open", req.Task)
	}
	s.purgeExpired(now)
	if s.pendingFor(p) >= s.limits.PendingPerPrincipal {
		return refused("%s already has %d open requests", p, s.limits.PendingPerPrincipal)
	}
	d := time.Duration(req.Duration)
	if d >= s.limits.TaskMax {
		return refused("duration %v is over the hard maximum lifetime (under %v)", d, s.limits.TaskMax)
	}
	g := model.Grant{
		ID: s.newID("g"), Principal: p, Action: a, Resource: r,
		Kind: model.KindTask, Task: req.Task, IssuedAt: now, EndsAt: now.Add(d),
	}
	if s.classOf(a, r) == model.ClassContained {
		if d > s.limits.ContainedMax {
			return refused("duration %v is over the contained maximum (%v)", d, s.limits.ContainedMax)
		}
		g.Approver = model.ApprovedByRule("contained")
		s.addGrant(g, now, "grant_created", "rule:contained")
		return Outcome{Approved: true, GrantID: g.ID}
	}
	id := s.newID("req")
	s.pending[id] = pending{grant: g, requester: p, at: now}
	return Outcome{Ask: true, RequestID: id, Approvers: s.approversFor(a, r, p, now)}
}

// Approve takes the approver as text, the way it arrives, and passes it to
// the engine as received: an approval from an agent reaches the engine as an
// agent, and the engine must refuse it (FR-11). depth is the delegation
// depth the approver allows on this human-only grant (FR-8, FR-18); 0 means
// it can't be handed down.
func (s *State) Approve(approverText, requestID string, depth int, now time.Time) Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	pr, ok := s.pending[requestID]
	if !ok {
		return refused("no open request %q", requestID)
	}
	if now.Sub(pr.at) > s.limits.PendingMax {
		delete(s.pending, requestID)
		return refused("request %s has expired; make a new request", requestID)
	}
	g := pr.grant
	in := engine.ApproveInput{
		Approver: approverText, Requester: pr.requester.String(),
		Action: string(g.Action), Resource: g.Resource.String(), Class: string(s.classOf(g.Action, g.Resource)),
	}
	approver, perr := model.ParsePerson(approverText)
	if perr == nil {
		in.ApproverIsAdmin = s.isAdmin(approver, now)
		in.AuthorityScopes = s.authorityScopes(approver, g.Action, now)
	}
	if res := s.engine.MayApprove(in); !res.Allow {
		return refused("%s", res.Reason)
	}
	if perr != nil {
		// Defence in depth: the engine allowed something that isn't a person.
		return refused("only a person can approve")
	}
	if depth < 0 || depth > s.limits.SystemMaxDepth {
		return refused("depth must be between 0 and %d", s.limits.SystemMaxDepth)
	}
	g.Depth = depth
	delete(s.pending, requestID)
	g.Approver = model.ApprovedByPerson(approver)
	g.IssuedAt = now
	g.EndsAt = now.Add(g.EndsAt.Sub(pr.grant.IssuedAt))
	if g.Kind == model.KindService {
		s.addGrant(g, now, "service_grant_created", approver.String())
	} else {
		s.addGrant(g, now, "grant_created", approver.String())
	}
	return Outcome{Approved: true, GrantID: g.ID}
}

// HandDown creates a child grant from a parent grant (FR-14). The layer
// checks the parent still covers the child's scope now, then gathers the
// facts; the engine decides whether the child is narrower, shorter, and
// covered by explicit sharing authority (FR-8).
func (s *State) HandDown(by model.Identity, parentID string, child model.Identity, r model.Resource, d time.Duration, depth int, now time.Time) Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	parent, ok := s.grants[parentID]
	if !ok {
		return refused("no grant %q", parentID)
	}
	if res := s.checkGrant(parent, r, now); !res.Allow {
		return refused("grant %s doesn't cover %s now: %s", parentID, r, res.Reason)
	}
	class := s.classOf(parent.Action, r)
	limit, hasLimit := s.sharingLimitFor(parent.Action, r)
	ends := now.Add(d).Truncate(time.Millisecond)
	res := s.engine.MayHandDown(engine.HandDownInput{
		Holder: by.String(), ParentPrincipal: parent.Principal.String(),
		ParentScope: s.effectiveResource(parent).String(), ParentEndsAt: s.effectiveEnd(parent),
		ParentDepth: parent.Depth, ParentKind: string(parent.Kind), Class: string(class),
		ChildScope: r.String(), ChildEndsAt: ends, ChildDepth: depth,
		SharingLimit: limit, HasSharingLimit: hasLimit,
		ChainLen: s.chainLength(parent) + 1, MaxDepth: s.limits.SystemMaxDepth,
	})
	if !res.Allow {
		return refused("%s", res.Reason)
	}
	if class == model.ClassContained {
		depth = 0 // contained grants carry no sharing fields
	}
	g := model.Grant{
		ID: s.newID("g"), Principal: child, Action: parent.Action, Resource: r,
		Kind: parent.Kind, Task: parent.Task, IssuedAt: now, EndsAt: ends,
		Parent: parent.ID, HandedBy: by, Depth: depth,
	}
	s.addGrant(g, now, "handed_down", by.String())
	return Outcome{Approved: true, GrantID: g.ID}
}

// chainLength counts grants handed down above g (a root grant has 0).
func (s *State) chainLength(g *model.Grant) int {
	n := 0
	for g.Parent != "" {
		n++
		g = s.grants[g.Parent]
	}
	return n
}

// sharingLimitFor returns the strictest standing sharing limit that covers
// the action on the resource.
func (s *State) sharingLimitFor(a model.Action, r model.Resource) (int, bool) {
	limit, found := 0, false
	for _, sl := range s.sharing {
		if sl.Action == a && sl.Resource.Covers(r) && (!found || sl.Depth < limit) {
			limit, found = sl.Depth, true
		}
	}
	return limit, found
}

// mayChange reports who may narrow, end or withdraw a grant (FR-4): its
// holder, its approver, anyone with authority over the resource now, or the
// parent grant's holder.
func (s *State) mayChange(by model.Identity, g *model.Grant, now time.Time) bool {
	if by == g.Principal {
		return true
	}
	if p, err := model.ParsePerson(by.String()); err == nil {
		if g.Approver.Person() == p || s.hasAuthorityOver(p, g.Resource, now) {
			return true
		}
	}
	if g.Parent != "" {
		if parent, ok := s.grants[g.Parent]; ok && parent.Principal == by {
			return true
		}
	}
	return false
}

// Narrow records a narrowing event. The grant record stays unchanged.
// Nobody may widen or extend a grant, its approver included (FR-4).
func (s *State) Narrow(by model.Identity, grantID string, r model.Resource, ends time.Time, now time.Time) Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grantID]
	if !ok {
		return refused("no grant %q", grantID)
	}
	if !s.mayChange(by, g, now) {
		return refused("%s may not change grant %s", by, grantID)
	}
	cur := narrowing{resource: s.effectiveResource(g), endsAt: s.effectiveEnd(g)}
	next := cur
	if !r.IsZero() {
		if !cur.resource.Covers(r) {
			return refused("a grant can't be widened; make a new request")
		}
		next.resource = r
	}
	if !ends.IsZero() {
		ends = ends.Truncate(time.Millisecond)
		if ends.After(cur.endsAt) {
			return refused("a grant can't be extended; make a new request")
		}
		next.endsAt = ends
	}
	s.narrowed[grantID] = next
	s.append("narrowed", now, by.String(), grantID)
	return Outcome{Approved: true, GrantID: grantID}
}

// Withdraw ends a grant early. Its descendants stop working too, because
// every check walks the chain (FR-15, FR-19).
func (s *State) Withdraw(by model.Identity, grantID string, now time.Time) Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grantID]
	if !ok {
		return refused("no grant %q", grantID)
	}
	if !s.mayChange(by, g, now) {
		return refused("%s may not withdraw grant %s", by, grantID)
	}
	if _, done := s.withdrawn[grantID]; !done {
		s.withdrawn[grantID] = withdrawal{by: by, at: now} // the first who and when are kept
	}
	s.append("withdrawn", now, by.String(), grantID)
	return Outcome{Approved: true, GrantID: grantID}
}

// AcceptTaskClosed and AcceptAgentEnded record closing signals. They only
// ever reduce access, so they are accepted from any source, and the source
// is recorded (FR-20). Each records its log position, so it ends only the
// grants created before it: a task opened again, or a new grant for the same
// agent, isn't ended by an old signal. A task that was never opened is
// refused, and AcceptAgentEnded takes an AgentID, so a person can't be
// "ended".
func (s *State) AcceptTaskClosed(task, source string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, known := s.openTasks[task]; !known {
		return fmt.Errorf("%w: task %q was never opened", ErrRefused, task)
	}
	s.openTasks[task] = false
	s.closedTasks[task] = s.append("signal_accepted", now, source, "task_closed:"+task)
	return nil
}

func (s *State) AcceptAgentEnded(agent model.AgentID, source string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos := s.append("signal_accepted", now, source, "agent_ended:"+agent.String())
	s.endedAgents[agent.Identity()] = pos
	return nil
}

// SetResourceInfo classifies a resource. Only an administrator can. A change
// that lowers the floor of four (taking the resource out of it, or clearing a
// flag) doesn't take effect: it becomes a proposal, which a second, different
// administrator must confirm through their own call (ConfirmResourceInfo).
// Otherwise one administrator could remove actions from the floor without
// any marking being refused. It returns the proposal ID when one is needed.
func (s *State) SetResourceInfo(by model.PersonID, info model.ResourceInfo, now time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isAdmin(by, now) {
		return "", fmt.Errorf("%w: only an administrator can classify a resource", ErrNotPermitted)
	}
	info.SetBy = by
	if s.lowersFloor(info) {
		id := s.newID("class")
		s.proposals[id] = proposal{info: info, by: by, at: now}
		s.append("classification_proposed", now, by.String(), info.Resource.String())
		return id, ErrNeedsSecond
	}
	s.applyResourceInfo(info, by, now)
	return "", nil
}

// ConfirmResourceInfo is the second administrator's own call. The proposer
// can't confirm their own proposal, and a proposal expires like a pending
// request.
func (s *State) ConfirmResourceInfo(by model.PersonID, proposalID string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pr, ok := s.proposals[proposalID]
	if !ok {
		return fmt.Errorf("%w: no proposal %q", ErrRefused, proposalID)
	}
	if !s.isAdmin(by, now) {
		return fmt.Errorf("%w: only an administrator can confirm", ErrNotPermitted)
	}
	if by == pr.by {
		return fmt.Errorf("%w: the proposer can't confirm their own proposal", ErrNotPermitted)
	}
	if now.Sub(pr.at) > s.limits.PendingMax {
		delete(s.proposals, proposalID)
		return fmt.Errorf("%w: proposal %s has expired", ErrRefused, proposalID)
	}
	delete(s.proposals, proposalID)
	s.applyResourceInfo(pr.info, by, now)
	return nil
}

// lowersFloor reports whether info would take its resource out of the floor
// of four, or clear a flag on an existing classification.
func (s *State) lowersFloor(info model.ResourceInfo) bool {
	after := s.withInfo(info)
	if floorByInfo(s.resourceInfo, info.Resource) && !floorByInfo(after, info.Resource) {
		return true
	}
	for _, old := range s.resourceInfo {
		if old.Resource == info.Resource {
			return (old.Production && !info.Production) || (old.RealSecret && !info.RealSecret) || (old.Money && !info.Money)
		}
	}
	return false
}

func (s *State) withInfo(info model.ResourceInfo) []model.ResourceInfo {
	after := make([]model.ResourceInfo, 0, len(s.resourceInfo)+1)
	replaced := false
	for _, old := range s.resourceInfo {
		if old.Resource == info.Resource {
			after = append(after, info)
			replaced = true
			continue
		}
		after = append(after, old)
	}
	if !replaced {
		after = append(after, info)
	}
	return after
}

func (s *State) applyResourceInfo(info model.ResourceInfo, by model.PersonID, now time.Time) {
	s.resourceInfo = s.withInfo(info)
	s.append("resource_classified", now, by.String(), info.Resource.String())
}

// SetMarking takes a PersonID, so an agent can't set a marking (FR-6,
// NFR-5 rule 12), and the engine decides too: only an administrator, and a
// floor-of-four action can never be marked contained.
func (s *State) SetMarking(by model.PersonID, a model.Action, r model.Resource, c model.Class, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !model.ValidClass(c) {
		return fmt.Errorf("%w: unknown class %q", ErrRefused, c)
	}
	res := s.engine.MaySetMarking(engine.MarkingInput{
		SetBy: by.String(), SetByIsAdmin: s.isAdmin(by, now),
		Resource: r.String(), Class: string(c), IsFloor: s.isFloor(a, r),
	})
	if !res.Allow {
		return fmt.Errorf("%w: %s", ErrRefused, res.Reason)
	}
	pos := s.append("marking_set", now, by.String(), string(a)+" "+r.String()+" "+string(c))
	s.markings = append(s.markings, model.Marking{Resource: r, Action: a, Class: c, SetBy: by, LogPosition: pos})
	return nil
}

// GiveAuthority gives approval authority, never wider than the giver's own,
// and never past its hard maximum (FR-7).
func (s *State) GiveAuthority(by, holder model.PersonID, actions []model.Action, r model.Resource, ends, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasAuthorityOver(by, r, now) {
		return fmt.Errorf("%w: %s has no authority over %s", ErrNotPermitted, by, r)
	}
	if !ends.After(now) || ends.Sub(now) > s.limits.AuthorityMax {
		return fmt.Errorf("%w: approval authority must end after now and within its hard maximum", ErrRefused)
	}
	pos := s.append("authority_given", now, by.String(), holder.String()+" "+r.String())
	s.authorities = append(s.authorities, model.ApprovalAuthority{Holder: holder, Actions: actions, Resource: r, GivenBy: by, EndsAt: ends, LogPosition: pos})
	return nil
}

// SetSharingLimit sets the standing sharing limit for a contained action
// (FR-8, [ASSUMPTION]).
func (s *State) SetSharingLimit(by model.PersonID, a model.Action, r model.Resource, depth int, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasAuthorityOver(by, r, now) {
		return fmt.Errorf("%w: %s has no authority over %s", ErrNotPermitted, by, r)
	}
	if s.classOf(a, r) != model.ClassContained {
		return fmt.Errorf("%w: a sharing limit applies only to contained actions", ErrRefused)
	}
	if depth < 1 || depth > s.limits.SystemMaxDepth {
		return fmt.Errorf("%w: depth must be between 1 and %d", ErrRefused, s.limits.SystemMaxDepth)
	}
	pos := s.append("sharing_limit_set", now, by.String(), string(a)+" "+r.String())
	s.sharing = append(s.sharing, model.SharingLimit{Resource: r, Action: a, Depth: depth, SetBy: by, LogPosition: pos})
	return nil
}

// AuthoriseService lets an agent create service grants (FR-16).
func (s *State) AuthoriseService(by model.PersonID, agent model.AgentID, actions []model.Action, r model.Resource, ends, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasAuthorityOver(by, r, now) {
		return fmt.Errorf("%w: %s has no authority over %s", ErrNotPermitted, by, r)
	}
	if !ends.After(now) || ends.Sub(now) > s.limits.AuthorityMax {
		return fmt.Errorf("%w: the authorisation must end after now and within its hard maximum", ErrRefused)
	}
	pos := s.append("service_authorisation_given", now, by.String(), agent.String()+" "+r.String())
	s.serviceAuth = append(s.serviceAuth, model.ServiceAuthorisation{Agent: agent, Actions: actions, Resource: r, GivenBy: by, EndsAt: ends, LogPosition: pos})
	return nil
}

func (s *State) serviceAuthorised(agent model.Identity, a model.Action, r model.Resource, now time.Time) bool {
	for _, sa := range s.serviceAuth {
		if sa.Agent.Identity() != agent || !now.Before(sa.EndsAt) || !sa.Resource.Covers(r) {
			continue
		}
		for _, act := range sa.Actions {
			if act == a {
				return true
			}
		}
	}
	return false
}

// CreateServiceGrant proposes a grant owned by a team or role that outlives
// its creator (FR-16). The engine decides who may propose one: an
// administrator, or an agent with explicit authorisation that holds the
// access itself, so creating one never gives anything new. Every service
// grant then waits for a person's approval, even for a contained action:
// long-lived access always has a human decision behind it, and the 4 h
// contained cap doesn't apply.
func (s *State) CreateServiceGrant(by, principal model.Identity, a model.Action, r model.Resource, owner string, d, review time.Duration, now time.Time) Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	if principal.IsZero() {
		return refused("a service grant needs a principal")
	}
	if d <= 0 || d > s.limits.ServiceMax || review <= 0 || review > d {
		return refused("the lifetime or review date is outside the service-grant limits")
	}
	s.purgeExpired(now)
	if s.pendingFor(by) >= s.limits.PendingPerPrincipal {
		return refused("%s already has %d open requests", by, s.limits.PendingPerPrincipal)
	}
	in := engine.ServiceGrantInput{Creator: by.String(), Owner: owner, Resource: r.String()}
	if p, err := model.ParsePerson(by.String()); err == nil {
		in.CreatorIsAdmin = s.isAdmin(p, now)
	} else {
		in.CreatorAuthorised = s.serviceAuthorised(by, a, r, now)
		in.CreatorHolds = s.check(by, a, r, now).Allow
	}
	if res := s.engine.MayCreateServiceGrant(in); !res.Allow {
		return refused("%s", res.Reason)
	}
	g := model.Grant{
		ID: s.newID("g"), Principal: principal, Action: a, Resource: r,
		Kind: model.KindService, IssuedAt: now, EndsAt: now.Add(d),
		Owner: owner, CreatedBy: by, ReviewDate: now.Add(review),
	}
	id := s.newID("req")
	s.pending[id] = pending{grant: g, requester: by, at: now}
	return Outcome{Ask: true, RequestID: id, Approvers: s.approversFor(a, r, by, now)}
}

// AddAdministrator and RemoveAdministrator keep at least two administrators
// with staggered expiries (FR-9).
func (s *State) AddAdministrator(by, p model.PersonID, ends, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isAdmin(by, now) || by == p {
		return fmt.Errorf("%w: only another administrator can add one", ErrNotPermitted)
	}
	if !ends.After(now) || ends.Sub(now) > s.limits.AuthorityMax {
		return fmt.Errorf("%w: an administrator must end after now and within the hard maximum", ErrRefused)
	}
	for _, a := range s.admins {
		gap := a.EndsAt.Sub(ends)
		if gap < 0 {
			gap = -gap
		}
		if gap < s.limits.AdminStagger {
			return fmt.Errorf("%w: administrators can't lapse within %v of each other", ErrRefused, s.limits.AdminStagger)
		}
	}
	pos := s.append("administrator_added", now, by.String(), p.String())
	s.admins[p] = model.Administrator{Person: p, AddedBy: by.String(), EndsAt: ends, LogPosition: pos}
	return nil
}

func (s *State) RemoveAdministrator(by, p model.PersonID, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isAdmin(by, now) || by == p {
		return fmt.Errorf("%w: only another administrator can remove one", ErrNotPermitted)
	}
	active := 0
	for q := range s.admins {
		if s.isAdmin(q, now) {
			active++
		}
	}
	if s.isAdmin(p, now) {
		active--
	}
	if active < s.limits.AdminMin {
		return fmt.Errorf("%w: an installation needs at least %d administrators", ErrRefused, s.limits.AdminMin)
	}
	delete(s.admins, p)
	s.append("administrator_removed", now, by.String(), p.String())
	return nil
}

// RemoveMarking undoes a marking on exactly this action and resource. It is
// a human-only change: it takes a PersonID, and the engine must allow it as
// it would a human-only marking by an administrator (FR-6). It's recorded as
// a marking_removed event.
func (s *State) RemoveMarking(by model.PersonID, a model.Action, r model.Resource, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := s.engine.MaySetMarking(engine.MarkingInput{
		SetBy: by.String(), SetByIsAdmin: s.isAdmin(by, now),
		Resource: r.String(), Class: string(model.ClassHumanOnly), IsFloor: s.isFloor(a, r),
	})
	if !res.Allow {
		return fmt.Errorf("%w: %s", ErrRefused, res.Reason)
	}
	kept := s.markings[:0]
	removed := false
	for _, m := range s.markings {
		if m.Action == a && m.Resource == r {
			removed = true
			continue
		}
		kept = append(kept, m)
	}
	if !removed {
		return fmt.Errorf("%w: no marking for %s on %s", ErrRefused, a, r)
	}
	s.markings = kept
	s.append("marking_removed", now, by.String(), string(a)+" "+r.String())
	return nil
}

// purgeExpired removes pending requests and proposals older than PendingMax.
func (s *State) purgeExpired(now time.Time) {
	for id, pr := range s.pending {
		if now.Sub(pr.at) > s.limits.PendingMax {
			delete(s.pending, id)
		}
	}
	for id, pr := range s.proposals {
		if now.Sub(pr.at) > s.limits.PendingMax {
			delete(s.proposals, id)
		}
	}
}

// pendingFor counts p's open requests.
func (s *State) pendingFor(p model.Identity) int {
	n := 0
	for _, pr := range s.pending {
		if pr.requester == p {
			n++
		}
	}
	return n
}
