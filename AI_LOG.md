# AI_LOG

How I used Claude (Opus, via Claude Code) and BMAD v6 on Countersign, and where I overruled them.

## Tools

- **Claude Code** with Claude Opus 5.5 (`claude-opus-5-5`), run from the VS Code extension.
- **BMAD v6.12**, installed with `npx bmad-method install`, with Claude Code as the chosen IDE.

## Session log

### 2026-10-30: setup and understanding the brief

- I asked Claude to read the Orvex intern brief PDF and summarise it.
- I set up the working repo: `git init` in `countersign/`, then renamed the branch to `main`.
- I started `bmad-product-brief`. Claude created the brief workspace at `_bmad-output/planning-artifacts/briefs/brief-countersign-2026-10-30/` and opened the discovery questions.
- **My steering:** I stopped the product brief before answering the discovery questions. I asked Claude to split the Orvex PDF into modules and quiz me on one module at a time, so that I understand the problem before I write the brief. The aim is for the brief to be my own thinking, not a reshuffled copy of the PDF.
- **Ground rule for the Q&A (set 2026-10-31):** my answers in the Q&A modules are only for understanding the problem. They are not decisions and must not be copied into the brief, PRD or architecture. Those documents will be written from scratch in their own BMAD steps.

### Q&A Module 1: the world, and why one engine

- **I identified:** revocation drift between the two products, delegated access that survives the parent being removed, and the trade-off of a library over a central service (fast checks, but revocations must reach every copy). The users I named were the developers who integrate the library and the security team.
- **Claude added:** approval shopping across inconsistent rules; the framing that the fix is "one source of truth", not "better sync"; library version skew between the two products; the clash between default deny and stale data; and two more user groups (approvers at risk of approval fatigue, and agents with their authors), plus customer admins implied by "10,000 installations".

### Q&A Module 2: default deny, and grants that end

- **I identified:** that errors from a stale copy are asymmetric (a stale withdrawal wrongly allows, which is the dangerous one); that "refuse all" during a network failure pushes teams towards fail-open bypasses; that "task closed" should be an event treated like a withdrawal; and that every task grant still needs a time limit as a safety net, because events get lost and tickets stay open.
- **My own choice:** I asked Claude to explain how task-bound grants could work, reviewed the options it gave me, and chose a task-closed event plus a mandatory time limit.
- **My clock-skew design:** a slow clock extends access, so the library counts elapsed time locally, and the grant expires as soon as either the clock or the timer says so.
- **Claude added:**
  - a middle option for staleness: a lease (bounded staleness) whose length depends on the risk of the action, with withdrawals on a faster path
  - authenticating the "task closed" signal, plus periodic checks so a lost event costs minutes, and short renewable time limits
  - the propagation-delay hole in a timer that starts on arrival; the fix is that the issuer states the remaining lifetime and a skew margin is subtracted
  - keeping wall-clock timestamps for replay

### Q&A Module 3: who approves

- **My own answers:**
  - Rules may approve only if every condition holds: the resource is non-production, the effect is small and reversible, and so on.
  - Production is known from a namespace label set by the system or a developer, never from the name.
  - An unlabeled resource is treated as production (fail closed).
  - My verdicts on the self-approval cases: (a) refused, (b) not needed, because rules auto-approve non-production, (c) allowed.
- **Where I used AI:** I used Claude to understand two parts.
  - The approval rule as a condition: a human approver, not the requester, not in the requester's lineage, and holding approver authority.
  - The "impossible to express" design: a separate `HumanApprover` Go type, approvals only from a human session, schema validation, and property-based tests with a decision-time check.
