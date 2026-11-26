---
title: "Rubric review: Countersign architecture spine"
reviewed: ARCHITECTURE-SPINE.md (updated 2026-11-17)
date: 2026-11-17
reviewer: independent reviewer (Claude), not the spine's author
method: good-spine checklist (divergence points, enforceable Rules, Deferred safety, verified tech, ratifies spike code, PRD coverage, operational envelope)
---

# Rubric review: Countersign architecture spine

## Verdict

The spine pins down the decision core well: the engine port, the input guard, the Cedar proofs, the immutable grants, the event log and the trust boundary. Most of it ratifies the spike faithfully. It is **not yet a safe build substrate**. Two Rules contradict each other on where decision records are written. Nothing stops Quornex, Houston and the store from running different policy versions. The event contract between store and replica has no schema or versioning. And the operational envelope (deployment, observability, backup and restore, upgrades) is almost entirely silent.

**Counts:** 2 critical, 12 high, 11 medium, 5 low.

## How to read the evidence

- `SPINE` = `ARCHITECTURE-SPINE.md` in this folder
- `PRD` = `prds/prd-countersign-2026-11-06/prd.md`
- `GS` = `grant-schema.md`
- `KL` = `spike/KNOWN-LIMITS.md`
- spike code paths are relative to `/home/skander/countersign/spike/`

## Checklist summary

| Criterion | Result |
|---|---|
| Fixes the real divergence points, misses none | **Partly.** Misses: where decision records are written (R1), version skew (R2), the event schema (R3), the write rules outside the engine (R5), replay (R6), store authentication (R7), task validity (R8), shared store/replica logic (R9), grant ordering (R16) |
| Every Rule is enforceable and prevents its divergence | **Partly.** AD-8 claims a structure the code doesn't have (R4). AD-6 claims every write rule goes through the engine (R5). AD-4's staleness reference point is undefined (R17) |
| Nothing under Deferred lets two units diverge | **No.** Store HA (R13), E17 caller authentication (R7) and where the horizon values live (R18) can each make units diverge |
| Named tech is verified-current | **Yes.** Go 1.27.3, cedar-go v1.9.2, gRPC-Go v1.85.0, pgx v5.13.0, OpenBao v2.7.2, SymCC 0.9.0 and cvc5 1.3.1 match go.mod, Cargo.toml, the module cache and .memlog entries 34 and 81. PostgreSQL has no minimum version (R23) |
| Ratifies the spike code rather than contradicting it | **Mostly.** Contradicted or overclaimed: AD-8's "each fixed rule is a forbid" (R4) and AD-6's "write-time rules go through the same engine" (R5). Decided in code but missing from the spine: "authority over a resource" means administrators only (R19) |
| Covers FR-1..FR-29 and NFR-1..NFR-6 | **Gaps:** FR-24 replay (R6), FR-3 task and agent validity (R8), the FR-9/16/20/25/28 alerts (R11), FR-25 tamper recording (R21), FR-29 record storage (R1, R20), NFR-4 version consistency (R2) |
| Operational envelope decided, deferred or open | **No.** Deployment and environments (R10), observability (R11), backup and restore (R12), HA (R13), upgrades and migrations (R2, R23) and event schema versioning (R3) are silent or only touched on |

## Findings

