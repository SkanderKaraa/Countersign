// Package cedarengine runs Countersign's rules in cedar-go. The adapter turns
// the facts the layer gathered into Cedar entities and a request; the
// decision comes only from policies.cedar.
package cedarengine

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	cedar "github.com/cedar-policy/cedar-go"
	"github.com/cedar-policy/cedar-go/types"

	"countersign/spike/engine"
)

//go:embed policies.cedar
var policyText []byte

type Engine struct{ ps *cedar.PolicySet }

// policyMaxDepth is the number of ancestor levels written out in
// policies.cedar.
const policyMaxDepth = 3

// New loads policies.cedar. cedar-go numbers policies by position, so each
// policy is re-added under its @id annotation, and explanations name the
// rule ("approve.forbid-self") rather than "policy5".
//
// policies.cedar writes out exactly policyMaxDepth ancestor levels, because
// Cedar has no recursion, so the engine refuses to load for any other
// system maximum depth.
func New(maxDepth int) (*Engine, error) {
	if maxDepth != policyMaxDepth {
		return nil, fmt.Errorf("policies.cedar is written for a maximum depth of %d, not %d", policyMaxDepth, maxDepth)
	}
	parsed, err := cedar.NewPolicySetFromBytes("policies.cedar", policyText)
	if err != nil {
		return nil, fmt.Errorf("loading Cedar policies: %w", err)
	}
	ps := cedar.NewPolicySet()
	for pos, p := range parsed.All() {
		id, ok := p.Annotations()["id"]
		if !ok || id == "" {
			return nil, fmt.Errorf("policy %s has no @id", pos)
		}
		if !ps.Add(cedar.PolicyID(id), p) {
			return nil, fmt.Errorf("duplicate policy @id %q", id)
		}
	}
	return &Engine{ps: ps}, nil
}

func (*Engine) Name() string { return "cedar" }

func uid(typ, id string) types.EntityUID {
	return types.NewEntityUID(types.EntityType(typ), types.String(id))
}

func action(name string) types.EntityUID { return uid("Action", name) }

// principalUID maps "person:x" to Person::"x" and "agent:x" to Agent::"x".
// Anything else can't be expressed, and the adapter refuses it.
func principalUID(s string) (types.EntityUID, bool) {
	kind, id, ok := strings.Cut(s, ":")
	if !ok || id == "" {
		return types.EntityUID{}, false
	}
	switch kind {
	case "person":
		return uid("Person", id), true
	case "agent":
		return uid("Agent", id), true
	}
	return types.EntityUID{}, false
}

// addPath adds a Path entity whose parents are every wider subtree, so
// Cedar's `in` means "covered by".
func addPath(es types.EntityMap, path string) types.EntityUID {
	u := uid("Path", path)
	if _, ok := es[u]; ok {
		return u
	}
	prefix := strings.TrimSuffix(path, "/**")
	segs := strings.Split(prefix, "/")
	var parents []types.EntityUID
	start := len(segs)
	if strings.HasSuffix(path, "/**") {
		start = len(segs) - 1 // a subtree's parents are strictly wider subtrees
	}
	for i := start; i >= 1; i-- {
		parents = append(parents, uid("Path", strings.Join(segs[:i], "/")+"/**"))
	}
	es[u] = types.Entity{UID: u, Parents: types.NewEntityUIDSet(parents...)}
	return u
}

func ms(t time.Time) types.Long { return types.Long(t.UnixMilli()) }

func (e *Engine) decide(es types.EntityMap, req types.Request) engine.Result {
	dec, diag := cedar.Authorize(e.ps, es, req)
	var ids []string
	for _, r := range diag.Reasons {
		ids = append(ids, string(r.PolicyID))
	}
	if len(diag.Errors) > 0 {
		// An evaluation error never allows (NFR-1).
		return engine.Result{Reason: "policy error: " + diag.Errors[0].String(), Policies: ids}
	}
	if dec == types.Allow {
		return engine.Result{Allow: true, Policies: ids}
	}
	if len(ids) > 0 {
		return engine.Result{Reason: "forbidden by " + strings.Join(ids, ", "), Policies: ids}
	}
	return engine.Result{Reason: "no policy permits this", Policies: []string{"default-deny"}}
}

func (e *Engine) Check(in engine.CheckInput) engine.Result {
	if len(in.Chain) == 0 {
		return engine.Result{Reason: "no grant; make a request"}
	}
	if in.MaxDepth != policyMaxDepth {
		return engine.Result{Reason: fmt.Sprintf("the Cedar policies support a maximum depth of %d only", policyMaxDepth)}
	}
	p, ok := principalUID(in.Principal)
	if !ok {
		return engine.Result{Reason: "malformed input"}
	}
	es := types.EntityMap{}
	res := addPath(es, in.Resource)
	// Build the chain from the root down, so each grant can point at its parent.
	var parent types.EntityUID
	for i := len(in.Chain) - 1; i >= 0; i-- {
		n := in.Chain[i]
		np, ok := principalUID(n.Principal)
		if !ok {
			return engine.Result{Reason: "malformed grant " + n.ID}
		}
		attrs := types.RecordMap{
			"principal":   np,
			"action":      action(n.Action),
			"scope":       addPath(es, n.Scope),
			"ends_at":     ms(n.EndsAt),
			"withdrawn":   types.Boolean(n.Withdrawn),
			"task_closed": types.Boolean(n.TaskClosed),
			"agent_ended": types.Boolean(n.AgentEnded),
		}
		if i < len(in.Chain)-1 {
			attrs["parent"] = parent
		}
		g := uid("Grant", n.ID)
		es[g] = types.Entity{UID: g, Attributes: types.NewRecord(attrs)}
		parent = g
	}
	r := e.decide(es, types.Request{
		Principal: p, Action: action(in.Action), Resource: res,
		Context: types.NewRecord(types.RecordMap{"now": ms(in.Now), "grant": parent}),
	})
	if !r.Allow && r.Reason == "no policy permits this" {
		r.Reason = explainRefusal(in)
	}
	return r
}

