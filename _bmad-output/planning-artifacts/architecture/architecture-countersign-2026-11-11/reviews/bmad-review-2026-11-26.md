# bmad-review, adversarial lens (2026-11-26)

Two documents were reviewed, each by its own reviewer in a fresh context, with the adversarial lens only, as Skander chose. The findings are in the reviewers' words, shortened to the problem and the proposed fix. bmad-review gives no severity or ranking.

## architecture.md (23 findings)

| # | Where | The problem | The proposed fix |
|---|---|---|---|
| A1 | Summary, engine | The "41 µs against 250 µs" headline is the typical prefiltered path only; the full path (259 µs) and the worst case (415 µs) miss 250 µs | Say so in the summary, and that the scope index isn't built |
| A2 | §9 | In the worst case OPA beat cedar-go (385 vs 415 µs), and the document never says so | State it, say why the rule judges the typical case only, and whether to re-run once the index exists |
| A3 | §5, §3 AD-9 | The 259 → 41 µs gain is credited to "the index and prefilter", and AD-9 is tagged Measured, but the scope index isn't built | Credit the prefilter alone; retag AD-9 "prefilter measured; scope index decided, not built" |
| A4 | §3 AD-6, §9 | A 2.2× margin on a noisy VM, with no plan to re-check on production-like hardware and no rule if the ratio drops below 2× | Add a before-v1 re-run on representative hardware, and the fallback |
| A5 | §11 B3, Summary | The critical risk is "deliberately left open" with no reason, no owner, no deadline and no interim mitigation | State the reason, the owner and "before v1"; add an interim rule: after any restore, stop the store and re-snapshot every replica |
| A6 | §5, §8 | "High-risk" is never defined, and §5 equates it with "not contained", while AD-14 also counts fixed-rule refusals | Add it to the glossary, matching AD-14 |
| A7 | §8 | The second horizon has no value and no stated cap, so a withdrawn contained grant can be honoured for an unbounded time | Give a tentative cap and the worst-case window |
| A8 | §8 diagram | No transitions to Halted from Starting or the stale states, no way out of Halted, and "Fresh answers every check" ignores the offset and version refusals | Add the transitions and a recovery path; change it to "answers checks" |
| A9 | §7 | A reused service-account name may inherit the old agent's OpenBao policies before the reconciler acts; "before" isn't defined | Say "by log position"; detach the policies on `agent_ended` before new grants, or bind entities to a per-instance ID |
| A10 | §6, §2 | The store takes the approver's identity as asserted by the product; mTLS and the residual risk (E17) aren't stated | Add a trust-boundary note |
| A11 | Summary, §1 | "Nobody approves their own request" is stated flatly, but Sam's case is open | Qualify it, pointing to §11 |
| A12 | §1 | Reading "a parent can approve for its child" as hand-down isn't among the questions for Danny | Add it to the questions |
| A13 | §6 | 2 s, 4 h and depth 3 are tentative in the spine but stated as fixed; depth counting in the example is ambiguous | Mark them tentative; define depth and label the example |
| A14 | §6 | Read-your-writes waits only for the writing product's replica; no timeout or result is given | State the scope, the timeout and the result |
| A15 | §7 | "done" is returned before OpenBao is updated; the 10 s OpenBao target is unmeasured | Mark it unmeasured; say "done" covers the log only, and what alerts on lag |
| A16 | §8 table | How the takeover counter stops a losing store writing to OpenBao is unexplained | Explain the mechanism, or say it's open |
| A17 | §10 | Asynchronous records can be lost, so "every decision is recorded" and 100% replay don't hold for contained checks | Say so, and link it to B10 |
| A18 | §9 memory | 650 bytes per grant is from run 2b, before the final policies, with store and replica counted together | Say so |
| A19 | Summary | "25 of 25 rows" includes the replacements of rows 22 and 24, which never ran | Say so in the summary |
| A20 | Summary, §9 | The 14 proved properties are Rust Cedar semantics, for agents and `push` only | Say so in the summary |
| A21 | §3 AD-14 | Tagged "Decided" although B10 leaves its core open, and the round trip is unmeasured | Retag it "decided in part; B10 open" |
| A22 | §7, §11 | No reconciliation interval, no drift method, no alert recipient; "each with an owner" names none | Give them, marked untested, or "TBD" |
| A23 | Glossary | Missing: high-risk, heartbeat, standing sharing limit, service grant, log position, `EnforcedScope`, administrator | Add them |

## PRD (23 findings)

