---
title: "Adversarial review 2: can two teams build the spine and diverge?"
reviewed: ARCHITECTURE-SPINE.md (updated 2026-11-17)
consulted: grant-schema.md, decision-table.md, spike/ (ref/, engine/, cedarengine/, model/)
date: 2026-11-17
---

# Adversarial review 2

## Verdict

**No.** Two independent teams that follow every AD exactly would still build systems that behave differently. At least four of the divergences can produce a **wrong allow**:

- a deposed store that keeps writing and reconciling (B1)
- an OpenBao policy wider than the replica's answer after an ancestor is narrowed (B2)
- a replica whose history silently forks after a store restore (B3)
- grants made after a task-closed signal that the signal never ends (B4)

Several more can produce a wrong allow depending on the reading a team picks: B5, B6, B7, B8, B9 and B10. The spine is strong where it was attacked before: one projection, closed outcomes, full-state narrowing, and limits held at the store. Its holes are at the seams nobody owns: the lock versus the connection pool, the log versus a restore, the heartbeat versus the commit, the grant versus its chain in OpenBao, and the snapshot versus the proto.

Severity scale:
- **critical:** a wrong allow that a team can reach by following the ADs literally.
- **high:** either a wrong allow under one plausible reading, or a guaranteed behavioural divergence on a decision path.
- **medium:** divergence in explanations, audit or availability, with no wrong allow.
- **low:** a documentation contradiction.

---

## Critical

### B1: The single writer isn't fenced. A deposed store keeps writing and keeps reconciling OpenBao (critical)
- **Units:** `store/` (write path) and `store/` (OpenBao reconciler), across two store processes
- **ADs:** AD-1 (D14 advisory lock), AD-2 (rules at N−1 inside the transaction), AD-13
- **Scenario:**
  - Team S builds the store the idiomatic way:
    - a `pgxpool` pool
    - a **session-level** `pg_advisory_lock` taken once on one pooled connection, X
    - each write as a transaction on whatever connection the pool hands out: check the rules against the **in-memory** projection, then `INSERT … position = (SELECT max(position)+1 …) FOR UPDATE`
  - Every AD is obeyed: the lock is held "for its whole lifetime", positions are gap-free, and the rules are checked "against the projection at N−1".
  - Connection X drops (a network blip, or a PgBouncer restart). PostgreSQL releases the lock. Store B starts or takes over, acquires the lock, and appends position N: `withdrawn g-42` (or `administrator_removed erin`).
  - Store A hasn't noticed yet, because it only learns when X errors. Its other pooled connections are healthy.
  - An approval arrives at A. A checks it against its in-memory projection, which is still at N−1, so g-42 is live and Erin is still an administrator. A then appends at N+1.
  - The append-only log now holds a hand-down from a withdrawn parent, or an approval by a removed administrator. Every replica folds it faithfully.
  - At the same time, A's reconciler keeps "bringing OpenBao into line" with A's stale projection. It re-creates `countersign-g-42` after B removed it. B's orphan sweep removes it again, then A re-creates it, and so on.
- **Can it produce a wrong allow?** Yes, both at write time (through the log, to every replica) and in OpenBao.
- **Unenforceable as written:** "A process without it serves no writes and runs no OpenBao reconciler." A process can't know at once that it has lost a session lock.
- **Fix (tighten AD-1 and AD-2):**
  - Every append transaction takes `pg_advisory_xact_lock(<installation key>)` **on its own connection**, or runs on the lock-holding connection.
  - It appends by **compare-and-append**: `INSERT … SELECT … WHERE (SELECT max(position) FROM log) = $expected_head`, where `$expected_head` is the position of the in-memory projection the rules were checked against. If no row is inserted, the store reloads and retries, or halts.
  - The store holds a **writer epoch**, incremented in a row when the lock is acquired. Each OpenBao write is preceded, within the same reconciler step, by a check that the epoch is still the store's own, and the epoch is written into each policy's metadata. A reconciler that sees a newer epoch in OpenBao stops.
  - Layer test: two store processes, one losing its lock mid-flight. Neither may append against a stale head, and the loser's reconciler must make no OpenBao write after the winner's first one.

