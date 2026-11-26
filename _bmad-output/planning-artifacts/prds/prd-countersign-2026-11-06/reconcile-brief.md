---
title: "Input reconciliation: brief vs PRD"
created: 2026-11-11
source: briefs/brief-countersign-2026-10-30/brief.md (final)
target: prd.md + addendum.md (prd-countersign-2026-11-06)
---

# Input reconciliation: product brief → PRD

The PRD covers nearly all of the brief's requirements, scope items and non-goals, and it keeps most of them word for word. The gaps below are mostly qualitative material that an FR structure tends to drop, plus one contradiction with a brief principle.

## 1. Real gaps

### G1. NFR-2 lets contained checks continue on a stale replica, which conflicts with "a wrong allow is not acceptable" (contradiction)
- **Brief:** Who This Serves ("A fast refusal is acceptable; a wrong allow is not"). Scope, In v1 ("Default deny: anything unknown or failing is a refusal"). Success Criteria ("Withdrawal cascades ... checks refuse them within 10 seconds").
- **PRD:** NFR-2 ("After the staleness horizon, high-risk checks are refused. Checks on contained actions may continue, because their damage is bounded"). NFR-6 refuses only *high-risk* checks when the clock offset is too large.
- **Issue:** a replica that can't confirm it is fresh might allow a contained grant that has already been withdrawn. That is a wrong allow, chosen for availability. It also conflicts with the PRD's own SM-C3 ("a withdrawn grant is never allowed after the withdrawal target") and with FR-19's 10 s cascade for contained grants. This may be a sound trade-off, but the brief puts security first on permissions, so the PRD should state it as a deliberate exception to NFR-1 and reconcile it with SM-C3. Right now it isn't stated.
- **Related, smaller:** FR-3 ("if the task source can't be reached later, the grant is not treated as ended") is also a "keep allowing when failing" choice. It is reasoned and bounded by the time limit, but it doesn't refer back to NFR-1.

### G2. The precedence rule between users was dropped
- **Brief:** Who This Serves ("security comes first for permissions and rules. Engineers come first for speed and ease of integration").
- **PRD:** §2 Target Users and §1 Vision keep only "A fast refusal is acceptable; a wrong allow is not". The rule for deciding conflicts between the security team and engineers is gone, and it is the rule the architecture needs for trade-offs such as G1.

### G3. Failure modes, and the "ease of integration / bypass" concern, have no requirement or metric
- **Brief:** Success Criteria ("This fails, even if the code runs, if people bypass Countersign because it is slow or annoying, approvers click 'yes' without reading ..., or a native tool such as OpenBao does the opposite ... and nobody notices"). Who This Serves ("If a check is slow or hard to integrate, they are the first to complain").
- **PRD:** §8 covers rubber-stamping (SM-C2) and OpenBao drift (SM-7, SM-C7), but only for *agent identities*. Nothing covers "people bypass Countersign because it is slow or annoying", and no NFR or metric covers ease of integration (NFR-4 only says "one Go library, embedded in both"). The brief's "this fails if" framing is dropped as a whole.

### G4. Two open questions for Danny were silently dropped
- **Brief:** Open Questions, To check with Danny ("Has Orvex had an incident with an agent's access?" and "Are approvers overloaded today?").
- **PRD:** §9 Open Questions keeps the service-grant and Argo CD questions and adds new ones, but not these two. They matter because SM-9 and SM-10 (90% by rules, fewer than 20 per approver per day) depend on today's baseline.

### G5. The problem statement, differentiation and growth path are missing
- **Brief:** The Problem (two rule sets drift; forgotten access; agents act in seconds and their access outlives the task). What Makes This Different (why not sync, why not buy Opal or Teleport, why not use OpenFGA or Cedar directly; "What we don't claim"). Executive Summary / alternatives ("built in Go, as embeddable open source"). Vision steps 1–3 (remaining native tools, typed customer configuration, use beyond Quornex and Houston).
- **PRD:** §1 Vision describes the product but gives no *why*. There is no alternatives or "what we don't claim" section. "Open source" is not mentioned anywhere (NFR-4 says only "Go library"). The candidate engines (OpenFGA, Cedar) aren't named, although NFR-5 refers to the two-engine spike. Of the vision's growth path, only "customers configuring their own rules" survives, as a later step in §7.2. "Use beyond Quornex and Houston" is dropped. The open-source licensing constraint is the most concrete loss here, because it limits the choice of embedded engine.

