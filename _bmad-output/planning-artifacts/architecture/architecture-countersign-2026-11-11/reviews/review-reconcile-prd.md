# Review: ARCHITECTURE-SPINE.md reconciled against the PRD and its addendum

- **Reviewed:** `ARCHITECTURE-SPINE.md` (updated 2026-11-17)
- **Inputs:** `prds/prd-countersign-2026-11-06/prd.md` (final, 2026-11-11), `prds/prd-countersign-2026-11-06/addendum.md`
- **Reviewer:** independent input-reconciliation pass, 2026-11-17. No file edited.
- **Path shorthand:** `SPINE` = `architecture/architecture-countersign-2026-11-11/ARCHITECTURE-SPINE.md`, `PRD` = `prds/prd-countersign-2026-11-06/prd.md`, `ADD` = `prds/prd-countersign-2026-11-06/addendum.md`, `GS` = `grant-schema.md` (companion, consulted only to calibrate severity).

## Verdict

The spine is faithful on the core (default deny, single writer, staleness, chain recomputation, OpenBao reconciliation, numbers 10 s / 250 µs / 50,000 / depth 3), but it drops NFR-5 rule 3 entirely, has no AD for FR-7 or the authority/withdrawal write rules, puts a synchronous store write in the check path without reconciling it with AD-1 and NFR-3, and binds FR-9 to an AD that addresses none of its consequences.

## Findings

### P1 — critical — NFR-5 rule 3 (every grant has a hard maximum lifetime), FR-3, SM-1
- **AD/section:** AD-5, AD-8 (the only ADs that enumerate NFR-5 rules); frontmatter binds NFR-5.
- **Finding:** Rule 3 is the only one of the 13 fixed rules bound by no AD: AD-5 lists rules 1, 4–6, 8, 9, 11–13 plus 7 and 10; AD-8 lists 1, 4–6, 8, 9, 11–13; AD-10 lists 2, 5, 12; AD-13 lists 10. The write-side check that every grant's end is within the hard maximum for its kind (task < 24 h, contained 4 h, service grant's own maximum) has no rule, no owner (store or engine) and no test obligation, even though the PRD classifies it as "Tests" precisely because the grant type has no validated constructor. SM-1 (100% of grants, service grants included, have a hard maximum) therefore has no architectural support.
- **Evidence:** PRD:589 (rule 3), PRD:187–189 (FR-3), PRD:648 (SM-1); SPINE:90, SPINE:119, SPINE:147, SPINE:174. AD-5's check-time list mentions "expiry and the hard maximum" (SPINE:93) but only as a recomputed fact, not as the write-time refusal FR-1 requires (PRD:163, "refused, not shortened; the refusal states the maximum").
- **Suggested fix:** Add rule 3 to AD-5's write-time list (store refuses any request or hand-down whose end exceeds the maximum for its kind, stating the maximum) and to AD-17's layer tests; state where the per-kind maxima live (config, with the tentative values) and that the service-grant maximum is still open (PRD §9 q7).

### P2 — high — FR-29 / FR-21 "records kept in full are written before the answer" vs NFR-3 and AD-1
- **AD/section:** AD-14 vs AD-1, AD-9; PRD §6 non-goal.
- **Finding:** AD-14 requires records for human-only actions and fixed-rule refusals to be written before the answer, a failed write refusing; AD-1 says only the store writes and checks make no network call. Together this means every high-risk check needs a synchronous gRPC round trip to the store, so the store is in the check path after all, the 250 µs p99 (NFR-3, SM-5) is unmeasured for those checks (the spike had no store write), and "store down" now refuses high-risk checks, which AD-4 attributes only to writes. The spine never reconciles these.
- **Evidence:** PRD:507 (FR-29 record before answer), PRD:569 (250 µs each check), PRD:614 (non-goal: not a service called over the network); SPINE:52–58 (AD-1, "no network call"), SPINE:188 (AD-14), SPINE:87 (AD-4, only writes fail closed).
- **Suggested fix:** Decide explicitly in AD-14 how a pre-answer record is made durable (synchronous store call, or a local durable write-ahead buffer in the replica forwarded to the store), amend AD-1's rule and title accordingly, state that NFR-3's 250 µs is measured for compact-record checks only or add the record write to the measurement, and add "store unreachable → high-risk checks refuse" to AD-4.

