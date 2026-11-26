// Package goref is the own-Go reference engine: Countersign's rules written
// directly in Go. In the spike it is a reference, not a candidate
// (spike-preregistration.md).
package goref

import (
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
)

type Engine struct{}

func New() Engine { return Engine{} }

func (Engine) Name() string { return "goref" }

func allow(policy string) engine.Result {
	return engine.Result{Allow: true, Policies: []string{policy}}
}

// deny names the rule that refused, using the same IDs as policies.cedar,
// so tests can assert which rule decided on every engine. "default-deny"
// means nothing permitted it.
func deny(rule, reason string) engine.Result {
	return engine.Result{Reason: reason, Policies: []string{rule}}
}

// Check: every grant on the chain must still be live, and its current scope
// must cover the resource (NFR-5 rules 8 and 9).
func (Engine) Check(in engine.CheckInput) engine.Result {
	if len(in.Chain) == 0 {
		return deny("default-deny", "no grant; make a request")
	}
	if len(in.Chain)-1 > in.MaxDepth {
		return deny("check.forbid-too-deep", "chain longer than the system maximum depth")
	}
	first := in.Chain[0]
	if first.Principal != in.Principal || first.Action != in.Action {
		return deny("default-deny", "grant "+first.ID+" is for another principal or action")
	}
	r, err := model.ParseResource(in.Resource)
	if err != nil {
		return deny("default-deny", "malformed input")
	}
	for _, n := range in.Chain {
		if n.Withdrawn {
			return deny("default-deny", "withdrawn by "+n.WithdrawnBy+" at "+n.WithdrawnAt.UTC().Format(time.RFC3339))
		}
		if !in.Now.Before(n.EndsAt) {
			return deny("default-deny", "grant "+n.ID+" has ended")
		}
		if n.TaskClosed {
			return deny("default-deny", "task "+n.Task+" closed")
		}
		if n.AgentEnded {
			return deny("default-deny", "agent "+n.Principal+" ended")
		}
		scope, err := model.ParseResource(n.Scope)
		if err != nil || !scope.Covers(r) {
			return deny("default-deny", "outside the scope of grant "+n.ID)
		}
	}
	return allow("check.grant-allows")
}

// MayApprove: only a person can be parsed from the approver text, so an agent
// can't approve, however it is worded (rule 5). Nobody approves their own
// request (rule 6).
func (Engine) MayApprove(in engine.ApproveInput) engine.Result {
	if _, err := model.ParsePerson(in.Approver); err != nil {
		if id, err := model.ParseIdentity(in.Approver); err == nil && id.IsAgent() {
			return deny("approve.forbid-agent", "only a person can approve")
		}
		return deny("default-deny", "only a person can approve")
	}
	if in.Approver == in.Requester {
		return deny("approve.forbid-self", "nobody approves their own request")
	}
	if in.Class != string(model.ClassHumanOnly) && in.Class != string(model.ClassContained) {
		return deny("approve.forbid-unknown-class", "unknown action class "+in.Class)
	}
	if in.ApproverIsAdmin {
		return allow("approve.authorised-person")
	}
	r, err := model.ParseResource(in.Resource)
	if err != nil {
		return deny("default-deny", "malformed resource")
	}
	for _, sc := range in.AuthorityScopes {
		if scope, err := model.ParseResource(sc); err == nil && scope.Covers(r) {
			return allow("approve.authorised-person")
		}
	}
	return deny("default-deny", in.Approver+" has no approval authority for "+in.Action+" on "+in.Resource)
}

// MayHandDown: never wider or longer than the parent (rule 8), and explicit
// sharing authority for every grant (rule 11). An unknown class refuses.
func (Engine) MayHandDown(in engine.HandDownInput) engine.Result {
	if in.ParentKind != string(model.KindTask) {
		return deny("hand_down.forbid-service-grant", "a service grant is never handed down")
	}
	if in.Holder != in.ParentPrincipal {
		return deny("default-deny", "only the holder of a grant can hand it down")
	}
	parent, err1 := model.ParseResource(in.ParentScope)
	child, err2 := model.ParseResource(in.ChildScope)
	if err1 != nil || err2 != nil || !parent.Covers(child) {
		return deny("default-deny", "a child grant can't be wider than its parent")
	}
	if in.ChildEndsAt.After(in.ParentEndsAt) {
		return deny("default-deny", "a child grant can't end later than its parent")
	}
	if in.ChainLen > in.MaxDepth {
		return deny("default-deny", "beyond the system maximum depth")
	}
	switch in.Class {
	case string(model.ClassHumanOnly):
		if in.ParentDepth <= 0 {
			return deny("default-deny", "this grant is not allowed to be handed down")
		}
		if in.ChildDepth >= in.ParentDepth {
			return deny("default-deny", "a child grant's depth must be below its parent's")
		}
	case string(model.ClassContained):
		if !in.HasSharingLimit {
			return deny("default-deny", "no sharing limit allows handing this down")
		}
		if in.ChainLen > in.SharingLimit {
			return deny("default-deny", "beyond the sharing limit")
		}
	default:
		return deny("hand_down.forbid-unknown-class", "unknown action class "+in.Class)
	}
	return allow("hand_down.within-parent")
}

// MaySetMarking: only an administrator (rule 12), and never contained for a
// floor-of-four action (rule 4).
func (Engine) MaySetMarking(in engine.MarkingInput) engine.Result {
	if id, err := model.ParseIdentity(in.SetBy); err == nil && id.IsAgent() {
		return deny("set_marking.forbid-agent", "only an administrator can set a marking")
	}
	if _, err := model.ParsePerson(in.SetBy); err != nil || !in.SetByIsAdmin {
		return deny("default-deny", "only an administrator can set a marking")
	}
	if in.Class != string(model.ClassHumanOnly) && in.Class != string(model.ClassContained) {
		return deny("set_marking.forbid-unknown-class", "unknown action class "+in.Class)
	}
	if in.Class == string(model.ClassContained) && in.IsFloor {
		return deny("set_marking.forbid-floor-contained", "a floor-of-four action can't be marked contained")
	}
	return allow("set_marking.administrator")
}

// MayCreateServiceGrant: the owner is never the creator; a person needs to
// be an administrator; an agent needs explicit authorisation and must hold
// the access itself (rule 13).
func (Engine) MayCreateServiceGrant(in engine.ServiceGrantInput) engine.Result {
	if in.Owner == "" || in.Owner == in.Creator {
		return deny("create_service_grant.forbid-owner-is-creator", "a service grant needs an owner that is a team or role, never its creator")
	}
	id, err := model.ParseIdentity(in.Creator)
	if err != nil {
		return deny("default-deny", "malformed creator")
	}
	if id.IsAgent() {
		if !in.CreatorAuthorised {
			return deny("default-deny", in.Creator+" is not authorised to create service grants for this")
		}
		if !in.CreatorHolds {
			return deny("default-deny", in.Creator+" doesn't hold this access itself")
		}
		return allow("create_service_grant.authorised-agent")
	}
	if !in.CreatorIsAdmin {
		return deny("default-deny", in.Creator+" has no authority over "+in.Resource)
	}
	return allow("create_service_grant.administrator")
}