// explainRefusal says which fact stopped the chain, so a refusal names its
// reason (FR-21). It only explains: Cedar has already refused.
func explainRefusal(in engine.CheckInput) string {
	for _, n := range in.Chain {
		switch {
		case n.Withdrawn:
			return "withdrawn by " + n.WithdrawnBy + " at " + n.WithdrawnAt.UTC().Format(time.RFC3339)
		case !in.Now.Before(n.EndsAt):
			return "grant " + n.ID + " has ended"
		case n.TaskClosed:
			return "task " + n.Task + " closed"
		case n.AgentEnded:
			return "agent " + n.Principal + " ended"
		}
	}
	return "outside the scope of a grant in the chain"
}

func (e *Engine) MayApprove(in engine.ApproveInput) engine.Result {
	p, ok := principalUID(in.Approver)
	if !ok {
		return engine.Result{Reason: "only a person can approve"}
	}
	rq, ok := principalUID(in.Requester)
	if !ok {
		return engine.Result{Reason: "malformed requester"}
	}
	es := types.EntityMap{}
	var scopes []types.Value
	for _, sc := range in.AuthorityScopes {
		scopes = append(scopes, addPath(es, sc))
	}
	reqUID := uid("Request", "pending")
	es[reqUID] = types.Entity{UID: reqUID, Attributes: types.NewRecord(types.RecordMap{
		"requester": rq,
		"target":    addPath(es, in.Resource),
		"action":    action(in.Action),
		"class":     types.String(in.Class),
	})}
	r := e.decide(es, types.Request{
		Principal: p, Action: action("approve"), Resource: reqUID,
		Context: types.NewRecord(types.RecordMap{
			"approver_is_admin": types.Boolean(in.ApproverIsAdmin),
			"authority_scopes":  types.NewSet(scopes...),
		}),
	})
	switch {
	case containsPolicy(r.Policies, "approve.forbid-agent"):
		r.Reason = "only a person can approve"
	case containsPolicy(r.Policies, "approve.forbid-self"):
		r.Reason = "nobody approves their own request"
	case !r.Allow:
		r.Reason = in.Approver + " has no approval authority for this"
	}
	return r
}

func (e *Engine) MayHandDown(in engine.HandDownInput) engine.Result {
	holder, ok1 := principalUID(in.Holder)
	pp, ok2 := principalUID(in.ParentPrincipal)
	if !ok1 || !ok2 {
		return engine.Result{Reason: "malformed identity"}
	}
	es := types.EntityMap{}
	g := uid("Grant", "parent")
	es[g] = types.Entity{UID: g, Attributes: types.NewRecord(types.RecordMap{
		"principal": pp,
		"scope":     addPath(es, in.ParentScope),
		"ends_at":   ms(in.ParentEndsAt),
		"depth":     types.Long(in.ParentDepth),
		"kind":      types.String(in.ParentKind),
	})}
	return e.decide(es, types.Request{
		Principal: holder, Action: action("hand_down"), Resource: g,
		Context: types.NewRecord(types.RecordMap{
			"child_scope":       addPath(es, in.ChildScope),
			"child_ends_at":     ms(in.ChildEndsAt),
			"child_depth":       types.Long(in.ChildDepth),
			"class":             types.String(in.Class),
			"has_sharing_limit": types.Boolean(in.HasSharingLimit),
			"sharing_limit":     types.Long(in.SharingLimit),
			"chain_len":         types.Long(in.ChainLen),
			"max_depth":         types.Long(in.MaxDepth),
		}),
	})
}

func (e *Engine) MaySetMarking(in engine.MarkingInput) engine.Result {
	p, ok := principalUID(in.SetBy)
	if !ok {
		return engine.Result{Reason: "only an administrator can set a marking"}
	}
	es := types.EntityMap{}
	return e.decide(es, types.Request{
		Principal: p, Action: action("set_marking"), Resource: addPath(es, in.Resource),
		Context: types.NewRecord(types.RecordMap{
			"set_by_is_admin": types.Boolean(in.SetByIsAdmin),
			"class":           types.String(in.Class),
			"is_floor":        types.Boolean(in.IsFloor),
		}),
	})
}

func (e *Engine) MayCreateServiceGrant(in engine.ServiceGrantInput) engine.Result {
	p, ok := principalUID(in.Creator)
	if !ok {
		return engine.Result{Reason: "malformed creator"}
	}
	es := types.EntityMap{}
	return e.decide(es, types.Request{
		Principal: p, Action: action("create_service_grant"), Resource: addPath(es, in.Resource),
		Context: types.NewRecord(types.RecordMap{
			"creator_is_admin":   types.Boolean(in.CreatorIsAdmin),
			"creator_authorised": types.Boolean(in.CreatorAuthorised),
			"creator_holds":      types.Boolean(in.CreatorHolds),
			"owner":              types.String(in.Owner),
			"owner_is_creator":   types.Boolean(in.Owner == in.Creator),
		}),
	})
}

func containsPolicy(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
