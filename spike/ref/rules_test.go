package ref

import (
	"strings"
	"testing"
	"time"

	"countersign/spike/model"
)

// One test per fixed rule in PRD NFR-5, numbered the same way. Rule 10
// (access that can't be enforced in OpenBao is not allowed) needs a real
// OpenBao, so it isn't tested here.

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

type fixture struct {
	s                *State
	ada, ben, erin   model.PersonID // ada and ben are administrators
	sam              model.PersonID
	apiRefactor      model.Identity
	dbMigrate, helpr model.Identity
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func res(s string) model.Resource { return must(model.ParseResource(s)) }

// newFixture builds an installation with:
//   - "repo/**" classified as non-production, and push on it marked contained
//   - "prod/**" classified as production (so everything there is in the floor)
//   - Erin holding approval authority for read on "prod/**"
//   - task ENG-88 open
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		ada:         must(model.ParsePerson("person:ada")),
		ben:         must(model.ParsePerson("person:ben")),
		erin:        must(model.ParsePerson("person:erin")),
		sam:         must(model.ParsePerson("person:sam")),
		apiRefactor: must(model.ParseIdentity("agent:api-refactor")),
		dbMigrate:   must(model.ParseIdentity("agent:db-migrate")),
		helpr:       must(model.ParseIdentity("agent:helper")),
	}
	f.s = must(NewState(DefaultLimits(), testEngine(t), t0, f.ada, f.ben))
	id, err := f.s.SetResourceInfo(f.ada, model.ResourceInfo{Resource: res("repo/**")}, t0)
	if err != ErrNeedsSecond {
		t.Fatalf("classifying repo/** as non-production should need a second administrator, got %v", err)
	}
	ok(t, f.s.ConfirmResourceInfo(f.ben, id, t0))
	_, err = f.s.SetResourceInfo(f.ada, model.ResourceInfo{Resource: res("prod/**"), Production: true}, t0)
	ok(t, err)
	ok(t, f.s.SetMarking(f.ada, model.ActionPush, res("repo/**"), model.ClassContained, t0))
	ok(t, f.s.GiveAuthority(f.ada, f.erin, []model.Action{model.ActionRead}, res("prod/**"), t0.Add(60*24*time.Hour), t0))
	f.s.OpenTask("ENG-88")
	return f
}

func ok(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func request(principal, action, resource, duration string) []byte {
	return []byte(`{"principal":"` + principal + `","action":"` + action +
		`","resource":"` + resource + `","task":"ENG-88","duration":"` + duration + `"}`)
}

// approvedGrant makes a human-only grant for db-migrate to read the orders
// secret, approved by Erin.
func (f *fixture) approvedGrant(t *testing.T) string {
	t.Helper()
	out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
	if !out.Ask {
		t.Fatalf("want ask, got %+v", out)
	}
	out = f.s.Approve("person:erin", out.RequestID, 0, t0)
	if !out.Approved {
		t.Fatalf("want approved, got %+v", out)
	}
	return out.GrantID
}

func TestRule1_DefaultDeny(t *testing.T) {
	f := newFixture(t)
	d := f.s.Check(f.dbMigrate, model.ActionRead, res("prod/secrets/orders"), t0)
	if d.Allow || d.Reason != "no grant; make a request" {
		t.Fatalf("no grant must refuse with a reason, got %+v", d)
	}
	if d := f.s.Check(model.Identity{}, model.ActionRead, res("prod/x"), t0); d.Allow {
		t.Fatal("a malformed check must refuse")
	}
}

func TestRule2_RequestNeverNamesItsApprover(t *testing.T) {
	f := newFixture(t)
	raw := []byte(`{"principal":"agent:db-migrate","action":"read","resource":"prod/secrets/orders","task":"ENG-88","duration":"2h","approver":"person:erin"}`)
	out := f.s.SubmitRequest(raw, t0)
	if out.Approved || out.Ask || !strings.Contains(out.Reason, "unknown field") {
		t.Fatalf("a request naming an approver must be refused, got %+v", out)
	}
}

func TestRule3_EveryGrantHasAHardMaximumLifetime(t *testing.T) {
	f := newFixture(t)
	out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "24h"), t0)
	if out.Approved || out.Ask || !strings.Contains(out.Reason, "hard maximum") {
		t.Fatalf("a request at or over the hard maximum must be refused, got %+v", out)
	}
	out = f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/feature-retry", "5h"), t0)
	if out.Approved || !strings.Contains(out.Reason, "contained maximum") {
		t.Fatalf("a contained request over 4 h must be refused, got %+v", out)
	}
}

