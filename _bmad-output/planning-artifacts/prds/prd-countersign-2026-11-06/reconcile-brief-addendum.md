# Reconciliation: brief addendum → PRD

- **Source input:** `briefs/brief-countersign-2026-10-30/addendum.md` (Danny's emails 2026-11-04 and 2026-11-06, the 2026-11-04 follow-up, mechanism notes, proposed test cases)
- **Compared against:** `prds/prd-countersign-2026-11-06/prd.md` and `prds/prd-countersign-2026-11-06/addendum.md` (Danny, 2026-11-11)
- **Date:** 2026-11-11

Superseded material, not counted as gaps:
- Skander's strict option (a): requests in the spawn chain count as the agent's own, task budget, separate oversight role. Replaced by Danny on 2026-11-06.
- Fallback option (b), the "didn't create the task, doesn't benefit" rule. Danny on 2026-11-06: no need to work out who benefits.
- The brief's splitting test case ("count together against the task's budget"). Danny on 2026-11-06 ruled out a task-budget system. The PRD swaps it for the FR-13 attack test and lists the task budget as out of scope (§7.2). That is correct.
- "Identity: not answered" and "Translation: not answered" (2026-11-04 §4). Both were answered on 2026-11-06, and the PRD reflects the answers (NFR-4, §6, FR-27).

## Gaps

### G1. The withdrawal clock vs. the staleness rules (contradiction)
- **Brief addendum** (Mechanism notes, "Withdrawal cascades"): the clock starts at the withdrawal, not when a replica hears of it. A replica that has lost contact refuses high-risk requests once its staleness horizon passes, so a withdrawal can't fail silently.
- **PRD:** FR-19 (lines 311–312) and SM-C3 (line 540) say a withdrawn grant is never allowed after the 10 s target, with the clock starting at the withdrawal. NFR-2 (lines 450–453) lets a stale replica keep answering high-risk checks until the staleness horizon, and contained checks until a second, longer horizon. Both values are open.
- **Problem:** a disconnected replica can keep honouring a withdrawn grant past 10 s: until the staleness horizon for a high-risk grant, and until the second horizon for a contained one. The PRD never says that the staleness horizon must be at most the withdrawal target. It also doesn't scope the 10 s target to connected replicas. One of the two has to give, and the PRD has to say which. SM-C4 (measure the slowest replica) makes the clash measurable but doesn't resolve it.

### G2. The floor of four is stated as settled, but it was Danny's open question
- **Brief addendum** (§2, "Still open"): "Which actions are human-only? Are production changes, real secrets, money and deletion always human-only?" Danny's 2026-11-06 and 2026-11-11 answers don't settle this. He says only "anything that requires a human goes to a human".
- **PRD:** the Glossary (line 68), FR-5, FR-6 and NFR-5 rule 4 treat the floor of four as fixed and impossible to remove. It isn't tagged `[ASSUMPTION]` or `[PROPOSAL]`, and it isn't listed in §9 or §10.
- **Problem:** an open client question is presented as a client decision. Either ask Danny to confirm it, or tag it as Skander's proposal.

### G3. Permissions are separate from Forgejo, and grants don't bypass infrastructure review (dropped)
- **Brief addendum** (§3): each installation's Forgejo holds the infrastructure configuration, and changes made through the console, the CLI or agents become Git commits with one reviewed history. Permissions live separately, in the permission store. Danny asked to correct "grants are Git files". Fast grants "do not bypass the review required for the infrastructure change itself".
- **PRD:** the Glossary has "Permission store" and "Installation" but never says the store is separate from Forgejo or Git. No requirement says a grant never replaces or bypasses the Forgejo review of an infrastructure change. FR-13 (line 240) calls the production gate "Skander's reading of Danny's Forgejo answer, not his words", and doesn't cite this sentence, which is the closest thing Danny actually said.
- **Problem:** a client correction and a client constraint are missing as requirements. The FR-13 note also understates how much client backing it has. Danny's sentence covers review, though, not a Countersign check, so question 6b is still needed.

### G4. Contained grants can never be shared: stricter than Danny, and not flagged
- **Brief addendum** (Danny, 2026-11-06): a parent agent can approve access for its child if it has permission to do so, and it can only pass on access it is allowed to share. The 2026-11-11 answer in the PRD addendum keeps this: a parent can share "only the permissions it has been explicitly authorised to share".
- **PRD:** the Glossary "Allowed to share" (line 71), F3 (line 199), FR-14 (line 255) and FR-18 say contained grants can never be shared in v1, and that "allowed to share" exists only for human-only grants.
- **Problem:** Danny's wording allows explicitly authorised sharing of any permission. The PRD drops this for contained grants. That may be a sound v1 simplification, since the child can make its own request. But it is Skander's decision, presented in the Glossary as if it came from Danny, and it is missing from §10 and §9.

### G5. Service access "that lasts on its own" vs. a mandatory hard maximum lifetime (minor tension)
- **Brief addendum** (Danny, 2026-11-06): someone with the right permission can give a service access that lasts on its own. For example, an employee leaving shouldn't break a service they created.
- **PRD:** FR-3 (line 126), FR-16 (line 289) and SM-1 require a hard maximum lifetime for service grants too, and renewal is a new approval (`[PROPOSAL]`).
- **Problem:** the lifetime and the renewal step are Skander's additions. The team or role ownership softens the "employee leaves" case, but a service can still break at the hard limit if nobody renews. §9 question 5 covers the review date, not the hard limit itself. Add the hard limit to the question for Danny.

### G6. Delay before Countersign learns a task closed (minor, dropped)
- **Brief addendum** (Mechanism notes, "Access ends with the task"): the delay between the tracker and the event is outside Countersign's control.
- **PRD:** SM-2 and FR-20 measure from when Countersign learns of or receives the signal, which is correct. But the PRD never says that the time from the task closing to the signal is unbounded and not Countersign's responsibility. The hard maximum lifetime is named as the backstop only for when a signal *never* arrives (F5, line 303), not for when it arrives late.

## Covered (no gap)
- No screens of its own: §6 Non-Goals.
- No self-approval, and human-only goes to a human: FR-11, FR-12, NFR-5 rules 5 and 6.
- Who marks an action human-only: a human with authority (F2, FR-6).
- Grants, approval authority and delegation limits are audited in the permission store: Glossary, FR-25.
- Identity: Keycloak for people, Kubernetes for agents and services; Countersign owns the decision (NFR-4, §6).
- A parent approves for its child only for access it is allowed to share; no rule about chains or who benefits: F3, FR-12 (exact requester only), FR-14.
- Delegated access ends with its parent: FR-15, FR-20.
- Splitting can't get around the final action, and there's no task budget: FR-13, §7.2.
- No translation into Keycloak or Kubernetes RBAC: §6.
- OpenBao enforcement, and access that can't be enforced is not allowed: FR-26, FR-27.
- Withdrawal cascade test (3 children and 1 grandchild, all 5 refused within 10 s): FR-19.
- "Ask Danny before publishing the email": kept in the PRD addendum.
