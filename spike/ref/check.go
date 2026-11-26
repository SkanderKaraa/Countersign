package ref

import (
	"sort"
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
)

// Decision is the answer to a check: allow or refuse, never "ask"
// (glossary: Check). Chain lists the grant IDs from the matching grant up
// to its root, and RootApprover is whoever approved the root grant.
type Decision struct {
	Allow        bool
	Reason       string
	Chain        []string
	RootApprover string
	Policies     []string
}

func refuse(reason string) Decision { return Decision{Reason: reason} }

// Check reads existing grants and never changes state (FR-21). The time is
// an input, so a replay with the same state and time gives the same answer.
func (s *State) Check(p model.Identity, a model.Action, r model.Resource, now time.Time) Decision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.check(p, a, r, now)
}

// check is Check without the lock, for use inside writes. For each
// candidate grant, in log order, the layer gathers the chain's facts from
// the log (withdrawal, narrowing, signals) and the engine decides. It allows
// on the first grant the engine allows; if none, the reason comes from the
// most recent grant tried, so the text doesn't depend on storage order.
func (s *State) check(p model.Identity, a model.Action, r model.Resource, now time.Time) Decision {
	if p.IsZero() || r.IsZero() || a == "" {
		return refuse("malformed input")
	}
	candidates := make([]*model.Grant, 0, len(s.byPrincipal[p]))
	for _, g := range s.byPrincipal[p] {
		if g.Action == a {
			candidates = append(candidates, g)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].LogPosition < candidates[j].LogPosition })
	var last string
	for _, g := range candidates {
		if s.prefilter && !s.effectiveResource(g).Covers(r) {
			continue // its own scope can't cover r, so every engine would refuse it
		}
		chain, ok := s.chainOf(g)
		if !ok {
			last = "broken chain above grant " + g.ID
			continue
		}
		res := s.engine.Check(engine.CheckInput{
			Principal: p.String(), Action: string(a), Resource: r.String(),
			Now: now, Chain: chain, MaxDepth: s.limits.SystemMaxDepth,
		})
		if res.Allow {
			ids := make([]string, len(chain))
			for i, n := range chain {
				ids[i] = n.ID
			}
			root := s.grants[chain[len(chain)-1].ID]
			return Decision{Allow: true, Reason: "allowed by grant " + g.ID, Chain: ids, RootApprover: root.Approver.String(), Policies: res.Policies}
		}
		last = res.Reason
	}
	if last != "" {
		return refuse(last)
	}
	return refuse("no grant; make a request")
}

// checkGrant asks the engine about one specific grant's chain, not about
// any grant the principal holds.
func (s *State) checkGrant(g *model.Grant, r model.Resource, now time.Time) engine.Result {
	chain, ok := s.chainOf(g)
	if !ok {
		return engine.Result{Reason: "broken chain above grant " + g.ID}
	}
	return s.engine.Check(engine.CheckInput{
		Principal: g.Principal.String(), Action: string(g.Action), Resource: r.String(),
		Now: now, Chain: chain, MaxDepth: s.limits.SystemMaxDepth,
	})
}

// chainOf builds the facts for g and its ancestors, up to one step beyond
// the system maximum depth so the engine can see a chain that's too long.
func (s *State) chainOf(g *model.Grant) ([]engine.Node, bool) {
	var chain []engine.Node
	for n := g; ; {
		chain = append(chain, s.nodeOf(n))
		if n.Parent == "" || len(chain) > s.limits.SystemMaxDepth+1 {
			return chain, true
		}
		parent, ok := s.grants[n.Parent]
		if !ok {
			return nil, false
		}
		n = parent
	}
}

func (s *State) nodeOf(g *model.Grant) engine.Node {
	n := engine.Node{
		ID: g.ID, Principal: g.Principal.String(), Action: string(g.Action),
		Scope: s.effectiveResource(g).String(), EndsAt: s.effectiveEnd(g), Task: g.Task,
	}
	if w, ok := s.withdrawn[g.ID]; ok {
		n.Withdrawn, n.WithdrawnBy, n.WithdrawnAt = true, w.by.String(), w.at
	}
	if pos, ok := s.closedTasks[g.Task]; ok && g.Kind == model.KindTask && g.LogPosition < pos {
		n.TaskClosed = true
	}
	if pos, ok := s.endedAgents[g.Principal]; ok && g.LogPosition < pos {
		n.AgentEnded = true
	}
	return n
}

func (s *State) effectiveEnd(g *model.Grant) time.Time {
	if nw, ok := s.narrowed[g.ID]; ok && !nw.endsAt.IsZero() {
		return nw.endsAt
	}
	return g.EndsAt
}

func (s *State) effectiveResource(g *model.Grant) model.Resource {
	if nw, ok := s.narrowed[g.ID]; ok && !nw.resource.IsZero() {
		return nw.resource
	}
	return g.Resource
}

// ListCheck answers "which of these items may p act on?" (FR-22). It gathers
// every candidate grant's chain once and asks the engine for the whole list.
// A refusal of the list as a whole refuses every item (all or nothing).
func (s *State) ListCheck(p model.Identity, a model.Action, items []model.Resource, now time.Time) ([]bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	in := engine.ListInput{Principal: p.String(), Action: string(a), Now: now, MaxDepth: s.limits.SystemMaxDepth}
	for _, it := range items {
		in.Items = append(in.Items, it.String())
	}
	for _, g := range s.byPrincipal[p] {
		if g.Action != a {
			continue
		}
		if chain, ok := s.chainOf(g); ok {
			in.Chains = append(in.Chains, chain)
		}
	}
	res := s.engine.(engine.Lister).List(in)
	if res.Reason != "" || len(res.Allowed) != len(items) {
		return make([]bool, len(items)), res.Method
	}
	return res.Allowed, res.Method
}
