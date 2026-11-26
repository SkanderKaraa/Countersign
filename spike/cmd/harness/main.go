// Command harness runs the spike: the 25 decision-table rows on every
// engine, then the measurements at 50,000 grants, and writes the results to
// results/. Thresholds come from spike-preregistration.md and were fixed
// before any of this ran.
//
//	go run ./cmd/harness
package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"countersign/spike/cedarengine"
	"countersign/spike/engine"
	"countersign/spike/goref"
	"countersign/spike/opaengine"
	"countersign/spike/ref"
)

// Pre-registered hard thresholds (spike-preregistration.md).
const (
	p99Threshold = 250 * time.Microsecond
	rowsRequired = 25
)

func main() {
	c, err := cedarengine.New(ref.DefaultLimits().SystemMaxDepth)
	if err != nil {
		panic(err)
	}
	o, err := opaengine.New()
	if err != nil {
		panic(err)
	}
	engines := []engine.Engine{c, o, goref.New()}
	role := map[string]string{"cedar": "candidate", "opa": "candidate", "goref": "reference, not a candidate (written by Claude, reviewed by Skander)"}

	var b strings.Builder
	fmt.Fprintf(&b, "# Spike results\n\nRun on %s. Go %s, %s/%s, %d CPUs available to the process, GOMAXPROCS=%d. Engines: cedar-go v1.9.2, OPA v1.22.4, and the own-Go reference. The machine is a VM with 4 virtual CPUs on a shared host, which shows CPU steal time; pinning reduces noise inside the VM only (change 7).\n\n",
		time.Now().UTC().Format("2006-01-02 15:04 MST"), runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.GOMAXPROCS(0))

	// The 25 rows.
	fmt.Fprintf(&b, "## Decision table: 25 checks, 5 of them attacks\n\n| Row | cedar-go | OPA | own Go (reference) |\n|---|---|---|---|\n")
	passed := map[string]int{}
	results := map[string][]rowResult{}
	for _, e := range engines {
		for _, row := range rows {
			r := row(e)
			results[e.Name()] = append(results[e.Name()], r)
			if r.Pass {
				passed[e.Name()]++
			}
		}
	}
	for i := range rows {
		fmt.Fprintf(&b, "| %d | %s | %s | %s |\n", i+1, cell(results["cedar"][i]), cell(results["opa"][i]), cell(results["goref"][i]))
	}
	fmt.Fprintf(&b, "\nPassed: cedar-go %d/25, OPA %d/25, own Go %d/25. Rows 22 and 24 are the replacements recorded as change 4 in spike-preregistration.md.\n\n",
		passed["cedar"], passed["opa"], passed["goref"])

	// The measurements.
	fmt.Fprintf(&b, "## Measurements at 50,000 grants\n\nOne installation's store, built through the normal write path. **Typical:** %d principals with %d grants each, 1 in %d with a hand-down chain of depth 3; 1 in 10 checks is for something not held. **Worst case** (change 6): the same store with one agent holding %d grants, a depth-3 chain, and half its grants withdrawn; every check is that agent's (or the chain's deepest child). Checks run on one goroutine while a second applies one event per %d checks, paced by checks done so every engine faces the same writes. %d checks per run after %d warm-up checks, except the worst case on the full path: %d checks per run with one event per 3 checks, reported as median and max, not p99, and not judged for stability. Before each run the harness forces a garbage collection, returns memory to the OS and waits 100 ms. Each measurement is %d runs; the median p99 is reported. If the fastest and slowest p99 differ by more than 20%%, the measurement is unstable and repeated, at most %d attempts. Paths: 'full' calls the engine once per candidate grant; 'prefiltered' first skips grants whose own scope can't cover the resource, the same for every engine, and changes no decision (TestPrefilterChangesNoDecision).\n\n",
		principals, totalGrants/principals, chainedShare, heavyGrants, checksPerEvent, measuredChecks, warmupChecks, worstFullChecks, runs, maxAttempts)
	fmt.Fprintf(&b, "| Case | Path | Engine | p50 | **p99 (median of %d)** | p99 of the runs | stable | attempts | p99.9 | max | bytes/check | allocs/check | list of 10,000 items | events |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n", runs)
	type key struct {
		sc        scenario
		path, eng string
	}
	med := map[key]benchResult{}
	stable := map[key]bool{}
	for _, sc := range []scenario{typical, worst} {
		for _, mode := range []struct {
			name      string
			prefilter bool
		}{{"full", false}, {"prefiltered", true}} {
			for _, e := range engines {
				k := key{sc, mode.name, e.Name()}
				var all []benchResult
				attempt := 0
				for attempt = 1; attempt <= maxAttempts; attempt++ {
					fmt.Fprintf(os.Stderr, "measuring %s %s %s, attempt %d...\n", sc, mode.name, e.Name(), attempt)
					all = all[:0]
					for r := 0; r < runs; r++ {
						all = append(all, bench(e, sc, mode.prefilter))
					}
					sort.Slice(all, func(x, y int) bool { return all[x].P99 < all[y].P99 })
					if sc == worst && !mode.prefilter {
						stable[k] = true // median and max only; the stability rule is for p99
						break
					}
					fast, slow := all[0].P99, all[len(all)-1].P99
					if float64(slow-fast) <= unstableSpread*float64(fast) {
						stable[k] = true
						break
					}
				}
				if attempt > maxAttempts {
					attempt = maxAttempts
				}
				r := all[len(all)/2]
				med[k] = r
				var runsTxt []string
				for _, x := range all {
					runsTxt = append(runsTxt, x.P99.Round(time.Microsecond).String())
				}
				p99 := "**" + r.P99.Round(time.Microsecond).String() + "**"
				if sc == worst && !mode.prefilter {
					p99 = fmt.Sprintf("n/a (%d checks)", r.Samples)
				}
				stab := yesNo(stable[k])
				if sc == worst && !mode.prefilter {
					stab = "n/a"
				}
				fmt.Fprintf(&b, "| %s | %s | %s (%s) | %v | %s | %s | %s | %d | %v | %v | %d | %d | %v (%d allowed, %s) | %d |\n",
					sc, mode.name, e.Name(), role[e.Name()], r.P50.Round(time.Microsecond), p99, strings.Join(runsTxt, ", "), stab, attempt,
					r.P999.Round(time.Microsecond), r.Max.Round(time.Microsecond), r.BytesPerCheck, r.AllocsPerCheck,
					r.ListCheckTime.Round(time.Millisecond), r.ListAllowed, r.ListMethod, r.EventsApplied)
			}
		}
	}

	fmt.Fprintf(&b, "\n## Against the pre-registered hard thresholds\n\nThe decision is taken on the prefiltered path. Rows threshold: 25 of 25, with 0 wrong allows.\n\n")
	for _, e := range engines {
		n := e.Name()
		tp, wp := med[key{typical, "prefiltered", n}], med[key{worst, "prefiltered", n}]
		tf, wf := med[key{typical, "full", n}], med[key{worst, "full", n}]
		fmt.Fprintf(&b, "- **%s** (%s): rows %d/25: %s. Typical p99 %v: %s%s. Worst-case p99 %v: %s%s.",
			n, role[n], passed[n], okFail(passed[n] == rowsRequired),
			tp.P99.Round(time.Microsecond), okFail(tp.P99 <= p99Threshold), unstableNote(stable[key{typical, "prefiltered", n}]),
			wp.P99.Round(time.Microsecond), okFail(wp.P99 <= p99Threshold), unstableNote(stable[key{worst, "prefiltered", n}]))
		if tf.P99 > p99Threshold {
			fmt.Fprintf(&b, " **The typical full path exceeds 250 µs** (p99 %v), so it passes only with the prefilter (change 6, point 3).", tf.P99.Round(time.Microsecond))
		}
		fmt.Fprintf(&b, " Worst case on the full path, reported only: median %v, max %v per check.", wf.Median.Round(time.Microsecond), wf.Max.Round(time.Microsecond))
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "\nChange 6, point 4: the 250 µs threshold applies to the **typical case only**, so the worst-case verdicts above are reported, not decisive. A worst-case failure that also affects the own-Go reference is a **layer finding** (E16) and doesn't trigger the \"both fail\" fallback. An unstable measurement supports no conclusion.\n")
	fmt.Fprintf(&b, "\nThe choice between cedar-go and OPA is made by Skander against spike-preregistration.md, not by this program.\n")

	if err := os.MkdirAll("results", 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile("results/results.md", []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
	fmt.Print(b.String())
}

func cell(r rowResult) string {
	note := strings.ReplaceAll(r.Note, "|", "/")
	if len(note) > 120 {
		note = note[:117] + "..."
	}
	switch {
	case r.Pass:
		return "✅ " + note
	case strings.HasPrefix(r.Note, "not measured"):
		return "⚪ " + note
	default:
		return "❌ " + note
	}
}

func okFail(ok bool) string {
	if ok {
		return "pass"
	}
	return "**fail**"
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "**no**"
}

func unstableNote(stable bool) string {
	if stable {
		return ""
	}
	return " (**unstable** after " + fmt.Sprint(maxAttempts) + " attempts)"
}
