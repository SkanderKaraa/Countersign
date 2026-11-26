---
title: "Spike pre-registration: cedar-go vs OPA"
created: 2026-11-17
status: fixed before any measurement
---

# Spike pre-registration

Skander wrote down these rules on 2026-11-17, **before any measurement**. Results are judged against them. Any change made after a measurement is recorded as a change, with its reason.

## Candidates

- **Real candidates:** cedar-go and OPA. Both are open source and on Orvex's list.
- **Reference, not a candidate:** an engine Skander writes in Go. It shows what the fixed rules look like as Go types, and it gives a speed baseline. The results say plainly that Skander wrote it.
- **Ruled out from documentation only, not measured:** Cerbos (can't be embedded in Go), SpiceDB (the embedded form has no list lookup) and Casbin (its own benchmark is about 24 ms per check at 110k rules).
- **Left out:** OpenFGA (it embeds a whole server, and its explanations are weak).

## Conditions, the same for every engine

- The same machine, the same 25 checks and the same load.
- 50,000 grants in one installation's store.
- Events being applied at a realistic rate during the measurement. Replicas apply events from the log; they don't reload snapshots.
- The explanation chain ("approved by whom") is built in Countersign's own layer from what each engine returns. No engine is modified.

## Hard thresholds (an engine must pass all of them)

1. p99 per check is at most 250 µs.
2. All 25 of 25 checks give the expected answer.
3. 0 wrong allows.

## Choosing between cedar-go and OPA, when both pass

Choose by this order, which follows how the brief judges:
1. Provable rules (NFR-5), the security criterion.
2. Explanations (FR-23).
3. The list check (FR-22).
4. Speed, among engines that already pass.

Speed decides earlier only if one engine is much slower, meaning **more than twice** the other's p99.

## When both fail a hard threshold

Take the engine that fails less, and compare it with the own-Go reference, measured by the same rules on the same machine, checks and load. If own Go is clearly better on a criterion named before running, it becomes the choice.

## The own-Go reference

- It stays a reference, and its numbers are reported as a data point. It doesn't change the choice between cedar-go and OPA.
- It can become the choice only if both engines fail a hard threshold.
- If it wins by a wide margin, that is noted in the results, and Danny is asked whether a custom engine is worth it.

## Cedar's analysis

- Install Rust and cvc5, and run Cedar's analysis tool (SymCC) on the Cedar policies. For example: prove that no policy lets an agent approve a human-only action.
- If the setup fails or takes too long, write plainly: "Cedar's analysis was not measured".
- Never write "provable" without having run it.

## Changes after measurement

Each change below was made after seeing results. It is recorded here with its reason, before any re-run.

1. **2026-11-17: event pacing (harness fix).** The first run paced events by the clock (100 per second), so slower engines faced more writes (cedar-go 82, OPA 455, own Go 18 events). That broke the "same load" condition above. Events are now paced by checks done: one per 200 checks, 100 per run, on a concurrent goroutine. The first run was discarded and is kept in the AI_LOG.
2. **2026-11-17: three runs per engine, median p99 reported.** Two identical runs gave cedar-go a p99 of 294 µs and then 579 µs, so one run can't support a verdict.
3. **2026-11-17: the same prefilter for every engine (Skander's decision).**
   - **The result:** after changes 1 and 2, both candidates failed the p99 threshold (cedar-go 466 µs, OPA 1,026 µs). Every engine failed "25 of 25" by construction, because rows 22 and 24 can't run in the spike.
   - **Skander's decision:** treat the result as inconclusive, because of known limit E16. The layer called the engine once per candidate grant, including grants whose own scope can't cover the resource.
   - **The change:** the layer skips those grants before calling any engine, in the same way for every engine. A test shows that skipping them changes no decision.
   - **What is reported:** the results report **both** paths for every engine, the full path and the prefiltered one. The decision is taken on the prefiltered run, against the thresholds and rules above, which are unchanged.
4. **2026-11-17: rows 22 and 24 replaced (Skander's decision).**
   - **The problem:** the original row 22 (a replica cut off from the store, NFR-2) and row 24 (replay before and after a withdrawal, FR-24) can't run in the spike, which has no replica, replication or state rebuild. Neither depends on the engine, so no engine could reach "25 of 25".
   - **The change:** the threshold stays **25 of 25 as written**. The two rows are replaced by rows an engine decides:
     - **New row 22:** a contained grant is handed down within the standing sharing limit, then the child grant checks (FR-8, FR-14).
     - **New row 24:** an "agent ended" signal. Grants created before it are refused, and a grant created after it is allowed (FR-20, FR-3).
   - **What is kept:** the original rows are kept in the decision table, in their own section. The first result (all engines 23 of 25, with rows 22 and 24 not measured) stays in the AI_LOG and in this file.
5. **2026-11-17: OPA's list check uses partial evaluation, as pre-registered (Skander's decision).**
   - **The problem:** the first runs used one check per item for every engine, but this file says "OPA: partial evaluation" for row 23. OPA wasn't being judged on the feature it was shortlisted for.
   - **The change:** the harness now runs OPA's list check through partial evaluation. cedar-go and the reference use one check per item (cedar-go's batch package is experimental and isn't used). Each row-23 result states its method.
6. **2026-11-17: additions to the decision rule, before the re-run (Skander).**
   1. **The worst case is a mandatory measurement.**
      - **What it is:** one agent holding 5,000 grants, a chain at the maximum depth (3), and a large withdrawn set (half of that agent's grants withdrawn), inside the same 50,000-grant store.
      - **Reported:** beside the typical case, with and without the prefilter.
      - **Implementation detail (Claude's, stated openly):** without the prefilter, each worst-case check calls the engine once per candidate grant, 5,000 times. So the full-path worst case is measured on 30 checks per run, reported as the median and maximum per check rather than a p99, and judged against 250 µs on those. The prefiltered worst case uses the normal 20,000 checks and a p99.
   2. **"More than twice" is judged on the median p99 of the 3 runs.**
      - **Stability:** if the fastest and slowest p99 of the 3 runs differ by more than 20%, the measurement is unstable and is repeated.
      - **Implementation detail (Claude's, stated openly):** at most 3 attempts. A measurement still unstable after 3 attempts is reported as unstable, with every value.
   3. **An engine that meets 250 µs only on the prefiltered path passes,** but the results state that its full path exceeds the threshold.
   4. **2026-11-17: added by Skander while run 3 was in progress, before any of its results were read.**
      1. **The 250 µs p99 threshold applies to the typical case only.** The worst case is a mandatory measurement, reported beside it, but it doesn't decide pass or fail.
      2. **A worst-case failure that also affects the own-Go reference is a layer finding** (known limit E16: no index, so the layer loops over every candidate grant). It is reported as such, and it doesn't trigger the "both fail" fallback.
      3. **A measurement still above 20% spread after 3 attempts is reported as unstable,** and no conclusion is drawn from it.
7. **2026-11-17: reduce measurement noise, and fix three report flaws (Skander's decision, recorded before the re-run).**
   - **Why:** in run 4, cedar-go's typical prefiltered p99 was **unstable after 3 attempts** (runs of 44, 76 and 156 µs), so no conclusion could be drawn from it (change 6, point 4.3). OPA's was stable at 144 µs. The pre-registration didn't cover one candidate being unstable.
   - **The machine:** a VM with 4 virtual CPUs on a shared host (AMD Ryzen 9 7950X3D). The VM shows CPU steal time from its host. The measures below reduce noise from *inside* the VM. They can't remove the host's, and the results say so.
   - **Noise reduction, applied identically to every engine (cedar-go, OPA and the own-Go reference):**
     1. **Pinning:** the harness runs on CPUs 1–3 only (`taskset -c 1-3`), with `GOMAXPROCS=3`. That gives one CPU for the checking goroutine, room for the event goroutine and the garbage collector's background work, and leaves CPU 0 for the operating system and other processes. Skander chose 3 CPUs over 2.
     2. **Settling:** before each measured run, the harness forces a garbage collection, returns memory to the operating system, and waits 100 ms.
     3. **Larger samples:** 50,000 measured checks per run instead of 20,000. Events stay at one per 200 checks (250 per run). The worst-case full path stays at 30 checks per run, with one event per 3 checks (10 per run) instead of none.
   - **Unchanged:** 3 runs, the median p99, the 20% stability rule, at most 3 attempts, the 250 µs threshold on the typical case only, 25 of 25 rows with 0 wrong allows, and the tie-break order.
   - **If cedar-go is still unstable after this re-measure (Skander's decision):** cedar-go is **not shown to meet the p99 threshold**. If OPA passes every hard threshold, OPA is the choice. cedar-go's runs are reported, with no conclusion drawn from them. There is no further re-measurement.
   - **Earlier runs stay in the results:** every earlier run is kept in `spike-results-history.md`, each dated, beside the new run. No earlier number is deleted or overwritten. Run 3 was stopped and never read.
   - **Three report fixes, in the same change (wording only; no number changes):**
     1. The "full path exceeds 250 µs" note is judged on the **typical** full path only. The worst-case full path is reported beside it (change 6, point 4.1).
     2. Worst-case full-path rows show "stable: **n/a**", because stability is evaluated only on p99.
     3. The worst-case full path applies one event per 3 checks, as in point 3 above, and the results say so.

## Decision

**2026-11-17: cedar-go is the chosen engine (Skander, applying the rules above to run 5).**
- **Hard thresholds:** both candidates pass on run 5:
  - 25 of 25 rows, with 0 wrong allows
  - a typical prefiltered p99 well under 250 µs, and stable: cedar-go 40 µs (39, 40, 41) and OPA 88 µs (87, 88, 94)
- **The tie-break:** "speed decides earlier only if one engine's p99 is more than twice the other's", judged on the median of 3. OPA's p99 is 2.2× cedar-go's, so speed decides, in favour of cedar-go.
- **Stated with the decision:**
  - The margin is thin (2.2× against a 2× line), and earlier unpinned runs gave different ratios.
  - cedar-go's typical full path is marginal (249 µs), so it relies on the prefilter, which the architecture must specify as an index.
  - The worst case fails on every engine, the reference included. That's a layer finding (E16), not an engine result.
  - OPA is the documented runner-up.
- **Provable rules:** Cedar's analysis tool (SymCC) hasn't run yet. That criterion didn't decide anything, and "provable" isn't claimed until part 5 runs it.

## Changes after the decision

8. **2026-11-17: re-measure after policies.cedar was rewritten for SymCC (Skander, recorded before the re-measure).**
   - **Why:** for part 5, three ancestor clauses in `cedarengine/policies.cedar` were rewritten from `!(grant has parent) || …` to `if grant has parent then … else true`, so Cedar's strict validator and SymCC could check them. All 75 tests pass on every engine after the rewrite. But run 5, on which cedar-go was chosen, measured the original text.
   - **The re-measure (run 6):**
     - **The same conditions as run 5:** pinned to CPUs 1–3 with `GOMAXPROCS=3`, settling before each run, 50,000 checks per run, 3 runs, the 20% stability rule with at most 3 attempts, both cases and both paths.
     - **All three engines.**
     - **Unchanged code:** the harness, OPA's policies and the reference are unchanged. Only `policies.cedar` differs from run 5.
   - **The same decision rule, unchanged:**
     - 25 of 25 rows with 0 wrong allows, and a typical prefiltered p99 of at most 250 µs
     - the stability rule
     - the tie-break order, with speed deciding earlier only if one engine's median p99 is more than twice the other's
     - change 7's rule for a still-unstable cedar-go

     **The rule is not changed after seeing run 6's numbers.**
   - **Reporting:** run 5 stays in the history, and runs 5 and 6 are reported side by side. If run 6 changes the decision, the results say so plainly.
   - **Result (run 6, 2026-11-17):**
     - **The decision is unchanged.** Both candidates pass every hard threshold: 25 of 25 rows, and a stable typical prefiltered p99 (cedar-go 41 µs, OPA 91 µs). OPA is 2.2× slower, so speed decides earlier, and cedar-go remains the choice.
     - **One statement changes:** cedar-go's typical full path is now 259 µs (it was 249 µs in run 5), so it passes only with the prefilter, as OPA does.
     - Runs 5 and 6 are compared side by side in `spike-results-history.md`.
