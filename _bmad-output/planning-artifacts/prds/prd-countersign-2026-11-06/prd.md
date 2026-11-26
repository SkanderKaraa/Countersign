---
title: "PRD: Countersign"
status: final
created: 2026-11-06
updated: 2026-11-26
---

# PRD: Countersign

## 0. Document Purpose

This PRD is for the engineers who will design and build Countersign, and for the Orvex reviewers. It turns the final product brief into testable requirements. It does not repeat the brief's "why it matters".

**Related artefacts**
- **The product brief and its addendum** (`briefs/brief-countersign-2026-10-30/`): what this PRD builds on.
- **This PRD's `addendum.md`:** Danny's 2026-11-11 answer, paraphrased, and the decision-table seeds.
- **The architecture** (`architecture/architecture-countersign-2026-11-11/`): `ARCHITECTURE-SPINE.md` (binding), `architecture.md`, the decision table and the two-engine spike. Changes made to this PRD after it was finalised are logged in this PRD's `.memlog.md`.
- **`AI_LOG.md`** at the repo root: the AI-assistance log, including the Q&A modules referenced by FR-29 and NFR-6.

**How it's structured**
- The Glossary (§3) fixes the vocabulary.
- Features (§4) group the functional requirements, numbered FR-1 onwards. Each FR has pass/fail consequences. The IDs are stable references, so some appear out of numeric order (for example FR-18 and FR-29).
- Cross-cutting requirements (§5) are NFR-1 onwards, and success metrics (§8) are SM-1 onwards. Both point back to FRs.
- Engines, data structures and mechanisms belong in the architecture, not here.

**Tags**
- `[OPEN]`: not decided yet. Collected in §9.
- `[ASSUMPTION]`: inferred, and needs confirming. Collected in §10.
- `[PROPOSAL]`: Skander's own suggestion, not from Danny.
- `[tentative]`: a target number Skander chose, to be confirmed in the spike.

## 1. Vision

Countersign is a Go library that Quornex and Houston embed and call before a person or an agent does anything. It decides what they may do. Every person and agent starts with no access, and anything unknown, malformed or failing is refused. Access comes only from a grant, which is created when a request is approved. The grant ends when its task closes, when its time runs out, when its agent ends, or when it is withdrawn.

Each check returns "allow" or "refuse", together with the grants and rules behind the answer. A request for access returns "approved", "ask" or "refuse". Requests for actions marked contained are approved by rules in seconds. Human-only actions wait for an authorised person. A parent agent may approve its child agent's request only for access it holds and is allowed to share. Nobody approves their own request, and splitting a request can't get around the approval of the final action. A service that must outlive its creator gets a separate service grant, owned by a team or role. Apart from service grants, access handed down can only shrink and ends with its parent grant, and withdrawing a grant withdraws everything handed down from it.

Identity comes from Keycloak (people) and Kubernetes (agents and services). Countersign's decisions are enforced in OpenBao. Every decision can be explained and replayed, and each check takes well under a millisecond.

Countersign has no screens, does not store secrets and does not establish identity. A fast refusal is acceptable; a wrong allow is not.

## 2. Target Users

### 2.1 Jobs To Be Done

- **Quornex and Houston engineers (primary):** embed one library and get an answer, instead of keeping two rule sets.
- **The security team:** answer "why was this allowed?" from one record.
- **Approvers:** decide only the requests that need a person.
- **Agents and their builders:** ask for what a task needs, and understand a refusal.
- **Customer admins:** `[OPEN]` their role.

**When these users want different things,** security comes first for permissions and rules, and engineers come first for speed and ease of integration. A fast refusal is acceptable; a wrong allow is not.

### 2.2 Key User Journeys

- **UJ-1. `api-refactor` pushes to a feature branch, and nobody is bothered.**
  `api-refactor` asks to push to `feature/retry`. The action is marked contained, so rules approve it in seconds, and no person is involved.

- **UJ-2. `db-migrate` waits for Erin before reading a production secret.**
  1. `db-migrate` asks to read the orders secret in production for task ENG-88, for 2 hours.
  2. Production secrets are human-only, so the request returns "ask". Erin gets the request, which shows who is asking, for what, for which task and for how long.
  3. Erin approves. `db-migrate` gets one grant that records all of this, and every later check names that grant.
  4. The access ends when ENG-88 closes or the grant's time limit is reached, or earlier if Erin withdraws it.

- **UJ-3. Erin withdraws `release-prep`, and everything below it stops.**
  `release-prep` holds access to push to the api repos and has handed three grants down to helper agents. Erin sees that `release-prep` is acting oddly, so she withdraws its grant.
  - **One action:** the withdrawal is one action and takes effect within 10 seconds `[tentative]`.
  - **The cascade:** the three handed-down grants stop working with it, and so would any grant below those. Nobody withdraws them one by one.
  - **What the helpers see:** on their next check they are refused. The answer says that Erin withdrew the access and when, and who could approve new access, so each agent knows why it was refused.
  - **`release-prep` itself:** it is also refused. Anything it already started, such as a build that is running, isn't undone by Countersign.
  - **Replay:** later, the security team can replay a check from before the withdrawal and see that it was allowed, then one from after it and see why it was refused.
  - **Service grants:** a service grant is different. It isn't a child grant of `release-prep`'s grant, so it stays unless someone withdraws it directly.

- **UJ-4. A month later, the security team asks why `db-migrate` could read the orders secret.**
  The question is: "Why could `db-migrate` read the orders secret at 09:21?" The answer names the grant, who approved it, the rules that applied, and the state of the permission store at that moment. Replaying the decision gives the same answer.

- **UJ-5. An agent reads a feed and sees only what it may see.**
  An agent opens a feed of 10,000 items. The product makes one list check, and the agent sees only the items its grants allow. One explanation names the grants that let each item through.

## 3. Glossary

_FRs, journeys and metrics use these terms exactly. No synonyms._

