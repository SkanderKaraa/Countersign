---
title: "Review: reconciling the architecture spine with the brief and the spike evidence"
reviewed: ARCHITECTURE-SPINE.md (updated 2026-11-17)
date: 2026-11-17
inputs: brief.md, addendum.md, spike-preregistration.md, spike-results.md, spike-results-history.md, symcc-results.md, decision-table.md, spike/KNOWN-LIMITS.md; spot checks in spike/ source (policies.cedar, ref/, engine/, cmd/harness/)
---

# Review: spine against the brief and the spike evidence

Paths are short: `SPINE` = ARCHITECTURE-SPINE.md, `PRE` = spike-preregistration.md, `RES` = spike-results.md, `HIST` = spike-results-history.md, `SYMCC` = symcc-results.md, `DT` = decision-table.md, `KL` = spike/KNOWN-LIMITS.md, `BRIEF` = brief.md, `ADD` = addendum.md, `CEDAR` = spike/cedarengine/policies.cedar.

**Verdict:** most numbers in the spine match run 6 exactly. The memory figures, the 14+2 SymCC count, the versions, the 2.2x decision and the 259 µs full-path figure are all correct. But the spine states four things more strongly than the evidence allows, or contrary to it:
- AD-8's "every fixed rule is a forbid"
- AD-8's proof scope
- AD-9's "no engine met 250 µs without the prefilter"
- AD-6's "write-time rules go through the engine"

It also drops two brief commitments (100% replay, 100% cascade within 10 s), and AD-4 is ADOPTED without the "decided, not measured" tag.

## Critical and high

### E1: high: AD-8: "each fixed rule is a forbid" is contradicted by the spike's policies
- **Finding:** AD-8 says each fixed rule is a Cedar `forbid` with an `@id`, except rule 1. In the policies that SymCC proved, rules 8, 9, 11 and 13 are **conditions inside `permit`s**, not forbids:
  - rules 8 and 9 sit in `check.grant-allows`
  - rules 8 and 11 sit in `hand_down.within-parent`
  - rule 13 sits in `create_service_grant.authorised-agent`

  Only rules 4, 5, 6 and 12 (plus depth, unknown-class, service-grant and owner) are forbids. A rule written as a permit condition can be bypassed by any future permit added for the same action, which is exactly what the forbid convention is meant to prevent.
- **Evidence:** SPINE:122 vs CEDAR:17-47, 85-96, 137-139 (forbids only at CEDAR:50, 66, 71, 75, 101, 106, 116, 121, 125, 142). KL:27 and SYMCC:13-19 prove the properties, not the forbid shape.
- **Fix:** either (a) state the real shape: "rules 4, 5, 6, 12 are forbids; rules 1, 8, 9, 11, 13 are conditions of the only permit for their action, proved as properties". Add a CI check that each such action has exactly one permit. Or (b) make the rewrite into forbids a v1 task, and mark it "decided, not measured" until SymCC re-proves it.

### E2: high: AD-8: the proof scope omits two known limits (people's grants, and check actions other than `push`)
- **Finding:** AD-8 mentions only the rule-6 limit (person requesters). KNOWN-LIMITS has two more:
  - the schema types `Grant.principal` as an `Agent`, so checks by **people** are not covered by any proof
  - the check properties (rule 1, rules 8/9, depth) were analysed only for `push`, not for the other five check actions

  So "proved as the property 'a check is allowed only by a matching grant'" claims more than was proved.
- **Evidence:** SPINE:122-123 vs KL:31-32 and SYMCC:16-18 (request environment "Agent, push, Path" only).
- **Fix:** add both limits under AD-8 beside the rule-6 limit. Then either widen the schema (`Grant.principal: Person | Agent`) and run the check properties for every check action in CI, or state that the CI suite proves rule 1, rules 8/9 and depth for agent principals on `push` only.

