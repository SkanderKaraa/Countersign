---
title: "Addendum: Countersign product brief"
created: 2026-11-04
updated: 2026-11-06
---

# Addendum

This is material that belongs in the PRD or the architecture rather than the brief.

## Clarifications from Danny (email, 2026-11-04)

Skander emailed Danny four questions. His answers are paraphrased from the reply, not quoted. Ask Danny before publishing the email itself.

### 1. Approval screens

Quornex and Houston provide the approval screens. Countersign has no screens of its own.

### 2. Agents can approve

- Approval does not always have to come from a human. Both products are designed for agent hierarchies.
- A supervising agent may approve another agent's request, within explicitly delegated limits.
- Fixed constraints:
  - Nobody can approve their own request.
  - Actions marked as human-only must still go to an authorised person.
- Danny considers the original brief's blanket statement, that agents never approve, too restrictive.
- **Still open:**
  - Which actions are human-only? Are production changes, real secrets, money and deletion always human-only?
  - Who marks an action as human-only?

### 3. Where permissions live

- Each Quornex installation has its own Forgejo instance. It holds the customer's infrastructure configuration: applications, databases, environments.
  - Changes made through the console, the CLI or agents become Git commits.
  - The result is one reviewed history of what should be running.
- Permissions live separately, in each installation's permission store, where every change is audited:
  - grants
  - approval authority
  - delegation limits
- An agent can therefore receive access in seconds without a pull request. Granting that access does not bypass the review required for the infrastructure change itself.
- Danny considers that the original brief's wording, that grants are Git files, needs correcting.

### 4. Not answered

- **Identity:** where identity comes from, and whether there is an existing identity provider or directory to rely on.
- **Translation:** whether "if a grant can't be translated without widening access, don't translate it; report it as untranslatable" is the right default.

## Follow-up to Danny: what "supervising" means, and splitting (sent 2026-11-04)

- **The scenario:** `orchestrator` spawns `release-prep`, which spawns `helper`. Can `release-prep` or `orchestrator` approve `helper`'s request?
- **The splitting attack:** `helper` turns one production change into ten small "contained" requests, each approved by its own supervisor. Together they add up to a change no person saw.
- **Skander's working position until Danny answers (option a, the strict one):**
  - A request made on an agent's behalf counts as that agent's own, so no agent in the spawn chain can approve it.
  - A task's requests count together against the task's budget.
  - Approval comes from a separate oversight role or a person.
- **The fallback (option b), if Danny means the agent above in the spawn chain:** allow an ancestor to approve only if it didn't create the task and doesn't benefit from the request. That rule would then have to be proved.

## Danny's answer to both emails (2026-11-06)

This is paraphrased, not quoted. Danny asked to keep things much simpler.

### Identity

- Keycloak says who a **person** is. Kubernetes says which **agent or service** is calling.
- Countersign decides what they may do, including who can approve requests and who can give access to others. Countersign owns the policy engine.

### Parent agents can approve

- A parent agent can approve access for its child, if it has permission to do so.
- It can only pass on access it is allowed to share.
- There is no need to rule out everyone in the same agent chain, or to work out who benefits.
- Still fixed:
  - Nobody approves their own request.
  - Anything that requires a human goes to a human.
- **This replaces Skander's working position (option a, strict) from the follow-up.**

### Access that outlives its parent

- Normally, access passed to a child ends when the parent's access ends.
- But someone with the right permission can give a service access that lasts on its own. Example: an employee leaving shouldn't break a service they created.

### Splitting

- Breaking a request into smaller pieces must not get around the approval needed for the final action.
- Danny wouldn't build a complicated task-budget system for the challenge.

### Translation and enforcement

- No translation into Keycloak or Kubernetes RBAC: both are used for identity only.
- For OpenBao, explain how Countersign's decision would be enforced.
- If a restriction can't be enforced, the access is not allowed.

## Mechanism notes moved from the brief's Success Criteria

- **Access ends with the task:** the 10-second target runs from when Countersign learns that the task closed. The delay between the tracker and the event is outside Countersign's control.
- **Withdrawal cascades:** the clock starts at the withdrawal, not when a replica hears of it. A replica that has lost contact refuses high-risk requests once its staleness horizon passes, so a withdrawal can't fail silently.

## Test cases proposed during the brief (for the decision table)

- **Withdrawal cascade:** withdraw a parent grant that has three children and one grandchild, then check that all five grants are refused within the withdrawal target (10 seconds, tentative).
- **Splitting attack:** one production change split into many small "contained" requests from the same task. They must count together against the task's budget and go to a person when the budget is exceeded.