### P3 — high — FR-7 (give approval authority), and FR-4 / FR-19 (who may narrow or withdraw)
- **AD/section:** frontmatter binds FR-7; AD-5, AD-6; no AD binds FR-7.
- **Finding:** FR-7 is in the frontmatter binds but no AD names it, and AD-6's closed list of engine decisions (`Check`, `MayApprove`, `MayHandDown`, `MaySetMarking`, `MayCreateServiceGrant`) has no decision for giving approval authority, setting a standing sharing limit, narrowing, withdrawing, or adding/removing administrators. Either those write rules escape the engine, contradicting AD-6's own "write rules escaping the engine" prevention, or they are undefined; FR-7's consequences (not wider than the giver's own authority, 90-day hard maximum, approved grants survive the approver's lapse) and FR-4's (holder, approver or current authority may narrow; never widen) have no rule.
- **Evidence:** PRD:234–243 (FR-7), PRD:196–202 (FR-4), PRD:403 (FR-19 "anyone who may end a grant early"); SPINE:11 (binds FR-7), SPINE:101–105 (AD-6), SPINE:94 (AD-5 write-time list omits narrowing/withdrawal authority and authority-giving). GS:72–79 defines the Approval authority record, so the data exists but no decision rule binds it.
- **Suggested fix:** Extend AD-6's list (e.g. `MayGiveAuthority`, `MaySetSharingLimit`, `MayNarrow`/`MayWithdraw`, `MayChangeAdministrators`) or state which of these are deliberately Go-only and why; add FR-4, FR-7 and FR-19 authority rules to AD-5's write-time list; bind FR-7 explicitly.