### B2: OpenBao policies use the grant's own effective scope, not its scope narrowed by its ancestors, so OpenBao allows what replicas refuse (critical)
- **Units:** `store/` OpenBao reconciler and `replica/` (and `projection/`)
- **ADs:** AD-13 (D6: "Its path is the **effective** scope, and its `expiration` is the **effective** end … the earliest over the chain"), AD-11, AD-5 ("overlap with every ancestor's *current* grant")
- **Scenario:**
  - This is decision-table row 16, moved onto OpenBao. Parent P holds `openbao/kv/app/**`. It hands child C the scope `openbao/kv/app/**`, which is allowed, because the child is not wider.
  - Erin narrows P to `openbao/kv/app/billing`. The store appends `narrowed(P, scope=…/billing)`. AD-11 says a `narrowed` event carries the full effective scope **of that grant**. Nothing says the store also emits `narrowed` events for descendants.
  - The replica computes the overlap with every ancestor, so C's check for `openbao/kv/app/payments` refuses.
  - The reconciler team reads AD-13 literally. C's effective scope is still `app/**`, because C has no `narrowed` event of its own. C's `expiration` is correctly the earliest end over the chain, but its **path** isn't intersected. "Any event that changes … an ancestor re-reconciles every descendant", so C is re-reconciled, to the same `app/**` path.
  - OpenBao serves C `app/payments` until C ends.
  - A second team reading "effective" as "intersected with the chain" builds a different reconciler. Both are compliant.
- **Can it produce a wrong allow?** Yes, in the enforcement point that actually guards secrets. The "before v1" agreement test would catch it only if it includes narrowing of an ancestor, which it isn't required to.
- **Fix (tighten AD-13 and AD-19):**
  - `projection/` defines `EnforcedScope(g)`: the intersection over the chain, using `model.Overlap` repeatedly. It is **empty** when any step doesn't overlap.
  - `EnforcedEnd(g)` is the minimum over the chain.
  - The reconciler writes `ToOpenBao(action, EnforcedScope(g))` with `expiration = EnforcedEnd(g) − margin`. It removes the policy when the scope is empty or any chain node isn't live.
  - Name this function in AD-13. Add a property test: for random chains with narrowings, OpenBao glob matching over `EnforcedScope` equals the replica check for every resource. Make that test a v1 gate, not a "before v1" intention.

### B3: Halt-on-rewind doesn't detect a forked history; after a store restore, replicas silently mix two logs (critical)
- **Units:** `replica/` (resume) and `store/` (restore, ID assignment); the Deferred list's "backup and restore"
- **ADs:** AD-21 ("halts … on a store whose head is below its own position"; "IDs unique across restarts"), AD-3 (resume after N), AD-4 (freshness by applied heartbeat)
- **Scenario:**
  - The event log is restored from a backup taken at position 90. Replica R had applied up to 100.
  - Positions 91–100 are lost. They included `grant_created g-77` for `agent:x` on `deploy` production.
  - After the restore, the store accepts writes again and reaches head 105 before R reconnects. The new positions 91–105 include `withdrawn g-50`.
  - R reconnects, resuming after 100. The head (105) is not below R's position (100), so **R doesn't halt**. It receives 101–105, and heartbeats say head 105, so R is **fresh**.
  - R permanently lacks `withdrawn g-50`, which is at the new position 95. It also still holds `g-77`, a grant that no longer exists in the store's history.
  - If the store's ID counter was also restored, a new grant may reuse the ID `g-77` with a different principal. R's chain lookups then resolve to the wrong record.
  - Every AD is obeyed.
- **Can it produce a wrong allow?** Yes. Withdrawn grants are honoured indefinitely, and grants that don't exist in the log are honoured, on a replica that reports itself fresh. Replay (AD-23) then folds a different history from the one the replica decided on.
- **Fix (new AD clause in AD-21, and an ops constraint in D16):**
  - The log carries a **history ID**, a UUID minted when the log is created and **re-minted on every restore or any operation that lowers the head**.
  - Each event carries `prev_hash`, a hash chain over its canonical proto bytes.
  - Resume sends `(history_id, position, last_event_hash)`. A mismatch on either halts the replica (stale) and forces a fresh snapshot.
  - The heartbeat and snapshot carry the `history_id`.
  - IDs are ULIDs or UUIDs, never counters derived from the log.
  - D16's restore procedure must re-mint the history ID.

---

## High

