---
title: "Decision table: 25 checks, 5 of them attacks"
created: 2026-11-17
status: reviewed by Skander, 2026-11-17 (attacks chosen by Skander; rows 6–25 drafted by Claude from the PRD and corrected by Skander)
sources: PRD prd-countersign-2026-11-06 (FRs, NFRs), PRD addendum (decision-table seeds)
---

# Decision table: 25 checks, 5 of them attacks

Each row is one test, a "check" in Orvex's sense of the word. It may exercise a check, a request, a hand-down, a signal or the replica. **Where decided** says which part produces the answer:
- **engine:** the policy engine under test (cedar-go, OPA, or the own-Go reference).
- **store:** a write-time rule. The store asks the **same engine** ("may X approve request R?", "may X hand down G?"), with the fixed rules written as engine policies. So attacks exercise the engine too. Go types stay as a second layer.
- **replica:** the Countersign layer around the engine on the replica, such as staleness or the chain overlap.

Results are filled in by the spike, against the pre-registration in `spike-preregistration.md`. The full run, with each row's notes and the measurements, is in `spike-results.md` (2026-11-17). Rows 22 and 24 were replaced after the first measurement (spike-preregistration.md, change 4); the originals are below.

## Attacks (chosen by Skander)

| ID | Scenario | Call | Expected | FRs | Where decided | Why it's an attack | cedar-go | OPA | own Go |
|---|---|---|---|---|---|---|---|---|---|
| 1 | A request names its own approver | request | refuse | FR-1 | store | Fails closed on a crafted request, the brief's "however it is worded" | ✅ pass | ✅ pass | ✅ pass |
| 2 | A parent agent hands down a contained grant without "allowed to share" | hand-down | refuse | FR-8, FR-14 | store | Danny: holding access never lets you pass it on, and "contained" gives no sharing authority | ✅ pass | ✅ pass | ✅ pass |
| 3 | A parent agent hands a child agent a grant that is wider (resource `api/**` when the parent holds `api/billing`) or ends later than its own | hand-down | refuse | FR-14 | store, plus engine at check time (overlap) | "Never widens" | ✅ pass | ✅ pass | ✅ pass |
| 4 | The requester approves their own request | request (approval) | refuse | FR-12 | store | "No self-approval" | ✅ pass | ✅ pass | ✅ pass |
| 5 | An agent approves a human-only action, however the approval is worded | request (approval) | refuse | FR-11 | store | The other half of "no self-approval" | ✅ pass | ✅ pass | ✅ pass |

## Normal and edge cases (only rows 1–5 are attacks)

