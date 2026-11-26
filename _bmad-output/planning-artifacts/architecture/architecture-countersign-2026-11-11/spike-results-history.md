---
title: "Spike results: every run, in order"
created: 2026-11-17
---

# Spike results: every run, in order

Every run of the harness, dated 2026-11-17, kept beside the latest one (spike-preregistration.md, change 7). No number is deleted or overwritten.

**Where the text comes from:**
- Runs 1 and 4 are the program's own results files.
- Runs 1b, 2 and 2b are the measurement tables the program printed during the session. Their files were overwritten by the next run, so the tables are copied verbatim from that output.
- Run 3 was stopped before it finished, and its output was never read.

## Run 1: events paced by the clock (discarded, change 1)

Events were paced by the clock, so a slower engine faced more writes. This broke the "same load" condition. The full results file follows.

## Spike results

Run on 2026-11-17 18:39 UTC. Go go1.27.3, linux/amd64, 4 CPUs available. Engines: cedar-go v1.9.2, OPA v1.22.4, and the own-Go reference.

### Decision table: 25 checks, 5 of them attacks

| Row | cedar-go | OPA | own Go (reference) |
|---|---|---|---|
| 1 | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" |
| 2 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: no sharing limit allows handing this down |
| 3 | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of grant g-2 / a child grant can't end later... |
| 4 | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request |
| 5 | ✅ all four wordings refused | ✅ all four wordings refused | ✅ all four wordings refused |
| 6 | ✅ approved by rules in 4.338µs (in-process, no network) | ✅ approved by rules in 5.561µs (in-process, no network) | ✅ approved by rules in 7.995µs (in-process, no network) |
| 7 | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] |
| 8 | ✅ no grant; make a request | ✅ no grant; make a request | ✅ no grant; make a request |
| 9 | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] |
| 10 | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin |
| 11 | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request |
| 12 | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) |
| 13 | ✅ grant g-2 has ended | ✅ grant g-2 has ended | ✅ grant g-2 has ended |
| 14 | ✅ task ENG-88 closed, refused 5.269µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 30.607µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 4.168µs after the store accepted the signal (in-process) |
| 15 | ✅ all 5 refused, 41.438µs after the withdrawal (in-process) | ✅ all 5 refused, 460.984µs after the withdrawal (in-process) | ✅ all 5 refused, 25.798µs after the withdrawal (in-process) |
| 16 | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed |
| 17 | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] |
| 18 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: this grant is not allowed to be handed down |
| 19 | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules |
| 20 | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: a floor-of-four action can't be marked contained |
| 21 | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for read on prod/secrets/... |
| 22 | ⚪ not measured: no replica, replication or staleness horizon in the spike (KNOWN-LIMITS.md) | ⚪ not measured: no replica, replication or staleness horizon in the spike (KNOWN-LIMITS.md) | ⚪ not measured: no replica, replication or staleness horizon in the spike (KNOWN-LIMITS.md) |
| 23 | ✅ 5000 of 10,000 items allowed, exactly the half under repo/feed/a; 116.254326ms by one check per item | ✅ 5000 of 10,000 items allowed, exactly the half under repo/feed/a; 504.364778ms by one check per item | ✅ 5000 of 10,000 items allowed, exactly the half under repo/feed/a; 18.924136ms by one check per item |
| 24 | ⚪ not measured: no state rebuild from the log by position in the spike (KNOWN-LIMITS.md) | ⚪ not measured: no state rebuild from the log by position in the spike (KNOWN-LIMITS.md) | ⚪ not measured: no state rebuild from the log by position in the spike (KNOWN-LIMITS.md) |
| 25 | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments |

Passed: cedar-go 23/25, OPA 23/25, own Go 23/25. Rows 22 and 24 can't run in the spike (see KNOWN-LIMITS.md), so 23 of 25 is the most any engine can reach.

### Measurements at 50,000 grants

One installation's store, built through the normal write path: 5000 principals with 10 grants each, 1 in 10 with a hand-down chain of depth 3. Checks run on one goroutine while a second applies 100 events/s (new contained grants, and withdrawals). 1 in 10 checks is for something not held. 20000 checks measured after 2000 warm-up checks. The layer calls the engine once per candidate grant, with no prefiltering (KNOWN-LIMITS.md, E16).

