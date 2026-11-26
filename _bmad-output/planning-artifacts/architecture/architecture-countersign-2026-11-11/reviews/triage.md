# Reviewer gate: triage (2026-11-17)

Five reviewers each read the spine in a fresh context. Each reviewer's own wording is in its file. This page records what happened to every finding.

| Reviewer | File | Verdict, in the reviewer's words (shortened) | Counts |
|---|---|---|---|
| Version check | review-versions.md | "The technology choices are real and current… One thing needs changing: Go 1.27.3 is a security patch behind." | 0 critical, 1 high, 3 medium, 6 low |
| PRD check | review-reconcile-prd.md | Matches the PRD on the core, "but it drops NFR-5 rule 3 entirely", and FR-7 and other authority rules have no AD | 1 critical, 3 high, 15 medium, 8 low |
| Evidence check | review-reconcile-evidence.md | "The spine's numbers match run 6. But it overstates four things" | 0 critical, 6 high, 10 medium, 9 low |
| Checklist walker | review-rubric.md | "Sound on the decision core but not yet a safe build substrate" | 2 critical, 12 high, 11 medium, 5 low |
| Two-teams attack | review-adversarial.md | "No. Two independent teams building from this spine would not end up with the same behaviour, and five of the gaps let teams that both follow every AD produce a wrong allow." | 5 critical, 10 high, 8 medium, 3 low |

Claude checked these claims directly before fixing:
- Rules 8 and 9 are `permit` conditions, not `forbid`s.
- Go 1.27.4 exists.
- cedar-go v1.9.2 ships `x/exp/batch`.
- The reference engine measured 30 µs (run 6) and 34 µs (run 5) on the full path.
- `classification_proposed` isn't in grant-schema.md.

The spine as it was before the gate is saved in the session scratchpad.

## Fixed in the spine (corrections of fact or evidence; no design choice)

| Finding | Fix |
|---|---|
| V1 | Stack: Go 1.27.4, with a note that the spike ran on 1.27.3 |
| V3 (fact part) | AD-13: OpenBao enforces `expiration` on its own clock and deletes expired policies itself, so a missing policy isn't drift |
| V4, E10 | AD-8: that Rust cedar-policy and cedar-go agree is stated as **assumed, not tested** |
| V5, E9 | AD-6: `x/exp/batch` exists, is experimental, and wasn't used |
| V7 | Stack: PostgreSQL 14 or later |
| V8, V9 (upgrade part) | AD-8 and Stack: cvc5 stays at 1.3.1, and SymCC, cedar-policy and cvc5 are upgraded together |
| P1, E21 | AD-5: rule 3, the hard maximum, is a write-time refusal, binding FR-1 and FR-3 |
| P12 (statement), E2 (statement) | AD-8: what the proofs don't cover (agent requesters, checks by people, actions other than `push`) |
| E1, R4 (statement) | AD-8 retitled. It states which rules are `forbid`s and which are `permit` conditions, and that the latter come back as `default-deny` |
| E3 | AD-9 Prevents: replaced with the true figures |
| E4, R5 (statement) | AD-6: lists the writes that are decided only in Go today |
| E5, E14 | AD-4: tagged "decided, not measured"; row 22 must pass before v1 |
| E6, E14 (replay obligation) | AD-14: row 24 must pass before v1 |
| E7 | AD-4: the contained exception to "100% within 10 s" is stated |
| E8 | AD-6: the thin margin (2.2× against 2×); SymCC decided nothing |
| E11 | AD-9: the replica's state is immutable and swapped atomically; the spike's p99 includes lock contention |
| E12 | AD-9: the scope index was never built |
| E13, R28 | AD-17: the reflection test isn't built yet |
| E15 | Deferred: the brief's "parent agent may approve" was reinterpreted, and is still open with Danny |
| E16 | AD-18: the production gate is a deviation from the brief's v1 commitment |
| E18 | AD-9: the harness prints "MB" but divides by 2^20; store and replica are counted together |
| E19 | AD-9: worst-case full-path memory as measured (12,390,551 bytes, 172,955 allocations) |
| E20 | AD-9: one 4-vCPU virtual machine, a single process |
| E22 | AD-12: tagged "partly measured" |
| E24, R24 | Deferred: a time budget for lists, with run 6's 51–60 ms |
| E25 | Frontmatter: spike-results.md added to the sources |
| R3 (names part) | Events convention: the names now match grant-schema.md, and `classification_proposed` is removed |
| R19, A23 | AD-16: "authority over a resource" means administrators only in v1 (overrule 4) |
| R26 | Deferred: the service grant's hard maximum (q7) |

**Not done yet:**
- **V6:** add the OpenBao Go client to the Stack (not verified).
- **V10:** research-engines.md still lists older versions.
- **P13:** the Deferred list doesn't name §9 questions 7, 9, 13 and 14 one by one. "Every question to Danny in PRD §9" covers them.

## Decided by Skander (2026-11-17), now in the spine

