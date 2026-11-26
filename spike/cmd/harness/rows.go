package main

import (
	"fmt"
	"strings"
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
	"countersign/spike/ref"
)

// The 25 rows of decision-table.md. Each row runs in a fresh installation
// built by setup, on one engine, and reports pass or fail with a note.

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

type world struct {
	s                        *ref.State
	ada, ben, erin           model.PersonID
	apiRefactor, dbMigrate   model.Identity
	helper, releasePrep, sam model.Identity
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func res(s string) model.Resource    { return must(model.ParseResource(s)) }
func id(s string) model.Identity     { return must(model.ParseIdentity(s)) }
func person(s string) model.PersonID { return must(model.ParsePerson(s)) }
func noErr(err error)                { must(struct{}{}, err) }
func req(p, a, r, d string) []byte {
	return []byte(fmt.Sprintf(`{"principal":%q,"action":%q,"resource":%q,"task":"ENG-88","duration":%q}`, p, a, r, d))
}

// setup builds the installation every row starts from:
//   - administrators Ada and Ben
//   - repo/** classified non-production (proposed by Ada, confirmed by Ben),
//     and push on it marked contained
//   - prod/** classified production
//   - Erin holds approval authority for read on prod/**
//   - task ENG-88 is open
func setup(e engine.Engine) *world {
	w := &world{
		ada: person("person:ada"), ben: person("person:ben"), erin: person("person:erin"),
		apiRefactor: id("agent:api-refactor"), dbMigrate: id("agent:db-migrate"),
		helper: id("agent:helper"), releasePrep: id("agent:release-prep"), sam: id("person:sam"),
	}
	w.s = must(ref.NewState(ref.DefaultLimits(), e, t0, w.ada, w.ben))
	pid, err := w.s.SetResourceInfo(w.ada, model.ResourceInfo{Resource: res("repo/**")}, t0)
	if err != ref.ErrNeedsSecond {
		panic(err)
	}
	noErr(w.s.ConfirmResourceInfo(w.ben, pid, t0))
	_, err = w.s.SetResourceInfo(w.ada, model.ResourceInfo{Resource: res("prod/**"), Production: true}, t0)
	noErr(err)
	noErr(w.s.SetMarking(w.ada, model.ActionPush, res("repo/**"), model.ClassContained, t0))
	noErr(w.s.GiveAuthority(w.ada, w.erin, []model.Action{model.ActionRead}, res("prod/**"), t0.Add(60*24*time.Hour), t0))
	w.s.OpenTask("ENG-88")
	return w
}

// erinApproved is UJ-2: db-migrate asks to read the orders secret for 2 h,
// and Erin approves with the given delegation depth.
func (w *world) erinApproved(depth int) (string, ref.Outcome) {
	ask := w.s.SubmitRequest(req("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
	if !ask.Ask {
		return "", ask
	}
	return ask.RequestID, w.s.Approve("person:erin", ask.RequestID, depth, t0)
}

type rowResult struct {
	ID     int
	Pass   bool
	Note   string
	Timing time.Duration // only where the row has a time target
}

func pass(id int, note string) rowResult { return rowResult{ID: id, Pass: true, Note: note} }
func fail(id int, note string) rowResult { return rowResult{ID: id, Note: note} }

func expect(id int, ok bool, good, bad string) rowResult {
	if ok {
		return pass(id, good)
	}
	return fail(id, bad)
}

var rows = []func(engine.Engine) rowResult{
	// 1. Attack: a request names its own approver.
	func(e engine.Engine) rowResult {
		w := setup(e)
		raw := []byte(`{"principal":"agent:db-migrate","action":"read","resource":"prod/secrets/orders","task":"ENG-88","duration":"2h","approver":"person:erin"}`)
		out := w.s.SubmitRequest(raw, t0)
		return expect(1, !out.Approved && !out.Ask, "refused: "+out.Reason, fmt.Sprintf("not refused: %+v", out))
	},
	// 2. Attack: a contained grant handed down without "allowed to share".
	func(e engine.Engine) rowResult {
		w := setup(e)
		p := w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
		out := w.s.HandDown(w.apiRefactor, p.GrantID, w.helper, res("repo/api/x"), time.Hour, 0, t0)
		return expect(2, !out.Approved, "refused: "+out.Reason, "handed down without a sharing limit")
	},
	// 3. Attack: a child grant wider, or ending later, than its parent.
	func(e engine.Engine) rowResult {
		w := setup(e)
		noErr(w.s.SetSharingLimit(w.ada, model.ActionPush, res("repo/**"), 2, t0))
		p := w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/api/billing", "3h"), t0)
		wider := w.s.HandDown(w.apiRefactor, p.GrantID, w.helper, res("repo/api/**"), time.Hour, 0, t0)
		longer := w.s.HandDown(w.apiRefactor, p.GrantID, w.helper, res("repo/api/billing"), 4*time.Hour, 0, t0)
		return expect(3, !wider.Approved && !longer.Approved, "both refused: "+wider.Reason+" / "+longer.Reason, "a wider or longer child grant was allowed")
	},
	// 4. Attack: the requester approves their own request.
	func(e engine.Engine) rowResult {
		w := setup(e)
		ask := w.s.SubmitRequest(req("person:erin", "read", "prod/secrets/orders", "2h"), t0)
		out := w.s.Approve("person:erin", ask.RequestID, 0, t0)
		return expect(4, ask.Ask && !out.Approved, "refused: "+out.Reason, fmt.Sprintf("self-approval: %+v", out))
	},
	// 5. Attack: an agent approves a human-only action, however it is worded.
	func(e engine.Engine) rowResult {
		w := setup(e)
		ask := w.s.SubmitRequest(req("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
		for _, who := range []string{"agent:api-refactor", "agent:erin", "PERSON:erin", "person:erin:x"} {
			if out := w.s.Approve(who, ask.RequestID, 0, t0); out.Approved {
				return fail(5, who+" approved")
			}
		}
		return pass(5, "all four wordings refused")
	},
	// 6. A contained request, approved by rules within 2 s.
	func(e engine.Engine) rowResult {
		w := setup(e)
		start := time.Now()
		out := w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/feature-retry", "1h"), t0)
		took := time.Since(start)
		r := expect(6, out.Approved && took < 2*time.Second, fmt.Sprintf("approved by rules in %v (in-process, no network)", took), fmt.Sprintf("%+v in %v", out, took))
		r.Timing = took
		return r
	},
	// 7. A check with the grant from row 6 allows and names it.
	func(e engine.Engine) rowResult {
		w := setup(e)
		g := w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/feature-retry", "1h"), t0)
		d := w.s.Check(w.apiRefactor, model.ActionPush, res("repo/feature-retry"), t0)
		return expect(7, d.Allow && strings.Contains(d.Reason, g.GrantID), d.Reason+" "+fmt.Sprint(d.Policies), d.Reason)
	},
	// 8. No grant: refused with "no grant; make a request".
	func(e engine.Engine) rowResult {
		w := setup(e)
		d := w.s.Check(w.dbMigrate, model.ActionRead, res("prod/secrets/orders"), t0)
		return expect(8, !d.Allow && d.Reason == "no grant; make a request", d.Reason, d.Reason)
	},
	// 9. A human-only request returns "ask" with approvers listed.
	func(e engine.Engine) rowResult {
		w := setup(e)
		out := w.s.SubmitRequest(req("agent:db-migrate", "read", "prod/secrets/orders", "2h"), t0)
		return expect(9, out.Ask && len(out.Approvers) > 0, fmt.Sprintf("ask, approvers %v", out.Approvers), fmt.Sprintf("%+v", out))
	},
	// 10. Erin approves; the check allows, naming the grant and Erin.
	func(e engine.Engine) rowResult {
		w := setup(e)
		_, ap := w.erinApproved(0)
		d := w.s.Check(w.dbMigrate, model.ActionRead, res("prod/secrets/orders"), t0)
		return expect(10, ap.Approved && d.Allow && d.RootApprover == "person:erin", d.Reason+", root approver "+d.RootApprover, fmt.Sprintf("%+v %+v", ap, d))
	},
	// 11. Erin narrows 2 h to 1 h; an extension is refused.
	func(e engine.Engine) rowResult {
		w := setup(e)
		_, ap := w.erinApproved(0)
		erin := w.erin.Identity()
		cut := w.s.Narrow(erin, ap.GrantID, model.Resource{}, t0.Add(time.Hour), t0)
		ext := w.s.Narrow(erin, ap.GrantID, model.Resource{}, t0.Add(3*time.Hour), t0)
		return expect(11, cut.Approved && !ext.Approved, "narrowed; extension refused: "+ext.Reason, fmt.Sprintf("%+v %+v", cut, ext))
	},
	// 12. A request over the hard maximum lifetime is refused, stating it.
	func(e engine.Engine) rowResult {
		w := setup(e)
		out := w.s.SubmitRequest(req("agent:db-migrate", "read", "prod/secrets/orders", "25h"), t0)
		return expect(12, !out.Approved && !out.Ask && strings.Contains(out.Reason, "24h"), out.Reason, fmt.Sprintf("%+v", out))
	},
	// 13. A check after the grant's time limit is refused.
	func(e engine.Engine) rowResult {
		w := setup(e)
		w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/x", "1h"), t0)
		d := w.s.Check(w.apiRefactor, model.ActionPush, res("repo/x"), t0.Add(time.Hour))
		return expect(13, !d.Allow, d.Reason, "allowed after the time limit")
	},
	// 14. "Task closed", then the check is refused.
	func(e engine.Engine) rowResult {
		w := setup(e)
		w.erinApproved(0)
		start := time.Now()
		noErr(w.s.AcceptTaskClosed("ENG-88", "product:quornex", t0))
		d := w.s.Check(w.dbMigrate, model.ActionRead, res("prod/secrets/orders"), t0)
		took := time.Since(start)
		r := expect(14, !d.Allow, fmt.Sprintf("%s, refused %v after the store accepted the signal (in-process)", d.Reason, took), "allowed after the task closed")
		r.Timing = took
		return r
	},
	// 15. Withdraw a parent with three child grants and a grandchild.
	func(e engine.Engine) rowResult {
		w := setup(e)
		noErr(w.s.SetSharingLimit(w.ada, model.ActionPush, res("repo/**"), 3, t0))
		p := w.s.SubmitRequest(req("agent:release-prep", "push", "repo/api/**", "3h"), t0)
		var kids []string
		for _, h := range []string{"agent:h1", "agent:h2", "agent:h3"} {
			kids = append(kids, w.s.HandDown(w.releasePrep, p.GrantID, id(h), res("repo/api/**"), time.Hour, 0, t0).GrantID)
		}
		w.s.HandDown(id("agent:h1"), kids[0], id("agent:g1"), res("repo/api/**"), 30*time.Minute, 0, t0)
		start := time.Now()
		w.s.Withdraw(w.erin.Identity(), p.GrantID, t0) // Erin isn't an approver of a contained grant
		w.s.Withdraw(w.ada.Identity(), p.GrantID, t0)
		refused := 0
		for _, who := range []string{"agent:release-prep", "agent:h1", "agent:h2", "agent:h3", "agent:g1"} {
			if !w.s.Check(id(who), model.ActionPush, res("repo/api/x"), t0).Allow {
				refused++
			}
		}
		took := time.Since(start)
		r := expect(15, refused == 5, fmt.Sprintf("all 5 refused, %v after the withdrawal (in-process)", took), fmt.Sprintf("%d of 5 refused", refused))
		r.Timing = took
		return r
	},
	// 16. A narrowed parent: the child is refused outside, allowed inside.
	func(e engine.Engine) rowResult {
		w := setup(e)
		noErr(w.s.SetSharingLimit(w.ada, model.ActionPush, res("repo/**"), 2, t0))
		p := w.s.SubmitRequest(req("agent:release-prep", "push", "repo/api/**", "3h"), t0)
		w.s.HandDown(w.releasePrep, p.GrantID, w.helper, res("repo/api/**"), time.Hour, 0, t0)
		w.s.Narrow(w.releasePrep, p.GrantID, res("repo/api/billing"), time.Time{}, t0)
		out := w.s.Check(w.helper, model.ActionPush, res("repo/api/payments"), t0)
		in := w.s.Check(w.helper, model.ActionPush, res("repo/api/billing"), t0)
		return expect(16, !out.Allow && in.Allow, "payments refused, billing allowed", fmt.Sprintf("payments %v, billing %v", out.Allow, in.Allow))
	},
	// 17. A human-only grant handed down with depth 1.
	func(e engine.Engine) rowResult {
		w := setup(e)
		_, ap := w.erinApproved(1)
		child := w.s.HandDown(w.dbMigrate, ap.GrantID, w.helper, res("prod/secrets/orders"), time.Hour, 0, t0)
		d := w.s.Check(w.helper, model.ActionRead, res("prod/secrets/orders"), t0)
		return expect(17, child.Approved && d.Allow, "child grant created; check allows, chain "+fmt.Sprint(d.Chain), fmt.Sprintf("%+v %+v", child, d))
	},
	// 18. The depth-0 child from row 17 can't hand down again.
	func(e engine.Engine) rowResult {
		w := setup(e)
		_, ap := w.erinApproved(1)
		child := w.s.HandDown(w.dbMigrate, ap.GrantID, w.helper, res("prod/secrets/orders"), time.Hour, 0, t0)
		again := w.s.HandDown(w.helper, child.GrantID, id("agent:g1"), res("prod/secrets/orders"), 30*time.Minute, 0, t0)
		return expect(18, child.Approved && !again.Approved, "refused: "+again.Reason, "handed down again")
	},
	// 19. An administrator marks an action contained; its request is then approved by rules.
	func(e engine.Engine) rowResult {
		w := setup(e)
		err := w.s.SetMarking(w.ada, model.ActionRead, res("repo/docs/**"), model.ClassContained, t0)
		out := w.s.SubmitRequest(req("agent:api-refactor", "read", "repo/docs/readme", "1h"), t0)
		return expect(19, err == nil && out.Approved, "marking accepted; request approved by rules", fmt.Sprintf("%v %+v", err, out))
	},
	// 20. An administrator marks a floor-of-four action contained by mistake.
	func(e engine.Engine) rowResult {
		w := setup(e)
		err := w.s.SetMarking(w.ada, model.ActionRead, res("prod/secrets/**"), model.ClassContained, t0)
		return expect(20, err != nil, fmt.Sprintf("refused: %v", err), "a floor-of-four action was marked contained")
	},
	// 21. Erin's approval authority expires: earlier grants stay valid, new approvals are refused.
	func(e engine.Engine) rowResult {
		w := setup(e)
		lab := person("person:lab-lead")
		noErr(w.s.GiveAuthority(w.ada, lab, []model.Action{model.ActionRead}, res("prod/**"), t0.Add(time.Hour), t0))
		ask := w.s.SubmitRequest(req("agent:db-migrate", "read", "prod/secrets/orders", "8h"), t0)
		ap := w.s.Approve("person:lab-lead", ask.RequestID, 0, t0)
		later := t0.Add(2 * time.Hour)
		earlier := w.s.Check(w.dbMigrate, model.ActionRead, res("prod/secrets/orders"), later)
		ask2 := w.s.SubmitRequest(req("agent:helper", "read", "prod/secrets/orders", "1h"), later)
		ap2 := w.s.Approve("person:lab-lead", ask2.RequestID, 0, later)
		return expect(21, ap.Approved && earlier.Allow && !ap2.Approved, "earlier grant still allows; new approval refused: "+ap2.Reason, fmt.Sprintf("%v %v %+v", ap.Approved, earlier.Allow, ap2))
	},
	// 22. A contained grant handed down within the standing sharing limit;
	// the child grant checks (replaces the original row 22, change 4).
	func(e engine.Engine) rowResult {
		w := setup(e)
		noErr(w.s.SetSharingLimit(w.ada, model.ActionPush, res("repo/**"), 2, t0))
		p := w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
		child := w.s.HandDown(w.apiRefactor, p.GrantID, w.helper, res("repo/api/x"), time.Hour, 0, t0)
		d := w.s.Check(w.helper, model.ActionPush, res("repo/api/x"), t0)
		return expect(22, child.Approved && d.Allow, "child grant created; check allows, chain "+fmt.Sprint(d.Chain), fmt.Sprintf("%+v %+v", child, d))
	},
	// 23. A list check over a feed of 10,000 items, by the engine's own
	// list method (OPA: partial evaluation; others: one check per item).
	func(e engine.Engine) rowResult {
		w := setup(e)
		w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/feed/a/**", "3h"), t0)
		items := feed()
		start := time.Now()
		allowed, method := w.s.ListCheck(w.apiRefactor, model.ActionPush, items, t0)
		took := time.Since(start)
		n, exact := 0, true
		for i, ok := range allowed {
			if ok {
				n++
			}
			if ok != (i%2 == 0) { // exactly the items under repo/feed/a
				exact = false
			}
		}
		r := expect(23, n == 5000 && exact, fmt.Sprintf("%d of 10,000 allowed, exactly those under repo/feed/a; %v by %s", n, took.Round(time.Millisecond), method), fmt.Sprintf("%d allowed (exact: %v), by %s", n, exact, method))
		r.Timing = took
		return r
	},
	// 24. An "agent ended" signal: a grant created before it is refused, a
	// grant created after it is allowed (replaces the original row 24, change 4).
	func(e engine.Engine) rowResult {
		w := setup(e)
		w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/api/a", "1h"), t0)
		noErr(w.s.AcceptAgentEnded(must(model.ParseAgent("agent:api-refactor")), "product:houston", t0))
		before := w.s.Check(w.apiRefactor, model.ActionPush, res("repo/api/a"), t0)
		later := t0.Add(time.Minute)
		w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/api/b", "1h"), later)
		after := w.s.Check(w.apiRefactor, model.ActionPush, res("repo/api/b"), later)
		return expect(24, !before.Allow && after.Allow, "earlier grant refused ("+before.Reason+"); later grant allowed", fmt.Sprintf("before %v, after %v", before.Allow, after.Allow))
	},
	// 25. An authorised agent proposes a service grant; a person approves it.
	func(e engine.Engine) rowResult {
		w := setup(e)
		agent := must(model.ParseAgent("agent:api-refactor"))
		noErr(w.s.AuthoriseService(w.ada, agent, []model.Action{model.ActionPush}, res("repo/**"), t0.Add(24*time.Hour), t0))
		w.s.SubmitRequest(req("agent:api-refactor", "push", "repo/api/**", "3h"), t0)
		prop := w.s.CreateServiceGrant(w.apiRefactor, w.helper, model.ActionPush, res("repo/api/billing"), "team:payments", 30*24*time.Hour, 7*24*time.Hour, t0)
		ap := w.s.Approve("person:ben", prop.RequestID, 0, t0)
		g, _ := w.s.Grant(ap.GrantID)
		return expect(25, prop.Ask && ap.Approved && g.Owner == "team:payments" && g.Kind == model.KindService,
			"proposal asked a person; created, owned by team:payments", fmt.Sprintf("%+v %+v", prop, ap))
	},
}

// feed is 10,000 items, alternating between repo/feed/a and repo/feed/b.
func feed() []model.Resource {
	items := make([]model.Resource, 10000)
	for i := range items {
		half := "a"
		if i%2 == 1 {
			half = "b"
		}
		items[i] = res(fmt.Sprintf("repo/feed/%s/item-%d", half, i))
	}
	return items
}