## 2. Minor or editorial (lower priority)

- **Nature of the hand-in.** Brief, Executive Summary: "This hand-in is its design: a PRD, an architecture, a grant schema, a decision table and a two-engine spike. It is not a working product." The PRD doesn't say this. The decision table (SM-8) and the spike appear only in passing.
- **"Not Git files".** Brief, Where permissions live: the permission store is deliberately not Git, so an agent gets access in seconds without a pull request, while infrastructure changes still go through Forgejo as reviewed commits. The PRD Glossary defines the permission store but drops the "not Git, and why" rationale. Forgejo appears only as the FR-13 gate.
- **"It does not replace OpenBao".** Brief, What Countersign does not do. PRD §6 Non-Goals lists "a secret store" but not "doesn't replace OpenBao". The meaning survives through F8.
- **UJ-2 drops the brief's step 5 cascade** ("Anything handed down from it ends too") and its step 6 (the security question). Both are covered elsewhere, by UJ-3 and UJ-4.
- **Self-approval is narrowed.** Brief: "Nobody approves their own request". PRD FR-12 `[ASSUMPTION]`: only the exact requester is excluded, and a person approving an agent that runs as them is open (Q4). This is visible and flagged, not silent, but it weakens the brief's wording if the assumption holds.
- **Approval-authority lifetime (90 days) and the administrator rules (FR-7, FR-9)** are new in the PRD and not in the brief. They are additions, not gaps.

## 3. Coverage confirmed (no action)

Default deny / fail closed (NFR-1). Hard maximum lifetime for every grant, including service grants (FR-3, SM-1). Access ends with the task within 10 s (FR-20, SM-2). Cascading withdrawal (FR-15, FR-19, SM-3). Floor of four, and markings only add to it (FR-5, FR-6). Only humans mark contained (FR-6). Nobody approves their own request (FR-12). Approval attached to the final action (FR-13). Service grants owned by a team or role, not cascading, withdrawable with their children, creation audited (FR-16, FR-17). Explanations, with who could approve (FR-21, FR-23). Replay for one month (FR-24, SM-6). Under 1 ms p99 (NFR-3, SM-5). List check (FR-22). OpenBao enforcement, "unenforceable means not allowed", and drift detection (FR-26 to FR-28, SM-7). Approver metrics as goals to measure (SM-9, SM-10). All Out-of-v1 items (§6, §7.2). Argo CD, Harbor and Grafana postponed with the rule stated and the gap stated (§7.2). Non-goal "big policy platform" (§6). Brief open questions on service-grant review date, Argo CD plan, "not yet proved", orphaned service grants, customer admins and the maximum-lifetime value (§9). "Allowed to share" answered.

## 4. Intentional supersessions (all visible, so none needs action)

- **Agents may create service grants with explicit authorisation.** Brief: an authorised person, or someone holding that permission. PRD: FR-16, Glossary, addendum (Danny, 2026-11-11). Visible.
- **Parent approval reframed as delegation; contained grants can never be shared.** Brief: who may approve. PRD: F3 intro, FR-8, FR-14, Glossary "Allowed to share" (Danny, 2026-11-11), with the F3 reading flagged `[OPEN]` (Q1). Visible.
- **Holding access never implies "allowed to share".** Brief open question. PRD: Q2 marked answered. Visible.
- **Keycloak admin role not trusted; administrators established in Countersign.** New, from FR-9 and Q3. Visible.
- **Keycloak/Kubernetes RBAC not translated, and no task budget.** These are the same in the brief and the PRD (§6, §7.2). No conflict.