| Engine | Role | p50 | **p99** | p99.9 | max | bytes/check | allocs/check | store heap | writes/s while building | list of 10,000 items | events applied |
|---|---|---|---|---|---|---|---|---|---|---|---|
| cedar | candidate | 29.585µs | **293.67µs** | 1.3018ms | 3.456909ms | 25642 | 356 | 31 MB | 176757 | 88ms (5000 allowed) | 82 |
| opa | candidate | 183.164µs | **1.166867ms** | 4.020214ms | 15.544373ms | 76492 | 1515 | 31 MB | 129363 | 448ms (5000 allowed) | 455 |
| goref | reference, not a candidate (written by Claude, reviewed by Skander) | 5.62µs | **51.557µs** | 588.372µs | 2.713586ms | 2662 | 53 | 31 MB | 144485 | 9ms (5000 allowed) | 18 |

### Against the pre-registered hard thresholds

- **cedar:** p99 293.67µs (threshold 250µs): **fail**; rows 23/25 (threshold 25/25, 0 wrong allows): **fail**
- **opa:** p99 1.166867ms (threshold 250µs): **fail**; rows 23/25 (threshold 25/25, 0 wrong allows): **fail**
- **goref:** p99 51.557µs (threshold 250µs): pass; rows 23/25 (threshold 25/25, 0 wrong allows): **fail**

The choice between cedar-go and OPA is made by Skander against spike-preregistration.md, not by this program.

## Run 1b: the same code run again by accident (clock-paced)

The program's measurement table, copied verbatim:

| Engine | Role | p50 | **p99** | p99.9 | max | bytes/check | allocs/check | store heap | writes/s while building | list of 10,000 items | events applied |
|---|---|---|---|---|---|---|---|---|---|---|---|
| cedar | candidate | 33.923µs | **578.905µs** | 2.404086ms | 6.335014ms | 25642 | 356 | 31 MB | 125117 | 83ms (5000 allowed) | 110 |
| opa | candidate | 188.173µs | **1.250744ms** | 3.851568ms | 14.136485ms | 76485 | 1515 | 31 MB | 71770 | 559ms (5000 allowed) | 461 |
| goref | reference, not a candidate (written by Claude, reviewed by Skander) | 4.97µs | **38.242µs** | 677.289µs | 2.036076ms | 2662 | 53 | 31 MB | 206779 | 8ms (5000 allowed) | 16 |

## Run 2: events paced by checks, 3 runs per engine, no prefilter (changes 1 and 2)

The program's measurement table, copied verbatim. The rows were 23 of 25 on every engine: the original rows 22 and 24 couldn't run.

| Engine | Role | p50 | **p99 (median of 3 runs)** | p99 range | p99.9 | max | bytes/check | allocs/check | store heap | writes/s while building | list of 10,000 items | events applied |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| cedar | candidate | 33.814µs | **465.712µs** | 287.98µs to 762.409µs | 1.95277ms | 9.65132ms | 25643 | 357 | 31 MB | 166090 | 81ms (5000 allowed) | 100 |
| opa | candidate | 176.571µs | **1.026424ms** | 826.59µs to 1.397249ms | 3.176744ms | 7.241273ms | 76472 | 1515 | 31 MB | 103689 | 480ms (5000 allowed) | 100 |
| goref | reference, not a candidate (written by Claude, reviewed by Skander) | 5.69µs | **46.087µs** | 39.243µs to 129.072µs | 442.139µs | 4.189602ms | 2662 | 53 | 31 MB | 171336 | 15ms (5000 allowed) | 100 |

## Run 2b: both paths, with the prefilter added (change 3)

The program's measurement table, copied verbatim. The list check used one check per item for every engine, which was the deviation corrected in change 5.

| Engine | Path | Role | p50 | **p99 (median of 3 runs)** | p99 range | p99.9 | max | bytes/check | allocs/check | store heap | writes/s while building | list of 10,000 items | events applied |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| cedar | full | candidate | 37.31µs | **945.352µs** | 713.929µs to 1.011847ms | 4.845915ms | 55.479736ms | 25643 | 356 | 31 MB | 169085 | 93ms (5000 allowed) | 100 |
| opa | full | candidate | 204.564µs | **1.411807ms** | 1.384886ms to 1.544496ms | 3.890453ms | 8.113681ms | 76479 | 1515 | 31 MB | 80520 | 423ms (5000 allowed) | 100 |
| goref | full | reference, not a candidate (written by Claude, reviewed by Skander) | 5.391µs | **96.3µs** | 57.909µs to 116.078µs | 589.495µs | 2.199493ms | 2662 | 53 | 31 MB | 138915 | 12ms (5000 allowed) | 100 |
| cedar | prefiltered | candidate | 6.653µs | **159.84µs** | 59.522µs to 165.18µs | 849.733µs | 3.490614ms | 4464 | 64 | 31 MB | 130397 | 53ms (5000 allowed) | 100 |
| opa | prefiltered | candidate | 32.33µs | **158.377µs** | 158.217µs to 191.659µs | 1.154405ms | 5.184109ms | 14659 | 292 | 31 MB | 76347 | 336ms (5000 allowed) | 100 |
| goref | prefiltered | reference, not a candidate (written by Claude, reviewed by Skander) | 2.886µs | **28.654µs** | 20.679µs to 29.866µs | 314.219µs | 6.288148ms | 636 | 14 | 31 MB | 159107 | 8ms (5000 allowed) | 100 |

