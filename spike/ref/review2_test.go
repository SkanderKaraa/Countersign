package ref

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"countersign/spike/model"
)

// Tests for Skander's second review round of part 1. Each was written to
// fail before its fix.

// 1. Only administrators classify resources, and clearing a flag needs a
// second administrator.
func TestReview2_1_ClassificationIsGuarded(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.SetResourceInfo(f.erin, model.ResourceInfo{Resource: res("prod/**")}, t0); err == nil {
		t.Fatal("a non-administrator must not classify a resource")
	}
	if _, err := f.s.SetResourceInfo(f.ada, model.ResourceInfo{Resource: res("prod/**")}, t0); err != ErrNeedsSecond {
		t.Fatalf("clearing the production flag alone must only propose, got %v", err)
	}
}

// 2. Every classification that covers a resource counts, so the strictest
// wins.
func TestReview2_2_ClassificationsUnion(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.SetResourceInfo(f.ada, model.ResourceInfo{Resource: res("prod/api/**")}, t0); err != nil {
		t.Fatalf("a narrower classification that doesn't lower the floor should apply directly: %v", err)
	}
	if err := f.s.SetMarking(f.ada, model.ActionPush, res("prod/api/**"), model.ClassContained, t0); err == nil {
		t.Fatal("marking a production action contained must still be refused")
	}
	if c := f.s.ClassOf(model.ActionPush, res("prod/api/x")); c != model.ClassHumanOnly {
		t.Fatalf("a narrower classification with no flags must not beat a wider production one, got %s", c)
	}
}

// 3. Authority over a resource means administrators only (v1).
func TestReview2_3_ApprovalAuthorityIsNotAuthorityOverTheResource(t *testing.T) {
	f := newFixture(t)
	if err := f.s.SetMarking(f.erin, model.ActionRead, res("repo/docs"), model.ClassContained, t0); err == nil {
		t.Fatal("an approver who isn't an administrator must not set markings")
	}
	if err := f.s.GiveAuthority(f.erin, f.sam, []model.Action{model.ActionRead}, res("prod/**"), t0.Add(time.Hour), t0); err == nil {
		t.Fatal("an approver who isn't an administrator must not give approval authority")
	}
}

// 4. "Ask" lists administrators too, so it is never empty while an
// administrator is active, and an administrator can approve.
func TestReview2_4_AskIncludesAdministrators(t *testing.T) {
	f := newFixture(t)
	out := f.s.SubmitRequest(request("agent:db-migrate", "deploy", "prod/api", "1h"), t0)
	if !out.Ask || len(out.Approvers) == 0 {
		t.Fatalf("ask must list someone who could approve, got %+v", out)
	}
	if got := f.s.Approve("person:ben", out.RequestID, 0, t0); !got.Approved {
		t.Fatalf("an administrator should be able to approve, got %+v", got)
	}
}

// 5. Removing an administrator counts only active ones.
func TestReview2_5_RemoveCountsActiveAdministrators(t *testing.T) {
	f := newFixture(t)
	c := must(model.ParsePerson("person:cleo"))
	ok(t, f.s.AddAdministrator(f.ada, c, t0.Add(30*24*time.Hour), t0))
	// Later, Ben's authority has lapsed. Only Ada and Cleo are active.
	later := t0.Add(80 * 24 * time.Hour)
	if f.s.isAdmin(f.ben, later) {
		t.Skip("fixture changed: Ben is still active")
	}
	if err := f.s.RemoveAdministrator(f.ada, c, t0.Add(29*24*time.Hour)); err != nil {
		t.Fatalf("removing one of three active administrators should work: %v", err)
	}
	f2 := newFixture(t)
	ok(t, f2.s.AddAdministrator(f2.ada, c, t0.Add(30*24*time.Hour), t0))
	if err := f2.s.RemoveAdministrator(f2.ada, c, later); err == nil {
		t.Fatal("removing an administrator must not leave fewer than two active ones")
	}
}

// 6. The refusal reason doesn't depend on the order grants were stored in.
func TestReview2_6_RefusalReasonIsDeterministic(t *testing.T) {
	f := newFixture(t)
	a := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/a", "1h"), t0)
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/b", "2h"), t0)
	f.s.Withdraw(f.ada.Identity(), a.GrantID, t0)
	at := t0.Add(90 * time.Minute) // grant a is withdrawn, grant b has ended
	want := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/a"), at).Reason
	gs := f.s.byPrincipal[f.apiRefactor]
	for i := 0; i < 20; i++ {
		rand.Shuffle(len(gs), func(i, j int) { gs[i], gs[j] = gs[j], gs[i] })
		if got := f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/a"), at).Reason; got != want {
			t.Fatalf("reason changed with storage order: %q vs %q", got, want)
		}
	}
}

// 7. Checks and writes can run at the same time. Run with -race.
func TestReview2_7_ConcurrentChecksAndWrites(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/x", "1h"), t0)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			f.s.Check(f.apiRefactor, model.ActionPush, res("repo/api/x"), t0)
		}
	}()
	wg.Wait()
}