### B4: Closing signals end only grants made before them, by position, but a request can be approved after its task closed; and the "agent IDs aren't reused" convention is unenforceable (high)
- **Units:** `store/` (approval, hand-down write rules) and `projection/` (task-closed and agent-ended facts); product adapters (agent identity)
- **ADs:** AD-11 ("end only grants created before them, by log position"), AD-5 (FR-3: "a grant whose task wasn't open … when it was made is refused at creation"), the Tasks convention ("never reopened"; "Agent IDs aren't reused"), AD-18 (D5: `agent:<namespace>.<serviceaccount>`)
- **Scenario A (the task):**
  - `db-migrate` submits request R for task `quornex:eng-88` at position 200, while the task is open. `task_closed eng-88` arrives at 205. Erin approves R at 210.
  - Store team S1 reads "when it was made" as **submission**, which the request event already validated, and creates `g` at 210.
  - Projection rule (AD-11): the close at 205 ends only grants with position below 205. `g` is at 210, so it is **live for its whole duration** (up to the 24 h task maximum) on a closed task.
  - Store team S2 re-checks task-open at approval and refuses. Both teams are compliant.
  - The same applies to hand-down: a child handed down after the close (if the parent somehow survives, for example a service-kind task grant) is never ended.
- **Scenario B (the agent):**
  - D5 makes the agent ID the ServiceAccount, so every pod of an agent type shares one ID. Kubernetes recreates ServiceAccounts with the same name.
  - "Agent IDs aren't reused" is therefore **unenforceable**: Countersign doesn't mint them.
  - Projection team P1 treats `agent_ended` positionally, as the spike does: later grants for the same ID live.
  - Store team S3 enforces the convention by refusing any request for an ended agent ID. After one pod's `agent_ended`, that agent type is locked out forever.
  - Store team S4 doesn't refuse. An `agent_ended` sent for one pod also kills every sibling pod's earlier grants.
  - Three compliant behaviours.
- **Can it produce a wrong allow?** Yes, in scenario A.
- **Fix (tighten AD-11 and AD-5):**
  - Because tasks are never reopened, **"task closed" is not positional**. A grant whose task is closed is ended, whatever its position. Every request, approval and hand-down transaction re-checks that the task is open against the projection at N−1. Pending requests for a closed task expire at the close.
  - For agents, either:
    - mint agent IDs that include a per-run discriminator the store can see (for example the pod UID, or a `run` claim), so "not reused" becomes true; **or**
    - drop the convention, and state that `agent_ended` is positional and ends every grant held by that ServiceAccount identity before it.
  - Pick one in AD-18. Add a layer test covering "request before close, approval after close".

### B5: Whether an action is contained is not bound to time: a rule-approved grant keeps allowing after its action is re-marked human-only (high)
- **Units:** `projection/` (`classOf`), `replica/` (high-risk routing, staleness), `cedarengine/` (`check.grant-allows` ignores class), `store/` (contained auto-approval, hand-down class)
- **ADs:** AD-5 ("the replica recomputes … the action class"), AD-4 ("high-risk, meaning anything not marked contained"), AD-14, AD-16, AD-8
- **Scenario:**
  - At 09:00 `push` on `forgejo/api/**` is contained, and rules approve a 4 h grant for `api-refactor` (approver = rule).
  - At 09:30 an administrator discovers the branch deploys to production. They add a human-only marking (strictest wins), or confirm a production classification.
  - Replica team R1 uses the class only for routing: the check is now high-risk, so it makes the synchronous record call, the store is up, and the record is written. Cedar's `check.grant-allows` has no class condition, so the check **allows until 13:00**. A rule approved a human-only action.
  - Replica team R2 reads "rules approve contained requests" as an invariant on the live grant. It refuses rule-approved grants whose current class is human-only.
  - Both are compliant. The same ambiguity hits hand-down:
    - a child created under a contained standing sharing limit survives the raise
    - a parent approved by a person at depth 0 can hand down after the marking is **lowered** to contained, because the class is computed at hand-down time
  - It also hits the hard maximum (AD-5 lists it both as a check-time recomputation and as a once-at-the-store rule): after a `limits_set` lowers the contained maximum, R1 keeps old grants and R2 refuses them.