- **Claude pushed back on:**
  - Changing a label must itself need human approval.
  - "Already holds" should be a pre-approved envelope.
  - A budget on automatic approvals to stop splitting.
  - Rules read structured fields only, never the free-text "why".
  - My reasoning for (c) was wrong: Erin and Sam are both humans.
  - "Who grants approver authority?"
  - Lineage can leave a small team with nobody able to approve.
  - The proof must also cover the translated native rules (for example, Argo CD sync).
  - A human token can be borrowed.
  - Property-based tests are not proof (an analyzer such as Cedar's can be).
- **Idea Claude suggested, which I rejected:** simplifying the fixed rule to "agents never approve anything" (see overrule 1).

### Q&A Module 4: delegation and withdrawal

- **My own answers:**
  - Verdicts: (a) valid, (b) refused (outlives parent), (c) refused (deploy is not push). A child is valid only if its action and expiry are equal to or below the parent's.
  - Wildcards: I proposed banning free wildcards and allowing only exact paths or an explicit subtree marker.
  - Cascade: delete-down-the-tree is fast but can miss a child or leave a stale copy active; check-up-the-chain is safer at the cost of walking the chain. To combine them, publish one withdrawal record to the whole system.
  - Approval: delegation inside the parent's envelope needs no new approval, and anything wider is a new request. My explanation format includes the snapshot: "all ancestors valid at snapshot N".
- **Claude added:**
  - Pick exactly two path forms (`api/billing`, `api/**`).
  - Every field must shrink, including environment and whether further delegation is allowed (a flag and a maximum depth).
  - Actions are flat or explicitly ordered.
  - Canonicalise paths to block `..`, case and encoding tricks (an attack case for the 25 checks).
  - Broadcast only the parent ID and check the chain at decision time, with delegation depth capped so the check stays cheap.
  - The stale-copy gap is bounded by the Module 2 lease.
  - Don't copy the human into the child's `approver` field (it falsely claims they approved it); record `delegated_by` and `parent` instead.

### Q&A Module 5: explaining and replaying answers

- **My own answers:**
  - A decision record holds the request, the decision, reasons (grant and rule IDs), the state snapshot ID, the decision time, the engine version and the approver chain.
  - Runtime changes that are not in Git go into an append-only log.
  - Refusals carry stable reason codes and cite the closest grants, with the condition each one failed.
  - "Who could approve" means humans with authority over the resource, excluding the requester and their lineage. I flagged that naming approvers or revealing that a resource exists can leak information.
  - Volume: store a compact record for every decision and rebuild explanations via replay; always store production, secrets, deletion, approvals and withdrawals in full.
- **Claude added:**
  - Time must be an explicit input (the engine never calls `now()`), so replay is deterministic.
  - The state version is a pair: (Git commit, event-log position).
  - The event log must be tamper-evident (hash-chained).
  - Decisions made natively by Kubernetes or OpenBao can't be replayed by Countersign (an open question).
  - Showing the agent its closest grants contradicts my own leak warning. The fix: the agent gets a reason code and an approval handle; humans with permission get the full explanation; use one reason code for "doesn't exist" and "forbidden".
  - Claude read my "outsiders" as attackers outside the system. I corrected it: I meant agents that are neither in the requester's lineage nor have authority over the resource, i.e. internal agents with no business knowing about it.
  - Retention must cover the replay window, and old engine versions must stay runnable.

### Q&A Module 6: translation into native tools

- **Mostly AI:** I used AI to answer these questions so I could understand how translation works. Most of the design is not mine:
  - the Kubernetes Role and RoleBinding plus the OpenBao policy, with a short-TTL token
  - expiry enforced twice (translator deletion and short-lived credentials)
  - errors always toward less access, so a grant is written only if native allows ⊆ Countersign allows, and otherwise it is "untranslatable"
  - agreement tests with `kubectl auth can-i` and OpenBao capabilities in CI
  - drift detection by watching RBAC events plus a periodic full comparison
- **Claude's review found problems in the AI-written answer:**
  - Pod creation, `exec`, `impersonate`, `bind` and `escalate` widen secret access beyond RBAC `get`.
  - Each agent needs its own service account.
  - Approver and task annotations leak information to anyone who can read RBAC; store only the grant ID.
  - Expiry can't un-read a secret, so rotate production secrets after the task.
  - Reject untranslatable grants at validation time, not at use.
  - Use finer-grained native mechanisms (for example, Harbor robot accounts).
  - "Translator never touches objects it didn't create" contradicts auto-revert.
  - Drift must be detected as "no matching grant", because the `managed-by` label can be forged.

### Q&A Module 7: the grant model

- **Status:** these are my Q&A answers and working hypotheses, not final decisions for the hand-in. **I'm not confident in the schema yet; it needs to be reworked** (during the brief or the architecture).
- **My own answers:**
  - Model: ABAC-style fixed fields at the core, no roles, no groups, and one `parent` pointer forming a plain tree, which is the only ReBAC-like part. Roles appear only as translation targets. Groups are expanded into individual grants at approval time.
  - I listed what is lost by not choosing each of the other models.
  - Schema: `id`, `principal`, `action` (closed enum), `resource` (canonical, exact or one subtree marker), `environment` (from the registry), `task`, `approver`, `expires_at`, `parent`, `delegated_by`, `max_depth` (default 0).
  - Left out on purpose, with a reason for each: deny grants, free-form conditions, time-of-day rules, IP ranges, roles, groups, extra wildcards.
  - Grants are data and policy is logic. The fixed rules live compiled in the library. The approver comes from the authenticated Git review, enforced by branch protection plus a CI check.
- **Claude added:**
  - Approver authority must itself be an `approve` grant held only by humans.
  - Expanding groups has join/leave costs.
  - Schema conflicts: `approver` must allow a rule ID (requirement 03 auto-approval) and be required only when `parent` is unset (Module 4).
  - `task` being required blocks on-call incidents with no ticket.
  - Add `issued_at`.
  - Re-read `environment` at decision time.
  - Attack case: approve, then push a widening commit (cancel stale approvals).
  - Git host admins can bypass branch protection, so they are the trust root.

### Q&A Module 8: speed, lists and scale

- **My own answers:**
  - A check uses an index by principal, then grant match, expiry and withdrawal check, then the answer. The p99 risks are the Go garbage collector, snapshot updates and the decision record.
  - Lists: filtering in the query is faster than filtering afterwards, but the filter must follow the grants exactly, including withdrawals mid-read.
  - Memory: about 500 B per grant, so about 25 MB for 50k.
  - p99 plan: Go benchmarks plus a load run (p50/p99/p99.9), the 25 cases plus generated traffic, a worst-case principal with 5,000 grants and maximum chain depth, reloads, logging and concurrency on.
  - Rejection criteria: p99 above 1 ms in the worst case, allocations on the hot path, or fixed rules that can't always win. **I fix the thresholds before running the spike.**
- **Claude added:**
  - The concrete lookup path (exact path, then `api/**`, then `**`): about hundreds of nanoseconds.
  - Mitigations: no allocations per check, copy-on-write snapshots with an atomic pointer swap, and an asynchronous decision log.
  - An open question: when the log buffer is full, drop the record or refuse? (Proposal: required for high-risk decisions, droppable for low-risk.)
  - Lists: combine the query filter with a per-item check for high-risk items; streaming subscriptions are cancelled on withdrawal; record one decision per filter; cap the filter size and have a fallback.
  - Memory is realistically 50–150 MB with indexes and the swap, and it is paid per process, not per installation.
  - "Memory per check" means allocations per check (`B/op`, `allocs/op`).

### End of the Q&A (2026-11-02)

- I stopped the Q&A after Module 8. I skipped Module 9 (engine choice and the 25-check spike) and Module 10 (deliverables and judging), because the BMAD architecture step covers the engine spike directly.
- I asked Claude to re-read the Orvex brief before going back to the BMAD workflow.

### 2026-11-02: product brief (BMAD step 2, `bmad-product-brief`)

- **Mode:** I chose the coaching path (section by section, content drawn from me) over the fast path (Claude drafts with `[ASSUMPTION]` tags). I want the brief to be my own thinking.
- **Research:** Claude ran a background research subagent on agent authorization in 2025–2026: standards, vendors, incidents, attenuation tokens and the six engines. The digest is saved, unverified, as `research-agent-authz.md` in the brief workspace. It is reading material, not brief content, and I need to check sources before citing any of it.
- **My discovery answers (my own words):**
  - The pain: nobody can tell what agents can do and why. Agents act in seconds and keep access after the task ends.
  - Three users: Quornex/Houston engineers, approvers, and the security team.
  - Success: access ends with the task, withdrawal cascades, and "why was this allowed?" is answered from one record.
  - Failure: people bypass it because it is slow, approvers rubber-stamp, or native tools diverge unnoticed.
- **Claude pushed back on:**
  - My pain statement covers only agents, though the brief covers people too.
  - The "two rule sets" half of the pain is missing.
  - I stated unverified facts (approvers overwhelmed today, past incidents). These are now logged as assumptions to check with Danny.
  - Two users are missing: agents with their builders, and customer admins.
  - The users have no ranking.
  - My success signals need numbers.
- **My second round of answers:**
  - People have the problem too (forgotten admin roles), but agents make it urgent.
  - The two rule sets drifting apart is part of the problem.
  - I admitted the approver-overload and incident claims were guesses; they are now questions for Danny.
  - Primary user: integrating engineers; second: the security team.
  - Conflict rule: security wins on permissions, engineers win on speed, and "a fast refusal is acceptable; a wrong allow is not".
  - Success targets: 100% of grants expire, access ends within 10 s of the task closing, no unmanaged agent access, p99 under 1 ms, and decisions are replayable.
- **Draft:** Claude drafted "The Problem" and "Who This Serves" in `brief.md` from my answers. Its own addition is the "Open, to check with Danny" note, and it kept the customer-admin scope as an `[ASSUMPTION]`. I will review and edit them.
- **Success numbers (mine):**
  - Access ends within 10 s of Countersign *learning* that the task closed (Claude asked "done according to whom?").
  - 100% of decisions are replayable for at least one month (I had typed "100M").
  - About 90% of requests are auto-approved, and each approver handles fewer than 20 requests a day.
- **Draft:** Claude drafted the "Success Criteria" table from these numbers. It left the maximum-lifetime safety net `[OPEN]` because I hadn't answered it yet.
- **Consistency check:** Claude noticed that the draft said agents "never approve anything" (taken from my own answer), which contradicted overrule 1. I kept overrule 1, and the line now states the brief's fixed rule.
- **More of my decisions:**
  - Every grant has a hard maximum lifetime that nothing can extend; the value is still open.
  - "90% auto-approved / fewer than 20 a day per approver" is an expected result, not a target. Claude pointed out that, as a target, it could push people to widen the auto-approval rules.
  - **I declined Claude's offer to cite public incidents (Replit, Amazon Q) in the brief. It stays about Orvex only.**
  - I didn't take Claude's suggested hard target of "0 high-risk requests approved by rules alone".
- **The Solution (my words):** Countersign is a library both products call. I described the `db-migrate` / Erin request from start to end. It is not an identity system (to confirm with Danny), does not store secrets, and sits above the native tools without replacing them. Claude drafted the section from my answer.

### 2026-11-04: questions to Danny, and his answers

- **The questions:** Claude flagged that the approval screen and "grants are Git files" vs "access in seconds" were unclear. I wrote an email with four questions and used Claude to correct its wording before sending it to Danny:
  - where identity and approver authority come from
  - whether "untranslatable means don't translate" is the right default
  - where a human sees and approves a request
  - Git vs seconds
- **Danny's answers** (paraphrased in the brief's `addendum.md`):
  - Quornex and Houston provide the approval screens.
  - **Agents can approve**: a supervising agent may approve another agent's request within explicitly delegated limits. Nobody approves their own request, and human-only actions still go to an authorised person. He considers the brief's blanket "agents never approve" too restrictive.
  - **Grants are not Git files.** They live in each installation's permission store (grants, approval authority and delegation limits, with every change audited). Forgejo holds the infrastructure configuration as reviewed commits, so access takes seconds without a pull request but doesn't skip review of the infrastructure change itself.
  - He didn't answer the identity question or the translation default.
- **Brief updated:**
  - added the common case (rules approve `api-refactor` in seconds)
  - added the permission store and the fact that Countersign has no screens
  - marked the agent-approval line and the customer-admin role for revision
  - recorded the clarifications in `addendum.md`
- **My decisions after Danny's answer:**
  - **Human-only floor:** production, real secrets, money and deletion are always human-only. Someone with authority can add markings to other actions but never remove from the floor, and an unmarked or unreadable action is treated as human-only.
  - **The new fixed approval rule, in my words:**
    - Nobody approves their own request.
    - A human-only action is approved only by an authorised person.
    - A supervising agent approves only what was explicitly delegated to it: the same or a narrower action and resource, never for longer than its own authority, never a human-only action.
- **Follow-up:** I sent Danny a follow-up on the two unanswered questions (identity source, untranslatable default). Until he answers, they are working assumptions in the brief's Open Questions.
- **Claude challenged two holes in my rules:**
  - **Fail-closed default vs the 90% goal.** If unmarked means human-only, rules approve nothing by default. I agreed and moved to three explicit classes: the floor of four, actions marked contained, and everything else human-only. Only a human with authority over a resource can mark an action contained. 90% becomes a goal to measure.
  - **Proxy self-approval.** I decided that a request made on my behalf counts as mine. `release-prep` can't approve `helper`'s request, approval must come from outside the lineage, and any doubt is treated as "own".
- **Lineage contradiction:** Claude pointed out that "a supervisor higher up" is itself in the lineage, so my rule blocks every ancestor. That may clash with Danny's "supervising agent" model. Claude offered three options: (a) strict, (b) narrow "own" to the task's chain, (c) ask Danny.
  - **I chose (a)** as my fail-closed position, and also emailed Danny. In the email I added the splitting attack (ten small "contained" requests adding up to an unseen production change) and proposed that a task's requests count together against a budget. My fallback, if he means the agent above, is that an ancestor approves only if it didn't create the task and doesn't benefit from the request.
