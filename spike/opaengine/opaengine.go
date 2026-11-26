// Package opaengine runs Countersign's rules in OPA (Rego). The adapter
// turns the facts the layer gathered into a Rego input; the decision comes
// only from policies.rego.
package opaengine

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"countersign/spike/engine"
)

//go:embed policies.rego
var policyText string

type Engine struct {
	check, approve, handDown, setMarking, serviceGrant rego.PreparedEvalQuery
}

// New prepares one query per decision, once, so each check only evaluates.
func New() (*Engine, error) {
	prep := func(rule string) (rego.PreparedEvalQuery, error) {
		return rego.New(
			rego.Query("data.countersign."+rule),
			rego.Module("policies.rego", policyText),
		).PrepareForEval(context.Background())
	}
	e := &Engine{}
	for _, q := range []struct {
		rule string
		dst  *rego.PreparedEvalQuery
	}{
		{"check", &e.check}, {"approve", &e.approve}, {"hand_down", &e.handDown},
		{"set_marking", &e.setMarking}, {"create_service_grant", &e.serviceGrant},
	} {
		pq, err := prep(q.rule)
		if err != nil {
			return nil, fmt.Errorf("preparing %s: %w", q.rule, err)
		}
		*q.dst = pq
	}
	return e, nil
}

func (*Engine) Name() string { return "opa" }

func ms(t time.Time) int64 { return t.UnixMilli() }

// eval runs one prepared query. An error, or anything but one well-formed
// decision object, is a refusal (NFR-1).
func eval(pq rego.PreparedEvalQuery, input map[string]any) engine.Result {
	rs, err := pq.Eval(context.Background(), rego.EvalInput(input))
	if err != nil {
		return engine.Result{Reason: "policy error: " + err.Error()}
	}
	if len(rs) != 1 || len(rs[0].Expressions) != 1 {
		return engine.Result{Reason: "policy returned no decision", Policies: []string{"default-deny"}}
	}
	obj, ok := rs[0].Expressions[0].Value.(map[string]any)
	if !ok {
		return engine.Result{Reason: "policy returned a malformed decision"}
	}
	allow, _ := obj["allow"].(bool)
	rule, _ := obj["rule"].(string)
	if allow {
		return engine.Result{Allow: true, Policies: []string{rule}}
	}
	reason := "forbidden by " + rule
	if rule == "default-deny" {
		reason = "no policy permits this"
	}
	return engine.Result{Reason: reason, Policies: []string{rule}}
}

func (e *Engine) Check(in engine.CheckInput) engine.Result {
	if len(in.Chain) == 0 {
		return engine.Result{Reason: "no grant; make a request", Policies: []string{"default-deny"}}
	}
	chain := make([]any, len(in.Chain))
	for i, n := range in.Chain {
		chain[i] = map[string]any{
			"id": n.ID, "principal": n.Principal, "action": n.Action, "scope": n.Scope,
			"ends_at_ms": ms(n.EndsAt), "withdrawn": n.Withdrawn,
			"task_closed": n.TaskClosed, "agent_ended": n.AgentEnded,
		}
	}
	r := eval(e.check, map[string]any{
		"principal": in.Principal, "action": in.Action, "resource": in.Resource,
		"now_ms": ms(in.Now), "max_depth": in.MaxDepth, "chain": chain,
	})
	if !r.Allow && r.Reason == "no policy permits this" {
		r.Reason = explainRefusal(in)
	}
	return r
}

// explainRefusal says which fact stopped the chain (FR-21). It only
// explains: the policy has already refused.
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
	scopes := make([]any, len(in.AuthorityScopes))
	for i, s := range in.AuthorityScopes {
		scopes[i] = s
	}
	r := eval(e.approve, map[string]any{
		"approver": in.Approver, "requester": in.Requester, "action": in.Action,
		"resource": in.Resource, "authority_scopes": scopes,
		"approver_is_admin": in.ApproverIsAdmin, "class": in.Class,
	})
	switch {
	case r.Allow:
	case len(r.Policies) == 1 && r.Policies[0] == "approve.forbid-agent":
		r.Reason = "only a person can approve"
	case len(r.Policies) == 1 && r.Policies[0] == "approve.forbid-self":
		r.Reason = "nobody approves their own request"
	case len(r.Policies) == 1 && r.Policies[0] == "default-deny":
		r.Reason = in.Approver + " has no approval authority for this"
	}
	return r
}

