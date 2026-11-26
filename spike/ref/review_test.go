package ref

import (
	"strings"
	"testing"
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
)

// Tests for the problems Skander found reviewing part 1. Each was written to
// fail before its fix.

// 1. A person creating a service grant needs authority over the resource.
func TestReview1_PersonCreatingServiceGrantNeedsAuthority(t *testing.T) {
	f := newFixture(t)
	out := f.s.CreateServiceGrant(f.sam.Identity(), f.helpr, model.ActionPush, res("repo/api/billing"), "team:payments", time.Hour, 30*time.Minute, t0)
	if out.Approved || out.Ask {
		t.Fatalf("a person without authority over the resource must be refused, got %+v", out)
	}
	out = f.s.CreateServiceGrant(f.ada.Identity(), f.helpr, model.ActionPush, res("repo/api/billing"), "team:payments", time.Hour, 30*time.Minute, t0)
	if !out.Ask {
		t.Fatalf("an administrator should be able to propose it for approval, got %+v", out)
	}
}

// 2. The approver sets the depth of a human-only grant; row 17 then works.
func TestReview2_ApproverSetsDepthAndHandDownWorks(t *testing.T) {
	f := newFixture(t)
	out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
	got := f.s.Approve("person:erin", out.RequestID, 1, t0)
	if !got.Approved {
		t.Fatalf("approve with depth 1 failed: %+v", got)
	}
	child := f.s.HandDown(f.dbMigrate, got.GrantID, f.helpr, res("prod/secrets/orders"), time.Hour, 0, t0)
	if !child.Approved {
		t.Fatalf("row 17: hand-down with allowed-to-share should succeed, got %+v", child)
	}
	if d := f.s.Check(f.helpr, model.ActionRead, res("prod/secrets/orders"), t0); !d.Allow {
		t.Fatalf("the child grant should allow within the parent's scope, got %+v", d)
	}
	// Row 18: the child has depth 0, so it can't hand down again.
	if out := f.s.HandDown(f.helpr, child.GrantID, must(model.ParseIdentity("agent:g1")), res("prod/secrets/orders"), time.Hour, 0, t0); out.Approved {
		t.Fatal("row 18: a depth-0 child grant must not be handed down again")
	}
	// Depth beyond the system maximum is refused.
	out = f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
	if got := f.s.Approve("person:erin", out.RequestID, 9, t0); got.Approved {
		t.Fatal("a depth above the system maximum must be refused")
	}
}

// 3. "Agent ended" ends only grants created before the signal.
func TestReview3_AgentEndedAppliesOnlyToEarlierGrants(t *testing.T) {
	f := newFixture(t)
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/a", "1h"), t0)
	ok(t, f.s.AcceptAgentEnded(must(model.ParseAgent("agent:api-refactor")), "product:houston", t0))
	if d := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/a"), t0); d.Allow {
		t.Fatal("a grant created before the signal must end")
	}
	later := t0.Add(time.Minute)
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/b", "1h"), later)
	if d := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/b"), later); !d.Allow {
		t.Fatalf("a grant created after the signal must not be killed by it, got %+v", d)
	}
}

// 4. The store assigns request IDs, and a pending request expires.
func TestReview4_StoreAssignsRequestIDsAndPendingExpires(t *testing.T) {
	f := newFixture(t)
	withID := []byte(`{"id":"mine","principal":"agent:db-migrate","action":"read","resource":"prod/secrets/orders","task":"ENG-88","duration":"2h"}`)
	if out := f.s.SubmitRequest(withID, t0); out.Ask || out.Approved {
		t.Fatalf("a caller-chosen request ID must be refused, got %+v", out)
	}
	a := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
	b := f.s.SubmitRequest(request("agent:helper", "read", "prod/secrets/orders", "2h"), t0)
	if a.RequestID == "" || a.RequestID == b.RequestID {
		t.Fatalf("each request needs its own store-assigned ID, got %q and %q", a.RequestID, b.RequestID)
	}
	tooLate := t0.Add(DefaultLimits().PendingMax + time.Second)
	if got := f.s.Approve("person:erin", a.RequestID, 0, tooLate); got.Approved {
		t.Fatal("an expired pending request must not be approvable")
	}
}

// 5. An unknown class refuses handing down.
func TestReview5_UnknownClassRefusesHandDown(t *testing.T) {
	f := newFixture(t)
	got := f.s.engine.MayHandDown(engine.HandDownInput{
		Holder: "agent:a", ParentPrincipal: "agent:a", ParentScope: "repo/**", ParentEndsAt: t0.Add(time.Hour),
		ParentDepth: 2, ParentKind: "task", Class: "future", ChildScope: "repo/a", ChildEndsAt: t0, ChainLen: 1, MaxDepth: 3,
		HasSharingLimit: true, SharingLimit: 3,
	})
	if got.Allow {
		t.Fatal("an unknown class must refuse")
	}
}

// Smaller points.

func TestReview_IdentityCharacterWhitelist(t *testing.T) {
	for _, bad := range []string{"person:erin\n", "person:érin", "person:Erin", "agent:a\tb", "person:a;b"} {
		if _, err := model.ParseIdentity(bad); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
}

func TestReview_AdministratorEndsIsLimited(t *testing.T) {
	f := newFixture(t)
	c := must(model.ParsePerson("person:cleo"))
	if err := f.s.AddAdministrator(f.ada, c, t0.Add(10*365*24*time.Hour), t0); err == nil {
		t.Fatal("an administrator's authority must have a hard maximum lifetime")
	}
}

func TestReview_WithdrawalKeepsFirstWhoAndWhen(t *testing.T) {
	f := newFixture(t)
	g := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/a", "1h"), t0)
	f.s.Withdraw(f.ada.Identity(), g.GrantID, t0)
	f.s.Withdraw(f.ben.Identity(), g.GrantID, t0.Add(time.Minute))
	d := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/a"), t0.Add(2*time.Minute))
	if !strings.Contains(d.Reason, "person:ada") {
		t.Fatalf("the first withdrawal must be kept, got %q", d.Reason)
	}
}

func TestReview_StrictestSharingLimitWins(t *testing.T) {
	f := newFixture(t)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 3, t0))
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/api/**"), 1, t0))
	if d, _ := f.s.sharingLimitFor(model.ActionPush, res("repo/api/x")); d != 1 {
		t.Fatalf("the strictest covering limit must apply, got %d", d)
	}
}

// 6. Every service grant needs a person's approval, contained or not.
func TestReview6_ServiceGrantAlwaysNeedsAPerson(t *testing.T) {
	f := newFixture(t)
	ok(t, f.s.GiveAuthority(f.ada, f.erin, []model.Action{model.ActionPush}, res("repo/**"), t0.Add(24*time.Hour), t0))
	out := f.s.CreateServiceGrant(f.ada.Identity(), f.helpr, model.ActionPush, res("repo/api/billing"), "team:payments", time.Hour, 30*time.Minute, t0)
	if out.Approved || !out.Ask {
		t.Fatalf("a contained service grant must still wait for a person, got %+v", out)
	}
	if got := f.s.Approve("person:erin", out.RequestID, 0, t0); !got.Approved {
		t.Fatalf("a person with approval authority should approve it, got %+v", got)
	}
}