- **What Makes This Different (my answers):**
  - Syncing fails because there would still be two sources of truth.
  - Buying a product is rejected on fit, and I marked that as "to confirm, not a proven gap".
  - A raw engine lacks task expiry, delegation, approval and translation, but Countersign may use one inside, and the spike decides.
  - The one thing: one shared model with approval, delegation and withdrawal built in.
  - Explicit non-claims, including "I haven't found one that does it all", not "none can".
  - Claude drafted the section from my words.
  - On Claude's suggestion, I added that "why not buy" is partly Orvex's own constraint (the PDF's "Built on Go, embeddable open source"), not only my judgement about fit.
- **Scope (my answers):**
  - In v1: the 8 requirements, the three action classes, the new fixed rules, and translation into Kubernetes RBAC and OpenBao first, with drift detection.
  - Postponed: Argo CD, Harbor and Grafana (coarser formats, same idea).
  - Out of v1: customer policy code, identity, screens, multi-installation, group and deny grants, and rich budget handling.
  - The brief states that my hand-in is a design, not a working product.
  - Claude drafted the section from my words.
  - Claude asked who governs Argo CD (production deploys) while it's postponed. I wrote the answer myself: agents get no Argo CD access, production syncs stay with people, and the gap is stated, not hidden.
- **Vision (my words):** Countersign becomes the one place Orvex answers "who can do what, and why?", and grows step by step. Its limit is that no feature may break "access only shrinks, ends on its own, every answer explained".
- **Executive Summary:** Claude drafted it from my sections and tagged it `[DRAFT by Claude]`, for me to rewrite in my own voice.
- **Vision correction:** Claude asked whether "customers write their own grants" was what I meant. I corrected it to "safe ways for customers to configure their own rules" (what counts as contained, who may approve, limits), typed and still without policy code.
- **Executive Summary, overruling Claude's emphasis:** Claude's draft put the four human-only classes up front. I changed the order to the promise first, then the problem, then the safeguards, with the human-only classes as just one safeguard. I wrote the opening line myself and asked for a clause on supervising agents (Danny's correction). Claude rewrote the rest around my line and order.

### 2026-11-04: brief finalize, memlog audit

- Claude classified all 63 memlog entries: in the brief, in Open Questions, in the addendum, superseded, deliberately kept out, or process noise.
- **The gap it found:** my success signal "withdrawal takes back everything handed down" had no row in Success Criteria.
- **My target:** 100% of descendants are withdrawn and refused within 10 s (tentative) of the withdrawal. The clock starts at the withdrawal, and a copy that has lost contact refuses high-risk requests after its staleness horizon.
- **My test case:** a parent with 3 children and 1 grandchild, where all 5 must be refused. It is saved, with the splitting attack, in the addendum for the decision table.

### 2026-11-04: brief polish (structure and prose reviews)

- **The reviews:** Claude ran BMAD's review lenses (structure, then prose) on the brief and the addendum as four subagent passes. There were about 55 findings, all about organisation and wording, none about decisions.
- **My decisions:**
  - **(a)** Keep the brief self-contained and only remove repetition, rather than trimming it to 1–2 pages. It went from about 2,300 to about 2,100 words.
  - Accept all the structure and wording fixes.
  - Merge the "grants that end" and "safety net" rows.
  - Mark the supervising-agent rule as partly pending in the Executive Summary.
  - **Keep British spelling** ("authorised") and **no serial comma**, overriding the review's Microsoft-style suggestions.
- **Where things moved:** the mechanism notes (tracker delay, replica staleness) moved from the Success Criteria table into the addendum.

### 2026-11-06: Danny's answer to both emails

- **Identity:** Keycloak identifies people and Kubernetes identifies agents and services. Countersign owns the policy engine, including who can approve and who can give access.
- **Parent agents:** a parent agent *can* approve its child's access if it has permission, passing on only what it may share. **This replaces my strict option (a)** (no approval from anywhere in the spawn chain). That change came from the client, not from overruling the AI.
- **Access that outlives its parent:** delegated access normally ends with the parent's, but an authorised person can give a service access that outlives its creator.
- **Splitting:** splitting must not bypass the approval needed for the final action, but he doesn't want a complex task-budget system.
- **Translation:** none into Keycloak or Kubernetes RBAC (identity only). For OpenBao, I must explain enforcement. If a restriction can't be enforced, the access is not allowed.
- **Logged:** Claude added a paraphrase to the brief's addendum and the memlog, and listed the brief changes for me to decide.

- **My decisions after Danny's answer:**
  - **Accepted the settled changes:** identity is now fact, "can't enforce means not allowed", the task budget is removed, and there is no Kubernetes RBAC translation (only OpenBao enforcement, with drift detection).
  - **New "Who may approve", in my words:** a parent approves only access it holds and may share, nobody approves their own request, and human-only actions go to an authorised person. On Claude's question "what stops a parent spawning a child just to approve it?", I answered that spawning gives no new power, but I haven't proved it, so it's an open question.
  - **Service grants (my design):** a separate kind of grant, not delegation. It's owned by a team or role, so nothing cascades when its creator leaves. It's still withdrawable with its children, human-only actions still need a person, and creation is audited. I proposed a review date (to check with Danny).
  - **Splitting:** approval is attached to the final action, as a principle. The architecture decides how to enforce it.
  - **Argo CD, Harbor and Grafana:** they stay postponed, with an open question to Danny.
  - **A conflict Claude found:** service grants with "no end" broke "100% of grants end". I chose (a): service grants also have a hard maximum lifetime, the review date comes before it, and renewing means a new approval.

- **The brief is final (2026-11-06):** I approved it after a last read. Its status is now `final`.

### 2026-11-06: PRD (BMAD step 3, `bmad-prd`)

- **Setup:** I started `bmad-prd`. The Skill tool couldn't load the BMAD skills, so Claude followed `bmad-prd`'s `SKILL.md` directly and created the PRD workspace at `_bmad-output/planning-artifacts/prds/prd-countersign-2026-11-06/`.
- **Inputs:** the final product brief and its addendum. The Q&A answers are not used.
- **Fixes I made to the final brief first:**
  - The "What Makes This Different" wording now says OpenBao enforcement instead of Kubernetes translation.
  - The 10 s task-end target is tagged `[tentative]`.
  - A status note says the `[OPEN]` and `[PROPOSAL]` items are waiting for Danny.
  - Two new open questions: where "allowed to share" comes from, and who takes over an orphaned service grant.
- **What I want the PRD to show the judges:**
  - Every FR has a pass/fail check, and the 25-case decision table traces back to the FRs.
  - The fixed rules are provable: which ones by construction and which only by tests.
  - Targets are labelled as targets, and every assumption is visible.
- **Mode:** the coaching path again, with Claude showing my progress through the PRD on every turn.
- **Entry point:** Vision + Features (capability-first), which was Claude's suggestion and which I confirmed. A few short user journeys will make the FRs concrete.
- **Vision:** I asked Claude to restate the brief's Executive Summary for an engineer about to build it, keeping "why it matters" in the brief only. Claude drafted §1 Vision.
- **Feature list (my words):** 15 capabilities, from default deny to staleness.
- **I extended the Vision:** with refusal of unknown, malformed or failing inputs, the service-grant exception, "no screens, no secrets, no identity", and "a fast refusal is acceptable; a wrong allow is not".
- **Grouping:** Claude proposed grouping my list into features F1–F8, with default deny, staleness, speed and integration as cross-cutting requirements. I agreed.
- **The four missing capabilities Claude pointed out (my answers):**
  - A request never names its approver.
  - Markings and approval authority are set only by humans, as audited changes.
  - Task and agent signals count as withdrawals, and untrusted ones are ignored.
  - A grant can only be narrowed or ended early, never widened or extended.
- **Untrusted closing signals:** Claude pointed out that ignoring them contradicted my own "a wrong allow is not acceptable" principle. I changed my answer to accept them, because a forged one only causes a recoverable denial of service. I added a condition: each one is audited with its source, a flood from one source raises an alert, and a signal that widens access is never accepted.
- **Who may narrow or end a grant (my answer):**
  - The holder may, for its own grant.
  - The approver or anyone with authority over the resource may, at that moment.
  - Nobody may widen it, including the approver.
  - A parent may narrow its child's grant only within "allowed to share" (still open).
- **Users and journeys (my words):**
  - One job to be done per user.
  - UJ-3 is a withdrawal scene I wrote: `release-prep` is withdrawn by Erin, the cascade reaches every descendant, running work is not undone, and service grants are unaffected.
  - I added UJ-4: the security team asks "why?" a month later.
  - Claude structured the journeys into the UJ format.
- **Glossary:** Claude drafted definitions from my decisions and flagged the three it was least sure of. I corrected it:
  - "Principal" has one meaning only, and the accountable human is the "owner".
  - I rejected "caller", because it means the calling product.
  - I kept "replica".
  - I added "human-only", "final action" and "installation".
  - I dropped "lineage", because Danny removed the chain rule.
  - I refined the floor of four, signal, grant and check.
- **"Owner" clash:** Claude pointed out that "owner" meant two things. I kept "owner" for service grants. There is no term for the human behind an agent unless an FR needs one.
- **What "ask" means:** Claude noticed that my "ask means a person must approve" was wrong once parent agents may approve. I corrected it: "ask" means approval is needed, and Countersign works out who and lists them. I rejected letting a parent approve before the request reaches Countersign, because a crafted request could then name its own approver.
- **Parent approval vs human-only:** Claude found that under my action classes, a parent agent could never approve anything. I chose (a): parent approval is delegation of a grant it already holds. For a human-only action it needs the approving person's separate "allowed to share" permission. I rejected (b) because parent approval would then add nothing, and I kept my reading of Danny's line as an open question.
- **FRs, method:** Claude drafts each feature's FRs from my logged decisions, tags each gap it fills as `[ASSUMPTION]`, and I review the feature before we move on. F1 was drafted as FR-1 to FR-4, with two assumptions.
- **F1 review (my answers):**
  - I confirmed that an over-long request is refused, and the refusal states the maximum.
  - I changed Claude's "unknown task or agent means ended" to three cases: never valid means the grant is invalid; valid but the source is unreachable means not ended (the time limit and staleness rules apply), so one outage doesn't end every grant; known to be gone means ended.
- **F2:** Claude drafted FR-5 to FR-8, with two assumptions (approval authority has a hard maximum lifetime; the root of authority).
- **F2 review (my answers):**
  - **Root of trust:** I agreed to installation administrators, identified by Keycloak, with my own conditions: at least two administrators, changes are human-only and audited, and first setup is left to the architecture. I'll ask Danny about Keycloak admin roles.
  - **Approval authority expires:** 90 days, tentative. Grants already approved stay valid. Administrators never lapse silently.
  - This became FR-9.
- **F3:** Claude drafted FR-10 to FR-13, with one assumption: a person and an agent running as that person are different principals.
- **Lost edits:** my editor saved an old buffer over `prd.md` and dropped FR-9 to FR-13. Claude noticed when I said they were missing, and re-applied them.
- **FR-12 (my answer):** I kept Danny's literal rule (only the exact requester is excluded). I added the "Sam approves their own agent's production deploy" case as an open question for Danny and as a decision-table case marked "depends on Danny's answer".
- **F4:** Claude drafted FR-14 to FR-17, with one assumption about narrowing a parent grant.
- **F4 review (my answers):**
  - When a parent is narrowed, the child is cut to fit rather than ended. At check time a child's access is the overlap with every ancestor's current grant.
  - Delegation depth (FR-18) defaults to 0, shrinks at each step, and has a system maximum of 3 (tentative). I deliberately brought it in from my own Q&A schema work and marked it `[PROPOSAL]`.
- **F5:** Claude drafted FR-19 (withdrawal, with my cascade test case) and FR-20 (closing signals).
- **F5 review (my answer):** the cascade applies to "agent ended" too, because a child outliving its parent would be a delegated grant escaping. A child that needs to finish work makes a new request or gets a service grant.
- **F6:** Claude drafted FR-21 and FR-22, with two assumptions: when a check returns "ask" rather than "refuse", and that a list check which runs out of time returns nothing.
- **F6 review (my answers):**
  - "Refuse" is for what a grant can't do, including a withdrawn grant and anything a rule always forbids, because "ask" would be false hope. "Ask" is for what a new approval could allow.
  - A list check that runs out of time is a refusal of the whole list with its reason, never a fake empty feed. It's all or nothing in v1.
- **F7 and F8:** Claude drafted FR-23 to FR-28, with three assumptions: audit records can't be changed, OpenBao access stops within 10 s, and drift raises an alert only.
- **F7 and F8 review (my answers):**
  - Audit records are append-only, and any attempt to tamper with one is recorded and alerted.
  - The two OpenBao withdrawal targets are measured separately, with the short access lifetime as the backstop.
  - Drift raises an alert only in v1, and access stays until a person removes it, so the gap is visible.
- **Cross-cutting:** Claude drafted NFR-1 to NFR-4 from my decisions. For NFR-5, it listed the 10 fixed rules for me to classify as provable by construction or by tests only.
- **Staleness (my change to Claude's assumption):** contained checks may continue on a stale replica, but not forever. I added a second, longer horizon after which everything is refused, so a withdrawn grant is never honoured indefinitely.
- **NFR-5 (my classification):** 6 rules hold by construction (2 of them also need tests) and 4 by tests only. Self-approval, default deny's error paths, the final-action rule (which I called the weakest) and OpenBao enforcement are tests only. I labelled it as my proposal, and the spike decides which engine can express the rules.
- **Non-goals and scope:** Claude drafted §6 and §7 from my brief's Scope section, updated for Danny's answer and for the decisions made in the PRD.
- **My additions:** "a central permission server" as a non-goal, and record retention beyond one month as out of scope. I also edited FR-2 directly, tagging "when it was issued" as an assumption.
- **Counter-metrics (all mine):** for each metric, I named how it could be gamed and what counterbalances it, for example "drift found: gamed by not looking, so a hand-added OpenBao access must be detected".
- **Metrics, open questions and assumptions:** Claude assembled §8 from my brief targets and counter-metrics, and compiled §9 (15 questions) and §10 (the assumptions and proposals index) from the tags in the PRD.
- **Splitting hole:** I brought back Danny's splitting line before finalizing. Claude showed that FR-13 only fires if the final action reaches Countersign, so ten contained config-file writes could skip it. I chose to combine a checked human-only gate on the merge or sync into production (the main mechanism) with "direct production writes are floor-of-four" (the backstop). I flagged that the gate is my reading of Danny's Forgejo answer, and added the ten-files attack to the decision table.
- **Follow-ups (my answers):** I upgraded NFR-5 row 7 to "construction and tests for the check itself", but stated that it depends on Quornex and Forgejo wiring, which makes it the rule with the most external dependency. I added the gate to scope as a *dependency Quornex builds*, not a Countersign deliverable, and as an assumption in the decision table.
- **FR-29 and NFR-6:** FR-29 is "record every decision". NFR-6 is "time and clock skew": the earlier of the wall clock and an elapsed-time timer, a clock-offset limit, and an early-expiry margin. Both were drafted with Claude's help from the Q&A Module 2 and 5 design discussion; I reviewed and adopted them. I corrected the provenance myself: I first submitted them as my own text, and Claude had logged them as written by me. Claude placed them, derived FR-29's testable consequences, and added the open values to §9. Those values (safety margin, maximum clock offset, sync recency) are mine to choose.

### 2026-11-11: Danny's answer during the PRD

- **Administrators:** they are established in Countersign at installation, and Keycloak's admin role is not trusted. This answers my FR-9 question.
- **Sharing:** holding access never implies sharing it, and a contained marking gives no sharing authority. This answers my question about sharing contained actions.
- **Service grants:** authorised users **or agents** may create them. This contradicts FR-16 as written ("an agent's attempt is refused").
- **Logged:** Claude applied the settled points (glossary, FR-8, FR-9, NFR-4, §9), added a paraphrase to the PRD's `addendum.md`, and flagged the conflicts for me to decide.
- **My decisions:**
  - **Agents creating service grants (FR-16):** only with a person's explicit, scoped, time-limited, audited authorisation. The grant is never wider than what the agent holds, the agent never approves it, its owner is a team or role, and each creation alerts the owner. Claude asked how to keep it from becoming an escape route for delegated access. My answer: it can only convert access the agent already holds, that a person authorised, and that a person approves if it is human-only.
  - **Contained grants can never be shared in v1:** a child makes its own request, which rules approve in seconds. Delegation depth applies to human-only grants only, and is never set by rules. I added a decision-table case.

### 2026-11-11: PRD finalize

- **Finalize start:** I asked Claude to propose changes as selectable items, so I can choose among them.
- **Step 1, the memlog audit:** Claude placed all 71 entries. It proposed three additions and I chose all three: traceability in SM-8, a §0 Document Purpose, and 8 decision-table seeds collected in the PRD addendum.
- **Step 2, input reconciliation:** three subagents compared the brief, the brief's addendum and the Orvex PDF against the PRD.
  - Coverage was strong: all 8 requirements and all 4 ground rules are covered.
  - **Gaps I chose to fix, from Claude's selectable list:**
    - Cap the first staleness horizon at 10 s, because it clashed with the withdrawal target.
    - Restore the brief's conflict rule.
    - Make "one model" testable, and add "open source".
    - Restore two questions for Danny.
    - Add NFR-5 rows for the relaxed approval rules.
    - Add a rule-approval speed target.
    - Add an OpenBao agreement test.
    - Label my own simplifications so they don't read as Danny's.
    - Keep the service-grant hard maximum lifetime as my `[PROPOSAL]`, with a question for Danny.
- **Step 3, rubric review:** the verdict was "good but has gaps", with 1 critical finding, 4 high, 6 medium and 3 low.
- **My choices on the rubric findings:**
  - **Critical:** checks never ask. A check returns "allow" or "refuse", and only a request returns "ask".
  - **Recording:** split by kind, so full records come before the answer and compact records are asynchronous.
  - **"High-risk":** anything not marked contained.
  - **Speed:** p99 of at most 250 µs per check (tentative), measured against one store of 50,000 grants.
  - **Medium findings:**
    - Stagger administrators' expiries and leave a break-glass path to the architecture.
    - If Danny says no to the gate, ship with a stated gap.
    - Approval rules check the contained marking, the normal request checks, and a shorter maximum duration for contained grants.
  - **Small fixes:** the clock for closing signals, glossary terms, bounds on the alert triggers, UJ-5 for feed reads, and the SM-C1 split.
- **Step 4, my values (all tentative):**
  - 2 s for rule approval
  - a 4 h maximum for contained grants
  - more than 20 closing signals per minute counts as a flood
  - an alert 14 days before an administrator lapses
- **NFR-5 rows 11–14 (my classifications):** with them, 11 of the 14 fixed rules hold at least partly by construction, and 3 by tests only.
- **Two hard maximums:** Claude noticed that my "under 24 h" figure would also cap service grants. I chose two maximums: task grants are under 24 h with an 8 h default (tentative), and service grants have their own longer limit, still open.
- **Triage:** I agreed that no open question blocks the architecture. Each one is deferred with an owner (Danny, the spike, or me) and a condition for when to revisit it.
- **Step 5, polish:** same style as the brief (British spelling, no serial comma). Claude first made the mechanical fixes (the date, scaffolding text, FR-16 duplicates, and a note on stable FR IDs), then ran the structure reviews on the PRD and its addendum.
- **Reversed "contained grants can never be shared":** Claude had offered it as one of two options ("who authorises sharing for contained grants, or can they simply never be shared?"). I chose it, and it went into the PRD. On re-reading Danny, I reversed it. He said a parent passes on only what it is explicitly authorised to share, and that "contained" only means automatic approval. He never said contained grants can't be shared. **I follow Danny:** sharing works the same way for every grant, a person authorises it explicitly, and contained grants get no exception. Claude updated the glossary, F3, FR-8, FR-14, FR-18, NFR-5 row 12, §10 and the decision-table seed.
- **Follow-up (my answers):** sharing on a contained grant is a standing sharing limit, set in advance by a person with authority over the resource and separate from the marking. It's still an `[ASSUMPTION]`, and I added it as question 6c for Danny. I deleted NFR-5 row 12 and folded it into row 11 ("sharing needs explicit authorisation, for every grant"), leaving 13 fixed rules.
- **Structure polish (my choices from Claude's grouped list of 35 findings):**
  - **Terms:** "hand down" only, "person with authority" only, "withdraw" only, and "child agent" / "child grant".
  - **Glossary:** missing terms added, definitions only, grouped by topic, plus a "related artefacts" list in §0.
  - **Repetition:** shorter feature intros, request rules moved to FR-1, one source for the gate, small cuts.
  - **Other:** SM-9b moved to Primary, the human-only marking clarified, and the addendum fixes.
  - **I kept NFR-5's "Why" column,** although the review suggested moving it to the architecture.
  - **How it was done:** a subagent applied the edits. Claude backed up the files first, to check the diff.
  - **Claude's check of the subagent's edits:**
    - All 58 IDs survived.
    - No requirement, number or target changed; the only new value is the withdrawal-target glossary entry.
    - Rules were moved into FRs, not cut.
- **Prose polish (my choices):**
  - The reviews found 34 issues in the PRD and 11 in the addendum. I accepted all of them.
  - That includes fixing three self-contradictions (FR-22, NFR-2, FR-8), the FR-1 "ask" wording, two misreadings, and two term slips.
  - I chose British "Realises" throughout.
  - A subagent applied the fixes, and Claude checks the diff.
  - **Claude's verification:** all IDs present, tag counts unchanged, and every number the same value.
  - **The two findings the subagent skipped as content (my answers):** the FR-9 window is 14 days (the alert window), and NFR-5 rule 12 is "construction and tests".
- **Decision-table wording:** the prose review had changed "25 checks" to "25 cases" to fit my glossary. I restored Orvex's exact wording, "25 checks, 5 of them attacks", with a glossary note that a decision-table check is one test row.
- **The PRD is final (2026-11-11):** `status: final`. All open questions are deferred with owners, and none blocks the architecture.

### 2026-11-11: architecture (BMAD step 4, `bmad-architecture`)

- **Setup:** the Skill tool couldn't load the BMAD skills, so Claude followed `bmad-architecture`'s `SKILL.md` directly. It created the workspace at `_bmad-output/planning-artifacts/architecture/architecture-countersign-2026-11-11/`, holding `ARCHITECTURE-SPINE.md` (from the template) and `.memlog.md`.
- **Inputs:** the final PRD and its addendum, the final brief and its addendum, and the Orvex PDF. The Q&A answers are not used.
- **Mode:** coaching. Deliverables in this run: the spine, an architecture document with diagrams, the grant schema and fixed rules in the chosen engine, the decision table, and the two-engine spike.
- **Engine research:** a subagent checked all six engines (in `research-engines.md`, unverified). Cerbos can't be embedded, Casbin is about 100× too slow at scale, and cedar-go and OPA look strongest.
- **System shape (my design):**
  - One store component per installation is the only writer.
  - Each product embeds the library with a read-only replica, so checks are local with no network call.
  - The store pushes changes and a heartbeat.
  - Full records go to a local append-only log before the answer.
  - My reason: "the hot path must be local and cheap, and anything that changes state needs one consistent history".
- **History:** I chose an append-only event log (event sourcing), Claude's recommended option, over a database with a change feed or periodic snapshots. The state version is the log position.
- **Store outage (my choice):** I accepted it and stated it. Writes fail closed, and checks degrade per NFR-2. I didn't lengthen the second horizon, because that would honour withdrawn grants for longer. High availability is an open operations question.
- **Check time vs write time (my split):** a check recomputes everything that depends on current state (expiry, ancestor overlap, the withdrawn set, the action class, staleness). Who may write is settled once, at the store. My reason: "a check only reads, so it can only go wrong by allowing too much".
- **Spike shortlist (my choice): cedar-go vs my own Go.** I judged the hardest requirement to be provable fixed rules (NFR-5), because the brief judges security hardest. I left out OPA (explanations are only traces) and OpenFGA (it embeds a server and explains poorly). I noted that Cerbos, SpiceDB and Casbin were ruled out from documentation only, not from my own measurements.
- **Making the spike fair (Claude raised three risks; my answers):**
  - **Cedar's analysis:** I'll actually run SymCC with Rust and cvc5. If the setup fails, I'll write "Cedar's analysis was not measured". I won't claim "provable" without having run it.
  - **Thresholds fixed before any measurement:** 250 µs p99 at 50,000 grants with a reload running, 25 out of 25 correct with 0 wrong allows, and the same machine and load for every engine.
  - **Shortlist revised:** two real open-source candidates, cedar-go vs OPA (OPA for its maturity and its partial evaluation for FR-22). My own Go becomes a *reference*, not a candidate, clearly labelled as written by me. The explanation chain is built in my own layer, not by modifying OPA.
- **My decision rule for the spike:**
  - Both engines must pass the hard thresholds. Then priority goes to provable rules, then explanations, then the list check, then speed. Speed decides earlier only if one engine is more than twice as slow.
  - If both fail, the less-failing engine is compared with my own Go.
  - Otherwise own Go stays a reference. If it wins by a wide margin, I'll ask Danny whether a custom engine is worth it.
  - Claude wrote this into `spike-preregistration.md`, dated before any measurement.
- **The store and its replicas (my choices):**
  - **Storage:** an append-only table in the installation's existing PostgreSQL, so a write, its audit record and its log position commit together, and the app's database role can't update or delete rows. I'll check with Danny that every installation has PostgreSQL.
  - **Delivery:** a gRPC push stream that resumes from the last log position. The heartbeat is the log position plus the store's time every second, which gives freshness and clock offset from one signal.
  - **Startup:** from a snapshot at position N, then the events after it. Snapshots are never a second source of truth.
  - **Log order:** Claude pointed out that a plain PostgreSQL sequence can leave gaps and commit out of order, so a replica could miss a withdrawal. I chose serialised appends (no gaps, commit order equals position order), with append throughput measured in the spike.
- **OpenBao enforcement:**
  - **My design:** option (b). The agent logs in with its own Kubernetes identity, the store attaches one policy per grant, and Countersign never touches a secret or a credential.
  - **Claude's challenge on crash order:** my order (policy first, grant after) left an orphan policy, which is a wrong allow. **I corrected it:** the log comes first, and the store keeps OpenBao in line with the log.
  - **Claude's challenge on "the task can't be enforced":** that would block every grant. **I narrowed FR-27** to restrictions OpenBao would have to check on each request. The task, time limits, withdrawal and the chain are enforced by removal.
  - **Web check (OpenBao v2.7.2 docs):** identity policies are evaluated per request, but token policies are fixed. I added the rule "grants attach only to the agent's identity, never to `token_policies`", and the policy `expiration` field as a backstop that works even if the store is down.
- **Decision table:** I picked the 5 attacks myself (A, C, D, E, F), each with a reason tied to the brief's judging words. Claude drafted the other 20 rows from my PRD in `decision-table.md`, for me to review.
- **Write-time rules go through the engine:** the table's "where decided" column showed that all 5 attacks were decided in the store's Go code, so the spike would never test the engines on security. I chose Claude's recommended option: the store asks the same engine, with the fixed rules as engine policies, and the Go types stay as a second layer.
- **My review of the decision table:**
  - I turned four attack-like rows (11, 19, 20, 21) into normal cases, so only rows 1–5 are attacks.
  - Rows 22–24 are "n/a, same result" for the engines.
  - I replaced Sam's case with the positive FR-16 case. Sam's case is tracked apart from the pass/fail count.
  - I reworded PRD SM-8, because 25 rows can't validate every FR.
- **Row 23 (list check):** Claude noticed that my "n/a" contradicted my own decision rule, where FR-22 is the third tie-break and part of why OPA was shortlisted. I made it engine-specific, with real results per engine (OPA partial evaluation, cedar-go batch or per-item, own Go prefix filter), and the method stated beside each result.
- **Who writes the code (my choice):** Claude writes it. I review each part before the next one starts, read every rule against the PRD, and test it myself before the spike uses it, because I have to explain each fixed rule without the code in front of me.
- **The grant schema (my own design, reworked from scratch):** 15 fields, each with a type and a validation rule.
  - **Claude's four challenges, and my fixes:**
    - `task` is required only for task grants.
    - A child grant's `approver` is empty, so it never claims Erin approved something she never saw.
    - Sharing fields exist only on human-only grants, and I dropped `may_share` because `depth` 0 already means no sharing.
    - I added `created_by`, and there is no `status` field, so grants are immutable and state comes from the log.
  - Claude drafted the other objects (request, approval, marking, authority, sharing limit, service authorisation, administrator, event types) in `grant-schema.md`, for my review.
  - **My review of Claude's draft objects:**
    - I accepted narrowing as an event, with the spike measuring the extra lookup.
    - I made five fixes: the authority hard maximum rather than a fixed number, the sharing-limit depth rule, a minimum of two administrators checked on removal, the agent's service-grant limit, and new `marking_removed` and `drift_detected` events.

- **Spike part 1 (code by Claude, for my review):**
  - The `spike/` Go module has the typed model and the own-Go reference: the check, plus the write-time rules.
  - There is one test per NFR-5 fixed rule.
  - Claude ran a mutation check: it broke the self-approval, ancestor-overlap and contained-sharing rules one at a time, and each made its test fail. That shows the tests really test something.
  - It listed five assumptions for me to review.
- **My review of part 1, by reading the code:** all the tests were green, but reading the code found **5 bugs**:
  - a person could create a service grant with no authority
  - the approver couldn't set a depth, so FR-8 and row 17 couldn't work
  - "agent ended" killed future grants too
  - a caller could overwrite another's pending request
  - the hand-down switch had no default case

  I also found 4 smaller issues. **I asked for a failing test before each fix.** Claude showed all 9 tests failing for the right reason, then fixed the code, and 22 tests now pass. I also had NFR-5 rule 3 reclassified to "tests", because the grant type has no validated constructor.
- **The contained cap on service grants:** I asked whether the 4 h contained cap should apply to service grants. Claude recommended no, but flagged that approval by rules would then mean long-lived access nobody reviewed. **I chose: every service grant needs a person's approval**, even for contained actions. This was done test-first, and PRD FR-16 and decision-table row 25 are updated.
- **My second review round of part 1 (`classes.go`, `state.go`), 7 issues:**
  - anyone with authority could clear "production" and empty the floor of four
  - a narrower classification beat a wider production flag
  - approval authority counted as authority over a resource (I decided: administrators only in v1, **overruling Claude's assumption**)
  - "ask" could list nobody
  - removal counted expired administrators
  - the refusal reason depended on storage order
  - there was a data race

  All were done test-first. One test passed for the wrong reason and had to be tightened, and the race detector caught the data race. 30 tests now pass with `-race`.

- **Engine interface (I chose Claude's recommended option):** I agreed to refactor part 1 so that the Countersign layer gathers the facts and a pluggable engine decides. The Go reference is one engine, Cedar and OPA are the others, and the same tests run on all three. Without this, Cedar would only rubber-stamp decisions Go had already made.

- **Spike part 2 (Cedar, code by Claude, for my review):**
  - Behind the new engine interface, the rules were moved out of the layer into a Go reference engine, and cedar-go v1.9.2 was added with `policies.cedar` (13 policies, every fixed rule an explicit `forbid`).
  - The same 30 tests pass on both engines.
  - Claude mutation-tested the *Cedar policies*, which caught a real gap: the "agents never approve" `forbid` could be removed without any test failing, because the layer's facts also blocked agents.
  - Claude added 6 engine contract tests with crafted facts. 36 tests now pass on both engines with `-race`.
  - A known issue for part 5: Cedar schemas have no union types, so `Grant.principal` (a person or an agent) needs remodelling before SymCC can run.

- **My review of part 2 (17 points), worked on test-first:**
  - Done:
    - **Floor of four:** lowering the floor needs a second administrator, confirmed through their own call; unknown marking classes are refused; markings can be removed.
    - **Engines must agree:** a shared input guard, a differential test across engines, Cedar's maximum depth enforced, and the engine deciding authority from scopes.
    - **Contract tests:** each one names the rule that decided, with `denyall` and `allowall` engines proving the tests can fail.
    - **D11–D13:** a service grant is never handed down (my choice), closing signals are tracked by log position, pending requests are capped, and end dates are validated.

    That brings it to 73 tests on each engine.
  - **Then I stopped the hardening** to move on to OPA and the measurements. D14 and E15–E17 are listed as known limits in `spike/KNOWN-LIMITS.md` for the README.

- **Spike part 3 (OPA):** Claude wrote `policies.rego` to mirror the Cedar rules, with the same rule IDs. All 73 tests pass on all three engines, and Rego mutations are caught.
- **Spike part 4 (the harness):** at my request, the 25 rows and the 50,000-grant measurements, against my pre-registration.
  - **Claude caught a flaw in its own harness:** events were paced by the clock, so OPA faced 5× Cedar's write load, against my "same load" condition. Two identical runs also gave Cedar a p99 of 294 µs and then 579 µs.
  - It fixed this to pace events by checks done, with 3 runs per engine, and recorded it as a change made after measurement.
  - **Results:** cedar-go p99 466 µs, OPA 1.03 ms, own-Go reference 46 µs. All 23 runnable rows pass on every engine, with 0 wrong allows. **Both candidates fail my 250 µs threshold**, and "25/25" can't be met because rows 22 and 24 can't run in the spike. The decision under my pre-registered rules is mine to make.

- **My decision on the first verdict:** both candidates failed p99 on the full path. I judged the result **inconclusive** rather than applying the "both fail" rule, and had the same prefilter added for every engine. It was recorded openly in the pre-registration as a change made after measurement, both paths are reported, and a test shows the prefilter changes no decision. On the prefiltered path, cedar-go (160 µs) and OPA (158 µs) both passed.
- **Then I kept "25 of 25" as written, and replaced rows 22 and 24** (which no engine could run) with two rows an engine decides. The originals, and where they must be tested, are kept. I also had **OPA's list check moved to partial evaluation**, as I had pre-registered, after Claude flagged that the harness had deviated from it.
- **Before re-running, I added three rules to the pre-registration:** a mandatory worst case (one agent with 5,000 grants, a maximum-depth chain, half withdrawn), a 20% stability rule on the 3 runs, and "passes only with the prefilter" stated in the results. Claude added two implementation details, labelled as its own: 30 checks for the worst-case full path, and at most 3 attempts.

- **Run 4:**
  - Every engine passed 25 of 25 rows. OPA's typical p99 passed and was stable (144 µs).
  - **cedar-go's typical p99 was unstable** (44, 76 and 156 µs), so my own rule allowed no conclusion.
  - The worst case failed for every engine *including the reference*. Under my rule, that's a layer finding (no index), not an engine failure.
  - Claude also reported three wording flaws in its own results output.
- **Change 7 (my decisions):**
  - I chose to reduce noise and re-measure, and asked to see the full text before anything was recorded.
  - Claude checked the machine first: it's a VM with host CPU steal, and with 2 pinned CPUs the garbage collector would compete with the checker. **I chose 3 pinned CPUs.**
  - **I decided in advance** that if cedar-go is still unstable, it's not shown to pass, OPA is the choice if it passes, and there is no further re-measurement.
  - Every earlier run is kept, dated, in `spike-results-history.md`.

- **Run 5 (pinned):** stable measurements. cedar-go 40 µs and OPA 88 µs typical p99, and both passed every hard threshold.
- **My engine decision: cedar-go.** I applied my pre-registered rule: OPA was 2.2× slower, which crosses my "more than twice" line, so speed decided. Claude didn't make the choice; it laid out how the rule applied. The decision states:
  - the thin margin, and that earlier unpinned runs differed
  - that cedar-go's full path (249 µs) depends on the prefilter
  - that the worst case is a layer finding
  - that OPA is the runner-up
  - that Cedar's analysis hasn't run yet, so I don't claim "provable" until it has

- **Spike part 5 (Cedar's analysis), as I pre-registered:**
  - **Setup:** Claude installed cvc5 1.3.1 and wrote the Cedar schema (an Orvex deliverable) and a small Rust program using SymCC.
  - **The problem Claude had predicted (union types) didn't happen.** The real problem was that Cedar's strict validator couldn't follow the `has parent` guards. That forced a rewrite of three clauses in `policies.cedar`, which was then re-verified by all the tests on all three engines and by a mutation.
  - **Result:** **14 of 14 properties proved** for every possible request and entity store. Two deliberately false claims were correctly refused, and breaking the self-approval rule produced a counterexample (an administrator approving their own request).
  - **"Provable" is now measured, not claimed.** The proof limits (the layer's facts, the schema's coverage, the rules outside Cedar) are in `spike/KNOWN-LIMITS.md`.

- **Re-measure after the SymCC rewrite (my request, change 8):** I had the rule fixed in writing before the run, with the same thresholds and the same 2× factor. Run 6 gave cedar-go 41 µs and OPA 91 µs, both stable, a ratio of 2.2×. **The decision is unchanged.** The plainly stated difference: cedar-go's full path is now 259 µs, so it too passes only with the prefilter.

- **The architecture spine:**
  - Claude distilled `ARCHITECTURE-SPINE.md` from the 40 decisions in the architecture memlog, all of them already made by me or by the spike: AD-1 to AD-16, the conventions, the stack (gRPC-Go and pgx versions checked through the Go module proxy), two diagrams and the deferred list.
  - BMAD's spine linter reports 0 findings. It's for my review before finalize.

- **My spine review (2026-11-17):** I asked for ten changes before the reviewers run, and Claude applied them:
  - AD-1, 2, 3, 13 and 14 are now marked "decided, not measured".
  - AD-6 and AD-15 now say the engine decides each item of a list by one check per item, because cedar-go has no list method.
  - AD-5 now says where rule 7 lives (the production gate, which depends on Quornex) and where rule 10 lives (the store's OpenBao reconciler).
  - It says "14 properties", not "14 rules". That applies to the spine, `KNOWN-LIMITS.md`, this log and the SymCC program. The SymCC program was re-run, and its result is unchanged: 14 of 14 proved, 2 of 2 controls.
  - AD-4 now says the clock rule has never run, because the spike has no replica.
  - AD-13 now says listing policies by the `countersign-` prefix comes from the OpenBao docs and is untested.
  - Deferred now lists "each installation has its own PostgreSQL" as an assumption to check with Danny.
  - Two ADs are new. AD-17 makes the layer the trust boundary (E15), tested apart from the Cedar proofs. AD-18 covers the boundary with Quornex and Houston: the approval screens, identity and the production gate.
  - AD-9 now has the memory figures from run 6, plus a statement that the scope index's memory isn't measured.
  - The spine linter still reports 0 findings. At my request, the reviewers wait for my approval.

- **My second spine review (2026-11-17), before the reviewers:**
  - **AD-9 memory:**
    - I asked for 620 bytes per grant. Claude checked the harness, which reports heap in MiB (2^20 bytes), so 31 MiB / 50,000 is about 650 bytes. 620 would assume decimal MB. Claude also found that the 31 MB comes from run 2b, not run 6, because runs 4–6 didn't record heap. I chose the wording "31 MiB, about 650 bytes, run 2b".
    - The full-path memory per check is added: 25,642 bytes and 356 allocations. I had written 25,643; run 6 says 25,642.
  - **AD-18:** the products' closing signals are now marked as an assumption to check with Danny.
  - **AD-17:** I chose to enforce "a new fact needs a layer test" with a CI reflection test over the engine input structs. A field without a registered layer test breaks the build.
  - **AD-4 and AD-12:** when the wall clock and the elapsed timer disagree, the later one decides. The replica computes `now` once per check and passes it in.
  - **AD-8:**
    - Rule 1 is Cedar's default deny, not a `forbid`.
    - Rule 6 is proved for person requesters only.
  - The "must make" wording for the worst case was already in AD-9, so it's unchanged. The linter reports 0 findings. The reviewers are still on hold.

- **The reviewer gate (2026-11-17), at my request after my approval:**
  - First, I had AD-4 reworded: `now` is the later of the two clocks, so a grant ends as soon as either clock reaches its end, and a wall clock that moves backwards can never lengthen access.
  - Five reviewers ran in fresh contexts: the version check, the PRD check, the evidence check, BMAD's checklist walker, and the "two teams" attack. They raised 116 findings, 8 of them critical.
  - **The two-teams reviewer's verdict:** two teams building from the spine would **not** end up with the same behaviour, and five gaps allow a wrong allow.
  - Claude checked the key claims against the files and the web before fixing anything.
  - **Fixed:** about 35 corrections of fact or evidence. Examples:
    - AD-8 said every fixed rule is a `forbid`, but rules 1, 8, 9, 11 and 13 are `permit` conditions.
    - AD-9 said no engine met 250 µs without the prefilter, but the reference engine did.
    - The Go version is now 1.27.4.
    - Rule 3 had no AD.
  - **20 decisions (D1–D20) are left for me.** Nothing has been applied for them yet. They're listed in `reviews/triage.md`.

- **My decisions on the gate's critical findings (2026-11-17):**
  - I chose Claude's recommended option for all four, from selectable options:
    - **D1:** a high-risk check makes one synchronous call to the store to record its decision. It's the only network call in the check path. 250 µs applies to contained checks, and a store that's down refuses high-risk checks.
    - **D3:** one `projection/` package is shared by the store and every replica (new AD-19). A `narrowed` event carries the full effective scope and end.
    - **D4:** a replica is fresh only once it has applied the heartbeat's position. With no heartbeat since start-up, it refuses everything.
    - **D5:** people are `person:<Keycloak sub>`, agents are `agent:<namespace>.<serviceaccount>`, built by one function in `model/`. The store accepts only mTLS from the products.
  - I haven't given reasons beyond choosing those options.

- **My decisions, second round (2026-11-17):** again Claude's recommended option, from selectable options, for:
  - **D6:** OpenBao policies follow the projection: the effective end minus the margin, removed when an ancestor ends, a 10 s lag target, and one `ToOpenBao` mapping.
  - **D2:** a replica whose policy hash differs from the store's refuses every check (new AD-20).
  - **D14:** a PostgreSQL advisory lock enforces one active store.
  - **D7:** the wire contract: `proto/`, a closed event list, exclusive resume, halting on unknown events, idempotency keys (new AD-21).

- **My decisions, third round (2026-11-17):** Claude's recommended option, from selectable options, for:
  - **D8:** limits are set at the store and carried by the log (new AD-22).
  - **D9:** markings combine with "strictest wins".
  - **D10:** a list filters out refused items, and is refused whole only on errors.
  - **D11:** read-your-writes, by position and wait.

- **My decisions, fourth round (2026-11-17):** Claude's recommended option, from selectable options, for:
  - **D12:** a fixed `Decision` shape and replay by log position (new AD-23).
  - **D13:** namespaced tasks and resources, and a `task_opened` event.
  - **D15:** the writes decided only in Go stay in Go, stated as not proved, with their FR rules written into AD-5.
  - **D16:** operations go to Deferred, each with an owner.

- **My decisions, fifth round (2026-11-17):**
  - **D17:** widening the proofs is v1 CI work, stated in AD-8.
  - **D18:** rules 8, 9, 11 and 13 stay as permit conditions, with a CI check that each action has exactly one `permit`.
  - **D19:** I sent all four questions to the README's open questions:
    - FR-17 vs FR-16
    - NFR-4 as a product obligation
    - Sam's case
    - unclassified counting as production
  - **D20:** Claude applied the small items as one batch, checking each against the PRD first.
  - **All 20 gate decisions are made.** The spine has 23 ADs, and the linter reports 0 findings.

- **The second two-teams reviewer pass (2026-11-17, at my request):**
  - **The reviewer's verdict:** two teams would still behave differently. It raised 3 critical findings.
  - **Several findings came from how Claude had worded my earlier decisions.** For example, D6's "effective scope" didn't include narrowing by an ancestor (B2).
  - Claude also found that my D18 check ("one permit per action") failed on the spike's own `create_service_grant`. It's corrected to one per action and principal type (B25).
  - **From selectable options, I chose to fix B1, B2, B4–B9 and B25.**
  - **I left open B3, a critical finding:** a restored log isn't detected. I also left open B10, B11, and the medium and low findings. They're recorded in Deferred and triage.md for the README.
  - I chose to stop the gate there, with no third pass.

- **B3 and B10, a request I withdrew (2026-11-17):**
  - I asked for B3 and B10 to be settled in the spine. Claude applied it. Then I withdrew the request and had every change reverted.
  - **B3 stays open,** first in the README's open questions, as the one critical risk left open. **B10 stays open too.**

- **The AD-14 contradiction (2026-11-17), at my request, and nothing else:**
  - AD-14 said approvals and withdrawals are recorded by a synchronous call from the replica, but they're store writes.
  - Their record is now the event in the write's own transaction.
  - B10 stays open.

- **The architecture document (2026-11-17):**
  - Claude drafted `architecture.md`, the human-facing document for the hand-in: 7 Mermaid diagrams and about 4,300 words. The spine stays binding.
  - Its contents:
    - the context
    - the decisions with how much evidence stands behind each
    - the packages
    - check, grant and withdrawal flows
    - staleness and failure modes
    - the engine choice with run 6's numbers and what SymCC does and doesn't prove
    - recording and replay
    - open risks (B3 first)
    - where the product differs from the spike
  - **Claude checked every fixed rule's evidence against the spike's test names.**
  - **It found two places where the spine moved away from the spike's code:** the spike allows reopening a closed task, and treats "task closed" by log position. The document says so.
  - **The diagrams weren't validated.** Running the Mermaid CLI through `npx` was blocked by the permission check.
  - For my review.

- **My review of the architecture document (2026-11-26), seven fixes:**
  1. B3 is stated plainly at the top as the open critical risk.
  2. Section 1 now says agents never approve a request, and that hand-down is how Danny's "a parent can approve for its child" is modelled.
  3. **Sam's case is not settled by the schema:** Sam and the agent are different principals, and the proof doesn't cover agent requesters. Claude's draft had wrongly said the schema settled it.
  4. The thin-margin note now includes the earlier unpinned ratios (about 1.0 in run 2b, 1.9 in run 4) and the VM's steal time. Claude checked both ratios and the reference engine's run 6 numbers against spike-results-history.md.
  5. The examples follow UJ-1: a contained push on `repo/api/...`, handed down within a standing sharing limit.
  6. AD-8's row says "proved once by hand, CI to build".
  7. A one-page summary box is at the top.

- **The spine closed (2026-11-26):**
  - At my request, Claude brought the spine into line with architecture.md:
    - The Deferred line on seed 2 now reads "Sam's case is an open question for Danny (PRD §9, question 4)".
    - AD-6's thin-margin note now has the earlier ratios (about 1.0 in run 2b, 1.9 in run 4) and the VM steal time.
    - The resource convention now says `repo` is a registered system segment, like `forgejo`, `openbao` and `houston`.
  - The document is dated 2026-11-26.
  - The spine is `status: final`, and the linter reports 0 findings.

- **bmad-review (2026-11-26):** from selectable options, I chose to review architecture.md and the PRD with the adversarial lens only. Two reviewers are running, each in a fresh context, one per document.

- **bmad-review results and my triage (2026-11-26):**
  - The two adversarial reviewers each found 23 findings.
  - Claude checked P12 (the NFR-5 count) and P13 (the "(to come)" link), and both were right. Claude also flagged that P3's fail-closed default for Sam's case would override "as Danny asked".
  - **From selectable options, I chose:**
    - the 15 wording fixes in architecture.md
    - resolving A4, A5 (B3's owner and an interim restore rule), A8 and A16
    - **a tentative 5-minute cap on the second horizon.** This went into the spine's AD-22 too.
    - fixing only P12 and P13 in the PRD, as post-final changes
    - sending the other PRD findings to the README's open questions
  - A9, A14 and A22 stay open.

- **The README (2026-11-26):**
  - Claude drafted `README.md` at the repo root. It contains:
    - B3 stated at the top as the open critical risk
    - a map of the hand-in
    - the design in brief, with one diagram
    - what the spike shows and what it doesn't
    - how to run the spike
    - the open questions: open risks (B3 first), questions for Danny, PRD changes to raise, and what's not decided by design
  - **Claude re-ran the suite first:** 75 of 75 tests passed on each engine, with `-race`.
  - **It doesn't link the Orvex PDF,** because whether it may be published is still to be decided.
  - For my review.

- **B3 settled (2026-11-26), decided, not built, not measured:**
  - I wrote the full fix and had Claude put it into the spine:
    - AD-2: a history ID, a ULID or UUID minted again on every restore, and a hash chain
    - AD-3: resume, the heartbeat and the snapshot carry the history ID and the head hash
    - AD-21: a replica halts until a snapshot is reloaded; the layer test; the known limit of a database recreated by hand
    - the IDs convention
    - D16's restore procedure
  - **Claude checked AD-1 and AD-14.** The compare-and-append now checks the head hash too. AD-14 had no contradiction, and the record's history ID went into AD-23.
  - architecture.md and the README now list B3 under "decided, not built". The README keeps the interim restore rule until it's built.
  - B10 stays open.
  - My earlier withdrawn attempt stays in the logs.
  - **Claude added one sentence of its own to the known limit:** without a new history ID, the hash chain alone still catches a fork wherever the restored events differ from what the replica applied. It's flagged for my review.

- **B3's known limit (2026-11-26):**
  - I kept Claude's sentence in the spine and had the same qualification added to architecture.md and the README. Without the restore procedure, the history ID doesn't change, but the hash chain still catches the fork when a replica resumes. The protection fails only if the restored events are identical to what the replica applied.
  - I kept the new IDs row and left the Identities row as it is.

- **The overrule list (2026-11-26):**
  - Claude identified two candidate overrules from this session: leaving B3 open against its recommendation, and the 5-minute cap.
  - I added both, then dropped the B3 one, so the list has 5 entries.
  - The cap's entry quotes my reason in full.

## Times I overruled the AI

<!-- The brief asks for 3–5. For each one: what the AI proposed, what I did instead, and why. -->

1. **Kept the brief's fixed rule (2026-10-31).** Claude proposed simplifying the fixed rule to "agents never approve anything", on the grounds that it is smaller than "agents never approve production". I kept the rule exactly as the brief states it: *"No agent approves a production change, and nobody approves their own request, however it is worded."* The fixed rules are Orvex's requirement, not mine to rewrite.
   - **Update (2026-11-04):** Danny's answer backs rejecting Claude's "agents never approve anything", and goes further: even the brief's blanket rule is too restrictive. Supervising agents may approve within delegated limits. The fixed rule now has to be reworded from his answer, so this overrule's reason needs revisiting.
2. **Changed the Executive Summary's emphasis (2026-11-04).** Claude's draft of the product brief's Executive Summary led with the rule that production, real secrets, money and deletion always wait for a person. I rejected that order. In my words: *"I wouldn't lead with the four human-only classes. I'd lead with the promise (agents start with nothing and lose access when the task ends), then the problem (two rule sets, nobody can say who can do what and why), then the safeguards, with the human-only classes as one of them."* I wrote the opening line myself, and Claude rewrote the rest around it.

3. **Reversed "contained grants can never be shared" (2026-11-11).** When we were deciding who authorises sharing for contained grants, Claude offered "can contained grants simply never be shared?" as one of two options. I chose it, and it went into the PRD, including FR-14, FR-18 and NFR-5. On re-reading Danny, I reversed it. In my words: *"I follow Danny. Sharing works the same way for every grant, and a person authorises it explicitly. Contained grants get no special exception."* Danny said a parent passes on only what it is explicitly authorised to share, and that "contained" only means automatic approval. He never said contained grants can't be shared.

4. **"Authority over a resource" means administrators only (2026-11-17).** In spike part 1, Claude's code assumed that anyone holding approval authority covering a resource also had authority over it, and listed this as an assumption for my review. Reading `classes.go`, I rejected it. In my words: *"I decided it means administrators only in v1. As coded, any approver can mark actions contained, classify resources, give authority and authorise service grants."* It was fixed test-first: the test failed on Claude's version and passes on mine.

5. **A tentative 5-minute cap on the second staleness horizon (2026-11-26).** BMAD recommended leaving the value open, so as not to invent a number nobody had measured. I chose a tentative cap of 5 minutes instead.
   - **My reason:** *"Leaving it open meant that a withdrawn contained grant could keep being honoured, for no stated time, on a cut-off replica. That is the kind of wrong allow the product is meant to prevent. A provisional number, marked tentative like my other targets, makes the window bounded and testable, and it is wrong in a visible way if it is too long. I accept that it is not measured and that the replica doesn't exist yet, so the value has to be measured and confirmed in v1."*

## What I checked by hand

<!-- Claims, numbers and benchmark results I verified myself rather than trusting the AI. -->
