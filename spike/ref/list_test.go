package ref

import (
	"fmt"
	"testing"
	"time"

	"countersign/spike/model"
)

// A list check must give, for every item, the same answer as a single check
// on that item (FR-22), whatever method the engine uses.
func TestListCheckMatchesSingleChecks(t *testing.T) {
	f := newFixture(t)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 3, t0))
	a := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/feed/a/**", "2h"), t0)
	f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/feed/c/item-7", "2h"), t0)
	f.s.HandDown(f.apiRefactor, a.GrantID, f.helpr, res("repo/feed/a/x/**"), time.Hour, 0, t0)
	w := f.s.SubmitRequest(request("agent:api-refactor", "push", "repo/feed/b/**", "2h"), t0)
	f.s.Withdraw(f.ada.Identity(), w.GrantID, t0)
	var items []model.Resource
	for _, d := range []string{"a", "a/x", "b", "c"} {
		for i := 0; i < 10; i++ {
			items = append(items, res(fmt.Sprintf("repo/feed/%s/item-%d", d, i)))
		}
	}
	for _, who := range []model.Identity{f.apiRefactor, f.helpr, f.dbMigrate} {
		for _, now := range []time.Time{t0, t0.Add(90 * time.Minute), t0.Add(3 * time.Hour)} {
			got, method := f.s.ListCheck(who, model.ActionPush, items, now)
			for i, it := range items {
				want := f.s.Check(who, model.ActionPush, it, now).Allow
				if got[i] != want {
					t.Fatalf("%s %s at +%v (%s): list says %v, single check says %v", who, it, now.Sub(t0), method, got[i], want)
				}
			}
		}
	}
}
