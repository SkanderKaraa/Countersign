package ref

import (
	"testing"
	"time"

	"countersign/spike/cedarengine"
	"countersign/spike/engine"
	"countersign/spike/goref"
	"countersign/spike/opaengine"
)

// B5. The same crafted inputs, through the same shared guard, must get the
// same answer from every engine.
func TestReview3_B5_EnginesAgreeOnCraftedInputs(t *testing.T) {
	c, err := cedarengine.New(DefaultLimits().SystemMaxDepth)
	if err != nil {
		t.Fatal(err)
	}
	o, err := opaengine.New()
	if err != nil {
		t.Fatal(err)
	}
	engines := []engine.Engine{engine.Guard(goref.New()), engine.Guard(c), engine.Guard(o)}
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	node := func(scope string) engine.Node {
		return engine.Node{ID: "g1", Principal: "agent:a", Action: "push", Scope: scope, EndsAt: now.Add(time.Hour)}
	}
	checks := map[string]engine.CheckInput{
		"non-canonical resource": {Principal: "agent:a", Action: "push", Resource: "repo/../prod", Now: now, MaxDepth: 3, Chain: []engine.Node{node("repo/**")}},
		"non-canonical scope":    {Principal: "agent:a", Action: "push", Resource: "prod/x", Now: now, MaxDepth: 3, Chain: []engine.Node{node("repo/../**")}},
		"malformed principal":    {Principal: "agent:a:x", Action: "push", Resource: "repo/x", Now: now, MaxDepth: 3, Chain: []engine.Node{node("repo/**")}},
		"capitalised resource":   {Principal: "agent:a", Action: "push", Resource: "REPO/x", Now: now, MaxDepth: 3, Chain: []engine.Node{node("repo/**")}},
	}
	for name, in := range checks {
		agree(t, name, engines, func(e engine.Engine) bool { return e.Check(in).Allow })
	}
	approvals := map[string]engine.ApproveInput{
		"approver with extra colon": {Approver: "person:erin:x", Requester: "agent:db", Action: "read", Resource: "prod/x", ApproverIsAdmin: true, Class: "human-only"},
		"approver with uppercase":   {Approver: "person:Erin", Requester: "agent:db", Action: "read", Resource: "prod/x", ApproverIsAdmin: true, Class: "human-only"},
	}
	for name, in := range approvals {
		agree(t, name, engines, func(e engine.Engine) bool { return e.MayApprove(in).Allow })
	}
	hand := engine.HandDownInput{
		Holder: "agent:a", ParentPrincipal: "agent:a", ParentScope: "repo/**", ParentEndsAt: now.Add(time.Hour),
		ParentDepth: 2, ParentKind: "task", Class: "human-only", ChildScope: "repo/x", ChildDepth: 1, ChainLen: 1, MaxDepth: 3,
	}
	sub := hand
	sub.ChildEndsAt = hand.ParentEndsAt.Add(500 * time.Microsecond)
	agree(t, "child ends 0.5 ms after its parent", engines, func(e engine.Engine) bool { return e.MayHandDown(sub).Allow })
	bad := hand
	bad.ChildEndsAt = now
	bad.ChildScope = "repo/./x"
	agree(t, "non-canonical child scope", engines, func(e engine.Engine) bool { return e.MayHandDown(bad).Allow })
}

func agree(t *testing.T, name string, engines []engine.Engine, ask func(engine.Engine) bool) {
	t.Helper()
	first := ask(engines[0])
	for _, e := range engines[1:] {
		if got := ask(e); got != first {
			t.Errorf("%s: %s says %v, %s says %v", name, engines[0].Name(), first, e.Name(), got)
		}
	}
	if first {
		t.Errorf("%s: crafted input was allowed", name)
	}
}