- **D1: a synchronous store call for high-risk records.** AD-1 retitled and amended, AD-14, AD-4, the diagram. 250 µs applies to contained checks.
- **D3: one projection package.** New AD-19, AD-11, AD-5, AD-3, the layer table and package tree, and grant-schema.md's `narrowed`.
- **D4: freshness counts applied events.** AD-4: the freshness instant, the start-up rule, the store's `now`, and a layer test.
- **D5: Keycloak `sub`, `namespace.serviceaccount`, mTLS.** AD-18 and the Identities convention. NFR-4's last bullet becomes a product obligation, a PRD change to raise.

- **D6: OpenBao policies follow the projection.** AD-13: effective end minus margin, ancestors, a 10 s lag target, `ToOpenBao`, an agreement test and a 5,000-policy measurement before v1.
- **D2: a mismatched policy hash refuses every check.** New AD-20.
- **D14: an advisory lock lease.** AD-1, AD-2 and Deferred.
- **D7: the contract rules are in the spine.** New AD-21, the `proto/` package, the Events convention and AD-3.

- **D8: the store sets limits, and the log carries them.** New AD-22.
- **D9: strictest wins.** AD-16; markings get IDs.
- **D10: lists filter items and are refused whole only on errors.** AD-15 retitled.
- **D11: read-your-writes by position and wait.** AD-3.

- **D12: a fixed `Decision` shape, and replay by position.** New AD-23; AD-12.
- **D13: namespaced tasks and resources, and `task_opened`.** Conventions, AD-18.
- **D15: keep these writes in Go, stated.** The FR-4, 5, 7, 8, 9 and 16 write rules are in AD-5; AD-6.
- **D16: operations deferred, with owners.** Deferred.

- **D17: widening the proofs is v1 CI work, stated.** AD-8.
- **D18: keep the permit conditions, plus a CI check for one `permit` per action.** AD-8.
- **D19: all four questions go to the README's open questions.** Deferred.
- **D20: the small items, applied as one batch, each checked against the PRD first:**
  - P7: the one-month replay floor (AD-23)
  - P11: depth 3 is tentative (AD-8)
  - P16: FR-10 (AD-5)
  - P20: the flood alert (AD-18)
  - P21: task and agent validity (AD-5)
  - P22: missing fields refuse (AD-5)
  - P23: the contained 10 s holds only on connected replicas (AD-4)
  - P24 and A25: the margin is added before truncating (AD-7)
  - P25: tighter binds (AD-1, AD-5, AD-10, AD-11, AD-16, AD-17)
  - P26: one list explanation (AD-15), the refusal names the restriction (AD-13), marking audit (AD-16)
  - V6: the OpenBao Go client v2.7.1, checked on the Go proxy (Stack)
  - V10: research-engines.md versions

**All 20 decisions are made. No finding is left open.**

## For Skander to decide (all now decided)

Each decision lists the findings it closes and Claude's recommendation. Nothing below has been applied.

| # | Decision | Findings | Claude's recommendation |
|---|---|---|---|
| D1 | Where high-risk decision records are written before the answer, given that AD-1 allows no network call | R1, A1, P2, R20, E6, P18 | One synchronous store call for high-risk checks only. It's the only network call allowed in the check path. 250 µs applies to contained checks. A store that's down refuses high-risk checks |
| D2 | One policy version per installation | R2, A7, P6 | The store announces the required policy hash in the snapshot and heartbeat. A replica with a different hash refuses every check. Records carry the hash and the library version |
| D3 | One projection package used by both the store and the replica | A3, R9, A20 | Yes. A `projection/` package; `narrowed` carries the full effective scope and end; snapshot-then-events must equal a full fold |
| D4 | Staleness counts applied events, not received heartbeats | A2, A17, A19, R17, R25 | Yes. A heartbeat refreshes a replica only once its position is applied. A replica with no heartbeat since start-up is stale. The store uses the same `now` rule as AD-4 |
| D5 | Identity mapping and channel authentication | A4, A26, R7, R27, R30, P4 | `person:<Keycloak sub>`; `agent:<namespace>.<serviceaccount>`; one mapping function in `model/`; the store accepts only mTLS from the products' service identities. NFR-4's "unauthenticated identity refused" becomes a product obligation, which needs the PRD owner's sign-off |
| D6 | OpenBao reconciliation | A5, A16, R14, V2, V3, P8, P9 | Policies follow the projection: effective end minus margin, removed when any ancestor ends, with a 10 s lag target. One `ToOpenBao` mapping in `model/`. The per-entity policy count is measured. Plus an agreement test against a real OpenBao |
| D7 | The wire contract and the event list | A6, R3, R15, R22, A18, A21, A22, A24, R12 (rewind) | A `proto/` package; a closed event list with request, task and classification events; resume is exclusive; a replica halts on an unknown event or a rewind; a closed outcome set; idempotency keys; store-assigned unique IDs |
| D8 | Limits are installation state | A8, R18 | Set at the store as a `limits_set` event. Replicas have no local configuration for them |
| D9 | How markings combine into a class | A9 | Strictest wins, as in the spike's `classOf`; markings have IDs |
| D10 | What a list refuses | A10, E23 | Items a single check refuses are filtered out. The whole list is refused only for malformed input, an engine error, the second horizon or the deadline |
| D11 | Read-your-writes | A11 | Writes return their log position. Checks may wait for that position, then refuse with `replica.behind` |
| D12 | The explanation contract and replay | A12, A13, R6, R16 | A fixed `Decision` shape; the lowest log position is named on allow; a reason precedence; closed layer-refusal IDs. Records carry the applied position. Replay is a fold up to that position |
| D13 | Tasks and resource namespaces | A14, A27, R8 | `<product>:<task-id>`; registered top-level resource segments; a `task_opened` event; no reopening in v1 |
| D14 | Enforcing the single writer | A15, R13 | A PostgreSQL advisory lock as the leader lease. Write rules are checked inside the appending transaction. "HA is an operations choice" becomes "one store process per installation in v1" |
| D15 | The writes decided only in Go: move them into the engine, or keep them in Go | E4, R5, P3, P5, P10, P14, P17 | Your choice. Moving them adds engine methods and proofs; keeping them is stated and covered by AD-17's layer tests |
| D16 | The operational envelope: deployment, observability and alerts, backup and restore, migrations, tamper alerts | R10, R11, R12, R21, R23, P19 | List them as deferred, with owners, rather than decide them now (feature altitude) |
| D17 | Widening the proofs | E2, V4, V9 | Optional spike work: `Grant.principal: Person \| Agent`, all six check actions, `check_never_errors`, and a cedar-go vs Rust conformance test |
| D18 | Rewriting rules 8, 9, 11 and 13 as `forbid`s | E1, R4 | Your choice. Either rewrite them (then re-prove and re-measure), or keep them as permit conditions with a CI check that each action has exactly one `permit` |
| D19 | Questions for the PRD or for Danny | P27, R29, E17 (FR-17 vs FR-16); P4; P12 (seed 2); P15 | Add them to the README's open questions |
| D20 | Small items | P7 (a one-month replay floor), P11 (depth 3 hard-coded), P16, P20–P26, A25 (margin before truncation) | Apply them as one batch once D1–D14 are settled |

