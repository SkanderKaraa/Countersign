package ref

import (
	"testing"
	"time"

	"countersign/spike/cedarengine"
	"countersign/spike/engine"
	"countersign/spike/model"
)

// Tests for Skander's part 2 review. Each was written to fail before its fix.

// A1. Classifying an unclassified resource with no flags lowers the floor of
// four (unclassified counts as production), so one administrator can't do it
// alone.
func TestReview3_A1_FirstClassificationLoweringTheFloorNeedsTwo(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.SetResourceInfo(f.ada, model.ResourceInfo{Resource: res("lab/**")}, t0); err == nil {
		t.Fatal("one administrator alone must not lower the floor for an unclassified resource")
	}
	if c := f.s.ClassOf(model.ActionPush, res("lab/x")); c != model.ClassHumanOnly {
		t.Fatalf("lab/x must still be in the floor, got %s", c)
	}
}

// A2. The second administrator consents through their own call: the proposer
// can't confirm their own proposal, a non-administrator can't confirm, and
// nothing changes until a second administrator does.
func TestReview3_A2_SecondAdministratorActsThroughOwnCall(t *testing.T) {
	f := newFixture(t)
	id, err := f.s.SetResourceInfo(f.ada, model.ResourceInfo{Resource: res("lab/**")}, t0)
	if err != ErrNeedsSecond || id == "" {
		t.Fatalf("want a proposal, got %q, %v", id, err)
	}
	if err := f.s.ConfirmResourceInfo(f.ada, id, t0); err == nil {
		t.Fatal("the proposer must not confirm their own proposal")
	}
	if err := f.s.ConfirmResourceInfo(f.erin, id, t0); err == nil {
		t.Fatal("a non-administrator must not confirm")
	}
	if c := f.s.ClassOf(model.ActionPush, res("lab/x")); c != model.ClassHumanOnly {
		t.Fatal("nothing may change before a second administrator confirms")
	}
	ok(t, f.s.ConfirmResourceInfo(f.ben, id, t0))
	ok(t, f.s.SetMarking(f.ada, model.ActionPush, res("lab/**"), model.ClassContained, t0))
	if c := f.s.ClassOf(model.ActionPush, res("lab/x")); c != model.ClassContained {
		t.Fatalf("after confirmation and a marking, push on lab/x should be contained, got %s", c)
	}
}

// A3. Only the two known classes can be set, and only an explicit
// "contained" makes an action contained.
func TestReview3_A3_UnknownMarkingClassNeverContains(t *testing.T) {
	f := newFixture(t)
	if err := f.s.SetMarking(f.ada, model.ActionRead, res("repo/docs"), model.Class("bogus"), t0); err == nil {
		t.Fatal("a marking with an unknown class must be refused")
	}
	f.s.markings = append(f.s.markings, model.Marking{Resource: res("repo/docs"), Action: model.ActionRead, Class: model.Class("bogus")})
	if c := f.s.ClassOf(model.ActionRead, res("repo/docs")); c != model.ClassHumanOnly {
		t.Fatalf("an unknown class stored by mistake must not make an action contained, got %s", c)
	}
}

// A4. A contained marking can be removed, by an administrator only, as an
// audited event; afterwards the action is human-only again.
func TestReview3_A4_ContainedMarkingCanBeRemoved(t *testing.T) {
	f := newFixture(t)
	if err := f.s.RemoveMarking(f.erin, model.ActionPush, res("repo/**"), t0); err == nil {
		t.Fatal("a non-administrator must not remove a marking")
	}
	ok(t, f.s.RemoveMarking(f.ada, model.ActionPush, res("repo/**"), t0))
	if c := f.s.ClassOf(model.ActionPush, res("repo/api/x")); c != model.ClassHumanOnly {
		t.Fatalf("after removal, push on repo must be human-only again, got %s", c)
	}
	log := f.s.Log()
	if log[len(log)-1].Type != "marking_removed" {
		t.Fatalf("the removal must be an event in the log, got %q", log[len(log)-1].Type)
	}
}

// B6. Cedar's check policies write out exactly three ancestor levels, so the
// Cedar engine must refuse to load for any other maximum depth, and refuse a
// check that asks for one.
func TestReview3_B6_CedarRefusesAnotherMaxDepth(t *testing.T) {
	if _, err := cedarengine.New(4); err == nil {
		t.Fatal("loading Cedar with a maximum depth other than 3 must fail")
	}
	e, err := cedarengine.New(3)
	if err != nil {
		t.Fatal(err)
	}
	now := t0
	n := engine.Node{ID: "g1", Principal: "agent:a", Action: "push", Scope: "repo/**", EndsAt: now.Add(time.Hour)}
	if e.Check(engine.CheckInput{Principal: "agent:a", Action: "push", Resource: "repo/x", Now: now, MaxDepth: 4, Chain: []engine.Node{n}}).Allow {
		t.Fatal("a check asking for another maximum depth must be refused")
	}
}