| # | Where | The problem | The proposed fix |
|---|---|---|---|
| P1 | FR-1, UJ-3, FR-19 | After a withdrawal, the agent asks again and rules re-approve contained access in seconds, so UJ-3's containment doesn't work for contained actions | A new FR: a withdrawal for cause blocks rule approval for that principal until a person lifts it |
| P2 | NFR-2, NFR-1 vs FR-19, SM-3, SM-C3 | A stale replica keeps allowing contained checks until an uncapped second horizon, against the 10 s promise | Limit the 10 s promise to high-risk, or refuse contained checks after the first horizon; cap the second horizon in the PRD |
| P3 | FR-12, §9 Q4 | The default reads a person and their own agent as different principals: the permissive reading of Sam's case | Default to fail closed for the floor of four until Danny answers |
| P4 | FR-16 vs FR-17 | A service grant is never handed down, yet withdrawing one withdraws its children | Pick one |
| P5 | FR-3 vs FR-16 | FR-3 ends every grant on task close or agent end, but service grants must outlive their creator | Split FR-3 for task grants and service grants |
| P6 | FR-3 vs FR-1 | FR-1 refuses a request without a duration, so FR-3's 8 h default can never apply | Say what the default is for |
| P7 | FR-10 vs FR-1 | A contained request over 4 h is refused, though a person could approve it, contradicting "refuse is for what no approval could allow" | Return "ask", or make 4 h absolute |
| P8 | FR-9 | The 14-day staggering forces a long administrator maximum, and an expiry can drop below two administrators with no "action" | Give the maximum a value, relax the spacing, treat that expiry as blocking |
| P9 | FR-9 | With two administrators, a compromised one can't be removed | Add a replacement first, or an emergency suspension |
| P10 | FR-20 | Forged closing signals under 20 a minute can end production grants quietly | Limit signals to the sender's own tasks and agents; alert when a human-only grant ends |
| P11 | FR-26, FR-27 vs FR-15, FR-4 | Narrowing doesn't trigger OpenBao removal, so "never wider, even for a moment" can't hold | Add narrowing to FR-26; reword FR-27 |
| P12 | NFR-5 summary | It says 7 of the 9 rules need tests too; the table shows 6 | Correct the count |
| P13 | §0 | The final PRD still calls the architecture "(to come)" | Link the spine; note what changed after the architecture |
| P14 | F3 | No FR for a pending request's lifecycle: approve or reject, lifetime, cancel, re-checking at approval | Add one |
| P15 | FR-1, FR-5 | No FR says how actions and resources become "known" | Add a catalogue FR |
| P16 | FR-1 vs rule 2 | "Names an approver in any field" can't be tested against a type with no such field | Reword it as "extra fields are malformed" |
| P17 | FR-22, FR-29 | One record or 10,000 per list, and how a list is replayed, isn't said | One record per list |
| P18 | FR-29 vs NFR-3 | Durable recording before the answer vs 250 µs | Say whether SM-5 covers these checks |
| P19 | FR-24, FR-29 | Replay needs history and old library versions kept; retention equals the replay window exactly | Keep versions; refuse "version unavailable" |
| P20 | FR-28, SM-7 | Drift detection has no frequency or target; SM-7 measures the environment | Add a target; test with injected drift |
| P21 | FR-16 | A person can create and approve their own service grant | Make creation a request under FR-12 |
| P22 | FR-13 | Worded as a guarantee that depends on Quornex | Reword it, and add an acceptance item for the gate |
| P23 | §2.1 vs FR-9 | Whether customer admins are the installation administrators isn't said | State it |

## Checked by Claude before presenting

- **P12 is right:** the table has 6 rules held by "construction and tests" (5, 7, 8, 11, 12, 13) and 3 by construction only (2, 4, 9).
- **P13 is right:** prd.md line 17 still says "(to come)".
- **P3 conflicts with the PRD's source:** FR-12's assumption says "Only the exact requester is excluded, as Danny asked." The fail-closed default would override Danny.
- **Overlap with the architecture review:**
  - A7 and P2 are the same gap (the uncapped second horizon).
  - A11 and P3 are both Sam's case.
  - P4 is D19's FR-17 vs FR-16.
  - A17 and P17 relate to B10.
  - A15 and P11 overlap the spine's AD-13.

## What happened to each finding (Skander's choices, 2026-11-26)

- **architecture.md, applied as wording:**
  - A1, A2, A3, A6, A10, A11, A12, A13, A15, A17, A18, A19, A20, A21, A23.
- **architecture.md, resolved:**
  - **A4:** a before-v1 re-run on representative hardware; below 2×, the pre-registered tie-break order decides.
  - **A5:** B3's owner is the store team, it must close before v1, and there's an interim rule: after a restore, stop the store and re-snapshot every replica.
  - **A8:** every state can go to Halted, and an operator's snapshot reload goes back to Starting.
  - **A16:** the writer-epoch mechanism is explained.
- **A7 and P2:** the second horizon is capped at 5 minutes (tentative, unmeasured), in architecture.md and in the spine's AD-22 and Deferred.
- **Still open, listed in architecture.md §11:** A9, A14 and A22.
- **PRD, fixed and logged as post-final changes in its memlog:**
  - **P12:** the NFR-5 count now reads "6 of them also need tests".
  - **P13:** §0 now links the architecture.
- **For the README's open questions, for Danny or the PRD owner:** P1, P3–P11 and P14–P23. P2 is settled by the 5-minute cap. P3 conflicts with FR-12's "as Danny asked".