## Run 3: stopped, never read

Stopped at Skander's request before it finished, so that the results would come straight from the program with change 6, point 4. It never wrote a results file. Its partial output was set aside unread.

## Run 4: replaced rows, OPA partial evaluation, worst case, stability rule (changes 4 to 6)

The full results file follows.

## Spike results

Run on 2026-11-17 18:59 UTC. Go go1.27.3, linux/amd64, 4 CPUs available. Engines: cedar-go v1.9.2, OPA v1.22.4, and the own-Go reference.

### Decision table: 25 checks, 5 of them attacks

| Row | cedar-go | OPA | own Go (reference) |
|---|---|---|---|
| 1 | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" |
| 2 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: no sharing limit allows handing this down |
| 3 | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of grant g-2 / a child grant can't end later... |
| 4 | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request |
| 5 | ✅ all four wordings refused | ✅ all four wordings refused | ✅ all four wordings refused |
| 6 | ✅ approved by rules in 7.193µs (in-process, no network) | ✅ approved by rules in 8.405µs (in-process, no network) | ✅ approved by rules in 4.128µs (in-process, no network) |
| 7 | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] |
| 8 | ✅ no grant; make a request | ✅ no grant; make a request | ✅ no grant; make a request |
| 9 | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] |
| 10 | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin |
| 11 | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request |
| 12 | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) |
| 13 | ✅ grant g-2 has ended | ✅ grant g-2 has ended | ✅ grant g-2 has ended |
| 14 | ✅ task ENG-88 closed, refused 11.211µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 33.602µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 2.054µs after the store accepted the signal (in-process) |
| 15 | ✅ all 5 refused, 137.194µs after the withdrawal (in-process) | ✅ all 5 refused, 222.23µs after the withdrawal (in-process) | ✅ all 5 refused, 12.222µs after the withdrawal (in-process) |
| 16 | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed |
| 17 | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] |
| 18 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: this grant is not allowed to be handed down |
| 19 | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules |
| 20 | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: a floor-of-four action can't be marked contained |
| 21 | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for read on prod/secrets/... |
| 22 | ✅ child grant created; check allows, chain [g-3 g-2] | ✅ child grant created; check allows, chain [g-3 g-2] | ✅ child grant created; check allows, chain [g-3 g-2] |
| 23 | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 93ms by one check per item | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 89ms by partial evaluation | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 6ms by one check per item |
| 24 | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed |
| 25 | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments |

Passed: cedar-go 25/25, OPA 25/25, own Go 25/25. Rows 22 and 24 are the replacements recorded as change 4 in spike-preregistration.md.

### Measurements at 50,000 grants

