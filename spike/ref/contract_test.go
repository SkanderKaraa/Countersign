package ref

import (
	"testing"
	"time"

	"countersign/spike/engine"
)

// Contract tests call the engine directly with crafted facts, to show each
// fixed rule holds in the engine itself and not only because the layer's
// facts happen to block it. They run on every engine.

func TestContract_AgentNeverApprovesEvenIfFactsSayAdmin(t *testing.T) {
	e := testEngine(t)
	got := e.MayApprove(engine.ApproveInput{
		Approver: "agent:api-refactor", Requester: "agent:db-migrate", Action: "read", Resource: "prod/x",
		ApproverIsAdmin: true, AuthorityScopes: []string{"prod/**"}, Class: "human-only",
	})
	wantRefusedBy(t, got, "approve.forbid-agent")
}

func TestContract_SelfApprovalRefusedEvenWithAuthority(t *testing.T) {
	e := testEngine(t)
	got := e.MayApprove(engine.ApproveInput{
		Approver: "person:erin", Requester: "person:erin", Action: "read", Resource: "prod/x",
		ApproverIsAdmin: true, AuthorityScopes: []string{"prod/**"}, Class: "human-only",
	})
	wantRefusedBy(t, got, "approve.forbid-self")
}

func TestContract_AgentNeverSetsMarkingEvenIfFactsSayAdmin(t *testing.T) {
	e := testEngine(t)
	got := e.MaySetMarking(engine.MarkingInput{SetBy: "agent:x", SetByIsAdmin: true, Resource: "repo/a", Class: "contained"})
	wantRefusedBy(t, got, "set_marking.forbid-agent")
}

func TestContract_FloorNeverContainedEvenForAdmin(t *testing.T) {
	e := testEngine(t)
	got := e.MaySetMarking(engine.MarkingInput{SetBy: "person:ada", SetByIsAdmin: true, Resource: "prod/a", Class: "contained", IsFloor: true})
	wantRefusedBy(t, got, "set_marking.forbid-floor-contained")
}

func TestContract_AgentServiceGrantNeedsAuthorisationAndHolding(t *testing.T) {
	e := testEngine(t)
	wantRefusedBy(t, e.MayCreateServiceGrant(engine.ServiceGrantInput{Creator: "agent:a", CreatorIsAdmin: true, CreatorAuthorised: false, CreatorHolds: true, Owner: "team:x", Resource: "repo/a"}), "default-deny")
	wantRefusedBy(t, e.MayCreateServiceGrant(engine.ServiceGrantInput{Creator: "agent:a", CreatorIsAdmin: true, CreatorAuthorised: true, CreatorHolds: false, Owner: "team:x", Resource: "repo/a"}), "default-deny")
	wantRefusedBy(t, e.MayCreateServiceGrant(engine.ServiceGrantInput{Creator: "person:ada", CreatorIsAdmin: true, Owner: "person:ada", Resource: "repo/a"}), "create_service_grant.forbid-owner-is-creator")
}

func TestContract_ChainTooDeepNeverAllows(t *testing.T) {
	e := testEngine(t)
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	node := func(id string) engine.Node {
		return engine.Node{ID: id, Principal: "agent:a", Action: "push", Scope: "repo/**", EndsAt: now.Add(time.Hour)}
	}
	got := e.Check(engine.CheckInput{
		Principal: "agent:a", Action: "push", Resource: "repo/x", Now: now, MaxDepth: 3,
		Chain: []engine.Node{node("g5"), node("g4"), node("g3"), node("g2"), node("g1")},
	})
	wantRefusedBy(t, got, "check.forbid-too-deep")
}

