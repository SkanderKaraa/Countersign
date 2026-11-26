package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"countersign/spike/engine"
	"countersign/spike/model"
	"countersign/spike/ref"
)

// The workload. These shapes are the spike's assumptions, not the PRD's:
// how 50,000 grants spread over principals is unknown, so this picks one
// and says so.
const (
	totalGrants    = 50000
	principals     = 5000  // 10 grants each
	chainedShare   = 10    // 1 in 10 principals also hands a grant down, up to depth 3
	measuredChecks = 50000 // change 7 (was 20,000)
	warmupChecks   = 2000
	checksPerEvent = 200 // one write applied, concurrently, per 200 checks: 100 per measured run [tentative]
	runs           = 3   // each engine is measured this many times; the median p99 is reported
	listItems      = 10000

	// The worst case (spike-preregistration.md, change 6): one agent with
	// heavyGrants grants, a depth-3 chain, and half its grants withdrawn,
	// inside the same 50,000-grant store.
	heavyGrants     = 5000
	worstFullChecks = 30 // the full path calls the engine 5,000 times per check
	worstFullWarmup = 3
	worstFullPace   = 3   // worst case, full path: one event per 3 checks (change 7)
	maxAttempts     = 3   // an unstable measurement is repeated at most this often
	unstableSpread  = 0.2 // fastest and slowest p99 of the runs differ by more than 20%
)

// scenario is the typical store, or the typical store with one heavy agent.
type scenario string

const (
	typical scenario = "typical"
	worst   scenario = "worst"
)

type benchResult struct {
	Engine           string
	BuildTime        time.Duration
	WritesPerSecond  float64
	HeapAfterBuildMB float64
	P50, P99, P999   time.Duration
	Max              time.Duration
	EventsApplied    int64
	BytesPerCheck    int64
	AllocsPerCheck   int64
	ListCheckTime    time.Duration
	ListAllowed      int
	ListMethod       string
	Samples          int
	Median           time.Duration
}

type target struct {
	who model.Identity
	act model.Action
	res model.Resource
}

// build makes one installation holding totalGrants grants, all created
// through the normal write path (contained push requests approved by rules,
// plus hand-downs for the chains).
func build(e engine.Engine, sc scenario) (*ref.State, []target, time.Duration) {
	w := setup(e)
	noErr(w.s.SetSharingLimit(w.ada, model.ActionPush, res("repo/**"), 3, t0))
	start := time.Now()
	var targets []target
	created := 0
	typicalGrants := totalGrants
	if sc == worst {
		typicalGrants = totalGrants - heavyGrants // the store stays at 50,000
	}
	for p := 0; p < principals && created < typicalGrants; p++ {
		who := fmt.Sprintf("agent:svc-%d", p)
		var first string
		for g := 0; g < totalGrants/principals; g++ {
			path := fmt.Sprintf("repo/svc-%d/r%d", p, g)
			out := w.s.SubmitRequest(req(who, "push", path, "3h"), t0)
			if !out.Approved {
				panic(out.Reason)
			}
			created++
			if g == 0 {
				first = out.GrantID
			}
			targets = append(targets, target{id(who), model.ActionPush, res(path)})
		}
		if p%chainedShare == 0 { // a chain of depth 3 from the first grant
			parent, holder := first, id(who)
			for d := 1; d <= 3; d++ {
				child := id(fmt.Sprintf("agent:svc-%d-h%d", p, d))
				out := w.s.HandDown(holder, parent, child, res(fmt.Sprintf("repo/svc-%d/r0", p)), time.Hour, 0, t0)
				if !out.Approved {
					panic(out.Reason)
				}
				parent, holder = out.GrantID, child
				targets = append(targets, target{child, model.ActionPush, res(fmt.Sprintf("repo/svc-%d/r0", p))})
			}
		}
	}
	if sc == worst {
		targets = heavy(w)
	}
	return w.s, targets, time.Since(start)
}