## Second pass: the two-teams reviewer again (review-adversarial-2.md)

**Verdict, in the reviewer's words:** "No. Two teams following every AD exactly would still behave differently, and at least four of the gaps (B1–B4) can produce a wrong allow under the literal reading. B5–B10 can too, depending on which reading a team picks."

It raised 3 critical, 8 high, 11 medium and 5 low findings. Several came from how Claude worded decisions D1, D3, D6 and D7.

**Fixed by Claude without a decision:**
- **B23:** the diagrams. The guard now calls the engine port, the adapter implements it, and Keycloak and Kubernetes feed both products.
- **B24:** `Lister` removed.

**Fixed, by Skander's choice:**
- **B1:** a fenced writer (AD-1)
- **B2:** `EnforcedScope` and `EnforcedEnd` (AD-13, AD-19)
- **B4:** "task closed" isn't positional; "agent ended" is positional (AD-11, Tasks convention)
- **B5:** the hard maximum is write-time only, the hand-down class, and a v1 `forbid` for grants approved by rules (AD-5, AD-8)
- **B6:** the heartbeat's order (AD-3)
- **B7:** validated limits, and the offset refusal (AD-22, AD-4)
- **B8:** the decision-code version (AD-20)
- **B9:** one owner for the snapshot (AD-19, AD-21)
- **B25:** D18 corrected to one `permit` per action and principal type (AD-8)

**Fixed on its own, at Skander's request (2026-11-17):** AD-14 had approvals and withdrawals recorded by a synchronous replica call. They're store writes, so their record is the event in the write's own transaction. B10 is unchanged and still open.

**Open, by Skander's choice to stop the gate here (no third pass), listed in the spine's Deferred and for the README:**
- **B10 (high):** the high-risk record call (re-deciding, where records live, retries, compact records, lists).
- **B11 (high):** candidates and tie-breaks in explanations.
- **B12–B22 (medium):**
  - B12: events missing from the closed list
  - B13: the registry against the stateless Guard
  - B14: the store's `now`
  - B15: retention against append-only
  - B16: OpenBao's clock, and grants held by people
  - B17: the store client's owner
  - B18: the format of Keycloak's `sub`
  - B19: the approver ID for contained auto-approval
  - B20: FR-4 and the parent's holder
  - B21: the outage during a store-first rollout
  - B22: the scope of idempotency keys
- **B26, B27 (low):** a task's product prefix against the product's identity; when a duration starts, and repeated withdrawals.

## B3 settled (2026-11-26), decided, not built, not measured

Skander settled B3 with the full fix. It's in AD-1 (the head hash), AD-2 (history ID and hash chain), AD-3 (resume, heartbeat and snapshot carry both), AD-21 (Halted, the layer test and the known limit), AD-23 (records carry the history ID), the new IDs convention (ULIDs, never counters), and Deferred (the restore procedure mints the ID; the interim rule until it's built).

AD-1 and AD-14 were checked. AD-1's compare-and-append now also checks the head hash. AD-14 has no contradiction; the record's history ID is in AD-23.

The earlier attempt to settle B3 and B10, which Skander withdrew, stays in the working log. B10 is still open.
