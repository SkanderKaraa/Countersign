package ref

import (
	"fmt"
	"testing"
	"time"

	"countersign/spike/model"
)

// The prefilter must change no decision: the same checks, with it on and
// off, give the same allow or refuse on every engine.
func TestPrefilterChangesNoDecision(t *testing.T) {
	f := newFixture(t)
	ok(t, f.s.SetSharingLimit(f.ada, model.ActionPush, res("repo/**"), 3, t0))
	var grants []string
	for i := 0; i < 20; i++ {
		out := f.s.SubmitRequest(request("agent:api-refactor", "push", fmt.Sprintf("repo/s%d/**", i%5), "2h"), t0)
		grants = append(grants, out.GrantID)
	}
	f.s.HandDown(f.apiRefactor, grants[0], f.helpr, res("repo/s0/x"), time.Hour, 0, t0)
	f.s.Withdraw(f.ada.Identity(), grants[3], t0)
	f.s.Narrow(f.apiRefactor, grants[6], res("repo/s1/a"), time.Time{}, t0)
	who := []model.Identity{f.apiRefactor, f.helpr, f.dbMigrate}
	paths := []string{"repo/s0/x", "repo/s1/a", "repo/s1/b", "repo/s3/y", "repo/s9/z", "prod/x"}
	when := []time.Time{t0, t0.Add(90 * time.Minute), t0.Add(3 * time.Hour)}
	for _, p := range who {
		for _, path := range paths {
			for _, now := range when {
				f.s.SetPrefilter(false)
				full := f.s.Check(p, model.ActionPush, res(path), now)
				f.s.SetPrefilter(true)
				pre := f.s.Check(p, model.ActionPush, res(path), now)
				if full.Allow != pre.Allow {
					t.Fatalf("%s push %s at %s: full path %v, prefiltered %v", p, path, now.Format(time.Kitchen), full.Allow, pre.Allow)
				}
			}
		}
	}
}