func TestRule4_MarkingsNeverRemoveFromTheFloorOfFour(t *testing.T) {
	f := newFixture(t)
	err := f.s.SetMarking(f.ada, model.ActionRead, res("prod/secrets/orders"), model.ClassContained, t0)
	if err == nil {
		t.Fatal("marking a floor-of-four action contained must be refused")
	}
	if c := f.s.ClassOf(model.ActionDelete, res("repo/api")); c != model.ClassHumanOnly {
		t.Fatalf("deletion is always in the floor, got %s", c)
	}
	if c := f.s.ClassOf(model.ActionPush, res("unclassified/x")); c != model.ClassHumanOnly {
		t.Fatalf("an unclassified resource fails closed, got %s", c)
	}
}

func TestRule5_OnlyAnAuthorisedPersonApprovesHumanOnly(t *testing.T) {
	f := newFixture(t)
	out := f.s.SubmitRequest(request("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
	if !out.Ask {
		t.Fatalf("want ask, got %+v", out)
	}
	// An agent, however the approval is worded, can't be parsed as a person.
	for _, who := range []string{"agent:api-refactor", "agent:erin", "person", "person:erin:x", "PERSON:erin"} {
		if got := f.s.Approve(who, out.RequestID, 0, t0); got.Approved {
			t.Fatalf("%q must not be able to approve", who)
		}
	}
	// A person without approval authority is refused too.
	if got := f.s.Approve("person:sam", out.RequestID, 0, t0); got.Approved {
		t.Fatal("a person without approval authority must not approve")
	}
	if got := f.s.Approve("person:erin", out.RequestID, 0, t0); !got.Approved {
		t.Fatalf("Erin holds approval authority and must succeed, got %+v", got)
	}
}

func TestRule6_NobodyApprovesTheirOwnRequest(t *testing.T) {
	f := newFixture(t)
	out := f.s.SubmitRequest(request("person:erin", "read", "prod/secrets/orders", "2h"), t0)
	if !out.Ask {
		t.Fatalf("want ask, got %+v", out)
	}
	for _, p := range out.Approvers {
		if p == f.erin {
			t.Fatal("the requester must not be listed as a possible approver")
		}
	}
	if got := f.s.Approve("person:erin", out.RequestID, 0, t0); got.Approved || !strings.Contains(got.Reason, "own request") {
		t.Fatalf("self-approval must be refused, got %+v", got)
	}
}

func TestRule7_ApprovalIsAttachedToTheFinalAction(t *testing.T) {
	f := newFixture(t)
	// Ten contained pushes, all approved by rules.
	for i := 0; i < 10; i++ {
		out := f.s.SubmitRequest([]byte(`{"principal":"agent:helper","action":"push","resource":"repo/api/cfg-`+itoa(i)+`","task":"ENG-88","duration":"1h"}`), t0)
		if !out.Approved {
			t.Fatalf("contained push %d should be approved by rules, got %+v", i, out)
		}
	}
	// The final action, a merge into production, has no grant of its own.
	if d := f.s.Check(f.helpr, model.ActionMerge, res("prod/api"), t0); d.Allow {
		t.Fatal("no number of contained grants may allow the production merge")
	}
	out := f.s.SubmitRequest(request("agent:helper", "merge", "prod/api", "1h"), t0)
	if !out.Ask {
		t.Fatalf("the request for the final action must return ask, got %+v", out)
	}
}

func TestRule8_DelegatedAccessOnlyShrinks(t *testing.T) {
	f := newFixture(t)
	parent := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 2, t0))
	if out := f.s.HandDown(f.apiRefactor, parent.GrantID, f.helpr, res("repo/**"), time.Hour, 0, t0); out.Approved {
		t.Fatal("a wider child grant must be refused")
	}
	if out := f.s.HandDown(f.apiRefactor, parent.GrantID, f.helpr, res("repo/api/billing"), 4*time.Hour, 0, t0); out.Approved {
		t.Fatal("a child grant that ends later than its parent must be refused")
	}
	child := f.s.HandDown(f.apiRefactor, parent.GrantID, f.helpr, res("repo/api/**"), time.Hour, 0, t0)
	if !child.Approved {
		t.Fatalf("a narrower child grant within the sharing limit should succeed, got %+v", child)
	}
	// Narrow the parent: the child is cut to the overlap at check time.
	if out := f.s.Narrow(f.apiRefactor, parent.GrantID, res("repo/api/billing"), time.Time{}, t0); !out.Approved {
		t.Fatalf("narrowing should succeed, got %+v", out)
	}
	if d := f.s.Check(f.helpr, model.ActionPush, res("repo/api/payments"), t0); d.Allow {
		t.Fatal("outside the parent's new scope must be refused")
	}
	if d := f.s.Check(f.helpr, model.ActionPush, res("repo/api/billing"), t0); !d.Allow {
		t.Fatalf("inside the parent's new scope must still be allowed, got %+v", d)
	}
}