- **Can it produce a wrong allow?** Yes. Under R1, an action that is human-only now is allowed on a grant no person approved.
- **Fix (tighten AD-5 and AD-8):**
  - Add `approved_by_rule` (or `root_approver_kind`) and the **current** class to each chain node in `CheckInput`.
  - Add a Cedar `forbid` with `@id("check.forbid-rule-approved-human-only")`: a chain whose root was approved by a rule can't allow an action whose current class is human-only. Prove it in SymCC.
  - State explicitly that the hard maximum is a **write-time** rule only, and that a later `limits_set` doesn't shorten existing grants. Or state the opposite, but say which.
  - State that the hand-down class is the **parent's root class at approval, or the current class, whichever is stricter**.

### B6: The heartbeat's (position, time) pair isn't required to be causally consistent, so a lagging heartbeat certifies freshness for withdrawals it doesn't cover (high)
- **Units:** `store/` (heartbeat producer) and `replica/` (freshness)
- **ADs:** AD-3 ("latest position plus the store's time, every 1 s"), AD-4 (D4: the freshness instant is the store time of the latest **applied** heartbeat)
- **Scenario:**
  - Store team S builds the heartbeat from an in-memory `head`, which a `LISTEN/NOTIFY` goroutine or the append path updates after the commit, plus `time.Now()` read **when the heartbeat is sent**.
  - Under load, `head` lags the committed log by 3 s. At store time T=12:00:05 the heartbeat says `(head=500, T)`, but `withdrawn g-9` committed at 12:00:03 as position 503.
  - The replica applies 500 and gets freshness instant 12:00:05. The stream then breaks. The replica believes it holds everything committed up to 12:00:05. It keeps honouring g-9 for high-risk checks until 12:00:05 plus the first horizon: up to 3 s past the 10 s target, and unboundedly more under a worse lag.
  - A store team that reads the clock **before** reading the head from the database is correct. Both are compliant.
- **Can it produce a wrong allow?** Yes: a withdrawal honoured past NFR-2's target, with the replica reporting itself fresh.
- **Fix (tighten AD-3):**
  - A heartbeat's time `T` is read from the store's clock **before** reading `max(position)` from the committed log, in that order, in one transaction or with a read barrier. It means "every event committed at or before T has position ≤ H".
  - Heartbeats are sent **in-band** after every event up to H.
  - The layer test injects a delay between commit and the head update, and asserts freshness never exceeds the commit time of an unapplied event.

### B7: Limit values are unvalidated, and `limits_set` has no defined shape: a partial or zero value silently widens access (high)
- **Units:** `store/` (`limits_set` write), `proto/`, `projection/` (fold of `limits_set`), `replica/` (horizons, margin, offset)
- **ADs:** AD-22 (only the depth is validated), AD-4 (first horizon "no more than" 10 s), AD-7 (margin), AD-11 (narrowed is full-state, but nothing says `limits_set` is)
- **Scenarios:**
  - **Partial or full.** An operator updates only the list deadline.
    - Proto3 scalars have no presence, so the store team emits a `limits_set` with the other fields at 0.
    - Projection team P1 replaces all limits, so the clock safety margin becomes 0 and the maximum offset 0.
    - Projection team P2 merges non-zero fields.
    - Under P1 the margin silently disappears. AD-7's "truncation can never lengthen access" and AD-13's `end − margin` both rest on the margin. Horizons of 0 refuse everything, and hard maxima of 0 refuse every write.
  - **Unvalidated values.** The store accepts `first_horizon = 1h`, or `second < first`, or `margin < 0`.
    - AD-4 says the first horizon is no more than 10 s, but only the depth is listed as refused at the store.
    - A 1 h first horizon means a cut-off replica honours a withdrawn high-risk grant for an hour.
  - **Maximum offset.** It appears only in AD-22's list. No AD says what a replica does when the offset exceeds it: refuse everything, refuse high-risk only, or alert. Three teams, three behaviours.
- **Can it produce a wrong allow?** Yes: a zero margin, a long first horizon, or an inverted pair of horizons.
- **Fix (tighten AD-22):**
  - `limits_set` always carries the **complete** limit set, with proto `optional` presence. The projection refuses (halts) on a `limits_set` with any field absent.
  - The store refuses, with a reason code, unless all of these hold:
    - 0 < first horizon ≤ 10 s < second horizon ≤ a stated cap
    - margin > 0
    - offset maximum > 0
    - every hard maximum > 0 and ≤ the PRD ceiling
    - depth = 3
  - Add a clause to AD-4: a replica whose measured |wall − store estimate| exceeds the maximum offset refuses **every** check with `replica.clock-offset`.