| ID | Scenario | Call | Expected | FRs | Where decided | cedar-go | OPA | own Go |
|---|---|---|---|---|---|---|---|---|
| 6 | `api-refactor` requests a push to `feature/retry`, a contained action (UJ-1) | request | approved by rules within 2 s `[tentative]` | FR-10 | store | ✅ pass | ✅ pass | ✅ pass |
| 7 | `api-refactor` checks that push, holding the grant from row 6 | check | allow, naming the grant | FR-2, FR-21 | engine | ✅ pass | ✅ pass | ✅ pass |
| 8 | A principal checks an action it holds no grant for | check | refuse: "no grant; make a request" | FR-21, NFR-1 | engine | ✅ pass | ✅ pass | ✅ pass |
| 9 | `db-migrate` requests read access to the production orders secret (UJ-2) | request | ask, with authorised people listed as approvers | FR-5, FR-11 | store | ✅ pass | ✅ pass | ✅ pass |
| 10 | Erin, who has approval authority, approves row 9, then `db-migrate` checks | request, then check | approved, then allow, naming the grant and Erin | FR-11, FR-2, FR-23 | store, then engine | ✅ pass | ✅ pass | ✅ pass |
| 11 | Erin cuts the grant from row 10 from 2 hours to 1, then a request to extend it is made | narrow, then request | narrowing accepted; extension refused (a new request is needed) | FR-4 | store | ✅ pass | ✅ pass | ✅ pass |
| 12 | A request for longer than the hard maximum lifetime | request | refuse, stating the maximum | FR-1, FR-3 | store | ✅ pass | ✅ pass | ✅ pass |
| 13 | A check after the grant's time limit has passed | check | refuse | FR-3 | engine | ✅ pass | ✅ pass | ✅ pass |
| 14 | A "task closed" signal for ENG-88, then `db-migrate` checks | signal, then check | refuse within 10 s `[tentative]` of the store accepting the signal | FR-20 | store, then engine | ✅ pass | ✅ pass | ✅ pass |
| 15 | Withdraw a parent grant with three child grants and one grandchild grant (UJ-3) | withdrawal, then checks | all five refused within 10 s `[tentative]` | FR-15, FR-19 | store, then engine | ✅ pass | ✅ pass | ✅ pass |
| 16 | A parent grant is narrowed from `api/**` to `api/billing`; its child grant then checks `api/payments` and `api/billing` | check | refuse `api/payments`, allow `api/billing` | FR-15, FR-14 | engine (overlap at check time) | ✅ pass | ✅ pass | ✅ pass |
| 17 | A parent agent hands down part of a human-only grant with "allowed to share" and depth 1 | hand-down, then check | child grant created; the check allows within the parent's scope | FR-8, FR-14, FR-18 | store, then engine | ✅ pass | ✅ pass | ✅ pass |
| 18 | The child grant from row 17 tries to hand down again (depth exhausted) | hand-down | refuse | FR-18 | store | ✅ pass | ✅ pass | ✅ pass |
| 19 | A person with authority marks an action contained, then a request for that action is made | settings write, then request | marking accepted; request approved by rules | FR-6, FR-10 | store | ✅ pass | ✅ pass | ✅ pass |
| 20 | An administrator marks a floor-of-four action contained by mistake | settings write | refuse | FR-6 | store | ✅ pass | ✅ pass | ✅ pass |
| 21 | Erin's approval authority expires; her earlier grants are checked, and she approves a new request | check, then request (approval) | earlier grants still allow; the new approval is refused | FR-7 | engine, then store | ✅ pass | ✅ pass | ✅ pass |
| 22 | A parent agent hands down a contained grant within the standing sharing limit, then the child grant checks | hand-down, then check | child grant created; the check allows within the parent's scope | FR-8, FR-14 | store, then engine | | | |
| 23 | A list check over a feed of 10,000 items | list check | exactly the items that single checks would allow, and one explanation. **Measured per engine:** correctness and speed over 10,000 items | FR-22 | engine-specific list filter. OPA: partial evaluation turning the agent's grants into a filter. cedar-go: its batch package or one check per item (both measured; report says which). Own Go: a prefix filter (reference) | ✅ pass | ✅ pass | ✅ pass |
| 24 | An "agent ended" signal; a grant created before it is checked, then a grant created after it | signal, then checks | the earlier grant is refused; the later grant is allowed | FR-20, FR-3 | engine | | | |
| 25 | A person authorises an agent to create service grants; the agent proposes one for access it holds, owned by a team, and a person with approval authority approves it | settings write, service-grant proposal, approval | authorisation accepted; the proposal returns "ask" (people only); after approval, the service grant is created with the team as owner, no wider than the agent's access | FR-16 | store | ✅ pass | ✅ pass | ✅ pass |

## Candidates not included (could replace a row)

- A person without approval authority approves a request, and is refused (FR-11).
- An agent tries to mark an action contained, and is refused (FR-6).
- An agent creates a service grant without authorisation, and is refused (FR-16).
- **Splitting:** `helper` writes ten contained config files, then the merge into production is checked (FR-13). `[OPEN, to check with Danny]` This depends on the production gate (PRD §9, question 6b).
- Access added to OpenBao by hand is detected (FR-28). This is an OpenBao test, not an engine test.
- A contained request longer than 4 h is refused (FR-10).
- A check with an unauthenticated identity is refused (NFR-4).
- An untrusted closing signal is accepted and recorded with its source (FR-20).

## Tracked apart from the pass/fail count

- **Original row 22: a replica cut off from the store after a withdrawal (NFR-2, FR-19).** High-risk checks are refused once the first horizon (10 s or less) passes, and contained checks are allowed until the second horizon. This belongs to the **replica layer** (staleness horizons, heartbeat), not to any engine. **Not tested in the spike**, which has no replica or replication. It must be tested in the layer's replica tests, before v1.
- **Original row 24: replaying a check from before a withdrawal, and one from after it (FR-24).** Before, it allows; after, it refuses, citing the withdrawal. This belongs to the **layer**: rebuild the store's state up to a log position, then ask the engine. **Not tested in the spike**, which has no state rebuild by log position. It must be tested in the layer's replay tests, before v1.
- **First result with the original rows (2026-11-17):** every engine scored 23 of 25 (rows 22 and 24 not measured), with 0 wrong allows.

- **Sam's case:** Sam starts `api-refactor`, which asks to deploy to production, and Sam approves. This depends on Danny's answer (PRD §9, question 4), so it is reported separately and not counted.

**Row 23 results** must say *how* each engine got its answer, so a speed difference isn't mistaken for a correctness difference.