One installation's store, built through the normal write path. **Typical:** 5000 principals with 10 grants each, 1 in 10 with a hand-down chain of depth 3; 1 in 10 checks is for something not held. **Worst case** (change 6): the same store with one agent holding 5000 grants, a depth-3 chain, and half its grants withdrawn; every check is that agent's (or the chain's deepest child). Checks run on one goroutine while a second applies one event per 200 checks, paced by checks done so every engine faces the same writes. 20000 checks per run after 2000 warm-up checks, except the worst case on the full path: 30 checks per run, reported as median and max, not p99. Each measurement is 3 runs; the median p99 is reported. If the fastest and slowest p99 differ by more than 20%, the measurement is unstable and repeated, at most 3 attempts. Paths: 'full' calls the engine once per candidate grant; 'prefiltered' first skips grants whose own scope can't cover the resource, the same for every engine, and changes no decision (TestPrefilterChangesNoDecision).

| Case | Path | Engine | p50 | **p99 (median of 3)** | p99 of the runs | stable | attempts | p99.9 | max | bytes/check | allocs/check | list of 10,000 items | events |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| typical | full | cedar (candidate) | 30µs | **247µs** | 234µs, 247µs, 272µs | yes | 3 | 1.114ms | 4.072ms | 25642 | 356 | 63ms (5000 allowed, one check per item) | 100 |
| typical | full | opa (candidate) | 170µs | **750µs** | 737µs, 750µs, 766µs | yes | 1 | 2.282ms | 8.487ms | 76489 | 1515 | 81ms (5000 allowed, partial evaluation) | 100 |
| typical | full | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 6µs | **80µs** | 64µs, 80µs, 86µs | **no** | 3 | 578µs | 2.828ms | 2662 | 53 | 5ms (5000 allowed, one check per item) | 100 |
| typical | prefiltered | cedar (candidate) | 6µs | **76µs** | 44µs, 76µs, 156µs | **no** | 3 | 552µs | 2.485ms | 4464 | 64 | 56ms (5000 allowed, one check per item) | 100 |
| typical | prefiltered | opa (candidate) | 31µs | **144µs** | 132µs, 144µs, 156µs | yes | 1 | 867µs | 6.225ms | 14658 | 292 | 81ms (5000 allowed, partial evaluation) | 100 |
| typical | prefiltered | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 2µs | **10µs** | 9µs, 10µs, 20µs | **no** | 3 | 78µs | 1.187ms | 636 | 14 | 4ms (5000 allowed, one check per item) | 100 |
| worst | full | cedar (candidate) | 32.621ms | n/a (30 checks) | 39.583ms, 58.899ms, 87.215ms | yes | 1 | 58.899ms | 61.662ms | 12390552 | 172955 | 72ms (5000 allowed, one check per item) | 0 |
| worst | full | opa (candidate) | 149.741ms | n/a (30 checks) | 173.204ms, 189.028ms, 223.438ms | yes | 1 | 189.028ms | 194.634ms | 34241947 | 671749 | 100ms (5000 allowed, partial evaluation) | 0 |
| worst | full | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 4.16ms | n/a (30 checks) | 11.578ms, 14.259ms, 14.406ms | yes | 1 | 14.259ms | 16.457ms | 1238632 | 24737 | 2ms (5000 allowed, one check per item) | 0 |
| worst | prefiltered | cedar (candidate) | 68µs | **735µs** | 724µs, 735µs, 822µs | yes | 1 | 2.301ms | 4.744ms | 45234 | 62 | 59ms (5000 allowed, one check per item) | 100 |
| worst | prefiltered | opa (candidate) | 124µs | **647µs** | 639µs, 647µs, 701µs | yes | 1 | 2.087ms | 7.031ms | 54657 | 272 | 115ms (5000 allowed, partial evaluation) | 100 |
| worst | prefiltered | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 63µs | **610µs** | 609µs, 610µs, 721µs | yes | 1 | 1.809ms | 5.763ms | 41474 | 13 | 3ms (5000 allowed, one check per item) | 100 |

### Against the pre-registered hard thresholds

The decision is taken on the prefiltered path. Rows threshold: 25 of 25, with 0 wrong allows.

- **cedar** (candidate): rows 25/25: pass. Typical p99 76µs: pass (**unstable** after 3 attempts). Worst-case p99 735µs: **fail**. **The full path exceeds 250 µs** (typical p99 247µs; worst case median 32.621ms, max 61.662ms per check), so it passes only with the prefilter (change 6, point 3).
- **opa** (candidate): rows 25/25: pass. Typical p99 144µs: pass. Worst-case p99 647µs: **fail**. **The full path exceeds 250 µs** (typical p99 750µs; worst case median 149.741ms, max 194.634ms per check), so it passes only with the prefilter (change 6, point 3).
- **goref** (reference, not a candidate (written by Claude, reviewed by Skander)): rows 25/25: pass. Typical p99 10µs: pass (**unstable** after 3 attempts). Worst-case p99 610µs: **fail**. **The full path exceeds 250 µs** (typical p99 80µs; worst case median 4.16ms, max 16.457ms per check), so it passes only with the prefilter (change 6, point 3).

Change 6, point 4: the 250 µs threshold applies to the **typical case only**, so the worst-case verdicts above are reported, not decisive. A worst-case failure that also affects the own-Go reference is a **layer finding** (E16) and doesn't trigger the "both fail" fallback. An unstable measurement supports no conclusion.

The choice between cedar-go and OPA is made by Skander against spike-preregistration.md, not by this program.

## Run 5: pinned to CPUs 1-3, GOMAXPROCS=3, settling, 50,000 checks per run (change 7)

The full results file follows.

## Spike results

Run on 2026-11-17 19:07 UTC. Go go1.27.3, linux/amd64, 3 CPUs available to the process, GOMAXPROCS=3. Engines: cedar-go v1.9.2, OPA v1.22.4, and the own-Go reference. The machine is a VM with 4 virtual CPUs on a shared host, which shows CPU steal time; pinning reduces noise inside the VM only (change 7).

### Decision table: 25 checks, 5 of them attacks

| Row | cedar-go | OPA | own Go (reference) |
|---|---|---|---|
| 1 | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" |
| 2 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: no sharing limit allows handing this down |
| 3 | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of grant g-2 / a child grant can't end later... |
| 4 | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request |
| 5 | ✅ all four wordings refused | ✅ all four wordings refused | ✅ all four wordings refused |
| 6 | ✅ approved by rules in 4.258µs (in-process, no network) | ✅ approved by rules in 3.607µs (in-process, no network) | ✅ approved by rules in 4.689µs (in-process, no network) |
| 7 | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] |
| 8 | ✅ no grant; make a request | ✅ no grant; make a request | ✅ no grant; make a request |
| 9 | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] |
| 10 | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin |
| 11 | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request |
| 12 | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) |
| 13 | ✅ grant g-2 has ended | ✅ grant g-2 has ended | ✅ grant g-2 has ended |
| 14 | ✅ task ENG-88 closed, refused 9.257µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 21.149µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 3.175µs after the store accepted the signal (in-process) |
| 15 | ✅ all 5 refused, 345.502µs after the withdrawal (in-process) | ✅ all 5 refused, 736.84µs after the withdrawal (in-process) | ✅ all 5 refused, 16.471µs after the withdrawal (in-process) |
| 16 | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed |
| 17 | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] |
| 18 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: this grant is not allowed to be handed down |
| 19 | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules |
| 20 | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: a floor-of-four action can't be marked contained |
| 21 | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for read on prod/secrets/... |
| 22 | ✅ child grant created; check allows, chain [g-3 g-2] | ✅ child grant created; check allows, chain [g-3 g-2] | ✅ child grant created; check allows, chain [g-3 g-2] |
| 23 | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 95ms by one check per item | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 97ms by partial evaluation | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 3ms by one check per item |
| 24 | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed |
| 25 | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments |

Passed: cedar-go 25/25, OPA 25/25, own Go 25/25. Rows 22 and 24 are the replacements recorded as change 4 in spike-preregistration.md.

### Measurements at 50,000 grants

One installation's store, built through the normal write path. **Typical:** 5000 principals with 10 grants each, 1 in 10 with a hand-down chain of depth 3; 1 in 10 checks is for something not held. **Worst case** (change 6): the same store with one agent holding 5000 grants, a depth-3 chain, and half its grants withdrawn; every check is that agent's (or the chain's deepest child). Checks run on one goroutine while a second applies one event per 200 checks, paced by checks done so every engine faces the same writes. 50000 checks per run after 2000 warm-up checks, except the worst case on the full path: 30 checks per run with one event per 3 checks, reported as median and max, not p99, and not judged for stability. Before each run the harness forces a garbage collection, returns memory to the OS and waits 100 ms. Each measurement is 3 runs; the median p99 is reported. If the fastest and slowest p99 differ by more than 20%, the measurement is unstable and repeated, at most 3 attempts. Paths: 'full' calls the engine once per candidate grant; 'prefiltered' first skips grants whose own scope can't cover the resource, the same for every engine, and changes no decision (TestPrefilterChangesNoDecision).

| Case | Path | Engine | p50 | **p99 (median of 3)** | p99 of the runs | stable | attempts | p99.9 | max | bytes/check | allocs/check | list of 10,000 items | events |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| typical | full | cedar (candidate) | 31µs | **249µs** | 241µs, 249µs, 282µs | yes | 2 | 897µs | 4.315ms | 25643 | 356 | 58ms (5000 allowed, one check per item) | 250 |
| typical | full | opa (candidate) | 177µs | **841µs** | 770µs, 841µs, 853µs | yes | 1 | 2.468ms | 16.167ms | 76466 | 1515 | 87ms (5000 allowed, partial evaluation) | 250 |
| typical | full | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 5µs | **34µs** | 32µs, 34µs, 38µs | yes | 3 | 332µs | 4.277ms | 2662 | 53 | 4ms (5000 allowed, one check per item) | 250 |
| typical | prefiltered | cedar (candidate) | 6µs | **40µs** | 39µs, 40µs, 41µs | yes | 1 | 268µs | 2.095ms | 4464 | 64 | 57ms (5000 allowed, one check per item) | 250 |
| typical | prefiltered | opa (candidate) | 30µs | **88µs** | 87µs, 88µs, 94µs | yes | 2 | 446µs | 2.26ms | 14657 | 292 | 89ms (5000 allowed, partial evaluation) | 250 |
| typical | prefiltered | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 1µs | **12µs** | 11µs, 12µs, 12µs | yes | 2 | 113µs | 991µs | 636 | 14 | 4ms (5000 allowed, one check per item) | 250 |
| worst | full | cedar (candidate) | 32.781ms | n/a (30 checks) | 38.048ms, 38.974ms, 41.3ms | n/a | 1 | 38.974ms | 42.384ms | 12390550 | 172955 | 54ms (5000 allowed, one check per item) | 10 |
| worst | full | opa (candidate) | 141.381ms | n/a (30 checks) | 150.285ms, 156.499ms, 156.963ms | n/a | 1 | 156.499ms | 162.928ms | 34240140 | 671740 | 83ms (5000 allowed, partial evaluation) | 10 |
| worst | full | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 4.266ms | n/a (30 checks) | 7.121ms, 7.19ms, 7.338ms | n/a | 1 | 7.19ms | 7.264ms | 1257744 | 25131 | 4ms (5000 allowed, one check per item) | 10 |
| worst | prefiltered | cedar (candidate) | 66µs | **479µs** | 403µs, 479µs, 493µs | **no** | 3 | 1.547ms | 4.488ms | 45234 | 62 | 71ms (5000 allowed, one check per item) | 250 |
| worst | prefiltered | opa (candidate) | 130µs | **378µs** | 364µs, 378µs, 431µs | yes | 1 | 1.525ms | 5.285ms | 54656 | 272 | 89ms (5000 allowed, partial evaluation) | 250 |
| worst | prefiltered | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 59µs | **377µs** | 370µs, 377µs, 427µs | yes | 1 | 1.256ms | 3.347ms | 41474 | 13 | 12ms (5000 allowed, one check per item) | 250 |

### Against the pre-registered hard thresholds

The decision is taken on the prefiltered path. Rows threshold: 25 of 25, with 0 wrong allows.

- **cedar** (candidate): rows 25/25: pass. Typical p99 40µs: pass. Worst-case p99 479µs: **fail** (**unstable** after 3 attempts). Worst case on the full path, reported only: median 32.781ms, max 42.384ms per check.
- **opa** (candidate): rows 25/25: pass. Typical p99 88µs: pass. Worst-case p99 378µs: **fail**. **The typical full path exceeds 250 µs** (p99 841µs), so it passes only with the prefilter (change 6, point 3). Worst case on the full path, reported only: median 141.381ms, max 162.928ms per check.
- **goref** (reference, not a candidate (written by Claude, reviewed by Skander)): rows 25/25: pass. Typical p99 12µs: pass. Worst-case p99 377µs: **fail**. Worst case on the full path, reported only: median 4.266ms, max 7.264ms per check.

Change 6, point 4: the 250 µs threshold applies to the **typical case only**, so the worst-case verdicts above are reported, not decisive. A worst-case failure that also affects the own-Go reference is a **layer finding** (E16) and doesn't trigger the "both fail" fallback. An unstable measurement supports no conclusion.

The choice between cedar-go and OPA is made by Skander against spike-preregistration.md, not by this program.

## Run 6: same conditions as run 5, policies.cedar rewritten for SymCC (change 8)

The full results file follows.

## Spike results

Run on 2026-11-17 19:31 UTC. Go go1.27.3, linux/amd64, 3 CPUs available to the process, GOMAXPROCS=3. Engines: cedar-go v1.9.2, OPA v1.22.4, and the own-Go reference. The machine is a VM with 4 virtual CPUs on a shared host, which shows CPU steal time; pinning reduces noise inside the VM only (change 7).

### Decision table: 25 checks, 5 of them attacks

| Row | cedar-go | OPA | own Go (reference) |
|---|---|---|---|
| 1 | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" | ✅ refused: malformed request: json: unknown field "approver" |
| 2 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: no sharing limit allows handing this down |
| 3 | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of a grant in the chain / no policy permits ... | ✅ both refused: grant g-2 doesn't cover repo/api/** now: outside the scope of grant g-2 / a child grant can't end later... |
| 4 | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request | ✅ refused: nobody approves their own request |
| 5 | ✅ all four wordings refused | ✅ all four wordings refused | ✅ all four wordings refused |
| 6 | ✅ approved by rules in 10.139µs (in-process, no network) | ✅ approved by rules in 4.849µs (in-process, no network) | ✅ approved by rules in 3.577µs (in-process, no network) |
| 7 | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] | ✅ allowed by grant g-2 [check.grant-allows] |
| 8 | ✅ no grant; make a request | ✅ no grant; make a request | ✅ no grant; make a request |
| 9 | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] | ✅ ask, approvers [person:ada person:ben person:erin] |
| 10 | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin | ✅ allowed by grant g-2, root approver person:erin |
| 11 | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request | ✅ narrowed; extension refused: a grant can't be extended; make a new request |
| 12 | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) | ✅ duration 25h0m0s is over the hard maximum lifetime (under 24h0m0s) |
| 13 | ✅ grant g-2 has ended | ✅ grant g-2 has ended | ✅ grant g-2 has ended |
| 14 | ✅ task ENG-88 closed, refused 10.069µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 30.408µs after the store accepted the signal (in-process) | ✅ task ENG-88 closed, refused 2.855µs after the store accepted the signal (in-process) |
| 15 | ✅ all 5 refused, 39.495µs after the withdrawal (in-process) | ✅ all 5 refused, 216.651µs after the withdrawal (in-process) | ✅ all 5 refused, 18.565µs after the withdrawal (in-process) |
| 16 | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed | ✅ payments refused, billing allowed |
| 17 | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] | ✅ child grant created; check allows, chain [g-4 g-2] |
| 18 | ✅ refused: no policy permits this | ✅ refused: no policy permits this | ✅ refused: this grant is not allowed to be handed down |
| 19 | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules | ✅ marking accepted; request approved by rules |
| 20 | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: forbidden by set_marking.forbid-floor-contained | ✅ refused: refused: a floor-of-four action can't be marked contained |
| 21 | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for this | ✅ earlier grant still allows; new approval refused: person:lab-lead has no approval authority for read on prod/secrets/... |
| 22 | ✅ child grant created; check allows, chain [g-3 g-2] | ✅ child grant created; check allows, chain [g-3 g-2] | ✅ child grant created; check allows, chain [g-3 g-2] |
| 23 | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 78ms by one check per item | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 93ms by partial evaluation | ✅ 5000 of 10,000 allowed, exactly those under repo/feed/a; 3ms by one check per item |
| 24 | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed | ✅ earlier grant refused (agent agent:api-refactor ended); later grant allowed |
| 25 | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments | ✅ proposal asked a person; created, owned by team:payments |