**Principals and installations**
- **Principal:** a person (identified by Keycloak) or an agent or service (identified by Kubernetes) that makes a request. It has this meaning only.
- **Installation:** one customer's deployment, with its own permission store.
- **Permission store:** each installation's store of grants, approval authority, markings and delegation limits. Every change to it is audited (FR-25).

**Requests and grants**
- **Request:** what a principal sends: who, what (action), which resource, which task and how long. It never names an approver (FR-1).
- **Request outcome:** "approved" (by rules, for a contained action), "ask" (approval is needed) or "refuse" (see FR-1).
- **Grant:** an approved request, and the only source of access. It records who, what, where, why (task), who approved and when it ends (FR-2). It can be narrowed or ended early, never widened or extended (FR-4).
- **Task grant:** a grant tied to a task, as opposed to a service grant (FR-3).
- **Hard maximum lifetime:** the limit after which every grant ends, whatever else happens. Task grants and service grants each have their own (FR-3).

**Action classes and authority**
- **Action:** something a principal can do on a resource, for example push or read.
- **Action class:** an action is in the floor of four, marked contained, or human-only by default (FR-5).
- **Floor of four:** production, real secrets, money and deletion, which are always human-only. No marking removes an action from it, and marking another action human-only doesn't add that action to it (FR-6).
- **Contained:** a marking, set by a person with authority over the resource, that lets rules approve the action (FR-6, FR-10).
- **Human-only:** an action that only an authorised person may approve.
- **Contained grant:** a grant for an action marked contained, approved by rules (FR-10).
- **Human-only grant:** a grant for a human-only action, approved by an authorised person (FR-11).
- **Authority over a resource:** the standing right to manage a resource in Countersign: setting its markings, giving approval authority for it, and narrowing or withdrawing grants on it. It comes from the installation administrators (FR-9).
- **Approval authority:** permission to approve a specific action on a specific resource (FR-7).

**Approval**
- **Final action:** the action a request is ultimately for. Approval is attached to it, not to the steps before it (FR-13).
- **Approval rule:** the rule that approves a contained request without a person (FR-10).
- **Fixed rule:** one of the rules no grant or setting can change, such as "nobody approves their own request" (listed in NFR-5).

**Delegation**
- **Parent agent / child agent:** a child agent is an agent spawned by a parent agent.
- **Parent grant / child grant:** a child grant is a grant handed down from a parent grant. Its descendants are its child grants, their child grants, and so on.
- **Hand down:** the one verb for delegation, meaning to give a child agent part of a grant. "Allowed to share" and "sharing limit" are only names of the permission that allows it.
- **Delegation:** handing part of a grant down to a child agent (FR-14, FR-15).
- **Allowed to share:** an explicit permission to hand part of a grant down to a child agent. Holding the access never implies it, and neither does a contained marking. For a human-only grant, the person who approved it sets it. For a contained grant, it is a standing sharing limit `[ASSUMPTION]` (see FR-8).
- **Delegation depth:** how many more times a grant may be handed down (FR-18).
- **Service grant:** a grant owned by a team or role, for a service that must outlive its creator. It is not a child grant of anyone's grant (FR-16).
- **Owner:** the team or role that owns a service grant.

**Ending and withdrawal**
- **Withdrawal:** ending a grant early. It also ends every grant handed down from it (FR-19).
- **Withdrawal target:** the time by which checks against a withdrawn grant and its descendants are refused: 10 seconds `[tentative]` from the withdrawal (FR-19).
- **Signal:** "task closed" or "agent ended", sent by the calling product and treated like a withdrawal (see FR-20).

**Checks and decisions**
- **Check:** one call that reads existing grants and returns a decision: "allow" or "refuse", never "ask" (see FR-21).
- **High-risk:** any action that is not marked contained, meaning every human-only action, the floor of four included. Used by the staleness and clock rules (NFR-2, NFR-6).

**Records and replay**
- **Explanation:** the grants and rules behind a decision, plus who could approve if it was refused.
- **Replay:** re-running a past decision against the permission store as it was then (FR-24).

**Replicas and time**
- **Replica:** a copy of the permission store's data, held by the library inside a product.
- **Staleness horizon:** how long a replica that can't confirm it is fresh keeps answering high-risk checks before it refuses them (NFR-2).

**Enforcement**
- **Enforcement:** turning a decision into short-lived access in OpenBao (FR-26, FR-27).
- **Drift:** access in OpenBao that Countersign didn't decide (FR-28).

