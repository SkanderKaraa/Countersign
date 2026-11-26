# Known limits of the spike

These are points found in review that the spike deliberately doesn't fix. On 2026-11-17, Skander stopped hardening parts 1 and 2 after C10, so the work could move on to OPA and the measurements. They feed the README's open questions.

## Skipped review points (part 2 review, groups D and E)

- **D14: what happens when a service grant's review date passes.** `review_date` is stored but never read. Whether it alerts the owner, or refuses high-risk checks after a grace period, is undecided.
- **E15: the layer is the trust boundary.** The layer computes the booleans each engine receives: "is an administrator", "holds the access", "is authorised", "is in the floor of four", and the chain facts (withdrawn, ended, task closed). An engine can't contradict a wrong fact, because it only combines facts. A bug in the layer is therefore a bug in every engine at once.
- **E16: no index on the grants.** A prefilter was added later, the same for every engine (change 3), so the engine is only called for grants whose scope covers the resource. But the layer still *loops over* every grant the principal holds for the action. With one agent holding 5,000 grants, every engine fails 250 µs, the reference included: a layer finding. A real replica needs an index on principal, action and scope.
- **E17: identities are trusted.** Every entry point trusts the identity it receives: `by`, the approver text, the request's principal. The product's authenticated session must supply the caller, and Countersign must refuse any mismatch. The prototype doesn't.

## Other limits of the prototype

- **A lock, not a swapped immutable state.** The `State` uses a read/write lock. The replica's design is an immutable state, swapped atomically, so checks never wait. The measured p99 therefore includes lock contention while events are applied.
- **Some writes aren't routed through the engine.** These are decided only in the layer's Go code:
  - who may narrow or withdraw a grant (FR-4)
  - classifying resources, and the two-administrator confirmation
  - approval authority, sharing limits, service-grant authorisations and administrators
- **Cedar writes out exactly three ancestor levels.** Cedar has no recursion. The engine refuses to load for any other maximum depth.
- **The Cedar schema's coverage is narrower than the model.** `policies.cedarschema` exists and `policies.cedar` validates strictly against it. The expected union-type problem didn't happen: Cedar accepts comparing a `Person` with an `Agent`. But the schema types `Request.requester` and `Grant.principal` one way each, which narrows what the proofs cover (see below).
- **No OpenBao.** Enforcement, the agreement test and drift detection (FR-26 to FR-28, NFR-5 rule 10) aren't in the spike.
- **No real store, replication or event log.** One in-memory `State` plays both the store and a replica. There is no PostgreSQL, no gRPC stream, no snapshots and no heartbeat, so staleness (NFR-2) and clock skew (NFR-6) aren't measured.
- **Single process, single machine.** The numbers come from a 4-core, 7 GB machine, not a production fleet.

## Limits of the SymCC proofs (part 5)

- **What is proved:** properties of `policies.cedar`, for every request and entity store that `policies.cedarschema` allows, in the request environments listed in `results/symcc.md`. Fourteen properties are proved; two negative controls found counterexamples, as they must; and breaking the self-approval `forbid` made SymCC find a counterexample. So the proofs aren't vacuous.
- **What isn't proved:**
  - **The layer's facts.** "Is an administrator", "holds the access", "is in the floor of four" and the chain facts are inputs to Cedar (E15). A wrong fact from the layer is outside the proof.
  - **Agent requesters on approvals.** The schema types `Request.requester` as a `Person`, so self-approval is proved for person requesters. Agents can't approve at all (proved separately), so an agent can never approve its own request.
  - **Grants held by people, at check time.** The schema types `Grant.principal` as an `Agent`, and checks are analysed for agent principals. Checks by people use the same policy, but the analysis doesn't cover them.
  - **Only the request environments listed.** For example, the depth and chain properties are analysed for `push`, not for each of the six check actions. The policies treat every check action the same way, but the analysis doesn't show that.
  - **Rules outside Cedar.** NFR-5 rules 2 (a request has no approver field), 3 (hard maximum lifetime), 7 (the production gate) and 10 (OpenBao) are enforced in the layer, so SymCC doesn't cover them.
- **`policies.cedar` was rewritten for the analysis.** Cedar's strict validator rejected the original ancestor clauses (`!(grant has parent) || …`), so they became `if grant has parent then … else true`. The meaning is unchanged as far as testing can show: all 75 tests pass on every engine, including the three-way differential test, and a mutation of the rewritten clause is still caught. Run 5 measured the original text, so the rewritten policies were re-measured in run 6 (change 8). The decision is unchanged, but cedar-go's typical full path moved from 249 µs to 259 µs, so it passes only with the prefilter.