Passed: cedar-go 25/25, OPA 25/25, own Go 25/25. Rows 22 and 24 are the replacements recorded as change 4 in spike-preregistration.md.

### Measurements at 50,000 grants

One installation's store, built through the normal write path. **Typical:** 5000 principals with 10 grants each, 1 in 10 with a hand-down chain of depth 3; 1 in 10 checks is for something not held. **Worst case** (change 6): the same store with one agent holding 5000 grants, a depth-3 chain, and half its grants withdrawn; every check is that agent's (or the chain's deepest child). Checks run on one goroutine while a second applies one event per 200 checks, paced by checks done so every engine faces the same writes. 50000 checks per run after 2000 warm-up checks, except the worst case on the full path: 30 checks per run with one event per 3 checks, reported as median and max, not p99, and not judged for stability. Before each run the harness forces a garbage collection, returns memory to the OS and waits 100 ms. Each measurement is 3 runs; the median p99 is reported. If the fastest and slowest p99 differ by more than 20%, the measurement is unstable and repeated, at most 3 attempts. Paths: 'full' calls the engine once per candidate grant; 'prefiltered' first skips grants whose own scope can't cover the resource, the same for every engine, and changes no decision (TestPrefilterChangesNoDecision).

| Case | Path | Engine | p50 | **p99 (median of 3)** | p99 of the runs | stable | attempts | p99.9 | max | bytes/check | allocs/check | list of 10,000 items | events |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| typical | full | cedar (candidate) | 31µs | **259µs** | 222µs, 259µs, 259µs | yes | 1 | 1.035ms | 3.896ms | 25642 | 356 | 51ms (5000 allowed, one check per item) | 250 |
| typical | full | opa (candidate) | 171µs | **705µs** | 693µs, 705µs, 706µs | yes | 1 | 1.798ms | 7.14ms | 76492 | 1515 | 81ms (5000 allowed, partial evaluation) | 250 |
| typical | full | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 5µs | **30µs** | 24µs, 30µs, 34µs | **no** | 3 | 286µs | 1.479ms | 2662 | 53 | 2ms (5000 allowed, one check per item) | 250 |
| typical | prefiltered | cedar (candidate) | 6µs | **41µs** | 39µs, 41µs, 43µs | yes | 2 | 337µs | 2.323ms | 4464 | 64 | 57ms (5000 allowed, one check per item) | 250 |
| typical | prefiltered | opa (candidate) | 29µs | **91µs** | 91µs, 91µs, 98µs | yes | 2 | 507µs | 3.134ms | 14657 | 292 | 83ms (5000 allowed, partial evaluation) | 250 |
| typical | prefiltered | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 2µs | **19µs** | 13µs, 19µs, 19µs | **no** | 3 | 200µs | 3.022ms | 636 | 14 | 5ms (5000 allowed, one check per item) | 250 |
| worst | full | cedar (candidate) | 28.405ms | n/a (30 checks) | 35.77ms, 36.123ms, 38.494ms | n/a | 1 | 36.123ms | 38.218ms | 12390551 | 172955 | 55ms (5000 allowed, one check per item) | 10 |
| worst | full | opa (candidate) | 140.891ms | n/a (30 checks) | 147.298ms, 151.717ms, 153.527ms | n/a | 1 | 151.717ms | 162.634ms | 34240185 | 671742 | 91ms (5000 allowed, partial evaluation) | 10 |
| worst | full | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 3.109ms | n/a (30 checks) | 6.357ms, 7.449ms, 7.932ms | n/a | 1 | 7.449ms | 7.951ms | 1254042 | 25055 | 4ms (5000 allowed, one check per item) | 10 |
| worst | prefiltered | cedar (candidate) | 68µs | **415µs** | 401µs, 415µs, 439µs | yes | 1 | 1.348ms | 3.148ms | 45234 | 62 | 60ms (5000 allowed, one check per item) | 250 |
| worst | prefiltered | opa (candidate) | 124µs | **385µs** | 365µs, 385µs, 405µs | yes | 3 | 1.477ms | 3.533ms | 54655 | 272 | 83ms (5000 allowed, partial evaluation) | 250 |
| worst | prefiltered | goref (reference, not a candidate (written by Claude, reviewed by Skander)) | 61µs | **364µs** | 352µs, 364µs, 392µs | yes | 1 | 1.273ms | 3.825ms | 41472 | 13 | 6ms (5000 allowed, one check per item) | 250 |

### Against the pre-registered hard thresholds

The decision is taken on the prefiltered path. Rows threshold: 25 of 25, with 0 wrong allows.

- **cedar** (candidate): rows 25/25: pass. Typical p99 41µs: pass. Worst-case p99 415µs: **fail**. **The typical full path exceeds 250 µs** (p99 259µs), so it passes only with the prefilter (change 6, point 3). Worst case on the full path, reported only: median 28.405ms, max 38.218ms per check.
- **opa** (candidate): rows 25/25: pass. Typical p99 91µs: pass. Worst-case p99 385µs: **fail**. **The typical full path exceeds 250 µs** (p99 705µs), so it passes only with the prefilter (change 6, point 3). Worst case on the full path, reported only: median 140.891ms, max 162.634ms per check.
- **goref** (reference, not a candidate (written by Claude, reviewed by Skander)): rows 25/25: pass. Typical p99 19µs: pass (**unstable** after 3 attempts). Worst-case p99 364µs: **fail**. Worst case on the full path, reported only: median 3.109ms, max 7.951ms per check.

Change 6, point 4: the 250 µs threshold applies to the **typical case only**, so the worst-case verdicts above are reported, not decisive. A worst-case failure that also affects the own-Go reference is a **layer finding** (E16) and doesn't trigger the "both fail" fallback. An unstable measurement supports no conclusion.

The choice between cedar-go and OPA is made by Skander against spike-preregistration.md, not by this program.

## Runs 5 and 6 side by side (change 8)

Only `policies.cedar` differs between the two runs. Run 6 uses the version rewritten for SymCC. The conditions are the same: pinned to CPUs 1–3, `GOMAXPROCS=3`, settling before each run, 50,000 checks per run, 3 runs each.

| Measure | cedar-go, run 5 | cedar-go, run 6 | OPA, run 5 | OPA, run 6 |
|---|---|---|---|---|
| rows | 25/25 | 25/25 | 25/25 | 25/25 |
| **typical prefiltered p99 (median of 3)** | **40 µs** (39, 40, 41), stable | **41 µs** (39, 41, 43), stable | **88 µs** (87, 88, 94), stable | **91 µs** (91, 91, 98), stable |
| OPA ÷ cedar-go, typical prefiltered | | | **2.2×** (88 ÷ 40) | **2.2×** (91 ÷ 41) |
| typical full p99 | 249 µs | **259 µs** (over 250) | 841 µs | 705 µs |
| worst prefiltered p99 (reported only) | 479 µs, unstable | 415 µs | 378 µs | 385 µs |
| list of 10,000 | 57 ms, one check per item | 57 ms, one check per item | 89 ms, partial evaluation | 83 ms, partial evaluation |

**The decision doesn't change.** Under the same rule, both candidates pass every hard threshold, OPA's typical prefiltered p99 is still more than twice cedar-go's (2.2×), so speed decides earlier, and cedar-go is still the choice.

**One statement does change.** In run 5, cedar-go's typical full path was 249 µs, just under 250. In run 6 it's 259 µs, just over. So cedar-go, like OPA, **passes only with the prefilter** (change 6, point 3). The difference is within the noise between runs, and it doesn't affect the decision, which is taken on the prefiltered path.