**Verification**
- **Decision table:** 25 checks, 5 of them attacks, each with its expected answer and the FRs it tests (Orvex's wording). It lives in the architecture (SM-8). In the decision table, a "check" is one test row, which may exercise a check, a request or a hand-down.
- **Two-engine spike:** the architecture's comparison of two engines against the decision table.

## 4. Features

### 4.1 F1. Requests and grants

A request asks for access, and an approved request becomes a grant, the only source of access. A grant ends on its own and can be narrowed or ended early, never widened or extended (FR-1 to FR-4).

**Functional Requirements**

#### FR-1: Make a request

A principal can request access by stating the action, the resource, the task and the duration. Realises UJ-1 and UJ-2.

**Consequences (testable):**
- A request that is missing any of these fields is refused.
- A request that names an approver, in any field, is refused. Countersign works out who may approve, so a crafted request can't name its own approver.
- A request for an unknown action or resource is refused.
- A request for longer than the hard maximum lifetime is refused, not shortened. The refusal states the maximum, so the agent can ask again for an allowed duration.

**Request outcomes (testable):**
- A request returns "approved", "ask" or "refuse".
- The requester never chooses who approves. Countersign works out who may approve from the action class and the approval authority, and an "ask" lists who could approve, so the product can show the right screen or notify the right agent. Every approval goes through Countersign.
- "Refuse" is for something no approval could allow, and "ask" is for something a new approval could allow.
- After a grant is withdrawn, a new request for the same access returns "ask", or is approved by rules if the action is contained.
- A request that a rule always forbids, such as someone approving their own request, returns "refuse", because "ask" would be false hope.

#### FR-2: An approved request becomes a grant

When a request is approved, Countersign creates one grant recording who, what, where, why (task), who approved, when it ends, and `[ASSUMPTION]` when it was issued (needed for replay). A grant is the only source of "allow". Realises UJ-2.

**Consequences (testable):**
- A check with no matching grant never returns "allow".
- Every "allow" names the grant behind it.
- A grant missing any of its recorded fields is treated as invalid, and checks against it are refused.

#### FR-3: A grant ends

A grant ends at the first of: its time limit, its task closing, its agent ending, or the hard maximum lifetime. Realises UJ-2.

**Consequences (testable):**
- A check after any one of these four events is refused.
- No grant can exist without a hard maximum lifetime.
- **Task grants:** the hard maximum lifetime is under 24 hours `[tentative]`. A human-approved task grant defaults to 8 hours `[tentative]`, and a contained grant is capped at 4 hours (FR-10).
- **Service grants:** they have their own, longer hard maximum lifetime. `[OPEN]` Its value. Giving service grants a hard maximum lifetime at all is `[PROPOSAL]` Skander's choice: Danny said they "last on their own" (§9, question 5).
- If the task or agent was not valid when the grant was made, the grant is invalid (fail closed).
- If the task or agent was valid but its source can't be reached later, the grant is not treated as ended. Its time limit and the staleness rules still apply, so one outage in the task system doesn't end every grant.
- A task or agent known to be gone ends the grant.

#### FR-4: Narrow or end a grant early

A grant's holder may end or narrow its own grant, because that only lowers its own power. Its approver, or anyone with authority over the resource at that moment, may narrow or end it. For example, Erin can cut 2 hours back to 1. Nobody may widen or extend a grant, including its approver. Realises UJ-3.

**Consequences (testable):**
- An attempt to widen or extend a grant is refused, and the refusal says that a new request is needed.
- An attempt to narrow another principal's grant, by someone without authority over the resource, is refused.
- An agent may narrow its child agent's grant only within what it is allowed to share.
- Every narrowing or early end is audited, recording who made the change.

### 4.2 F2. Action classes and approval settings

Every action is in the floor of four (production, real secrets, money, deletion), which is always human-only, or it is marked contained, or human-only by default. These settings are made in the permission store by a person with authority over the resource, never by an agent:
- marking an action contained
- giving someone approval authority
- deciding what a parent agent may hand down

All of them are human-only changes, and all are audited. A marking never removes an action from the floor of four.

**Functional Requirements**

#### FR-5: Every action has one action class

Every action is in exactly one action class: the floor of four, marked contained, or human-only by default. Realises UJ-1 and UJ-2.

**Consequences (testable):**
- A request for an action with no marking returns "ask", with authorised people as the only approvers.
- An action whose marking can't be read is treated as human-only.
- An action in the floor of four is always human-only, whatever its markings say.

#### FR-6: Mark an action contained or human-only

A person with authority over a resource can mark an action on it as contained, or as human-only. A marking never removes an action from the floor of four. Marking an action human-only makes it human-only, but doesn't add it to the floor of four. Realises UJ-1.

**Consequences (testable):**
- An agent's attempt to set or change any marking is refused.
- A person without authority over the resource is refused.
- An attempt to mark an action in the floor of four as contained is refused.
- Every marking change is audited: who made it, when, and the old and new values.

#### FR-7: Give approval authority

A person with authority over a resource can give a person approval authority for specific actions on that resource. Realises UJ-2.

**Consequences (testable):**
- An agent's attempt to give or receive approval authority for human-only actions is refused.
- A person can't give approval authority wider than their own authority over the resource.
- Every change is audited.
- Approval authority has a hard maximum lifetime, like a grant. It can be longer than a grant's (target: 90 days `[tentative]`). Renewing it is a new decision by a person with authority over the resource.
- Grants that a person already approved stay valid when that person's approval authority expires. Only the power to approve new ones ends.

#### FR-8: Allow handing down a grant

A person can explicitly allow a grant's holder to hand part of it down to a child agent. This sets "allowed to share", and the same rules apply to every grant:
- **For a human-only grant,** the person who approved it sets "allowed to share" on that grant.
- **For a contained grant,** `[ASSUMPTION]` a person with authority over the resource sets it in advance, as a standing sharing limit for that resource and action. The sharing limit isn't set per grant, because no person approves each contained grant. Setting it is a separate act from the marking: marking an action contained never allows handing down by itself (Danny, 2026-11-11). The sharing limit follows the same rules as every "allowed to share" permission:
  - an agent can never set it
  - delegation depth defaults to zero, so with no sharing limit nothing can be handed down
  - the child grant still gets the same or less, and ends no later than its parent grant
  - every change is audited

Realises UJ-3.

**Consequences (testable):**
- Without "allowed to share", a parent agent can't hand down any part of any grant.
- For any grant, holding the access alone never allows handing it down. A contained marking allows automatic approval only, never handing down.
- An agent can't set or widen "allowed to share" on any grant, including its own.
- Every change is audited.

#### FR-9: Installation administrators are the root of authority

The first trusted installation administrators are established in Countersign during installation. They hold the first authority over the installation's resources and hand out approval authority from there. Keycloak identifies them, but Countersign doesn't automatically trust Keycloak's admin role (Danny, 2026-11-11).

**Consequences (testable):**
- An installation has at least two administrators. An action that would leave fewer is refused.
- Adding or removing an administrator is a human-only change, approved by another administrator and audited.
- An administrator's authority also has a hard maximum lifetime, but it never lapses silently. Renewal needs another administrator, and an alert fires at least 14 days `[tentative]` before it lapses.
- No two administrators' authority may lapse within 14 days `[tentative]` of each other, the same window as the alert, so their expiries are always staggered. A last-resort recovery path for a fully lapsed installation is for the architecture (§9).

**Notes:**
- How the very first administrator is set at installation time is for the architecture.

### 4.3 F3. Approval

Rules approve contained requests, an authorised person approves human-only actions, and nobody approves their own request. Approval is attached to the final action, so splitting a request can't get around it. A parent agent's "approval" of its child agent's request is delegation, specified in F4 (FR-8, FR-14). `[OPEN, to check with Danny]` This PRD reads Danny's "anything requiring a human still goes to a human" as "a person decides once, when approving the grant and its sharing limit".

**Functional Requirements**

#### FR-10: Rules approve contained requests

A request for an action marked contained is approved by rules, with no person involved, and becomes a grant. The approval rule verifies only:
- that the action is marked contained
- that the normal request checks pass: a known principal and a valid task
- that the duration is within the maximum for contained grants (4 hours `[tentative]`), which is shorter than the hard maximum lifetime

No other conditions apply. Realises UJ-1.

**Consequences (testable):**
- A request for a contained action is approved without anyone being asked, within 2 seconds `[tentative]`.
- A request for a contained action that is longer than the contained maximum is refused, and the refusal states the maximum.
- A request for any other action returns "ask".
- The grant records the rule that approved it, as its approver.

#### FR-11: A person approves human-only actions

Only a person with approval authority for that action on that resource can approve a request for a human-only action. Realises UJ-2.

**Consequences (testable):**
- An approval by a person without approval authority for that action and resource is refused.
- An approval of a human-only action by an agent is refused, however the approval is worded.
- The grant records the person who approved it.

#### FR-12: Nobody approves their own request

The principal that made a request can never approve it. Realises UJ-2.

**Consequences (testable):**
- An approval where the approver is the requester is refused, whichever identity source the approver uses.
- `[ASSUMPTION]` Countersign treats a person's Keycloak identity and any agent identity running as that person as different principals. Only the exact requester is excluded, as Danny asked.
- `[OPEN, to check with Danny]` Sam starts `api-refactor`, it asks to deploy to production, and Sam approves. Is that self-approval, or the normal case of a person approving their own agent? The decision table includes this case, marked open until §9 question 4 is answered.

#### FR-13: Approval is attached to the final action

A change reaches production only through an action that Countersign checks and that is human-only, however many contained grants came before it. Two mechanisms work together:
- **The main mechanism, a checked gate:** the merge or sync of a change into production (through the installation's Forgejo) is itself an action Countersign checks, and it is human-only. However a change was split into pieces, they all meet that one gate.
- **The backstop:** anything that writes directly to a production resource is in the floor of four, so such a piece is human-only on its own.

Realises UJ-1 and UJ-2.

**Consequences (testable):**
- An agent holding grants for every step leading to a human-only action is refused when it checks that action (it has no grant for it), and its request for it returns "ask".
- No number of contained grants ever produces "allow" for a human-only action.
- **Attack test case:** `helper` writes ten config files, each contained, into a branch that later becomes a production change. The check on the merge or sync into production is refused, and the request for it returns "ask", with only people as approvers.
- A direct write to a production resource is human-only, whatever the step is called.

**Notes:**
- **The gate is `[OPEN, to check with Danny]`.** This relies on the merge or sync into production really passing through a Countersign check, which Quornex has to build. It is Skander's reading of Danny's Forgejo answer (brief addendum, 2026-11-04): fast access "doesn't bypass the review required for the infrastructure change itself". That sentence covers review, but not that the merge or sync is checked by Countersign, so question 6b stays open. If the gate is missing, Countersign alone can't stop the splitting attack through merges.
- **If Danny says no:** v1 ships with a stated gap, like the Argo CD gap (§7.2). Production merges and syncs stay with people outside Countersign, and the backstop still holds (direct production writes are human-only). Splitting through merges is a stated gap, not a hidden one.

### 4.4 F4. Delegation and service grants

Access handed down can only shrink and ends with its parent grant. A service grant is separate. An authorised person creates it, or an agent that a person has explicitly authorised. It is approved like any grant, is owned by a team or role, and does not cascade from its creator.

**Functional Requirements**

#### FR-14: Hand down part of a grant

A parent agent can hand part of a grant it holds down to its child agent, as a new child grant. Realises UJ-3.

**Consequences (testable):**
- A child grant with a wider action or resource than its parent grant's is refused.
- A child grant that would end later than its parent grant is refused.
- Handing down any part of a grant without "allowed to share" is refused, whether the grant is contained or human-only. **Decision-table case:** a parent agent tries to hand down a contained grant without "allowed to share" (refuse), then with it (allowed).
- The child grant records its parent grant and who handed it down. Its explanation shows the chain up to the person or rule that approved the first grant.
- At check time, a child agent's access is the overlap of its own grant and every ancestor grant as it is now. A child grant is never wider than its parent grant at any moment, not only when it is created.

#### FR-15: Delegated access ends with its parent grant

When a parent grant ends, is withdrawn or is narrowed, its child grants follow. Realises UJ-3.

**Consequences (testable):**
- After a parent grant ends or is withdrawn, checks against its child grants, and theirs, are refused.
- After a parent grant is narrowed, a check against a child grant for anything outside the parent grant's new scope is refused, and the child grant stays valid inside it. If Erin cuts `release-prep` from all api repos to one, a helper working on that one repo keeps working.

#### FR-18: Delegation depth

`[PROPOSAL]` Each grant carries a delegation depth: how many more times it may be handed down. The proposal comes from Skander's own earlier schema work, not from Danny. Realises UJ-3.

**Consequences (testable):**
- The depth defaults to zero, so a child grant can't be handed down again unless allowed.
- A child grant's depth is always smaller than its parent grant's.
- Delegation depth is set by the person who authorises handing down, as part of "allowed to share" (FR-8). Rules never set it, so a contained marking never gives any depth.
- A system maximum depth (target: 3 `[tentative]`, untested) keeps every chain short enough to check within the speed target. Handing down beyond it is refused.

#### FR-16: Create a service grant

An authorised person can create a service grant for a service that must outlive its creator. An agent can also create one, but only if a person has explicitly authorised it to, within limits that make it safe (Danny, 2026-11-11). Realises UJ-3.

**Consequences (testable):**
- An agent's attempt to create a service grant without explicit authorisation is refused.
- That authorisation comes from a person with authority over the resource. It is scoped to specific actions and resources, has a hard maximum lifetime, and is audited, under the same rules as approval authority.
- A service grant created by an agent can't be wider than what the agent holds at that moment, or than its authorisation. Creating one never gives anything new.
- The creating agent never approves its own service grant.
- Every service grant is approved by a person with approval authority, never by its creator, and never by rules, even for a contained action. Long-lived access always has a human decision behind it. The 4-hour contained maximum doesn't apply to service grants.
- A service grant names an owner (a team or role), never the creating agent. A service grant without an owner is refused.
- A service grant is not a child grant of its creator's grant: withdrawing the creator's grants, or the creator leaving, doesn't end it.
- A service grant is never handed down. A helper that needs long-lived access gets its own service grant.
- A service grant has its own hard maximum lifetime, longer than a task grant's (FR-3). `[PROPOSAL, to check with Danny]` A review date comes before that limit, and renewal is a new approval.
- Every creation is audited. A creation by an agent also alerts the owner.

**Notes:** A service grant outlives its creator, so an agent could otherwise turn short-lived access into long-lived access. These rules close that escape: the agent can do that only for access it already holds, that a person authorised it to convert, and that a person approves if it is human-only. FR-15 is untouched, because a service grant is not delegation.

#### FR-17: Withdraw a service grant

A person with authority over the resource can withdraw a service grant, which also withdraws its own child grants. Realises UJ-3.

**Consequences (testable):**
- After a service grant is withdrawn, checks against it and its child grants are refused.

**Notes:** `[OPEN]` Who takes over a service grant if its owning team or role disappears?

### 4.5 F5. Ending and withdrawal

Withdrawing a grant withdraws everything handed down from it, within seconds. Closing signals from the calling product are treated like a withdrawal, and only ever reduce access (FR-19, FR-20).

**Functional Requirements**

#### FR-19: Withdraw a grant

Anyone who may end a grant early (see FR-4) can withdraw it in one action. Every grant handed down from it is withdrawn with it. Realises UJ-3.

**Consequences (testable):**
- **Test case:** withdraw a parent grant that has three child grants and one grandchild grant. All five grants are refused within 10 seconds `[tentative]` of the withdrawal.
- The clock starts at the withdrawal, not when a replica hears of it.
- A refusal after withdrawal says who withdrew the grant and when.
- Service grants that aren't descendants of the withdrawn grant are unaffected.
- Every withdrawal is audited.

#### FR-20: Accept closing signals

Countersign treats a "task closed" or "agent ended" signal from the calling product as a withdrawal of the grants for that task or agent. Realises UJ-2.

**Consequences (testable):**
- After a valid closing signal, checks against the affected grants are refused within 10 seconds `[tentative]` of the permission store accepting it. The clock starts there, not when a replica hears of it, mirroring FR-19.
- A closing signal is accepted even when its source can't be fully trusted, because it only reduces access. A forged one causes a recoverable denial of service, not a wrong allow.
- Every accepted signal is recorded with its source in the audit trail.
- A flood of closing signals from one source raises an alert. A flood means more than 20 signals per minute from one source `[tentative]`.
- A signal that would widen access is never accepted.
- If a genuine signal never arrives, the hard maximum lifetime is the backstop (FR-3).
- When a parent agent ends, the grants it handed down end too (FR-15), whatever ended it. A child grant that outlived its parent grant would let delegated access escape, which is the attack this defends against.

**Notes:**
- A child agent that still has work to finish needs a new request, or a service grant if it must outlive its parent agent.
- How signals are authenticated is for the architecture.

### 4.6 F6. Checks and lists

A check returns "allow" or "refuse", with the grants and rules behind it, and who could approve if it is refused. A request's outcomes are in FR-1. A list check answers "what may this agent see?" for feed reads.

**Functional Requirements**

#### FR-21: One check returns a decision and its explanation

A product calls Countersign with a principal, an action and a resource, and gets back one decision ("allow" or "refuse") with its explanation. A check only reads existing grants and never changes state. Asking for access is a separate call: a request (FR-1). Realises UJ-1, UJ-2 and UJ-3.

**Consequences (testable):**
- Every decision comes with an explanation: the grants and rules behind it.
- A check with no matching grant returns "refuse", with the reason "no grant; make a request".
- A "refuse" says why, and who could approve new access.
- A check with an unknown principal, action or resource, a malformed input, or an internal error returns "refuse".
- A check using a withdrawn grant returns "refuse", and the explanation says it was withdrawn, by whom and when.

#### FR-22: A list check answers "what may this agent see?"

A product can ask which items of a feed, or of any list of resources, a principal may see. Realises UJ-5.

**Consequences (testable):**
- For every item, a list check returns the same decision as a single check on that item against the same permission store state.
- An item whose decision isn't "allow" is left out.
- The list result has one explanation, naming the grants that let each item through.
- A list check that can't finish within its time budget returns a refusal of the whole list rather than a partial list. The refusal says why (time budget exceeded). It is never an empty result the product could read as "no items". The product can try again or page.
- Returning only the items already checked is out of v1. The v1 rule is all or nothing.

### 4.7 F7. Explanation, replay and audit

Every decision can be explained and re-run later. Every change to the permission store is audited.

**Functional Requirements**

#### FR-23: Explain any decision

For any decision, past or present, Countersign can give its full explanation. Realises UJ-4.

**Consequences (testable):**
- An explanation names the grants behind the decision, each grant's chain up to the person or rule that approved it, the rules that applied, and the time used for the decision.
- An explanation identifies the state of the permission store that the decision was made against.

#### FR-24: Replay a decision

A decision can be re-run later against the permission store as it was at that moment, and gives the same decision and explanation. Realises UJ-3 and UJ-4.

**Consequences (testable):**
- 100% of decisions can be replayed for at least one month.
- **Test case:** replaying a check from before a withdrawal gives "allow". Replaying one from after it gives "refuse", with the withdrawal as the reason.
- A replay that can't rebuild the permission store state says so. It never guesses an answer.

#### FR-25: Audit every change to the permission store

Every change to the permission store is recorded: grants, withdrawals, markings, approval authority, "allowed to share", administrators and accepted signals. Realises UJ-4.

**Consequences (testable):**
- Each audit record says who made the change, when, from which source, and the old and new values.
- A change that can't be audited is not made.
- Audit records are append-only. Nobody, including an administrator, can change or delete one while it is being kept.
- Any attempt to change or delete an audit record is itself recorded, and raises an alert.

**Notes:** `[OPEN]` What happens when retention ends, and records may be removed, is for the architecture.

#### FR-29: Record every decision

_Source: drafted with Claude's help from the Module 2 and 5 design discussion, reviewed and adopted by Skander._

Every check is recorded with:
- the request and the decision
- the grants and rules behind it
- the state version of the permission store
- the time used and the clock offset
- the version of the fixed rules and of the library

Records for human-only actions, approvals, withdrawals and refusals under the fixed rules are kept in full. Others may be kept compactly and explained by replay. Realises UJ-4.

**Consequences (testable):**
- Every check produces a record with all of these fields.
- **Kinds kept in full** (human-only actions, approvals, withdrawals, refusals under the fixed rules) are recorded before the answer is returned. If recording fails, the check is refused.
- **Compact records** are written asynchronously and never slow a check. How the volume is stored is for the architecture.
- A record of each kind kept in full contains its whole explanation, without needing a replay.
- Replaying a compactly kept record reproduces the same decision and explanation (FR-24).

### 4.8 F8. Enforcement in OpenBao

A decision becomes short-lived access in OpenBao. If a restriction can't be enforced, the access is not allowed. Drift in OpenBao is detected.

**Functional Requirements**

#### FR-26: Enforce a decision in OpenBao

When a grant allows access to a secret, Countersign's decision is enforced in OpenBao as short-lived access. Realises UJ-2.

**Consequences (testable):**
- The OpenBao access never lasts longer than the grant behind it.
- When the grant ends or is withdrawn, two targets are measured separately: Countersign refuses within 10 seconds `[tentative]`, and the OpenBao access is removed within 10 seconds `[tentative]` of the withdrawal. The second depends on OpenBao revoking promptly. If it is slower, the access's short lifetime (first bullet) is the backstop.
- The OpenBao access is no wider than the grant: the same secret, the same operations.

#### FR-27: Access that can't be enforced is not allowed

If a restriction in a grant can't be enforced in OpenBao, the access is not allowed. Realises UJ-2.

**Consequences (testable):**
- A grant whose restriction OpenBao can't express exactly gets no OpenBao access, and the refusal says which restriction couldn't be enforced.
- OpenBao access is never wider than the grant, even for a moment.
- **Agreement test:** over the decision table, OpenBao allows exactly what Countersign allows, tested against a real OpenBao.

#### FR-28: Detect drift in OpenBao

Access in OpenBao for agent identities that Countersign didn't decide is detected and reported. Realises UJ-4.

**Consequences (testable):**
- **Test case:** access added to OpenBao by hand for an agent identity is detected and reported.
- Detected drift raises an alert and is reported to people. Countersign doesn't remove it on its own in v1.

**Notes:**
- Drifted access stays until a person removes it, so the gap between "detected" and "removed" is visible.
- Automatic removal could lock out people who legitimately added access by hand, and Countersign can't tell the difference without more context.
- Automatic removal for agent identities is a later option.

## 5. Cross-Cutting Requirements

These apply to every feature.

### NFR-1: Default deny, and fail closed

- Every principal starts with no access.
- Anything unknown, malformed or failing is refused, including internal errors and timeouts.
- A fast refusal is acceptable; a wrong allow is not.
- **The one stated exception:** a stale replica may keep allowing checks on contained actions until its second horizon (NFR-2), because their damage is bounded.

### NFR-2: Staleness

A replica that can't confirm it is fresh refuses in two steps, so a withdrawal can't be missed silently and a withdrawn grant is never honoured forever:
- **After the staleness horizon,** high-risk checks are refused. This horizon may be no longer than the withdrawal target (10 seconds `[tentative]`), so even a disconnected replica refuses high-risk checks within the time FR-19 promises. Checks on contained actions may continue, because their damage is bounded.
- **After a second, longer horizon,** everything is refused.
- `[OPEN]` The second horizon's value, and whether the first should be shorter than 10 seconds, to be measured in the spike.

### NFR-3: Speed and scale

- Each check takes at most 250 µs at p99 `[tentative]`, well under the 1 ms ceiling in Orvex's brief.
- This is measured at the scale in Orvex's brief: 10,000 installations with 50,000 grants each. Each installation has its own permission store, so the per-check target is measured against one store holding 50,000 grants. The fleet figure sizes the totals, such as record volume. Both p99 latency and memory per check are measured, not estimated.
- `[OPEN]` The time budget for a list check.

### NFR-4: Integration and identity

- Countersign is one Go library, embeddable and open source, used by both Quornex and Houston. Both use the same grant model.
- The same check from Quornex and from Houston, against the same permission store state, gives the same decision.
- No permission rules live outside the permission store. Neither product keeps its own copy of the rules.
- Identity comes in from Keycloak (people), typically connected to an external identity provider such as Google or Microsoft SSO, and from Kubernetes (agents and services). Countersign doesn't establish identity. It decides what principals may do.
- A check with an identity that isn't authenticated is refused.

### NFR-5: The fixed rules are provable

For each fixed rule, the architecture shows how it holds: **by construction** (what would break it can't be expressed or reached) or **by tests only**. The classification below is Skander's proposal. Which engine can express each rule is for the two-engine spike to decide.

| # | Fixed rule | How it holds | Why |
|---|---|---|---|
| 1 | Default deny (NFR-1) | Tests | "No grant means no allow" can be built in, because only a matched grant can produce an allow. But the error and timeout paths need tests. |
| 2 | A request never names its approver (FR-1) | Construction | The request type has no approver field, and only typed IDs are accepted, no free text. |
| 3 | Every grant has a hard maximum lifetime (FR-3) | Tests | Every write path checks the expiry against the hard maximum for its kind, but the grant type has no validated constructor, so it's shown by tests, not by construction (reclassified after the spike's part 1 review). |
| 4 | Markings never remove an action from the floor of four (FR-6) | Construction | The action class is the floor combined with the markings, so a marking can't remove from it. |
| 5 | Only an authorised person approves a human-only action (FR-11) | Construction and tests | Construction for "an agent can't be the approver": approvers are a separate type that agents can't create. Tests for the authority check. |
| 6 | Nobody approves their own request (FR-12) | Tests | It is a comparison of identities, and it depends on the identity being right. |
| 7 | Approval is attached to the final action (FR-13) | Construction and tests for the check itself | Countersign can show by construction that the final action is refused without its own grant, and that its request returns "ask" for a human-only action, however many contained grants exist. But "nothing reaches production except through the gate" depends on the production gate, which Countersign can't guarantee: `[OPEN, to check with Danny]`, see FR-13. **This is the rule with the most external dependency.** |
| 8 | Delegated access only shrinks, at every moment (FR-14) | Construction at check time, and tests | A child grant's access is the overlap with every ancestor grant as it is now, so a wider child grant has no effect. Tests cover the check when a child grant is created. |
| 9 | A child grant ends with its parent grant (FR-15) | Construction | A grant's validity is computed up its chain, so a child grant has no validity of its own. |
| 10 | Access that can't be enforced is not allowed (FR-27) | Tests | Against a real OpenBao. |
| 11 | Handing down needs explicit authorisation, for every grant (FR-8) | Construction and tests | Delegation depth defaults to zero, and a marking and a sharing limit are separate fields, so a marking can't allow handing down. Tests cover the hand-down check itself. |
| 12 | Agents never set markings, approval authority or "allowed to share" (FR-6 to FR-8) | Construction and tests | Those changes accept only the human approver type, which agents can't create (the same mechanism as rule 5). Tests cover the refusal path. |
| 13 | An agent-created service grant gains nothing new (FR-16) | Construction and tests | By construction at check time: its access is limited to what the agent held and was authorised to convert. Tests cover the creation limit and the approval of human-only actions. |

Summary: 9 of the 13 rules hold at least partly by construction (6 of them also need tests), and 4 by tests only (rules 1, 3, 6 and 10). Rule 7 also depends on wiring outside Countersign.

### NFR-6: Time and clock skew

_Source: drafted with Claude's help from the Module 2 and 5 design discussion, reviewed and adopted by Skander._ The open values (safety margin, maximum clock offset, sync recency) are Skander's to choose.

- A grant expires when either the wall clock or a local timer of elapsed time says so, whichever comes first, so a slow clock can't extend access.
- Each replica estimates its clock offset from the permission store's time. If the offset is too large, or the replica hasn't synced recently, high-risk checks are refused.
- Expiry is evaluated slightly early, with a safety margin, so skew can only shorten access. `[OPEN]` The value of the margin.

## 6. Non-Goals

Countersign is not, and in v1 will not become:
- **A central permission server.** Countersign is a library inside each product, not a service called over the network.
- **An identity system.** Keycloak and Kubernetes say who is calling.
- **A screen or approval interface.** Quornex and Houston provide the approval screens.
- **A secret store.** OpenBao stores secrets.
- **A translator into Keycloak or Kubernetes RBAC.** Both are used for identity only.
- **A policy language for customers.** Customers write no policy code.
- **An undo system.** Countersign doesn't undo work an agent already started.
- **A big policy platform nobody can reason about.** No feature goes in if it breaks "access only shrinks, ends on its own, and every answer can be explained".

## 7. MVP Scope

### 7.1 In Scope

- Features F1 to F8 (§4) and cross-cutting requirements NFR-1 to NFR-6 (§5).
- Enforcement in OpenBao only.
- **A dependency, not something Countersign delivers:** Quornex checks Countersign before any merge or sync into production. It is Quornex's side to build, and it is the gate FR-13 relies on. `[OPEN, to check with Danny]`, see FR-13 and §9 question 6b.

### 7.2 Out of Scope for v1

- **Argo CD, Harbor and Grafana.** Postponed, with the rule specified now: never more access than the grant. Until Argo CD is covered, agents get no Argo CD access through Countersign, and production syncs stay with people. Countersign doesn't govern them yet, and that gap is stated, not hidden. `[OPEN, to check with Danny]` Are they still in the plan?
- **Multi-installation features.** Each installation has its own permission store, and nothing is shared across installations.
- **Group grants.** A team is expanded into individual grants.
- **Deny grants, free-form conditions, and time-of-day or IP rules.**
- **A task-budget system for splitting.** FR-13 covers splitting as a principle.
- **Partial list results.** A list check is all or nothing (FR-22).
- **Automatic removal of drift.** A person removes it (FR-28).
- **Customers configuring their own rules.** A later step in the brief's vision.
- **Keeping records for longer than one month.** `[OPEN]` How long records are kept, and what happens when retention ends (FR-25).

## 8. Success Metrics

_All numbers are targets, not promises. A value marked `[tentative]` is Skander's own, and is to be confirmed in the spike._

**Primary**
- **SM-1: Every grant ends.** 100% of grants, including service grants, have a hard maximum lifetime. Validates FR-3 and FR-16.
- **SM-2: Access ends with the task.** Checks are refused within 10 seconds `[tentative]` of the permission store accepting the "task closed" signal. Validates FR-20.
- **SM-3: Withdrawal cascades.** 100% of descendants are refused within 10 seconds `[tentative]` of the withdrawal. Validates FR-15 and FR-19.
- **SM-4: OpenBao access is removed.** OpenBao access is removed within 10 seconds `[tentative]` of the withdrawal, measured separately from SM-3. Validates FR-26.
- **SM-5: Speed.** Each check takes at most 250 µs at p99 `[tentative]`, against one installation's store of 50,000 grants (the fleet is 10,000 installations). Memory per check is measured. Validates NFR-3 and FR-21.
- **SM-6: Explainability.** 100% of decisions can be replayed for at least one month. Validates FR-23, FR-24 and FR-29.
- **SM-7: No bypass.** Drift checks find no access in OpenBao for agent identities that Countersign didn't decide. Validates FR-28.
- **SM-8: Decision table.** All 25 checks, 5 of them attacks, give their expected answer, with 0 wrong allows. Each row cites the FRs it tests.
- **SM-9b: Rule approval is fast.** A contained request is approved within 2 seconds `[tentative]`. Validates FR-10.

**Secondary (goals to measure, not targets)**
- **SM-9: Rules approve most requests.** About 90% of requests are approved by rules alone. This depends on how many actions are marked contained. Validates FR-10.
- **SM-10: Approvers aren't overwhelmed.** Fewer than 20 requests per approver per day. Validates FR-11.

**Counter-metrics (don't optimise these; they keep the primary metrics honest)**
- **SM-C1:** the number of contained markings, watched alongside SM-8's 0 wrong allows. Counterbalances SM-9, which could be gamed by marking too much as contained.
- **SM-C2:** the time each approver takes to decide, plus the number of people holding approval authority. Counterbalances SM-10, which could be gamed by bulk "yes" clicks.
- **SM-C3:** no check against a withdrawn grant returns "allow" after the withdrawal target. Counterbalances SM-5, which could be gamed by over-caching.
- **SM-C4:** the slowest replica is measured too. Counterbalances SM-3, which could be gamed by measuring only fresh replicas.
- **SM-C5:** the median grant duration. Counterbalances SM-1, which could be gamed by absurdly long limits.
- **SM-C6:** a sampled replay must reproduce the explanation, not just the decision. Counterbalances SM-6, which could be gamed by thin records.
- **SM-C7:** access added to OpenBao by hand must be detected. Counterbalances SM-7, which could be gamed by not looking.

## 9. Open Questions

**To check with Danny**
1. Reading "anything requiring a human still goes to a human" as "a person decides once, when approving a grant and its sharing limit" (F3). Is that right?
2. ~~For contained actions, is holding the access enough to hand it down?~~ Answered 2026-11-11: no. Handing down always needs explicit authorisation.
3. ~~Does Keycloak have an admin role Countersign should trust?~~ Answered 2026-11-11: no. Administrators are established in Countersign at installation.
4. Sam starts `api-refactor`, it asks to deploy to production, and Sam approves. Is that self-approval? (FR-12)
5. Service grants: Danny said they "last on their own". Is a hard maximum lifetime, with a review date before it and renewal by a new approval, acceptable? (FR-3, FR-16)
5b. Has Orvex had an incident with an agent's access?
5c. Are approvers overloaded today? SM-9 and SM-10 depend on today's baseline.
6. Argo CD, Harbor and Grafana: are they still in the plan? (§7.2)
6c. For contained grants, is a standing sharing limit, set in advance by a person with authority over the resource and separate from the marking, the right way to authorise handing down? (FR-8)
6b. Does every merge or sync into production pass through a Countersign check, so it can act as the final-action gate? See FR-13.

**For the architecture or the spike**
7. ~~The value of the hard maximum lifetime~~ Set for task grants: under 24 hours, with an 8-hour default `[tentative]`. Still open: the service-grant hard maximum.
8. The values of both staleness horizons (NFR-2).
9. The time budget for a list check (NFR-3).
9b. The expiry safety margin, the maximum clock offset, and how recent a sync must be (NFR-6).
10. How long records are kept, and what happens when retention ends (FR-25).
11. How closing signals are authenticated (FR-20).
12. How the first administrator is set at installation time (FR-9).
12b. The last-resort recovery path if every administrator has lapsed (FR-9).

**Still undecided**
13. The role of customer admins (§2.1).
14. Who takes over a service grant if its owning team or role disappears (FR-17).
15. Not yet proved: a parent agent approving its child agent's request never gives either of them more than the parent agent could do itself.

## 10. Assumptions and Proposals Index

- **FR-2 `[ASSUMPTION]`:** a grant records when it was issued, which replay needs.
- **FR-12 `[ASSUMPTION]`:** a person's identity and an agent running as that person are different principals. Only the exact requester is excluded.
- **FR-16 `[PROPOSAL]`:** a service grant gets a review date before its hard maximum lifetime.
- **FR-18 `[PROPOSAL]`:** delegation depth (default 0, system maximum 3 `[tentative]`), from Skander's own schema work.
- **NFR-5:** the classification of the fixed rules is Skander's proposal.
- **FR-13 and §7.1 `[OPEN, to check with Danny]`:** the production gate, Quornex routing every production merge or sync through a Countersign check. The decision table records it as an assumption. See FR-13 and §9 question 6b.
- **Floor of four as settled:** it comes from Orvex requirement 03 ("production, real secrets, money and deletion wait for a person"). That its actions are *always* human-only, and that no marking removes an action from it, is Skander's decision. Danny hasn't confirmed it.
- **FR-8 `[ASSUMPTION]`:** for a contained grant, handing down is authorised by a standing sharing limit for the resource and action, set in advance by a person with authority over the resource, and separate from the marking.
- **Service grants have a hard maximum lifetime:** `[PROPOSAL]` Skander's choice (FR-3, FR-16).
- **Tentative targets:**
  - a task-grant hard maximum under 24 hours, with an 8-hour default (FR-3)
  - a 4-hour maximum for contained grants (FR-3, FR-10)
  - 250 µs p99 per check (NFR-3, SM-5)
  - 2-second rule approval (FR-10, SM-9b)
  - the 10-second withdrawal target (UJ-3, FR-19, FR-20, FR-26, NFR-2's first horizon, SM-2, SM-3, SM-4)
  - more than 20 closing signals per minute counts as a flood (FR-20)
  - an alert 14 days before an administrator lapses (FR-9)
  - 90 days of approval authority (FR-7)
  - a delegation depth of 3 (FR-18)