// heavy adds the worst-case agent: heavyGrants grants, a depth-3 chain from
// its first grant, and half its grants withdrawn. The checks target only it.
func heavy(w *world) []target {
	who := id("agent:heavy")
	var ids []string
	var targets []target
	for g := 0; g < heavyGrants; g++ {
		path := fmt.Sprintf("repo/heavy/r%d", g)
		out := w.s.SubmitRequest(req(who.String(), "push", path, "3h"), t0)
		if !out.Approved {
			panic(out.Reason)
		}
		ids = append(ids, out.GrantID)
		targets = append(targets, target{who, model.ActionPush, res(path)})
	}
	parent, holder := ids[0], who
	for d := 1; d <= 3; d++ {
		child := id(fmt.Sprintf("agent:heavy-h%d", d))
		out := w.s.HandDown(holder, parent, child, res("repo/heavy/r0"), time.Hour, 0, t0)
		if !out.Approved {
			panic(out.Reason)
		}
		parent, holder = out.GrantID, child
	}
	targets = append(targets, target{holder, model.ActionPush, res("repo/heavy/r0")})
	for g := 1; g < heavyGrants; g += 2 { // half withdrawn (keep r0, the chain's root)
		w.s.Withdraw(w.ada.Identity(), ids[g], t0)
	}
	return targets
}

func heapMB() float64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / (1 << 20)
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	i := int(float64(len(sorted)-1) * p)
	return sorted[i]
}

func bench(e engine.Engine, sc scenario, prefilter bool) benchResult {
	r := benchResult{Engine: e.Name()}
	before := heapMB()
	s, targets, buildTime := build(e, sc)
	s.SetPrefilter(prefilter)
	r.BuildTime = buildTime
	r.WritesPerSecond = float64(totalGrants) / buildTime.Seconds()
	r.HeapAfterBuildMB = heapMB() - before

	rng := rand.New(rand.NewSource(1))
	now := t0.Add(time.Minute)
	pick := func() target {
		t := targets[rng.Intn(len(targets))]
		if rng.Intn(10) == 0 { // 1 in 10 checks is for something not held
			t.res = res(t.res.String() + "-other")
		}
		return t
	}
	measured, warmup, pace := measuredChecks, warmupChecks, int64(checksPerEvent)
	if sc == worst && !prefilter {
		measured, warmup, pace = worstFullChecks, worstFullWarmup, worstFullPace
	}
	// Settle before the measured run (change 7): one run's garbage mustn't
	// land in the next run's p99.
	runtime.GC()
	debug.FreeOSMemory()
	time.Sleep(100 * time.Millisecond)
	r.Samples = measured
	for i := 0; i < warmup; i++ {
		t := pick()
		s.Check(t.who, t.act, t.res, now)
	}

	// Apply events while measuring, on a second goroutine: new contained
	// grants and withdrawals. They are paced by checks done, not by the
	// clock, so every engine faces the same writes per check (a slower
	// engine would otherwise face more).
	var checksDone, events atomic.Int64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ada := person("person:ada").Identity()
		for i := int64(0); ; {
			select {
			case <-stop:
				return
			default:
			}
			if i >= checksDone.Load()/pace {
				time.Sleep(20 * time.Microsecond)
				continue
			}
			out := s.SubmitRequest(req(fmt.Sprintf("agent:churn-%d", i), "push", fmt.Sprintf("repo/churn/%d", i), "1h"), now)
			if i%2 == 1 && out.Approved {
				s.Withdraw(ada, out.GrantID, now)
			}
			events.Add(1)
			i++
		}
	}()

	lat := make([]time.Duration, measured)
	for i := range lat {
		t := pick()
		start := time.Now()
		s.Check(t.who, t.act, t.res, now)
		lat[i] = time.Since(start)
		checksDone.Add(1)
	}
	for events.Load() < int64(measured)/pace { // let the last events land
		time.Sleep(50 * time.Microsecond)
	}
	close(stop)
	wg.Wait()
	r.EventsApplied = events.Load()
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	r.P50, r.P99, r.P999, r.Max = percentile(lat, 0.50), percentile(lat, 0.99), percentile(lat, 0.999), lat[len(lat)-1]
	r.Median = r.P50

	// Allocations per check, on a quiet store.
	br := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			t := targets[i%len(targets)]
			s.Check(t.who, t.act, t.res, now)
		}
	})
	r.BytesPerCheck, r.AllocsPerCheck = br.AllocedBytesPerOp(), br.AllocsPerOp()

	// Row 23 at this scale: one agent's feed of 10,000 items, one check per item.
	who := id("agent:feed-reader")
	s.SubmitRequest(req(who.String(), "push", "repo/feed/a/**", "3h"), now)
	items := feed()
	start := time.Now()
	allowed, method := s.ListCheck(who, model.ActionPush, items, now)
	r.ListCheckTime = time.Since(start)
	r.ListMethod = method
	for _, ok := range allowed {
		if ok {
			r.ListAllowed++
		}
	}
	return r
}
