package engine

import (
	"time"

	"countersign/spike/model"
)

// Guard validates every input in one shared place before any engine runs, so
// engines only ever see well-formed facts and can't disagree on malformed
// ones: identities, paths, actions and classes go through the same model
// parsers, and anything malformed is refused. Grant times must be whole
// milliseconds (the layer stores them that way, and Cedar works in
// milliseconds); "now" is truncated down, which can never make an ended
// grant live again.
func Guard(e Engine) Engine { return guard{e} }

type guard struct{ inner Engine }

func (g guard) Name() string { return g.inner.Name() }

func invalid(what string) Result { return Result{Reason: "invalid input: " + what} }

func validIdentity(s string) bool { _, err := model.ParseIdentity(s); return err == nil }
func validResource(s string) bool { _, err := model.ParseResource(s); return err == nil }
func validAction(s string) bool   { _, err := model.ParseAction(s); return err == nil }
func validClass(s string) bool    { return model.ValidClass(model.Class(s)) }
func wholeMs(t time.Time) bool    { return t.Equal(t.Truncate(time.Millisecond)) }

func (g guard) Check(in CheckInput) Result {
	if !validIdentity(in.Principal) || !validAction(in.Action) || !validResource(in.Resource) {
		return invalid("check")
	}
	for _, n := range in.Chain {
		if !validIdentity(n.Principal) || !validAction(n.Action) || !validResource(n.Scope) || !wholeMs(n.EndsAt) {
			return invalid("grant " + n.ID)
		}
	}
	in.Now = in.Now.Truncate(time.Millisecond)
	return g.inner.Check(in)
}

func (g guard) MayApprove(in ApproveInput) Result {
	if !validIdentity(in.Approver) {
		return invalid("approver")
	}
	if !validIdentity(in.Requester) || !validClass(in.Class) || !validAction(in.Action) || !validResource(in.Resource) {
		return invalid("approval")
	}
	for _, sc := range in.AuthorityScopes {
		if !validResource(sc) {
			return invalid("authority scope")
		}
	}
	return g.inner.MayApprove(in)
}

func (g guard) MayHandDown(in HandDownInput) Result {
	if !validIdentity(in.Holder) || !validIdentity(in.ParentPrincipal) ||
		!validResource(in.ParentScope) || !validResource(in.ChildScope) ||
		!wholeMs(in.ParentEndsAt) || !wholeMs(in.ChildEndsAt) || !validClass(in.Class) {
		return invalid("hand-down")
	}
	return g.inner.MayHandDown(in)
}

func (g guard) MaySetMarking(in MarkingInput) Result {
	if !validIdentity(in.SetBy) || !validResource(in.Resource) || !validClass(in.Class) {
		return invalid("marking")
	}
	return g.inner.MaySetMarking(in)
}

func (g guard) MayCreateServiceGrant(in ServiceGrantInput) Result {
	if !validIdentity(in.Creator) || !validResource(in.Resource) {
		return invalid("service grant")
	}
	return g.inner.MayCreateServiceGrant(in)
}

// List validates the list, then uses the engine's own list method if it has
// one, or asks it one item at a time.
func (g guard) List(in ListInput) ListResult {
	if !validIdentity(in.Principal) || !validAction(in.Action) {
		return ListResult{Reason: "invalid input: list"}
	}
	for _, it := range in.Items {
		if !validResource(it) {
			return ListResult{Reason: "invalid input: list item"}
		}
	}
	for _, chain := range in.Chains {
		for _, n := range chain {
			if !validIdentity(n.Principal) || !validAction(n.Action) || !validResource(n.Scope) || !wholeMs(n.EndsAt) {
				return ListResult{Reason: "invalid input: grant " + n.ID}
			}
		}
	}
	in.Now = in.Now.Truncate(time.Millisecond)
	if l, ok := g.inner.(Lister); ok {
		return l.List(in)
	}
	out := ListResult{Allowed: make([]bool, len(in.Items)), Method: "one check per item"}
	for i, it := range in.Items {
		for _, chain := range in.Chains {
			if g.inner.Check(CheckInput{Principal: in.Principal, Action: in.Action, Resource: it, Now: in.Now, Chain: chain, MaxDepth: in.MaxDepth}).Allow {
				out.Allowed[i] = true
				break
			}
		}
	}
	return out
}