### B8: AD-20 pins the policy bundle, not the library: Quornex and Houston can run different `projection/` and `Guard` code against the same log (high)
- **Units:** the Quornex and Houston product adapters (embedded library versions); `projection/`; replay
- **ADs:** AD-20 (only the policy hash is enforced), AD-19 (one package, but not one version), AD-21 (a proto schema minimum only), AD-23 (replay uses "the engine named by the recorded hash")
- **Scenario:**
  - Library v1.4 fixes a fold bug in `projection/`, for example how a `narrowed` event that carries only an end combines with an earlier scope narrowing, or the precedence of chain facts.
  - Quornex ships v1.4. Houston stays on v1.3. Both embed the same `policies.cedar`, so both pass the hash check, and both speak the same proto schema.
  - For the same `(principal, action, resource, now, position)`, Houston allows and Quornex refuses.
  - Replay folds with whatever `projection/` the replaying tool links, so a replay of Houston's records raises false alerts, or hides the bug.
  - "Two products deciding the same check differently" is exactly what AD-20 claims to prevent.
- **Can it produce a wrong allow?** Yes: a fixed safety bug stays live in whichever product lags.
- **Fix (tighten AD-20):**
  - The installation's required identity is `(policy_hash, decision_core_version)`. `decision_core_version` is a semantic version of `model/` + `engine/` (Guard) + `projection/` + `replica/` decision code, bumped on any change that can alter a `Decision`.
  - The store announces the **minimum** `decision_core_version` in the snapshot and heartbeat, and replicas below it refuse every check (`replica.version-mismatch`).
  - Replay must run the recorded `decision_core_version`. Keep tagged builds, with the replay tool vendoring each.

### B9: The snapshot has two owners, `proto/` and `projection/`, and the property test bypasses the wire (high)
- **Units:** `proto/` (store team) and `projection/`
- **ADs:** AD-21 ("`proto/` … defines … the snapshot"), AD-19 ("The snapshot is this package's serialised state"; the property test), AD-3
- **Scenario:**
  - The projection team adds a fact, for example `closedTasks` becoming non-positional per B4, or an enforced-scope cache. It serialises the fact through its own `MarshalBinary`. The AD-19 property test (snapshot at N, plus events, equals the fold of 1..M) passes, **using `projection/`'s own encoding**.
  - The proto team's `Snapshot` message, which is the one actually sent to replicas, isn't updated. It has no field for the new fact.
  - New replicas bootstrapped from the proto snapshot start without it. With `withdrawn` or the closed-task positions missing, they **allow** grants that a replica built by replay refuses.
  - Alternatively, the proto team defines `Snapshot { bytes projection_state }`. Then the schema version (AD-21) no longer versions the snapshot at all.
  - Each team is compliant with its own AD.
- **Can it produce a wrong allow?** Yes, on any replica bootstrapped from a snapshot.
- **Fix (tighten AD-19 and AD-21):**
  - One owner: `projection/` defines the canonical snapshot as a proto message in `proto/`, and `projection/` is the only code that writes or reads it.
  - The AD-19 property test must round-trip through the **proto wire bytes**.
  - A reflection test fails if any field of the projection state struct has no snapshot field.
  - The snapshot carries `history_id` (B3), the policy hash and `decision_core_version` (B8).

### B10: The high-risk record call has undefined semantics: whether the store re-decides, where records live, retries, and the compact-record path (high)
- **Units:** `replica/` (the high-risk path) and `store/` (the record endpoint); `proto/`
- **ADs:** AD-14 (D1), AD-1, AD-21 (closed event list; idempotency only for "writes"), AD-23, the dependency invariant ("The replica never writes, except to send a high-risk decision record")
- **Scenarios:**
  - **Re-decide or record.** Erin withdraws g at position 900. Replica R is at 897, inside the first horizon, and makes a high-risk check on g.
    - Store team S1 persists R's allow record and acknowledges, so the check allows, with the store **knowing** g was withdrawn.
    - Store team S2 re-checks the record against its head and refuses with `replica.behind`.
    - Both are compliant. Under S1 the "durable record" is a durable record of a wrong allow that the store could have prevented. Under S2 the record call is in effect a remote check, with a different latency and availability profile.
  - **Where records go.** AD-21's event list is closed and has no `decision_recorded`.
    - Team A appends records to the log. That breaks the closed list, and every replica halts on the unknown type. Even if the type is added, every high-risk check then advances the position replicas must apply.
    - Team B writes records to a side table, outside AD-2's single-transaction and append-only guarantees, so FR-25 tamper evidence doesn't cover them.
  - **Retries.** The record call times out after the store committed it. The replica refuses, but the record says allow. A retry writes a second record. AD-21's idempotency key covers "writes" only.
  - **Compact records.** AD-14 says compact records are "written asynchronously", but the invariant forbids any replica write other than the high-risk record, and no unit owns them. Team A keeps them in process memory and sends them in batches, losing them on a crash. Team B writes them to a local file. AD-23's "100% of decisions can be replayed" is then unenforceable.
  - **Lists.** A list of 10,000 human-only items needs 10,000 synchronous records under AD-14, or one batch record. Unstated.