### E3: high: AD-9: "no engine met 250 µs without the prefilter" is false
- **Finding:** On the typical full path (no prefilter), two engines met 250 µs:
  - the own-Go reference: 30 µs in run 6 (unstable) and 34 µs in run 5 (stable)
  - cedar-go in run 5: 249 µs

  Only cedar-go in run 6 (259 µs) and OPA failed. The true statement is narrower: "neither candidate met it reliably without the prefilter, and every engine, prefiltered or not, failed the worst case".
- **Evidence:** SPINE:130 vs RES:43-45, HIST:227-229 (run 5: cedar 249 µs, goref 34 µs stable), HIST:332, PRE:114.
- **Fix:** replace the clause with: "in the spike, cedar-go's typical full path was 249 µs (run 5) and 259 µs (run 6), around the 250 µs line, so the decision relied on the prefilter. With 5,000 grants for one principal, every engine failed 250 µs even with the prefilter (cedar-go 415 µs, OPA 385 µs, reference 364 µs, run 6), because the prefilter still loops over every grant (E16)."

### E4: high: AD-6 and AD-5: "write-time rules go through the same engine" contradicts a known limit, and the spine doesn't say who decides the rest
- **Finding:** AD-6 says write-time rules go through the engine, and lists only five engine decisions (`Check`, `MayApprove`, `MayHandDown`, `MaySetMarking`, `MayCreateServiceGrant`). In the spike, the following writes are decided **only in Go**, outside any engine and outside SymCC:
  - who may narrow or withdraw a grant (FR-4)
  - classification and the two-administrator confirmation
  - approval authority
  - sharing limits
  - service-grant authorisations
  - administrators

  The spine neither routes these through the engine (there is no `MayNarrow`, `MayWithdraw`, `MayClassify` or `MaySetAuthority`) nor says they stay in Go. AD-16 (the floor lowered only by two administrators) is ADOPTED without a tag, yet it was decided in Go only.
- **Evidence:** SPINE:103, 105, 94, 200-207 vs KL:15-18. Tests exist in Go only: spike/ref/review3_test.go (`TestReview3_A1_…`, `TestReview3_A2_…`), spike/ref/rules_test.go (`TestRule12_…`).
- **Fix:** add an explicit list to AD-6: "decided by the engine: …; decided in the store's Go code, tested but not proved: FR-4 narrow/withdraw authority, classification and the two-administrator confirmation, approval authority, sharing limits, service-grant authorisations, administrators". Then either add engine actions for them as v1 work, or accept the gap and carry it to AD-17's layer tests.

### E5: high: AD-4 is ADOPTED without "decided, not measured", but nothing in it was measured
- **Finding:** AD-4's horizons, its two-step fail-closed and its "later of two clocks" rule were never exercised: the spike has no replica, heartbeat or staleness. The spine admits this inline for the clock rule and the horizons, but the tag is missing, so AD-4 reads as evidence-backed next to AD-1 to AD-3, which carry the tag. The original row 22, the replica cut off from the store, was removed from the spike and marked "must be tested in the layer's replica tests, before v1". The spine doesn't carry that obligation.
- **Evidence:** SPINE:80, 86 vs DT:65, KL:22, PRE:69.
- **Fix:** retag AD-4 as `[ADOPTED; decided, not measured]` and add: "the original decision-table row 22 must pass in the replica's tests before v1".

### E6: high: AD-14 and Deferred drop the brief's "100% of decisions can be replayed for at least one month"
- **Finding:** The brief commits that every decision can be replayed, with 100% replayable for at least a month. The spine:
  - writes compact records **asynchronously**, so a crash can lose them, and a lost record can't be replayed
  - defers record retention entirely
  - never says replay (the original row 24) was **not tested** in the spike and must be tested before v1

  The commitment is weakened silently.
