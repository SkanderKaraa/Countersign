---
title: "Product Brief: Countersign"
status: final
created: 2026-10-30
updated: 2026-11-06
---

# Product Brief: Countersign

> **Status note:** this brief is final. The `[OPEN]` items and the `[PROPOSAL]` on service grants are waiting for Danny's answer, and the PRD will pick them up.

## Executive Summary

Countersign is the permissions engine that Quornex and Houston call before a person or an agent does anything: agents start with no access, ask for exactly what a task needs, and lose it when the task ends.

Today each product keeps its own permission rules, so nobody can say in one sentence who can do what, and why. Agents make this urgent: they act in seconds, create sub-agents and pass their access on, and that access can outlive the task it was given for.

Countersign gives both products one grant model, as a Go library they embed. Its safeguards:
- Access, and everything handed down from it, ends when its task closes, when its time runs out, or when it is withdrawn.
- Delegated access can only shrink. A service that must outlive its creator gets a separate service grant, owned by a team or role.
- Requests for actions marked contained are approved by rules in seconds. A parent agent may approve its child's request only for access it holds and is allowed to share.
- Approval is attached to the final action, so splitting a request into smaller pieces can't get around it.
- Production, real secrets, money and deletion always wait for an authorised person.
- Nobody approves their own request.
- Every decision names the grants and rules behind it and can be replayed.
- Each check takes well under a millisecond.

Version 1 (v1) covers the core model plus enforcement in OpenBao. This hand-in is its design: a PRD, an architecture, a grant schema, a decision table and a two-engine spike. It is not a working product.

## The Problem

Orvex runs two products that hand out power:

- **Quornex**, a developer cloud, gives people and agents access to clusters, secrets, registries and deploys.
- **Houston** runs fleets of AI agents that need that access to do their work.

The symptom already shows with people: access is granted and then forgotten, like an old admin role nobody remembers giving. AI agents make the problem urgent. They act in seconds, create sub-agents and share their permissions with them. Their access can stay active after the task is over, and nobody is watching it.

Having two rule sets makes this worse. When each product handles expiry, delegation and withdrawal on its own, the two drift apart without anyone noticing.

## Who This Serves

- **Primary: Quornex and Houston engineers** who integrate the library. They stop maintaining two sets of rules: they call one library and get an answer. If a check is slow or hard to integrate, they are the first to complain.
- **Secondary: the security team.** Whether they trust Countersign decides whether Orvex relies on it. When something goes wrong, they can ask "why could this agent read that secret?" and get one precise answer, instead of searching two systems.
- **Approvers.** Human approvers see only the requests that really need a person.
- **Agents and the developers who build them.** Agents ask for what a task needs. When a request is refused, the refusal explains why and who can approve it.
- **Customer admins.** `[OPEN]` Their role (see Open Questions).

**When these groups want different things:** security comes first for permissions and rules. Engineers come first for speed and ease of integration. A fast refusal is acceptable; a wrong allow is not.

## The Solution

Countersign is a library inside each product, not a service people log into.

**The common case.** `api-refactor` asks to push to `feature/retry`. Rules approve it in seconds, and no person is involved.

**A request that waits for a person, start to end.**
1. `db-migrate` needs to read the orders secret in production for task ENG-88, so it asks for that permission.
2. Production secrets wait for a person. Erin gets the request, which shows who is asking, for what, for which task and for how long.
3. Erin approves. `db-migrate` gets one grant that records all of this.
4. Every check names that grant. A refusal says why, and who could approve.
5. The access ends when ENG-88 closes or the grant's time limit is reached, or earlier if Erin withdraws it. Anything handed down from it ends too.
6. Later, the security team can ask why it was allowed and get one answer.

**Three classes of action.**
- **Always human-only:** production, real secrets, money and deletion. Markings can add to this floor but never remove from it.
- **Marked contained:** rules may approve. Only a human with authority over the resource can mark an action as contained, never an agent.
- **Everything else, including an action whose marking is missing or can't be read:** human-only (fail closed).

**Who may approve.**
- Nobody approves their own request.
- Anything marked human-only goes to an authorised person.
- A parent agent may approve its child's request only for access the parent holds and is allowed to share. Spawning a child gives the parent no new power: the child never gets more than the parent could do itself, and the human-only classes are untouched.
- Approval is attached to the final action, not to the steps before it. No number of contained grants ever adds up to approval for a human-only action; that action still goes to a person when it is attempted.

**Service grants.** A service that must outlive the person who created it gets a service grant. This is not delegation:
- An authorised person, or someone holding that permission, creates it, and it is approved like any other grant.
- Its named owner is a team or role, not the person who created it. It is not a child of their grant, so nothing cascades when they leave.
- It can still be withdrawn, and withdrawing it takes back its own children.
- Human-only classes still need a person to approve it.
- Who may create one is a sensitive power, so every creation is audited.
- Like every grant, it has a hard maximum lifetime. `[PROPOSAL, to check with Danny]` A review date comes before that limit, and renewing the grant means a new approval.

Delegated access still only shrinks, and still ends with its parent.

**Where permissions live.** Each installation has a permission store holding its grants, approval authority and delegation limits, and every change to it is audited. These are not Git files, so an agent can get access in seconds without a pull request. Getting access doesn't skip review: the infrastructure change itself still goes through the installation's Forgejo, where console, CLI and agent changes become reviewed Git commits.