// D11a. Handing down checks that this parent grant is valid, not that the
// principal holds some grant for the action.
func TestReview3_D11_HandDownValidatesThisParent(t *testing.T) {
	f := newFixture(t)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 2, t0))
	old := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/**", "1h"), t0)
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
	f.s.Withdraw(f.ada.Identity(), old.GrantID, t0)
	if out := f.s.HandDown(f.apiRefactor, old.GrantID, f.helpr, res("repo/api/x"), 30*time.Minute, 0, t0); out.Approved {
		t.Fatal("a withdrawn parent grant must not be handed down, even if the holder has another valid grant")
	}
}

// D11b. A service grant is never handed down (Skander, 2026-11-17): a helper
// that needs long-lived access gets its own service grant.
func TestReview3_D11_ServiceGrantIsNeverHandedDown(t *testing.T) {
	f := newFixture(t)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 2, t0))
	out := f.s.CreateServiceGrant(f.ada.Identity(), f.apiRefactor, model.ActionPush, res("repo/api/**"), "team:payments", time.Hour, 30*time.Minute, t0)
	sg := f.s.Approve("person:ben", out.RequestID, 0, t0)
	if !sg.Approved {
		t.Fatalf("service grant: %+v", sg)
	}
	if child := f.s.HandDown(f.apiRefactor, sg.GrantID, f.helpr, res("repo/api/x"), 30*time.Minute, 0, t0); child.Approved {
		t.Fatal("a service grant must never be handed down")
	}
}

// D12a. A closing signal for a task that was never opened is refused.
func TestReview3_D12_UnknownTaskClosedIsRefused(t *testing.T) {
	f := newFixture(t)
	if err := f.s.AcceptTaskClosed("NEVER-OPENED", "product:quornex", t0); err == nil {
		t.Fatal("closing a task that was never opened must be refused")
	}
}

// D12b. Closing a task ends the grants created before it, not forever: a
// task that is opened again can be used for new requests.
func TestReview3_D12_ReopenedTaskCanBeUsedAgain(t *testing.T) {
	f := newFixture(t)
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/a", "1h"), t0)
	ok(t, f.s.AcceptTaskClosed("ENG-88", "product:quornex", t0))
	if d := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/a"), t0); d.Allow {
		t.Fatal("a grant created before the task closed must end")
	}
	later := t0.Add(time.Minute)
	f.s.OpenTask("ENG-88")
	out := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/b", "1h"), later)
	if !out.Approved {
		t.Fatalf("a reopened task must accept new requests, got %+v", out)
	}
	if d := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/b"), later); !d.Allow {
		t.Fatalf("a grant created after the task reopened must not be ended by the old closure, got %+v", d)
	}
}

// D12c. Pending requests are purged when they expire and capped per principal.
func TestReview3_D12_PendingRequestsArePurgedAndCapped(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < DefaultLimits().PendingPerPrincipal; i++ {
		if out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "1h"), t0); !out.Ask {
			t.Fatalf("request %d should wait for approval, got %+v", i, out)
		}
	}
	if out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "1h"), t0); out.Ask {
		t.Fatal("one principal must not have more than the cap of open requests")
	}
	later := t0.Add(DefaultLimits().PendingMax + time.Second)
	if out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "1h"), later); !out.Ask {
		t.Fatalf("expired requests must be purged, freeing the cap, got %+v", out)
	}
	if n := len(f.s.pending); n != 1 {
		t.Fatalf("expired requests must be removed, %d remain", n)
	}
}

// D13a. Authority, authorisations and administrators can't end in the past.
func TestReview3_D13_EndDatesMustBeInTheFuture(t *testing.T) {
	f := newFixture(t)
	past := t0.Add(-time.Hour)
	if err := f.s.GiveAuthority(f.ada, f.sam, []model.Action{model.ActionRead}, res("prod/**"), past, t0); err == nil {
		t.Error("approval authority ending in the past must be refused")
	}
	if err := f.s.AuthoriseService(f.ada, must(model.ParseAgent("agent:a")), []model.Action{model.ActionPush}, res("repo/**"), t0, t0); err == nil {
		t.Error("a service authorisation ending now must be refused")
	}
	if err := f.s.AddAdministrator(f.ada, must(model.ParsePerson("person:cleo")), past, t0); err == nil {
		t.Error("an administrator ending in the past must be refused")
	}
}

// D13b. A service grant needs a principal and a positive duration.
func TestReview3_D13_ServiceGrantIsValidated(t *testing.T) {
	f := newFixture(t)
	if out := f.s.CreateServiceGrant(f.ada.Identity(), model.Identity{}, model.ActionPush, res("repo/a"), "team:x", time.Hour, time.Minute, t0); out.Ask || out.Approved {
		t.Error("a service grant with no principal must be refused")
	}
	if out := f.s.CreateServiceGrant(f.ada.Identity(), f.helpr, model.ActionPush, res("repo/a"), "team:x", -time.Hour, -2*time.Hour, t0); out.Ask || out.Approved {
		t.Error("a service grant with a negative duration must be refused")
	}
}