// B7. The engine checks authority for this action on this resource itself,
// from the approver's authority scopes, and refuses an unknown class.
func TestContract_B7_EngineChecksAuthorityScopeAndClass(t *testing.T) {
	e := testEngine(t)
	base := engine.ApproveInput{
		Approver: "person:erin", Requester: "agent:db-migrate", Action: "read",
		AuthorityScopes: []string{"prod/**"}, Class: "human-only",
	}
	in := base
	in.Resource = "prod/secrets/orders"
	if got := e.MayApprove(in); !got.Allow {
		t.Fatalf("authority covering the resource must allow, got %+v", got)
	}
	in.Resource = "repo/x"
	if got := e.MayApprove(in); got.Allow {
		t.Fatal("authority that doesn't cover the resource must refuse")
	}
	in = base
	in.Resource, in.Class = "prod/secrets/orders", "bogus"
	if got := e.MayApprove(in); got.Allow {
		t.Fatal("an unknown class must refuse an approval")
	}
	if got := e.MaySetMarking(engine.MarkingInput{SetBy: "person:ada", SetByIsAdmin: true, Resource: "repo/a", Class: "bogus"}); got.Allow {
		t.Fatal("an unknown class must refuse a marking")
	}
}

// wantRefusedBy asserts a refusal and names the rule that made it. Both
// engines use the same rule IDs (the @id annotations in policies.cedar), and
// "default-deny" when nothing permits.
func wantRefusedBy(t *testing.T, got engine.Result, id string) {
	t.Helper()
	if got.Allow {
		t.Fatalf("want a refusal by %s, got allow", id)
	}
	for _, p := range got.Policies {
		if p == id {
			return
		}
	}
	t.Fatalf("want a refusal by %s, got %v (%s)", id, got.Policies, got.Reason)
}

// C9. The permit side: an engine that refuses everything must fail these.

func TestContract_C9_PersonWithAuthorityApproves(t *testing.T) {
	got := testEngine(t).MayApprove(engine.ApproveInput{
		Approver: "person:erin", Requester: "agent:db-migrate", Action: "read",
		Resource: "prod/secrets/orders", AuthorityScopes: []string{"prod/**"}, Class: "human-only",
	})
	wantAllowedBy(t, got, "approve.authorised-person")
}

func TestContract_C9_AdministratorMarks(t *testing.T) {
	got := testEngine(t).MaySetMarking(engine.MarkingInput{SetBy: "person:ada", SetByIsAdmin: true, Resource: "repo/a", Class: "contained"})
	wantAllowedBy(t, got, "set_marking.administrator")
}

func TestContract_C9_AuthorisedAgentHoldingAccessProposesServiceGrant(t *testing.T) {
	got := testEngine(t).MayCreateServiceGrant(engine.ServiceGrantInput{
		Creator: "agent:a", CreatorAuthorised: true, CreatorHolds: true, Owner: "team:payments", Resource: "repo/a",
	})
	wantAllowedBy(t, got, "create_service_grant.authorised-agent")
}

func TestContract_C9_AdministratorProposesServiceGrant(t *testing.T) {
	got := testEngine(t).MayCreateServiceGrant(engine.ServiceGrantInput{
		Creator: "person:ada", CreatorIsAdmin: true, Owner: "team:payments", Resource: "repo/a",
	})
	wantAllowedBy(t, got, "create_service_grant.administrator")
}

func TestContract_C9_LiveChainAllowsCheck(t *testing.T) {
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	child := engine.Node{ID: "g2", Principal: "agent:h", Action: "push", Scope: "repo/api/**", EndsAt: now.Add(time.Hour)}
	root := engine.Node{ID: "g1", Principal: "agent:a", Action: "push", Scope: "repo/**", EndsAt: now.Add(2 * time.Hour)}
	got := testEngine(t).Check(engine.CheckInput{Principal: "agent:h", Action: "push", Resource: "repo/api/x", Now: now, MaxDepth: 3, Chain: []engine.Node{child, root}})
	wantAllowedBy(t, got, "check.grant-allows")
}

func wantAllowedBy(t *testing.T, got engine.Result, id string) {
	t.Helper()
	if !got.Allow {
		t.Fatalf("want an allow by %s, got a refusal: %s %v", id, got.Reason, got.Policies)
	}
	for _, p := range got.Policies {
		if p == id {
			return
		}
	}
	t.Fatalf("want an allow by %s, got %v", id, got.Policies)
}

// C10. Hand-down refusals, and chains with a withdrawn, ended or closed grant.
// An engine that allows everything must fail these.