- **Can it produce a wrong allow?** Yes under S1, where the store knowingly records an allow it would refuse. The other scenarios produce divergence and audit holes.
- **Fix (tighten AD-14 and AD-21):**
  - The record call carries `(decision, applied_position, history_id, idempotency_key)`. The store **refuses the record**, which makes the check refuse with `replica.behind`, if any event after `applied_position` touches the decision's chain, the principal's grants for that action, or the class. The store is the tie-breaker for high-risk checks.
  - Decision records live in a dedicated append-only table, written by the store under the same role constraints as AD-2. They are not log events, and the closed event list says so.
  - Compact records go to the store over the same channel, in batches, with a bounded local buffer. If the buffer overflows, the replica refuses high-risk checks, and FR-29's "100%" is restated as "100% of high-risk; compact records best-effort within the buffer". Or make the buffer durable.
  - A list sends **one** record for all its high-risk items, before answering.

### B11: The prefilter and scope index (AD-9) contradict "precedence over **all** candidates, prefiltered or not" (AD-23), so the same refusal is explained differently (high; no wrong allow)
- **Units:** two teams building `replica/`; the replay tool
- **ADs:** AD-9 (the index reaches "only grants whose scope can cover the resource"), AD-23 (refusal precedence includes "narrowed out", over all candidates, "prefiltered or not"), `TestPrefilterChangesNoDecision` comparing the whole `Decision`
- **Scenario:**
  - g covered `api/**` and was narrowed to `api/billing`. The agent checks `api/payments`.
  - Replica team R1 builds the scope index over **effective** scopes, which AD-9 requires for 5,000-grant independence. g isn't a candidate, so the result is `default-deny` with no chain fact.
  - Replica team R2 indexes by **original** scope, so g is a candidate. Its effective scope doesn't cover, so the result is chain fact `narrowed out`, by Erin, at 10:02.
  - Even if R1 tries to honour "prefiltered or not", it must scan all of the principal's grants for this action on every refusal, which reintroduces the O(n) worst case AD-9 forbids.
  - The precedence also has no tie-break between two withdrawn ancestors, or two candidates with the same chain-fact kind. Which `grant` goes into `{kind, grant, by, at}`?
  - Replays built by the other team raise disagreement alerts.
- **Can it produce a wrong allow?** No. But AD-23's own "Prevents" is violated, and replay alerts become noise.
- **Fix (tighten AD-23 and AD-9):**
  - Define "candidate" as **every grant of (principal, action) whose original scope covers the resource**, and index by original scope. The effective scope only narrows, so the index stays sound.
  - Tie-break each chain-fact kind by the nearest ancestor first, then the lowest log position.
  - Close the layer-refusal list (remove "and so on").
  - Define the reason code for each chain-fact kind: `chain.withdrawn`, `chain.task-closed`, `chain.agent-ended`, `chain.expired`, `chain.narrowed-out`.

---

## Medium

- **B12 (medium): the closed event list is missing events needed to build the ADs.** These events are missing:
  - a direct (floor-raising) classification. AD-21 has only the proposal and its confirmation; the spike has `resource_classified`
  - the system registry (Conventions: installation state, D13)
  - revoking approval authority, a sharing limit or a service authorisation before expiry
  - the required policy hash changing (AD-20)
  - the store's write-time bundle

  Any team that adds one halts every other team's replicas (AD-21). **Fix:** enumerate these in AD-21 now, with payloads in grant-schema.md.