func (e *Engine) MayHandDown(in engine.HandDownInput) engine.Result {
	return eval(e.handDown, map[string]any{
		"holder": in.Holder, "parent_principal": in.ParentPrincipal,
		"parent_scope": in.ParentScope, "parent_ends_at_ms": ms(in.ParentEndsAt),
		"parent_depth": in.ParentDepth, "parent_kind": in.ParentKind, "class": in.Class,
		"child_scope": in.ChildScope, "child_ends_at_ms": ms(in.ChildEndsAt), "child_depth": in.ChildDepth,
		"sharing_limit": in.SharingLimit, "has_sharing_limit": in.HasSharingLimit,
		"chain_len": in.ChainLen, "max_depth": in.MaxDepth,
	})
}

func (e *Engine) MaySetMarking(in engine.MarkingInput) engine.Result {
	return eval(e.setMarking, map[string]any{
		"set_by": in.SetBy, "set_by_is_admin": in.SetByIsAdmin,
		"resource": in.Resource, "class": in.Class, "is_floor": in.IsFloor,
	})
}

func (e *Engine) MayCreateServiceGrant(in engine.ServiceGrantInput) engine.Result {
	return eval(e.serviceGrant, map[string]any{
		"creator": in.Creator, "creator_is_admin": in.CreatorIsAdmin,
		"creator_authorised": in.CreatorAuthorised, "creator_holds": in.CreatorHolds,
		"owner": in.Owner, "resource": in.Resource,
	})
}

// List answers a list check by partial evaluation: the candidate chains are
// known data, the item is the unknown input. OPA evaluates the policy once
// for the list, leaving a residual query over input.resource, then evaluates
// only that residual for each item.
func (e *Engine) List(in engine.ListInput) engine.ListResult {
	chains := make([]any, len(in.Chains))
	for i, ch := range in.Chains {
		nodes := make([]any, len(ch))
		for j, n := range ch {
			nodes[j] = map[string]any{
				"principal": n.Principal, "action": n.Action, "scope": n.Scope,
				"ends_at_ms": ms(n.EndsAt), "withdrawn": n.Withdrawn,
				"task_closed": n.TaskClosed, "agent_ended": n.AgentEnded,
			}
		}
		chains[i] = nodes
	}
	store := inmem.NewFromObject(map[string]any{"listctx": map[string]any{
		"principal": in.Principal, "action": in.Action, "now_ms": ms(in.Now),
		"max_depth": in.MaxDepth, "chains": chains,
	}})
	ctx := context.Background()
	pr, err := rego.New(
		rego.Query("data.countersign.list_allows"),
		rego.Module("policies.rego", policyText),
		rego.Store(store),
	).PartialResult(ctx)
	if err != nil {
		return engine.ListResult{Reason: "partial evaluation failed: " + err.Error(), Method: "partial evaluation"}
	}
	pq, err := pr.Rego().PrepareForEval(ctx)
	if err != nil {
		return engine.ListResult{Reason: "preparing the residual failed: " + err.Error(), Method: "partial evaluation"}
	}
	out := engine.ListResult{Allowed: make([]bool, len(in.Items)), Method: "partial evaluation"}
	for i, it := range in.Items {
		rs, err := pq.Eval(ctx, rego.EvalInput(map[string]any{"resource": it}))
		if err != nil {
			return engine.ListResult{Reason: "evaluating the residual failed: " + err.Error(), Method: "partial evaluation"}
		}
		out.Allowed[i] = len(rs) == 1 && len(rs[0].Expressions) == 1 && rs[0].Expressions[0].Value == true
	}
	return out
}