**What Countersign does not do**
- **It is not an identity system.** Keycloak says who a person is, and Kubernetes says which agent or service is calling. Countersign decides what they may do, including who can approve and who can give access to others.
- **It has no screens.** Quornex and Houston provide the approval screens.
- **It does not store secrets.** OpenBao does.
- **It does not translate permissions into Keycloak or Kubernetes RBAC.** Both are used for identity only.
- **It does not replace OpenBao.** Countersign's decision is enforced in OpenBao. If a restriction can't be enforced, the access is not allowed.

## What Makes This Different

**The one thing:** one grant model that both products share, with approval, delegation and withdrawal built in. Access only shrinks as it moves down a chain, it ends on its own, and every answer says why.

**Why not the alternatives**
- **Keep two rule sets and sync them better.** Syncing doesn't give the two copies one meaning. Expiry, delegation and withdrawal would still be enforced in two places, and the weaker copy wins. The problem is two sources of truth, not slow sync.
- **Buy an access product (for example Opal or Teleport).** Part of the answer is Orvex's own decision: Countersign is to be built in Go, as embeddable open source. The rest is fit. It must be embedded in both products, check in well under 1 ms, and enforce decisions in the tools Orvex ships (OpenBao in v1). That part is something to confirm, not a proven gap.
- **Use an open-source engine directly (for example OpenFGA or Cedar).** These engines answer "is this allowed?", but they don't know about expiry tied to a task, delegation chains, approval rules, or enforcement in OpenBao. All of that would have to be built around the engine anyway. Countersign may well use one of them inside; the two-engine spike decides.

**What we don't claim**
- That it is a better policy language than Cedar or OPA.
- That it is a better access-request product than Opal or Teleport.
- That the alternatives can't do this. We claim only that we haven't found one that does it all in one embedded model.

## Success Criteria

| Signal | Target |
|---|---|
| Grants that end | Every grant, including service grants, has a hard maximum lifetime, so 100% of grants end. If the "task closed" event never arrives, the grant ends at that limit, and nothing can extend it. `[OPEN]` the value of the limit. |
| Access ends with the task | Access stops within 10 seconds `[tentative]` of Countersign learning that the task closed. |
| Withdrawal cascades | 100% of descendants are withdrawn, and checks refuse them, within 10 seconds `[tentative]` of the withdrawal being made. |
| No bypass | Drift checks find no access for agent identities in OpenBao that Countersign didn't decide. |
| Speed | Each check takes well under 1 ms at p99. |
| Explainability | 100% of decisions can be replayed from their record for at least one month. |
| Approvers not overwhelmed | A goal to measure, not a given: about 90% of requests approved by rules alone, and fewer than 20 requests per approver per day. It depends on how many actions are marked contained. |

**This fails, even if the code runs, if** people bypass Countersign because it is slow or annoying, approvers click "yes" without reading because there are too many requests, or a native tool such as OpenBao does the opposite of what Countersign says and nobody notices.

## Scope

**In v1**
- Default deny: anything unknown or failing is a refusal.
- Grants that end when their time runs out, their task closes, or their agent ends, plus a hard maximum lifetime.
- Three action classes: always human-only (the floor of four), marked contained, and everything else human-only.
- The fixed approval rules (see Who may approve).
- Approval attached to the final action, so splitting a request can't get around it.
- Service grants that outlive their creator, owned by a team or role.
- Delegation that can only shrink, and withdrawal that cascades.
- A replayable explanation for every decision, including who could approve.
- Checks well under 1 ms, plus answering "what may this agent see?" when a read returns a list, such as a feed.
- Enforcement of Countersign's decisions in OpenBao, with drift detection.

**Postponed.** `[OPEN]` Argo CD, Harbor and Grafana use the same translator idea, but their formats are coarser (Harbor and Grafana have fixed roles), so they add effort without changing the design. The rule for them is specified now (never more access than the grant) and implemented later.

Until Argo CD is translated, agents get no Argo CD access through Countersign, and production syncs stay with people. Countersign does not govern them yet, and that gap is stated, not hidden.

**Out of v1**
- Customers writing policy code.
- Identity and approval screens (see What Countersign does not do).
- Multi-installation features: each installation has its own permission store, and nothing is shared across installations.
- Group grants: a team is expanded into individual grants.
- Deny grants, free-form conditions, and time-of-day or IP rules.
- Translation into Keycloak or Kubernetes RBAC: both are used for identity only.
- A task-budget system for splitting.

## Vision

If v1 works, Countersign becomes the one place where Orvex answers "who can do what, and why?" for every person and agent. It grows step by step:
1. the remaining native tools (Argo CD, Harbor, Grafana)
2. ways for customers to configure their own rules (such as what counts as contained, who may approve, and their own limits) in a typed, safe form, still without policy code
3. use beyond Quornex and Houston

What it must not become is a big policy platform nobody can reason about. Every new feature has to keep the promise that access only shrinks, ends on its own, and every answer can be explained. If a feature breaks that promise, it doesn't go in.

## Open Questions

- **To check with Danny:**
  - Service grants: a review date before the hard maximum lifetime, with renewal by a new approval. Is that right?
  - Argo CD, Harbor and Grafana: are they still in the plan as postponed?
  - Has Orvex had an incident with an agent's access?
  - Are approvers overloaded today?
- **Not yet proved:** a parent approving its child's request never gives either of them more than the parent could do itself.
- **Allowed to share:** where does a parent's "allowed to share" come from? Is it a separate permission, or is holding the access enough?
- **Orphaned service grants:** who takes over a service grant if its owning team or role disappears?
- **Customer admins:** grants live in the permission store, not in Git, so what is their role?
- **The hard maximum lifetime:** what value?
