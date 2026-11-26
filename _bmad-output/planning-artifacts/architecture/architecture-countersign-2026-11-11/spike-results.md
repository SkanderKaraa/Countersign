# Spike results

Run on 2026-11-17 19:31 UTC. Go go1.27.3, linux/amd64, 3 CPUs available to the process, GOMAXPROCS=3. Engines: cedar-go v1.9.2, OPA v1.22.4, and the own-Go reference. The machine is a VM with 4 virtual CPUs on a shared host, which shows CPU steal time; pinning reduces noise inside the VM only (change 7).

## Decision table: 25 checks, 5 of them attacks

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

## Measurements at 50,000 grants

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

## Against the pre-registered hard thresholds

The decision is taken on the prefiltered path. Rows threshold: 25 of 25, with 0 wrong allows.

- **cedar** (candidate): rows 25/25: pass. Typical p99 41µs: pass. Worst-case p99 415µs: **fail**. **The typical full path exceeds 250 µs** (p99 259µs), so it passes only with the prefilter (change 6, point 3). Worst case on the full path, reported only: median 28.405ms, max 38.218ms per check.
- **opa** (candidate): rows 25/25: pass. Typical p99 91µs: pass. Worst-case p99 385µs: **fail**. **The typical full path exceeds 250 µs** (p99 705µs), so it passes only with the prefilter (change 6, point 3). Worst case on the full path, reported only: median 140.891ms, max 162.634ms per check.
- **goref** (reference, not a candidate (written by Claude, reviewed by Skander)): rows 25/25: pass. Typical p99 19µs: pass (**unstable** after 3 attempts). Worst-case p99 364µs: **fail**. Worst case on the full path, reported only: median 3.109ms, max 7.951ms per check.

Change 6, point 4: the 250 µs threshold applies to the **typical case only**, so the worst-case verdicts above are reported, not decisive. A worst-case failure that also affects the own-Go reference is a **layer finding** (E16) and doesn't trigger the "both fail" fallback. An unstable measurement supports no conclusion.

The choice between cedar-go and OPA is made by Skander against spike-preregistration.md, not by this program.