func TestContract_C10_HandDownRefusals(t *testing.T) {
	e := testEngine(t)
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	ok := engine.HandDownInput{
		Holder: "agent:a", ParentPrincipal: "agent:a", ParentScope: "repo/api/**", ParentEndsAt: now.Add(time.Hour),
		ParentDepth: 2, ParentKind: "task", Class: "human-only", ChildScope: "repo/api/x", ChildEndsAt: now.Add(30 * time.Minute),
		ChildDepth: 1, ChainLen: 1, MaxDepth: 3,
	}
	wantAllowedBy(t, e.MayHandDown(ok), "hand_down.within-parent")
	cases := map[string]struct {
		edit func(*engine.HandDownInput)
		rule string
	}{
		"wider":                   {func(in *engine.HandDownInput) { in.ChildScope = "repo/**" }, "default-deny"},
		"longer":                  {func(in *engine.HandDownInput) { in.ChildEndsAt = in.ParentEndsAt.Add(time.Second) }, "default-deny"},
		"unknown class":           {func(in *engine.HandDownInput) { in.Class = "future" }, "hand_down.forbid-unknown-class"},
		"human-only at depth 0":   {func(in *engine.HandDownInput) { in.ParentDepth = 0 }, "default-deny"},
		"child depth not below":   {func(in *engine.HandDownInput) { in.ChildDepth = 2 }, "default-deny"},
		"contained without limit": {func(in *engine.HandDownInput) { in.Class = "contained" }, "default-deny"},
		"contained beyond limit": {func(in *engine.HandDownInput) {
			in.Class, in.HasSharingLimit, in.SharingLimit, in.ChainLen = "contained", true, 1, 2
		}, "default-deny"},
		"not the holder":           {func(in *engine.HandDownInput) { in.Holder = "agent:b" }, "default-deny"},
		"beyond the maximum depth": {func(in *engine.HandDownInput) { in.ChainLen = 4 }, "default-deny"},
	}
	for name, c := range cases {
		in := ok
		c.edit(&in)
		t.Run(name, func(t *testing.T) { wantRefusedBy(t, e.MayHandDown(in), c.rule) })
	}
}

func TestContract_C10_BrokenChainsRefuse(t *testing.T) {
	e := testEngine(t)
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	child := engine.Node{ID: "g2", Principal: "agent:h", Action: "push", Scope: "repo/api/**", EndsAt: now.Add(time.Hour), Task: "ENG-88"}
	root := engine.Node{ID: "g1", Principal: "agent:a", Action: "push", Scope: "repo/**", EndsAt: now.Add(2 * time.Hour), Task: "ENG-88"}
	cases := map[string]func(c, r *engine.Node){
		"root withdrawn":   func(c, r *engine.Node) { r.Withdrawn, r.WithdrawnBy, r.WithdrawnAt = true, "person:erin", now },
		"root ended":       func(c, r *engine.Node) { r.EndsAt = now },
		"root task closed": func(c, r *engine.Node) { r.TaskClosed = true },
		"root agent ended": func(c, r *engine.Node) { r.AgentEnded = true },
		"child withdrawn":  func(c, r *engine.Node) { c.Withdrawn, c.WithdrawnBy, c.WithdrawnAt = true, "person:erin", now },
		"root narrowed":    func(c, r *engine.Node) { r.Scope = "repo/web/**" },
	}
	for name, edit := range cases {
		c, r := child, root
		edit(&c, &r)
		t.Run(name, func(t *testing.T) {
			got := e.Check(engine.CheckInput{Principal: "agent:h", Action: "push", Resource: "repo/api/x", Now: now, MaxDepth: 3, Chain: []engine.Node{c, r}})
			wantRefusedBy(t, got, "default-deny")
		})
	}
}

func TestContract_D11_ServiceGrantNeverHandedDown(t *testing.T) {
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	got := testEngine(t).MayHandDown(engine.HandDownInput{
		Holder: "agent:a", ParentPrincipal: "agent:a", ParentScope: "repo/**", ParentEndsAt: now.Add(time.Hour),
		ParentDepth: 2, ParentKind: "service", Class: "human-only", ChildScope: "repo/x",
		ChildEndsAt: now.Add(time.Minute), ChildDepth: 1, ChainLen: 1, MaxDepth: 3,
	})
	wantRefusedBy(t, got, "hand_down.forbid-service-grant")
}