func TestRule9_AChildGrantEndsWithItsParent(t *testing.T) {
	f := newFixture(t)
	parent := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 3, t0))
	kids := []string{}
	for _, name := range []string{"agent:h1", "agent:h2", "agent:h3"} {
		out := f.s.HandDown(f.apiRefactor, parent.GrantID, must(model.ParseIdentity(name)), res("repo/api/**"), time.Hour, 0, t0)
		if !out.Approved {
			t.Fatalf("hand-down to %s failed: %+v", name, out)
		}
		kids = append(kids, out.GrantID)
	}
	h1 := must(model.ParseIdentity("agent:h1"))
	grand := f.s.HandDown(h1, kids[0], must(model.ParseIdentity("agent:g1")), res("repo/api/**"), time.Hour, 0, t0)
	if !grand.Approved {
		t.Fatalf("grandchild hand-down failed: %+v", grand)
	}
	later := t0.Add(time.Minute)
	if out := f.s.Withdraw(f.ada.Identity(), parent.GrantID, later); !out.Approved {
		t.Fatalf("withdrawal failed: %+v", out)
	}
	for _, name := range []string{"agent:api-refactor", "agent:h1", "agent:h2", "agent:h3", "agent:g1"} {
		d := f.s.Check(must(model.ParseIdentity(name)), model.ActionPush, res("repo/api/x"), later)
		if d.Allow || !strings.Contains(d.Reason, "withdrawn by person:ada") {
			t.Fatalf("%s must be refused after the withdrawal, citing it, got %+v", name, d)
		}
	}
}

func TestRule11_SharingNeedsExplicitAuthorisationForEveryGrant(t *testing.T) {
	f := newFixture(t)
	// A contained grant, with no sharing limit set.
	parent := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
	if out := f.s.HandDown(f.apiRefactor, parent.GrantID, f.helpr, res("repo/api/**"), time.Hour, 0, t0); out.Approved {
		t.Fatal("handing down a contained grant without a sharing limit must be refused")
	}
	// A human-only grant at depth 0 (the default).
	g := f.approvedGrant(t)
	if out := f.s.HandDown(f.dbMigrate, g, f.helpr, res("prod/secrets/orders"), time.Hour, 0, t0); out.Approved {
		t.Fatal("handing down a human-only grant with depth 0 must be refused")
	}
}

func TestRule12_AgentsNeverSetMarkingsAuthorityOrSharing(t *testing.T) {
	// By construction: SetMarking, GiveAuthority, SetSharingLimit and
	// AuthoriseService take a model.PersonID, and an agent identity can't be
	// parsed as one. This test checks the parser holds.
	if _, err := model.ParsePerson("agent:api-refactor"); err == nil {
		t.Fatal("an agent must never parse as a person")
	}
}

func TestRule13_AnAgentCreatedServiceGrantGainsNothingNew(t *testing.T) {
	f := newFixture(t)
	agent := must(model.ParseAgent("agent:api-refactor"))
	// Not yet authorised: refused.
	out := f.s.CreateServiceGrant(f.apiRefactor, f.helpr, model.ActionPush, res("repo/api/billing"), "team:payments", time.Hour, 30*time.Minute, t0)
	if out.Approved {
		t.Fatal("an agent without authorisation must not create a service grant")
	}
	ok(t, f.s.AuthoriseService(f.ada, agent, []model.Action{model.ActionPush}, res("repo/**"), t0.Add(24*time.Hour), t0))
	// Authorised, but it doesn't hold the access itself: refused.
	out = f.s.CreateServiceGrant(f.apiRefactor, f.helpr, model.ActionPush, res("repo/api/billing"), "team:payments", time.Hour, 30*time.Minute, t0)
	if out.Approved {
		t.Fatal("an agent must not create a service grant for access it doesn't hold")
	}
	// Holding it: allowed, owned by the team.
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
	out = f.s.CreateServiceGrant(f.apiRefactor, f.helpr, model.ActionPush, res("repo/api/billing"), "team:payments", time.Hour, 30*time.Minute, t0)
	if !out.Ask {
		t.Fatalf("an authorised agent holding the access should be able to propose it, got %+v", out)
	}
	ok(t, f.s.GiveAuthority(f.ada, f.erin, []model.Action{model.ActionPush}, res("repo/**"), t0.Add(24*time.Hour), t0))
	out = f.s.Approve("person:erin", out.RequestID, 0, t0)
	if !out.Approved {
		t.Fatalf("a person with approval authority should approve it, got %+v", out)
	}
	g, _ := f.s.Grant(out.GrantID)
	if g.Owner != "team:payments" || g.CreatedBy != f.apiRefactor || g.Kind != model.KindService {
		t.Fatalf("unexpected service grant %+v", g)
	}
}

func TestResourceParserRefusesNonCanonicalPaths(t *testing.T) {
	for _, bad := range []string{"", "/repo", "repo/", "repo//api", "repo/../prod", "repo/./api", "Repo/api", "repo/**/x", "**", "repo/a b"} {
		if _, err := model.ParseResource(bad); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
}