- **Evidence:** SPINE:189, 302, 164-170 vs BRIEF:25, 118; DT:66 (replay "not tested in the spike… must be tested … before v1"); PRD FR-24 (prd.md:476).
- **Fix:** either make compact records durable before the answer (or batched with a bounded loss window that is stated against the 100% target), or record an explicit deviation from the brief. Add a one-month minimum retention as a binding floor, not a deferral. Add the obligation that "the original row 24 replay test passes before v1" to AD-12 or AD-14.

## Medium

- **E7: medium: AD-4 vs brief: cascade of 100% within 10 s isn't met for contained grants on a cut-off replica.** AD-4 allows contained checks until the second, longer horizon, but the brief promises 100% of descendants refused within 10 s, and task-closed within 10 s. ADD:96 narrows this to high-risk only, and the spine should state that as an explicit exception to BRIEF:114-115. Evidence: SPINE:84-85 vs BRIEF:114-115, ADD:96. Fix: state the contained exception and its bound (the second horizon) under AD-4, or cap the second horizon at 10 s.
- **E8: medium: AD-6: the thin decision margin is omitted.** The spine says cedar-go was "chosen by the pre-registered rule", but leaves out the caveats recorded with the decision:
  - 2.2x against a 2x line
  - earlier unpinned runs gave different ratios (run 2b: about 1.0x, run 4: about 1.9x with cedar-go unstable)
  - provable rules (SymCC) decided nothing

  Evidence: SPINE:106 vs PRE:113, 117; HIST:102-103, 159-160. Fix: add one sentence carrying these caveats.