- **B13 (medium): the system registry is installation state, but AD-7's `Guard` is stateless and lives in `engine/` (model only).** One team checks a resource's first segment against the registry, and another can't. An unregistered system passes in one replica and is refused in another. **Fix:** the registry check moves into `replica/` and `store/` before `Guard`, with a fixed refusal ID `input.unknown-system`.
- **B14 (medium): the store's `now` isn't defined.** AD-4 says the store "follows the same rule" (the later of two clocks, plus the margin), but the store has no heartbeat estimate. Teams also differ on whether `issued_at`, `ends_at` = now + d, event `at` and the heartbeat time include the margin. A margined heartbeat time makes replicas fresher than reality by the margin. **Fix:** the store uses its raw wall clock for stored times and heartbeats, and adds the margin only when evaluating its own rules.
- **B15 (medium): AD-23's retention ("events kept at least one month") contradicts AD-2 (no DELETE), AD-19 and replay ("fold events 1 to N").** Grants, authority and administrators outlive a month, so pruning drops state. Pruning `administrator_added` can lock out every administrator. **Fix:** never prune log events; retention applies to decision records only. Or prune only behind a durable, store-signed snapshot, with replay starting from it.
- **B16 (medium): OpenBao enforces `expiration` on its own clock (AD-13), and no AD bounds OpenBao's clock offset.** A grant that the D17 typing allows to have a person principal has no OpenBao identity to attach a policy to. Some teams refuse such a grant (FR-27) and some create it without a policy. **Fix:** the reconciler also removes policies at `EnforcedEnd − margin` by the store's clock, and person principals on `openbao/` resources are refused with "not expressible".
- **B17 (medium): no unit owns the store client.** Each product writes its own, with its own minimum-position (read-your-writes) handling, idempotency-key generation, retry rules and outcome mapping. **Fix:** add a `client/` package to the tree, owned by the store team, which the products must use.
- **B18 (medium): Keycloak `sub` values can fall outside `[a-z0-9._-]+`.** Examples are federated `f:<provider>:<user>` IDs and mixed case. One adapter refuses, another lowercases or escapes. **Fix:** D5's function refuses any non-conforming `sub`, with `identity.unsupported`, and the conformance test includes such tokens.
- **B19 (medium): the contained auto-approval sits neither in an engine method nor in AD-6's list of rules kept in Go.** Its approver string has no defined form: the spike uses `rule:contained`, while the Rule-ID convention is `<action>.<rule>`. **Fix:** add a `MayAutoApprove` engine decision with a fixed `@id`, and use that ID as the approver.
- **B20 (medium): who may change a child grant isn't defined.** AD-5's FR-4 list omits the parent's holder, which the spike allows, and doesn't say whether the root approver counts as "its approver" on a child. **Fix:** enumerate who may change a child grant.
- **B21 (medium): AD-20's store-first rollout makes every policy change a full outage of checks** until the products upgrade, and the switch isn't a log event, so it isn't replayable (see B12). **Fix:** replicas carry the previous and the next bundle, the store switches by a `policy_bundle_set` event, and the store's own bundle must equal the announced hash.
- **B22 (medium): idempotency keys have no defined scope or lifetime, and no rule for a reused key with a different body.** If they're held only in memory, they are lost on failover. **Fix:** keys are recorded in the event, unique per (caller, key), and a reused key with a different body is refused.

## Low

- **B23 (low): the invariants diagram draws `engine.Guard → cedarengine`** (the port importing its adapter), omits `Guard → model` (AD-7), and shows no product → `cedarengine` wiring edge. The structural seed links Keycloak only to Quornex and Kubernetes only to Houston, which contradicts AD-18.
- **B24 (low): the layer table and package tree still list `Lister` in `engine/`,** although AD-6 and AD-15 say one check per item and OPA isn't used.
- **B25 (low): D18 ("exactly one `permit` per action") conflicts with the spike's two `create_service_grant` permits,** and the merged form isn't specified. A rule-13 refusal isn't a chain fact, so "names its chain fact through AD-23" doesn't apply to it.
- **B26 (low): nothing ties a task's `<product>:` prefix to the mTLS identity of the product that opens or closes it.** Houston can open `quornex:` tasks.
- **B27 (low): the spine doesn't say whether a request's duration starts at submission or at approval, or whether a second `withdrawn` event for an already withdrawn grant is appended or refused.**