### P4 — high — NFR-4 "a check with an identity that isn't authenticated is refused"
- **AD/section:** AD-18, Deferred (E17).
- **Finding:** The PRD makes this a testable Countersign consequence; the spine hands it to the products for v1 and defers authenticating the caller (E17), so no Countersign test can show it. Combined with unauthenticated closing signals (also deferred) and rule 6 depending on correct identity (PRD:592), the identity a check or approval carries is trusted as passed in.
- **Evidence:** PRD:579 (NFR-4), PRD:592 (rule 6 "depends on the identity being right"); SPINE:227, SPINE:303.
- **Suggested fix:** Either state in AD-18 the minimum Countersign enforces in v1 (e.g. the store accepts approvals only over an authenticated channel bound to the product's service identity, and the library API takes a verified-identity type the product must construct from a Keycloak/Kubernetes token) or record explicitly that NFR-4's last bullet is downgraded to a product obligation and needs the PRD owner's sign-off.

### P5 — medium — FR-9 bound to AD-16, which addresses none of FR-9's consequences
- **AD/section:** AD-16.
- **Finding:** AD-16 binds FR-9 but only covers resource classification and two-administrator floor lowering; FR-9's consequences (at least two administrators, add/remove approved by another administrator, administrator hard maximum with a 14-day alert, staggered expiries) appear in no AD. GS:108–116 covers the record, so this is a binding/rule gap rather than a data gap.
- **Evidence:** PRD:267–271; SPINE:200–207; GS:108–116.
- **Suggested fix:** Add an AD (or extend AD-16) for administrator lifecycle with these rules, or remove FR-9 from AD-16's binds and bind it where it is addressed.

### P6 — medium — FR-29 / FR-24 record fields: version of fixed rules and library
- **AD/section:** AD-12, AD-14.
- **Finding:** FR-29 requires each record to carry the fixed-rules and library version; the spine records only time and clock offset (AD-12) and rule IDs/chain (AD-14). Without the policy/library version, replay across an upgrade cannot guarantee "same decision and explanation" (FR-24).
- **Evidence:** PRD:496–501, PRD:473; SPINE:170, SPINE:190.
- **Suggested fix:** Add to AD-12 or AD-14: every record carries the policy-set hash and library version; replay loads the matching policy set or says it can't.

### P7 — medium — FR-24 / SM-6 one-month replay minimum treated as open
- **AD/section:** Deferred ("record retention").
- **Finding:** Retention beyond one month is open (PRD §9 q10), but the one-month minimum is a firm requirement; the spine defers "record retention" wholesale and states no minimum, and doesn't say replay refuses rather than guesses when state can't be rebuilt.
- **Evidence:** PRD:476, PRD:478, PRD:653, PRD:641; SPINE:302.
- **Suggested fix:** State in AD-2 that the log and records are kept at least one month, and that a replay outside retention reports "cannot rebuild".

### P8 — medium — FR-26 / SM-4 OpenBao removal within 10 s; effective scope of child grants
- **AD/section:** AD-13.
- **Finding:** AD-13 says the store "continuously" reconciles but sets no target, so SM-4's separately measured 10 s has no architectural commitment. It also writes one policy per grant covering the grant's exact path, but a child grant's effective access is its overlap with every ancestor's current grant (FR-14), and AD-13 doesn't say a parent's narrowing or withdrawal re-reconciles descendants' policies.
- **Evidence:** PRD:524, PRD:651, PRD:348; SPINE:179–182.
- **Suggested fix:** Add the 10 s reconciliation target (and that it is measured apart from SM-3), and state that a policy encodes the grant's effective scope and end (after narrowing and ancestor overlap) and is rewritten on any event in its chain.

### P9 — medium — NFR-5 rule 10 / FR-27 agreement test against a real OpenBao
- **AD/section:** AD-13.
- **Finding:** The PRD holds rule 10 "by tests, against a real OpenBao" and FR-27 requires an agreement test over the decision table; the spine commits to neither, and notes the prefix listing is untested.
- **Evidence:** PRD:596, PRD:534; SPINE:180.
- **Suggested fix:** Add to AD-13 a required CI test against a real OpenBao v2.7.2 for rule 10, drift detection (FR-28) and decision-table agreement.

### P10 — medium — FR-8 / §10 standing sharing limit for contained grants, §9 q6c
- **AD/section:** AD-5, AD-6, AD-10.
- **Finding:** The standing sharing limit (PRD assumption, open with Danny as q6c, and addendum seed 4) is not mentioned in any AD; AD-10's list of PersonID-only operations omits "allowed to share" and the sharing limit, which rule 12 covers.
- **Evidence:** PRD:249–253, PRD:682, PRD:598, ADD:37; SPINE:94, SPINE:151; GS:83.
- **Suggested fix:** Name the sharing limit in AD-5 (write-time, PersonID-only) and AD-10, and mark it as resting on the FR-8 assumption / q6c.

### P11 — medium — FR-18 depth 3 made structural
- **AD/section:** AD-8.
- **Finding:** The PRD's depth 3 is a tentative, untested proposal; AD-8 hard-codes it so the engine "refuses to load for any other" value, turning a tunable target into a policy rewrite. FR-18's default-zero and strictly-decreasing depth rules are also not stated.
- **Evidence:** PRD:363–366, PRD:705, PRD:720; SPINE:126.
- **Suggested fix:** Mark depth 3 as tentative in AD-8 with what changing it costs (policy regeneration plus re-proof), and add the default-0 / decreasing rules to AD-5's write-time list.

### P12 — medium — FR-12 / §9 q4 implicitly answered by the Cedar schema
- **AD/section:** AD-8.
- **Finding:** Typing `Request.requester` as `Person` and relying on "agents can never approve" embeds the FR-12 assumption (person and their agent are different principals), which makes Sam approving `api-refactor` allowed by construction. The spine doesn't say this is q4's still-open case or what changes (a launched-by fact, a new forbid, a re-proof) if Danny answers "yes, self-approval". Rule 6 is also proved for person requesters only, narrower than FR-12's "whichever identity source".
- **Evidence:** PRD:311–313, PRD:677, ADD:35; SPINE:123.
- **Suggested fix:** Add to AD-8: "this encodes the FR-12 assumption; if q4 is answered yes, add a `launched_by` fact and an `approve.forbid-own-agent` rule", and require a layer test for agent requesters.

### P13 — medium — §9 architecture/undecided questions not deferred
- **AD/section:** Deferred.
- **Finding:** Deferred lists q8, q9b, q10, q11, q12, q12b and "every question to Danny", but omits q7 (service-grant hard maximum), q9 (list time budget; AD-15 relies on a timeout of unspecified value), q13 (customer admins), q14 (service-grant owner disappearing) and q15 (parent approving child never gains more; unclear whether one of the 14 SymCC properties covers it).
- **Evidence:** PRD:686–698; SPINE:296–306, SPINE:198.
- **Suggested fix:** Add q7, q9, q13, q14 to Deferred and state whether q15 is one of the proved properties.

### P14 — medium — NFR-5 rule 13 moved from check time to write time
- **AD/section:** AD-5.
- **Finding:** The PRD holds rule 13 "by construction at check time"; AD-5 places it only "once, at the store, when a write is accepted". This may be the right call (a service grant must not cascade from its creator), but it is a silent reclassification of a fixed rule.
- **Evidence:** PRD:599; SPINE:94.
- **Suggested fix:** State the reclassification and its reason in AD-5, as the PRD did for rule 3.

### P15 — medium — §10 floor of four and AD-16 "unclassified counts as production"
- **AD/section:** AD-16.
- **Finding:** The floor of four being absolute is Skander's decision, unconfirmed by Danny; AD-16 widens it so every unclassified resource is production, meaning no action on it can be marked contained until an administrator classifies it. That adds a precondition to FR-6 (a person with authority over the resource marks contained) and to UJ-1, and the spine doesn't flag the floor as an assumption.
- **Evidence:** PRD:708, PRD:226, PRD:57; SPINE:204–206.
- **Suggested fix:** Note in AD-16 that this rests on the §10 floor-of-four assumption and that UJ-1 requires prior classification.

### P16 — medium — FR-10 / SM-9b 2-second rule approval, and contained 4 h maximum
- **AD/section:** AD-1, AD-6.
- **Finding:** Rule approval goes through the single-writer store over gRPC, but no AD carries the 2 s target, the 4 h contained maximum, or "the grant records the rule as approver".
- **Evidence:** PRD:287, PRD:292–295, PRD:656; SPINE:52–58, SPINE:103.
- **Suggested fix:** Add these to AD-5's write-time list and give the 2 s target to the store path.

### P17 — medium — FR-16 service-grant rules largely unbound
- **AD/section:** AD-5, AD-6.
- **Finding:** Beyond "gaining nothing new" and `MayCreateServiceGrant`, the spine has no rule for: approval only by a person (never rules or creator), owner required, never handed down, not a child of the creator, owner alert on agent creation. The review date (a PROPOSAL pending q5) is treated as existing in Deferred (D14).
- **Evidence:** PRD:373–382, PRD:678; SPINE:94, SPINE:303.
- **Suggested fix:** List these in AD-5/AD-6 and mark the review date as pending q5.

### P18 — medium — PRD §6 non-goal "not a service called over the network"
- **AD/section:** AD-1, Structural Seed.
- **Finding:** The spine introduces a store process reached over gRPC for requests, approvals and signals; this is reasonable, but the spine never reconciles it with the non-goal, and after P2 it may also be on the check path.
- **Evidence:** PRD:614; SPINE:39, SPINE:268–276.
- **Suggested fix:** State in AD-1 that the non-goal is read as "no network call on checks", and get that reading confirmed.

### P19 — medium — FR-25 tamper attempts recorded and alerted
- **AD/section:** AD-2.
- **Finding:** AD-2 denies `UPDATE`/`DELETE` to the app role, but FR-25 also requires any attempt to change or delete an audit record to be recorded and to raise an alert; and old/new values plus source per record are not stated.
- **Evidence:** PRD:485–488; SPINE:69.
- **Suggested fix:** Add a trigger or log-based detection of attempts (including by other roles) to AD-2, and the required audit fields.

### P20 — low — FR-20 flood alert (more than 20 signals per minute from one source)
- **Evidence:** PRD:420; SPINE (absent). **Fix:** add to AD-18 or the store's signal handling.

### P21 — low — FR-3 task/agent validity at creation and unreachable source later
- **Evidence:** PRD:190–192; SPINE (absent). **Fix:** state in AD-5 that the store validates task and agent at write time and that an unreachable source never ends a grant.

### P22 — low — FR-1 missing fields refused; FR-2 incomplete grant invalid
- **Evidence:** PRD:160, PRD:179; SPINE:152 (only extra fields refuse). **Fix:** add "any missing field refuses" to AD-10.

### P23 — low — FR-19 / SM-3 10 s vs NFR-1 contained exception
- **Evidence:** PRD:406, PRD:650 vs PRD:558; SPINE:84. **Fix:** state in AD-4 that a cut-off replica may honour a withdrawn *contained* grant until the second horizon, so FR-19's 10 s holds for contained grants only on connected replicas.

### P24 — low — AD-7 truncates `now` down
- **Evidence:** SPINE:114 vs PRD:609 ("skew can only shorten access"). Truncating down can lengthen access by under 1 ms unless the margin is added first. **Fix:** state the order (add margin, then truncate) or truncate up.

### P25 — low — loose binds
- **Evidence:** SPINE:53 (AD-1 "all FRs that change state"), SPINE:156 (AD-11 range FR-14–FR-20 sweeps in FR-16–FR-18, which it doesn't address), SPINE:210 (AD-17 binds FR-24). **Fix:** list specific FRs.

### P26 — low — FR-22 single explanation for the list, FR-27 refusal names the restriction, FR-6 audit old/new values
- **Evidence:** PRD:453, PRD:532, PRD:232; SPINE:192–198, SPINE:182. **Fix:** one clause each in AD-15, AD-13, AD-2.

### P27 — low — PRD-internal tension carried over: FR-17 "withdraws its own child grants" vs FR-16 "a service grant is never handed down"
- **Evidence:** PRD:388, PRD:380. **Fix:** flag back to the PRD; the spine needn't resolve it.

## Numbers check

| Number | PRD | Spine | Status |
|---|---|---|---|
| Withdrawal target 10 s | PRD:124, 406, 417 | SPINE:84 (first horizon ≤ 10 s) | matches; OpenBao 10 s (SM-4) absent (P8) |
| 250 µs p99 | PRD:569, 652 | SPINE:130 | matches; unreconciled with pre-answer records (P2) |
| 50,000 grants per store | PRD:570 | SPINE:143 | matches; 31 MiB / 50,000 ≈ 650 B arithmetic checks out |
| Depth 3 | PRD:366 (tentative) | SPINE:126 | value matches, but hard-coded (P11) |
| 24 h / 8 h / 4 h maxima | PRD:188, 287 | absent | P1, P16 |
| 2 s rule approval | PRD:292, 656 | absent | P16 |
| 20 signals/min | PRD:420 | absent | P20 |
| 14-day admin alert, 90-day authority | PRD:270, 242 | absent | P3, P5 |
| 1 month replay | PRD:476 | absent | P7 |
| 25 decision-table checks | PRD:144 | companion only | ok |

## Addendum check

- Danny's answer: identity (SPINE:227), several administrators (implied by AD-16's "different administrator"), Keycloak admin role not trusted (not stated in spine; GS:113 covers it), sharing needs explicit authorisation (AD-5), agents may create service grants (AD-6). No contradiction.
- Seeds 1, 3, 5, 6, 7, 8: consistent with AD-10, AD-18, AD-5, AD-4/AD-11, AD-2/AD-12, AD-13. Seed 2 (Sam) implicitly decided by the schema (P12). Seed 4: relies on the sharing limit the spine doesn't name (P10).

## Coverage table

| PRD item | ADs that address it | Status |
|---|---|---|
| FR-1 | AD-7, AD-10, AD-18 | partial: over-maximum refusal (P1), missing fields (P22) |
| FR-2 | AD-11, AD-12 | ok; invalid-grant rule (P22) |
| FR-3 | AD-5 (check-time expiry), AD-11 | partial: no write-time maximum (P1), validity rules (P21) |
| FR-4 | AD-5, AD-11 | partial: who may narrow (P3) |
| FR-5 | AD-16 | ok (strengthened, P15) |
| FR-6 | AD-5, AD-6, AD-16 | ok; audit fields (P26) |
| FR-7 | none (AD-1 generic, AD-10 PersonID) | **missing** (P3) |
| FR-8 | AD-5, AD-6 | partial: sharing limit (P10) |
| FR-9 | AD-16 (wrong bind) | **missing in spine** (P5) |
| FR-10 | AD-6 | partial (P16) |
| FR-11 | AD-5, AD-10, AD-18 | ok |
| FR-12 | AD-8, AD-10 | partial (P12) |
| FR-13 | AD-5, AD-18 | ok (dependency stated) |
| FR-14 | AD-5, AD-6, AD-11 | ok |
| FR-15 | AD-5, AD-11 | ok |
| FR-16 | AD-5, AD-6 | partial (P17) |
| FR-17 | AD-11 (range only) | weak (P25, P27) |
| FR-18 | AD-8 | partial (P11) |
| FR-19 | AD-3, AD-4, AD-11 | ok; authority (P3), contained exception (P23) |
| FR-20 | AD-3, AD-11, AD-18 | ok; flood alert (P20), auth deferred |
| FR-21 | AD-6, AD-7, AD-9, AD-17 | ok |
| FR-22 | AD-9, AD-15 | ok; budget undeferred (P13) |
| FR-23 | AD-6, AD-14, AD-17 | ok |
| FR-24 | AD-2, AD-11, AD-12 | partial (P6, P7) |
| FR-25 | AD-2, AD-14 | partial (P19) |
| FR-26 | AD-13 | partial (P8) |
| FR-27 | AD-13 | partial (P9) |
| FR-28 | AD-13 | ok (untested listing) |
| FR-29 | AD-2, AD-12, AD-14 | partial (P2, P6) |
| NFR-1 | AD-4, AD-7, AD-17 | ok |
| NFR-2 | AD-3, AD-4 | ok (values deferred) |
| NFR-3 | AD-1, AD-9 | partial (P2) |
| NFR-4 | AD-1, AD-7, AD-18 | partial (P4) |
| NFR-5 r1 | AD-5, AD-8, AD-7 | ok |
| NFR-5 r2 | AD-10 | ok |
| NFR-5 r3 | none | **missing** (P1) |
| NFR-5 r4 | AD-5, AD-8, AD-16 | ok |
| NFR-5 r5 | AD-5, AD-8, AD-10 | ok |
| NFR-5 r6 | AD-5, AD-8 | partial (P12) |
| NFR-5 r7 | AD-5, AD-18 | ok (dependency) |
| NFR-5 r8 | AD-5, AD-8 | ok |
| NFR-5 r9 | AD-5, AD-8 | ok |
| NFR-5 r10 | AD-5, AD-13 | partial (P9) |
| NFR-5 r11 | AD-5, AD-8 | partial (P10) |
| NFR-5 r12 | AD-5, AD-8, AD-10 | partial (P10) |
| NFR-5 r13 | AD-5, AD-8 | reclassified (P14) |
| NFR-6 | AD-4, AD-12 | ok; truncation (P24) |
| SM-1 | none | missing (P1) |
| SM-2, SM-3 | AD-3, AD-4 | ok |
| SM-4 | AD-13 | no target (P8) |
| SM-5 | AD-9 | partial (P2) |
| SM-6 | AD-2, AD-12, AD-14 | partial (P6, P7) |
| SM-7 | AD-13 | ok |
| SM-8 | companion decision-table.md | ok |
| SM-9b | none | missing (P16) |