- **E9: medium: AD-6: "cedar-go has no list method" is inaccurate.** cedar-go has an experimental batch package. It wasn't used, and despite DT:48's "both measured", it was never measured. Evidence: SPINE:104 vs PRE:76, DT:48. Fix: "cedar-go's batch package is experimental and wasn't used or measured; lists use one check per item."
- **E10: medium: AD-8: proofs are on Rust Cedar semantics, but cedar-go is what ships.** SymCC analysed with cedar-policy 4.15.0 (Rust), while the runtime is cedar-go v1.9.2. "Proved" covers cedar-go only as far as cedar-go conforms to Rust Cedar, which nothing in the spike established. Only the three-way differential test touches this. Evidence: SPINE:125, 248, 253 vs SYMCC:3. Fix: state the assumption, and name the guard (the differential test, plus cedar-go's conformance suite if it has one).
- **E11: medium: KNOWN-LIMITS' immutable swapped state is omitted.** The replica's intended design (an immutable state swapped atomically, so checks never wait) isn't in the spine, and the measured p99 includes read/write-lock contention that the design would remove. Evidence: KL:14 vs SPINE:20-22, 128-144 (silent). Fix: add the swap rule to AD-9 or the Design Paradigm, and note that the measured latencies include lock contention.
- **E12: medium: AD-9 is ADOPTED without a tag, but its index was never built.** The spike's only index is by principal, and the prefilter loops over every grant (E16). The (principal, action, scope) index and "worst case independent of grant count" are unmeasured. Evidence: SPINE:128, 132-134 vs KL:9; spike/ref/state.go:87; spike/ref/check.go:41-42. Fix: tag the index rule "decided, not measured", and keep the memory figures as measured basis.
- **E13: medium: AD-17's CI reflection-registry test is stated as "Enforced in CI" but doesn't exist in the spike.** No reflect-based test, and no registry of facts mapped to tests, exists in the spike. Evidence: SPINE:220 vs spike/ref/*_test.go (no such test). Fix: phrase it as a v1 requirement, "decided, not built".
- **E14: medium: Original decision-table rows 22 and 24 must be tested before v1, and the spine doesn't carry this.** Evidence: DT:65-66 vs SPINE (absent). Fix: add both as v1 acceptance obligations (see E5, E6).
- **E15: medium: Brief vs AD-10/AD-18: "a parent agent may approve its child's request" becomes agents never approve, with hand-down instead.** This follows PRD:278, which is still `[OPEN, to check with Danny]`. The spine doesn't record that it reinterprets the brief or that this is open. Evidence: BRIEF:21, 73; ADD:69 vs SPINE:150-151, 226; prd.md:278. Fix: one line under AD-10 or AD-18 naming the reinterpretation and the open question.
- **E16: medium: Brief vs AD-5/AD-18: "approval attached to the final action, so splitting can't get around it" is in v1 scope, but the spine makes it conditional on an unconfirmed Quornex gate.** The gap is disclosed, but it is a dropped brief commitment, not just a dependency. Evidence: BRIEF:22, 130 vs SPINE:96, 228. Fix: list it explicitly as a deviation from v1 scope until question 6b is answered.

## Low

- **E17: low: Brief vs spike: service grants' "own children".** BRIEF:79 says withdrawing a service grant takes back its children, but the spike decided that service grants are never handed down. The spine is silent. Evidence: BRIEF:79 vs CEDAR:98-103, SYMCC:15. Fix: state the rule in the spine and flag the brief change.
- **E18: low: AD-9's store-memory figure.** The results label it "31 MB". The spine says MiB, and "rounds to the nearest MiB" can't be verified, because run 2b's printing code was overwritten (the current harness divides by 1<<20 but no longer prints it). Truncation would give about 650–670 bytes. The figure also measures the combined store-plus-replica `State`, not the store alone. Evidence: SPINE:143 vs HIST:102; spike/cmd/harness/bench.go:152-171. Fix: say "31 (as printed; MiB per the harness code)", and "State (store and replica together)".
- **E19: low: AD-9's worst-case full-path memory is "not comparable", though it was measured** (12,390,551 bytes and 172,955 allocations per check). Evidence: SPINE:141 vs RES:49. Fix: give the number with its caveat.
- **E20: low: The measurement environment caveat is missing.** The spine gives no mention of the 4-vCPU VM with host steal time, single process, single machine. Evidence: KL:23, RES:3 vs SPINE:135-143. Fix: one line under the measured basis.
- **E21: low: NFR-5 rule 3 (hard maximum lifetime) isn't bound by any AD,** although AD-5 recomputes it. Evidence: SPINE:90, 119, 147, 173; KL:33. Fix: add rule 3 to AD-5's Binds.
- **E22: low: AD-12 is ADOPTED untagged, but decision records with clock offset were never built.** Evidence: SPINE:164, 170 vs KL:22. Fix: tag that part "decided, not measured".
- **E23: low: AD-15's list timeout is untested.** The spike's `ListCheck` has no timeout. Evidence: SPINE:198 vs spike/ref/check.go:142-162. Fix: tag the timeout "decided".
- **E24: low: The list latency evidence is absent.** cedar-go took 51–78 ms per 10,000 items (run 6), and the brief scopes lists into v1. The spine sets no list budget. Evidence: RES:23, 43, 46 vs BRIEF:134. Fix: record the measured figure and a budget, or defer it explicitly.
- **E25: low: The frontmatter `sources` omits spike-results.md** (the run-6 file whose numbers AD-9 quotes). Evidence: SPINE:12. Fix: add it.

## Checked and found consistent

- Memory per check: 4,464/64 and 25,642/356 (typical), 45,234/62 (worst, prefiltered).
- SymCC: 14 properties plus 2 controls, cvc5 1.3.1, cedar-policy-symcc 0.9.0.
- Versions: Go 1.27.3, cedar-go v1.9.2, OPA v1.22.4.
- The decision on runs 5 and 6, with OPA as runner-up.
- Max depth 3, and the engine refusing to load for any other.
- `TestPrefilterChangesNoDecision`, `TestListCheckMatchesSingleChecks`, the three-way differential test, `engine.Guard` and millisecond truncation all exist.
- The rule-6 person-requester limit.
- D14 and E17 deferred; E15 covered by AD-17; E16 covered by AD-9.
- AD-1, 2, 3, 13 and 14 carry the tag.
- Untested OpenBao prefix listing disclosed.