### R1 — critical — AD-1 vs AD-14 (and FR-29)
**Finding.** AD-14 requires high-risk decision records to be written durably **before the answer**, but AD-1 forbids any network call on a check, and the dependency rule says "the replica never writes" and "nothing but the store writes to PostgreSQL". So a replica has no permitted way to make a pre-answer durable write. The same gap leaves compact-record writers unspecified.
**Evidence.** SPINE:58 ("`check` and `list` run in-process … with no network call"); SPINE:50 ("the replica never writes, and nothing but the store writes to PostgreSQL"); SPINE:188–189 (records "written before the answer. A failed write refuses."); PRD:507 (FR-29: kinds kept in full are "recorded before the answer is returned"); PRD:614 (non-goal: not a central server).
**Suggested fix.** Add an AD that chooses one path and states its cost against NFR-3. Option (a): a synchronous gRPC "record" call to the store for full-record kinds only, so human-only checks leave the 250 µs budget, stated as an explicit exception to AD-1. Option (b): a local durable append-only spool per product process, which the store drains and owns as the only writer to PostgreSQL. Then amend AD-1 and the dependency rule to match. Name the owner of compact records (the replica's async batcher, sending to the store) and say what happens when that queue is full: refuse, or drop and count.

### R2 — critical — missing AD (versioning across Quornex, Houston and the store); NFR-4, FR-24, FR-29
**Finding.** `policies.cedar` is compiled into each binary with `go:embed`, and the action list is a closed compiled-in enum. So Quornex, Houston and the store can each run a different library and policy version against the same log. Nothing in the spine forbids it, detects it, or says which version a replay uses. That breaks NFR-4 ("the same check from Quornex and from Houston … gives the same decision") and leaves FR-29's "version of the fixed rules and of the library" with no source of truth.
**Evidence.** cedarengine/cedarengine.go:18–19 (`//go:embed policies.cedar`); model/records.go:413–425 (closed action list); SPINE:18–30 (no versioning AD anywhere); PRD:576 (NFR-4); PRD:501 (FR-29 records the fixed-rule and library versions).
**Suggested fix.** Add an AD for the **policy and library version contract**:
- The store publishes the policy version (a hash of `policies.cedar`, the schema and the action list) in the snapshot and in every heartbeat.
- A replica whose embedded version differs refuses everything (fail closed), or at least refuses high-risk checks.
- Every decision record carries that hash.
- Replay loads the policy text by hash from a store-kept archive.
- State an upgrade order (store first, then products, with a compatibility window) and a rule for adding actions.

### R3 — high — AD-2, AD-3, Consistency Conventions (Events)
**Finding.** The event log is the contract between the store team and the replica team, yet the spine fixes only event *names*. It leaves open the payload schema, the wire encoding, a schema version per event, what a replica does on an unknown event type or version, and the snapshot format. The names it lists don't even match its own cited source: `classification_proposed` is in the spine but not in grant-schema.md, and the spike also emits `resource_classified`, which neither document lists.
**Evidence.** SPINE:238 (events convention citing grant-schema.md); GS:121–134 (list without `classification_proposed`, `resource_classified`, request or proposal events); ref/state.go:47–53 (`Event{Type, Subject string}`, untyped); ref/writes.go:271, 278 (payload encoded as `"task_closed:"+task`); ref/writes.go:299, 365 (`classification_proposed`, `resource_classified`); SPINE:78 (snapshot format not specified).
**Suggested fix.** Add a Rule to AD-2/AD-3:
- Events are protobuf messages in a named `.proto` package, each with an `event_type`, a `schema_version`, `log_position`, `store_time`, `actor` and `source`.
- Fields are only ever added. A type is never repurposed.
- A replica that meets an unknown type or version stops applying events and fails closed (it must not skip).
- Snapshots carry the same version and the policy hash (R2).
- Make grant-schema.md's event list the single normative list, and add the classification and request events the code already emits.

### R4 — high — AD-8
**Finding.** AD-8's Rule says each fixed rule in Cedar is a `forbid` with an `@id` (rule 1 excepted). In the spike, rules 8 and 9 (check-time overlap and liveness), 11 (sharing needs authority) and 13 (agent service grants) are **permit conditions**, and rule 12 exists in Cedar only for markings, not for approval authority or "allowed to share". So refusals under rules 8, 9, 11 and 13 come back as `default-deny`. That makes AD-14's "fixed-rule refusals are written before the answer" uncomputable for them, and the convention "tests assert the ID" can't name them.
**Evidence.** SPINE:122 (the forbid claim); SPINE:118–121 (binds rules 1, 4–6, 8, 9, 11–13); cedarengine/policies.cedar:17–47 (rules 8 and 9 inside `check.grant-allows` permit); policies.cedar:85–96 (rules 8 and 11 inside `hand_down.within-parent` permit); policies.cedar:137–139 (rule 13 as a permit); policies.cedar:120–122 (rule 12 for markings only); spike-results.md rows 2 and 18 ("no policy permits this"); SPINE:188 (fixed-rule refusals written first); SPINE:237 (tests assert the ID).
**Suggested fix.** Either restate AD-8 truthfully, or make it true:
- **Restate:** "rules 4, 5, 6 and 12 (markings) are forbids; rules 1, 8, 9, 11 and 13 are permit conditions proved by SymCC properties". Then define how the layer classifies a `default-deny` refusal as a fixed-rule refusal for AD-14. One way is an extra forbid per rule that mirrors the permit's negated condition and carries its own `@id`.
- **Make it true:** rewrite those rules as forbids.

In both cases, take rule 12's authority and sharing halves out of AD-8's Binds until they exist in Cedar (see R5).

### R5 — high — AD-5, AD-6
**Finding.** AD-6 says "write-time rules go through the same engine", but the port has only five decision methods. In the spike, these rules are decided only in Go:
- who may narrow or withdraw (FR-4)
- giving approval authority (FR-7)
- setting a sharing limit (FR-8)
- adding and removing administrators (FR-9)
- classification and its two-administrator confirmation (AD-16)

The spine doesn't decide whether these move into the engine (new methods, new input structs, AD-17 registry entries) or stay in Go. So the engine team and the store team will diverge, and NFR-5 rule 12's authority and sharing halves stay outside the proofs.
**Evidence.** SPINE:103–105; engine/engine.go:97–104 (five methods); KL:15–18 ("Some writes aren't routed through the engine"); ref/writes.go:189–205 (`mayChange` in Go); ref/writes.go:391–423 (`GiveAuthority`, `SetSharingLimit` in Go); ref/writes.go:289–332 (classification in Go).
**Suggested fix.** List every write operation in AD-6 with its decider. Either add `MayChangeGrant`, `MayGiveAuthority`, `MaySetSharingLimit`, `MayClassify` and `MayChangeAdministrators` to the port (with AD-17 registry entries and SymCC properties), or state explicitly that these stay in Go and are held by tests only, and reclassify them in AD-5 and NFR-5.

### R6 — high — missing AD (FR-24 replay); also FR-23, FR-29, SM-6
**Finding.** No AD decides how replay works:
- which component rebuilds state at a log position (the store, a CLI, or a replica-in-a-box)
- whether it starts from the nearest snapshot or from position 0
- which policy version it uses (R2)
- how a compact record is turned back into a full explanation
- how "a replay that can't rebuild says so" is detected

The decision table explicitly defers replay to "the layer's replay tests" with no owner.
**Evidence.** PRD:471–478 (FR-24); decision-table.md:200 (original row 24 not tested, "must be tested in the layer's replay tests"); SPINE:164–170 (AD-12 covers only time); SPINE:155–156 (AD-11 binds FR-24 but gives no mechanism).
**Suggested fix.** Add an AD: replay = load the snapshot at or below position P, apply events up to P, load the policy by the record's hash, and run the same `replica` check code with the recorded `now`. Replay refuses with "can't rebuild" if any event, snapshot or policy hash is missing. Name the package that owns it, require a test equivalent to `TestReplayMatchesOriginal`, and tie snapshot retention to the one-month guarantee.

### R7 — high — AD-18, Deferred (E17); missing transport-security decision
**Finding.** The store is a network service whose write API takes identities as text, including the approver. Yet the spine decides no authentication or authorisation for the gRPC channel: no mTLS, no Kubernetes service-account token check, no list of permitted calling services. It also decides nothing for the confidentiality of the event stream, which carries every grant. Deferring E17 to "the product's authenticated session" works only if the store accepts calls from the products alone. As written, any pod that can reach the store could submit `Approve("person:erin", …)`, and the Quornex and Houston teams are free to convey caller identity in different ways.
**Evidence.** SPINE:227 (identity check is "the product's job in v1 (E17; deferred)"); SPINE:303 (E17 deferred); SPINE:39 (`SC -->|gRPC| S`, no auth); ref/writes.go:80 (`Approve(approverText, …)`); KL:10 (E17: "Every entry point trusts the identity it receives").
**Suggested fix.** Add an AD for **store API trust**:
- mTLS between library and store (or a projected Kubernetes service-account token verified by the store).
- An allow-list of calling workloads per installation (Quornex and Houston only).
- TLS on the replication stream.
- One fixed way the product passes the end-user identity (a signed field or a forwarded Keycloak token), so the later E17 work doesn't change the API.

### R8 — high — missing AD (task and agent validity); FR-3, FR-20, AD-18
**Finding.** FR-3 makes a grant invalid if its task or agent wasn't valid when the grant was made. But the spine never says where task and agent validity come from, or that opening a task is an event. In the spike, `OpenTask` mutates state **without appending to the log**. So a replica fed by the log, or a replay, can't reproduce task validity, and the two product teams have no contract for registering tasks.
**Evidence.** ref/state.go:197–203 (`OpenTask` sets `openTasks` with no `append`); ref/writes.go:47–49 (requests refused unless the task is open); PRD:190–192 (FR-3 task and agent validity rules); SPINE:229 (AD-18 lists only closing signals); GS:121–134 (no `task_opened` event).
**Suggested fix.** In AD-18, add a product duty to send "task opened" (and, if needed, "agent started") to the store. Add a `task_opened` event to the log. Decide whether validity can also be checked by asking the task system, and what happens when that system is unreachable (FR-3: a grant is not ended by an outage).

### R9 — high — Design Paradigm table, Structural Seed (replica/ vs store/)
**Finding.** The spine splits the spike's `ref/` into `replica/` (the check path) and `store/` (the write path). But the store's write rules call the check path and the projection:
- `CreatorHolds` runs a full check.
- Hand-down re-checks parent coverage.
- `classOf` and `approversFor` compute class and approvers.

The spine doesn't say the store reuses the replica's projection and check code. So two teams will each build chain facts, narrowing and class logic, and those two copies can diverge silently. AD-17's "layer computes every fact" doesn't say which layer.
**Evidence.** SPINE:29–30; SPINE:288–292; ref/writes.go:478 (`in.CreatorHolds = s.check(by, a, r, now).Allow`); ref/writes.go:48–66 (`classOf` in the request path); ref/classes.go:48–70.
**Suggested fix.** Add a Rule: there is one projection package (for example `state/` or `projection/`) that applies events and computes every fact. Both `replica/` and `store/` import it, and the store never reimplements a fact. Add a test that a store and a replica fed the same log produce identical facts at each position.

### R10 — high — Operational envelope: deployment and environments (silent)
**Finding.** The spine never decides how the store is deployed per installation:
- the Kubernetes workload kind, namespace and replica count
- how the embedded library finds the store (service discovery)
- where the store's own configuration lives (limits, horizons, OpenBao address and credentials)
- what environments exist (dev, staging, production) and how a test installation is built

It isn't listed under Deferred or open questions either.
**Evidence.** SPINE:257–284 (the Structural Seed shows boxes but no deployment unit or config); SPINE:296–306 (Deferred has no deployment item); ref/state.go:20–45 (`Limits` are code defaults, including a placeholder `ServiceMax`).
**Suggested fix.** Add a short Deployment AD: the store is one Kubernetes Deployment per installation (see R13 for the count), reached by a fixed Service name. Configuration is a versioned ConfigMap or Secret read only by the store, and products receive limits through the snapshot and heartbeat. The OpenBao credentials are the store's own Kubernetes auth role. Or list each of these explicitly under Deferred with an owner.

### R11 — high — Operational envelope: observability and alerting (silent)
**Finding.** Five FRs require alerts, and NFR-2 / SM-C4 require measuring the slowest replica. But the spine decides no metrics, logs, traces or alert sink, nor who emits each alert. The alerts are:
- FR-9: an administrator's authority lapsing
- FR-16: an agent creating a service grant
- FR-20: a signal flood
- FR-25: a tamper attempt
- FR-28: drift

Without that, each team will invent its own, or none.
**Evidence.** PRD:270 (FR-9 alert); PRD:382 (FR-16 alert); PRD:420 (FR-20 flood alert); PRD:488 (FR-25 alert); PRD:542 (FR-28 alert); PRD:666 (SM-C4); SPINE (no section mentions metrics, alerts or logs).
**Suggested fix.** Add an Observability AD:
- The store owns every alert and emits it as a log event (for example `alert_raised`) and as a metric, sent to the installation's existing alerting (name it, or ask Danny).
- A required metric set: replica lag in positions and seconds, staleness state, clock offset, check latency histogram, refusals by rule ID, reconciler lag, drift count, record-queue depth.
- OpenTelemetry or Prometheus, chosen and pinned in the Stack.

### R12 — high — AD-2, AD-3 (backup, restore, log rewind)
**Finding.** The spine doesn't decide backup and restore for the event log, which is the source of truth and the audit record. Nor does it say what a replica does when the store's head position is **lower** than the replica's last applied position, as after a point-in-time restore or a database switch. AD-3's "resume from last applied position" would then silently splice two histories, or hang.
**Evidence.** SPINE:67–70 (positions without gaps, but no epoch); SPINE:76–78 (resume from last position); SPINE:298–299 (PostgreSQL ownership and HA deferred, backup not mentioned).
**Suggested fix.** Add a Rule:
- The log carries an installation ID and a **log epoch** (a generation number) that changes on any restore.
- The heartbeat and stream carry both.
- A replica that sees a different epoch, or a head below its own position, discards its state and reloads a snapshot, refusing everything until it has.
- Also state the backup expectation (PostgreSQL PITR, owned by the installation's operators) and that a restore must never lose positions that a decision record already cites. If it does, replay must report "can't rebuild".

### R13 — high — Deferred ("Store high availability is an operations choice")
**Finding.** Calling store HA "an operations choice per installation" means an operator may run two store instances. That breaks single-writer assumptions the spine relies on:
- two OpenBao reconcilers fight each other (AD-13)
- two in-memory projections can each accept writes validated against different state
- pending requests and classification proposals held in memory diverge (ref/state.go:91–92)

This is an architecture decision, not an operations choice.
**Evidence.** SPINE:299; SPINE:52–58 (AD-1 single writer); ref/state.go:91–92 (`pending`, `proposals` maps in memory); SPINE:179 (AD-13 continuous reconciliation).
**Suggested fix.** Decide it in AD-1: "Exactly one active store per installation, enforced by a PostgreSQL advisory lock or lease. A standby holds no state and takes over only after acquiring the lock, rebuilding from the log." Leave only "whether to run a standby" as an operations choice.

### R14 — high — AD-13 (OpenBao mapping)
**Finding.** AD-13 fixes "one policy per grant, attached to the entity". But it leaves open:
- how `agent:<id>` maps to an OpenBao entity (the Kubernetes auth alias defaults to the service-account **UID**, not a name)
- how a canonical path and its `/**` map to OpenBao paths and globs
- how actions map to OpenBao capabilities
- how fast reconciliation must run against FR-26's 10 s removal target
- how the spine's own worst case (5,000 grants for one principal, so 5,000 policies on one entity) fits OpenBao's warning against thousands of policies per entity

Two engineers would build two incompatible reconcilers, and FR-27's "can't express, so not allowed" has no defined test.
**Evidence.** SPINE:179–182; research-openbao.md:313 (alias = service-account UID); research-openbao.md:331 ("avoid thousands of policies on one entity"); SPINE:134 (5,000-grant worst case); PRD:524 (FR-26 removal within 10 s); KL:21 (no OpenBao in the spike).
**Suggested fix.** Add to AD-13:
- a normative mapping table (identity to entity alias, path grammar to OpenBao path, action to capabilities, with every unmappable case refused)
- an event-driven reconcile on each log event plus a periodic full pass, with a latency target matching SM-4
- a cap or aggregation rule for many grants per entity

Mark the 10 s removal as unmeasured.

### R15 — medium — Structural Seed (public library API and store API)
**Finding.** The seed's package tree has no package for the product-facing API (`Check`, `List`, `Request`, `Approve`, `Withdraw`, signals), the store client, or the gRPC `.proto` definitions. Those are exactly the contract the Quornex and Houston teams code against, and the mermaid diagram shows a "store client" that has no home.
**Evidence.** SPINE:37 (store client in the diagram); SPINE:286–294 (package tree without client/api/proto).
**Suggested fix.** Add `countersign/` (the public facade), `client/` (store client) and `api/` (the protobuf definitions for writes, replication and snapshots) to the seed. State that products import only the facade.

### R16 — medium — AD-9, AD-14 (which grant an allow names)
**Finding.** In the spike, a check allows on the **first** allowed grant in log-position order and names it. But AD-9 introduces a scope index whose iteration order differs, and no Rule fixes which grant an allow names or which reason a refusal gives. So two replicas, or a replica and a replay, can return the same decision with different explanations, which breaks FR-24's "same decision and explanation".
**Evidence.** ref/check.go:47 (sort by `LogPosition`); ref/check.go:62–68 (first allowed grant named); ref/review2_test.go (`TestReview2_6_RefusalReasonIsDeterministic`); SPINE:132–134 (index structure deferred); PRD:473 (FR-24).
**Suggested fix.** Add to AD-9: "Candidates are evaluated in ascending log position whatever the index. An allow names the lowest-position allowing grant, and a refusal reports the reason of the highest-position candidate tried." Keep the determinism test as a required test.

### R17 — medium — AD-4 (staleness reference point)
**Finding.** AD-4 sets the first horizon at "no more than" 10 s but doesn't say what staleness is measured from: when the last heartbeat was received (local monotonic clock), or the store time inside it plus the offset. If it's measured from receipt, a heartbeat delayed in flight lets a withdrawn high-risk grant be honoured for longer than 10 s after the withdrawal, which breaks FR-19's "the clock starts at the withdrawal".
**Evidence.** SPINE:77 (heartbeat carries store time); SPINE:84–86; PRD:407 (FR-19 clock starts at the withdrawal).
**Suggested fix.** State: staleness = (AD-4's `now`) − (store time of the last applied heartbeat) − offset bound. The first horizon is 10 s minus the maximum one-way delay minus the margin. Name the replica test that must prove it (original decision-table row 22).

### R18 — medium — Deferred (horizon, margin and offset values)
**Finding.** The values are deferred, but so is **where they live**. If each product configures them, Quornex and Houston can answer the same stale-replica check differently (NFR-4), and an operator could set a first horizon above 10 s.
**Evidence.** SPINE:300; PRD:563 (first horizon ≤ withdrawal target); PRD:576 (NFR-4).
**Suggested fix.** Decide now that these values are store-owned, distributed in the snapshot and heartbeat (or compiled constants), validated by the store (first horizon ≤ 10 s), and never set per product. Defer only the numbers.

### R19 — medium — AD-16 / AD-5 ("authority over a resource")
**Finding.** The spike decides that in v1 only administrators have "authority over a resource": markings, giving approval authority, sharing limits, service authorisations, and narrowing others' grants. The Cedar marking permit also accepts only administrators. The spine records only "only administrators classify resources", so this narrowing of the PRD's FR-6/FR-7 wording isn't ratified.
**Evidence.** ref/classes.go:77–82 ("In v1 only administrators have it (Skander, 2026-11-17)"); cedarengine/policies.cedar:112–114 (`set_marking.administrator`); SPINE:204 (classification only); PRD:226, 236 (FR-6, FR-7 "person with authority over a resource").
**Suggested fix.** Add to AD-16: "In v1, authority over a resource = active administrator. Delegating it to non-administrators is deferred." List that under Deferred.

### R20 — medium — AD-14, FR-29, NFR-3 (record volume and storage)
**Finding.** FR-29 says outright that "how the volume is stored is for the architecture", and NFR-3 says the fleet figure sizes record volume. But the spine names no store, table or partitioning for decision records, no volume estimate (every check is recorded), and no tie to the one-month replay guarantee. Retention is deferred, but storage isn't even mentioned.
**Evidence.** PRD:508; PRD:570; SPINE:189 ("Compact records are written asynchronously"); SPINE:302 (retention deferred).
**Suggested fix.** Decide where records go (for example a separate PostgreSQL table partitioned by day, written only by the store), what a compact record contains (log position, principal, action, resource, decision, rule IDs, `now`, offset, policy hash), and give a rough checks-per-second estimate. Or list it under Deferred with the risk stated.

### R21 — medium — AD-2 (FR-25 tamper attempts)
**Finding.** AD-2 removes `UPDATE`/`DELETE` from the application's role. But FR-25 also requires that any **attempt** to change or delete an audit record is recorded and raises an alert, and nothing in the spine covers attempts, or changes made by a database superuser or migration role.
**Evidence.** SPINE:69; PRD:487–488.
**Suggested fix.** Add a hash chain over events (each event stores the hash of the previous one). The store verifies it on start-up and periodically, and alerts on a break. Optionally add a PostgreSQL trigger that records and rejects `UPDATE`/`DELETE` attempts. State that superuser tampering is detected, not prevented.

### R22 — medium — AD-1, AD-2 (pending requests and proposals are not events)
**Finding.** AD-1 lists requests among what the store writes. But in the spike, a pending request ("ask") and a classification proposal live only in in-memory maps and produce no event. A store restart loses every open request and proposal, and the audit can't show who asked for what before approval.
**Evidence.** ref/writes.go:72–73 (`s.pending[id] = …`, no `append`); ref/state.go:91–92; SPINE:56; GS:121–134 (no `request_submitted` or `request_expired`).
**Suggested fix.** Add `request_submitted`, `request_expired` and `classification_confirmed` events (and make `classification_proposed` normative). The pending set then becomes a projection of the log.

### R23 — medium — Stack, operational envelope (schema migrations and PostgreSQL floor)
**Finding.** PostgreSQL is "the installation's existing instance (version not pinned)", and no Rule covers who runs database schema migrations, under which role, how they coexist with the no-`UPDATE`/`DELETE` role, or how the store and an older replica coexist across a migration. CI also has no minimum version to test against.
**Evidence.** SPINE:249; SPINE:69.
**Suggested fix.** Pin a minimum supported PostgreSQL major version and test it in CI. Add a Rule: migrations are forward-only, run by a separate migration role at store start-up, never alter or delete log rows, and are additive with respect to the event schema (R3).

### R24 — medium — AD-15, Deferred (list time budget)
**Finding.** AD-15 refuses a list "on timeout", but the budget is neither decided nor deferred (PRD §9 question 9 is addressed to the architecture). With cedar-go measured at 57–78 ms per 10,000 items, the budget decides whether UJ-5 works at all.
**Evidence.** SPINE:198; PRD:571, 688; spike-results.md:236, 251 (cedar list times).
**Suggested fix.** Either set a default budget (for example 100 ms per 10,000 items, with paging above that), or add it explicitly to Deferred together with the measured figures.

### R25 — medium — AD-3, AD-4 (replica start-up)
**Finding.** The spine doesn't say what a replica answers before it has loaded its first snapshot and caught up. Nor does it say how long a snapshot load takes at 50,000 grants (about 31 MiB). A product pod that serves checks during start-up could treat an empty state as current.
**Evidence.** SPINE:78; SPINE:143.
**Suggested fix.** Add to AD-4: "Until a replica has applied a snapshot and a heartbeat newer than it, it refuses everything and reports not-ready." Require the start-up time to be measured when the replica is built.

### R26 — low — Deferred (service-grant hard maximum)
**Finding.** The service-grant hard maximum (PRD §9 question 7) isn't listed under Deferred, while the spike uses a 30-day placeholder that could leak into the product.
**Evidence.** ref/state.go:38 (`ServiceMax: 30 * 24 * time.Hour, // placeholder`); PRD:686; SPINE:296–306.
**Suggested fix.** List it under Deferred and require a refusing default (no service grants) until it's set.

### R27 — low — Consistency Conventions (owner identifiers)
**Finding.** Service-grant owners (`team:payments`) have no format in the conventions, don't parse as an identity, and aren't validated by `engine.Guard`, so the two teams can encode owners differently.
**Evidence.** SPINE:235; engine/guard.go:203–208 (`Owner` not validated); spike-results.md:238 (`team:payments`).
**Suggested fix.** Add an Owners row (`team:<id>` or `role:<id>`, same character set) and a Guard check.

### R28 — low — AD-17 (registry test not yet built)
**Finding.** AD-17 says the field-registry reflection test is "Enforced in CI", but no such test exists in the spike, so the statement reads as current fact.
**Evidence.** SPINE:220; `grep -rn reflect spike/` finds nothing.
**Suggested fix.** Mark it "to build (not in the spike)", and extend it to any new port methods from R5.

### R29 — low — FR-17 vs AD-5/AD-8 (service-grant children)
**Finding.** PRD FR-17 says withdrawing a service grant "also withdraws its own child grants". But the spine (through AD-8's proof and the spike's forbid) makes service grants impossible to hand down. The contradiction is left unresolved.
**Evidence.** PRD:388; cedarengine/policies.cedar:101–103; symcc-results.md (FR-16 "a service grant is never handed down", proved).
**Suggested fix.** Note in AD-11 or AD-18 that FR-17's child-grant clause is empty by construction, and raise a PRD correction.

### R30 — low — Deferred (signal authentication vs flood alert)
**Finding.** While closing-signal authentication is deferred, FR-20's "flood from one source" alert has no trustworthy notion of "source", since a sender can name any source.
**Evidence.** SPINE:302; PRD:420; ref/writes.go:264, 275 (`source string` taken as given).
**Suggested fix.** Define the source as the authenticated calling workload from R7, not a payload field.

## What the spine does well (kept short)

- AD-7, AD-10 and AD-11 ratify the spike exactly: the Guard, typed identities, strict request parsing and the immutable grant all match `engine/guard.go`, `model/ids.go` and `model/records.go`.
- AD-8's depth-3 Rule matches `cedarengine.New`, which refuses any other depth (cedarengine.go:34–37).
- AD-9 and AD-15 name tests that exist and pass (`TestPrefilterChangesNoDecision`, `TestListCheckMatchesSingleChecks`).
- It is honest about what was never measured (AD-1/2/3/13/14 tagged; AD-4's clock rule flagged as never run).
