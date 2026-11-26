---
title: "Input reconciliation: Orvex brief PDF vs Countersign PRD"
created: 2026-11-11
input: Orvex-AI-Intern-Brief-Countersign-Permissions-Engine.pdf (2 pages)
compared: prd.md, addendum.md (prd-countersign-2026-11-06)
---

# Input reconciliation: Orvex brief PDF vs PRD

Intentional client changes after the PDF (Danny's emails), not treated as gaps:
- Kubernetes RBAC is identity-only, not translated (PRD §6 Non-Goals, NFR-4).
- "Agents never approve" relaxed to parent approval via delegation within explicit sharing authority; nobody approves their own request; human-only actions go to a person (F3, FR-8, FR-11, FR-12, FR-14).
- Grants live in a permission store, not Git files (Glossary "Permission store", FR-25).
- No task-budget system (§7.2).

**Overall verdict:** strong coverage. All 8 requirements and all 4 ground rules are covered at requirement level, with the client changes applied. Of the 8 questions, 5 are answered at requirement level, and 3 (Model choice, Engine, Translation mechanics) are properly left to the architecture. There are a few real gaps, listed in the last section.

## 1. The 8 requirements ("What it must do")

| # | Brief requirement | Coverage | Where | Notes |
|---|---|---|---|---|
| 01 | Start from nothing: no rights until a grant; unknown or failing means refuse | Covered | NFR-1, FR-2, FR-21, FR-1, FR-5, NFR-4 (unauthenticated identity refused), NFR-5 row 1 | Complete. |
| 02 | Grants that end: record who/what/where/why/approver; end on time, task or agent | Covered | Glossary "Grant", FR-2, FR-3, FR-20, SM-1, SM-2, SM-C5 | Adds a hard maximum lifetime and an issued-at time (FR-2 `[ASSUMPTION]`). |
| 03 | The right approver: rules approve contained requests in seconds; production, real secrets, money, deletion wait for a person | Mostly covered | FR-5, FR-6, FR-10, FR-11, Glossary "Floor of four", SM-9, SM-10 | **Gap:** "in seconds" has no target. FR-10 has no time consequence and no SM measures rule-approval latency (see Gap 2). |
| 04 | Agents never approve; nobody approves their own request, however worded | Covered (relaxed by client) | FR-1 (no approver field), FR-11 ("however the approval is worded"), FR-12, FR-13, FR-8, FR-14, FR-16 | FR-12 still has an open self-approval case (Sam approving his own agent; §9 Q4). That is open, not missing. |
| 05 | Hand on only what you hold: a subset, never for longer | Covered | FR-14, FR-15, FR-18, NFR-5 rows 8 and 9 | Overlap with every ancestor's current grant at check time is stronger than the brief asks. |
| 06 | Withdraw in one click: takes back everything handed down, within seconds | Covered | FR-19, FR-15, FR-17, UJ-3, SM-3, SM-C4 | 10 s `[tentative]`. The brief's mock shows 4 s, which is consistent. |
| 07 | Explain every answer: names grants and rules, can be replayed, refusals say who could approve | Covered | FR-21, FR-23, FR-24, FR-29, SM-6, SM-C6 | Complete. |
| 08 | Fast enough to sit everywhere: every push, message, read and tool call, well under 1 ms | Covered | NFR-3, SM-5, FR-21 | Which call types go through the check is integration work for the architecture. FR-29 (record every check, refuse if it can't be recorded) puts pressure on the latency target. That is for the architecture, not a PRD gap. |

## 2. Ground rules

| Ground rule | Coverage | Where | Notes |
|---|---|---|---|
| One model, two products: one grant model, one Go library, neither keeps its own copy of the other's rules | Mostly covered | NFR-4, §1 Vision, §2.1 | **Gap (minor):** "neither keeps its own copy" has no testable consequence. Nothing says the same check from Quornex and from Houston gives the same decision, or that neither product holds rules outside the permission store (see Gap 4). |
| Grants are data, not code; customers write no policy code in v1 | Covered (Git changed to a permission store by the client) | Glossary "Permission store", FR-25, §6 ("A policy language for customers"), §7.2 | The audit trail (FR-25) replaces Git review as the reviewable record. That follows from the client change. |
| Translate, don't replace: grants become native rules in K8s RBAC, OpenBao, Argo CD, Harbor, Grafana | Partly covered, with deliberate deferral | FR-26, FR-27, FR-28 (OpenBao), §6 (K8s identity-only, a client change), §7.2 (Argo CD, Harbor, Grafana postponed) | The Argo CD, Harbor and Grafana deferral is **not** among the client-confirmed changes. It is stated openly and still pending (§9 Q6). That is a scope risk to settle with Danny, not a hidden gap. |
| Prove the fixed rules: "agents never approve" by test or analysis; on any error, refuse | Mostly covered | NFR-5 (10-row table), NFR-1 | **Gap:** the table doesn't cover the relaxed form of "agents never approve" (see Gap 1). |

## 3. "What your documents must answer"

| Question | PRD coverage | Left to architecture | Notes |
|---|---|---|---|
| **Model.** RBAC, ABAC or relationships? Grant fields? What was left out on purpose? | Grant fields: Glossary "Grant", FR-2, FR-8 (allowed to share), FR-14 (parent grant, who handed it down), FR-18 (depth). Left out on purpose: §7.2 (group grants, deny grants, free-form conditions, time-of-day or IP rules, partial lists) | The RBAC/ABAC/ReBAC choice; the formal schema | Correct split. Grant fields are spread over several FRs with no single field list, which the architecture schema should collect. |
| **Approval.** What can rules approve? What stops a crafted request? | FR-10, FR-5, FR-6, FR-1 (no approver field, typed fields only), FR-12, FR-13 (splitting attack), NFR-5 row 2, SM-8 (5 attacks) | How the request type is enforced | Covered. |
| **Engine.** Which engine, what was rejected and why? | NFR-5 mentions the two-engine spike. SM-8 requires the 25-case decision table | All of it: shortlist two, run 25 cases, decide | Correctly left to the architecture. |
| **Translation.** One grant to a K8s Role and an OpenBao policy, and how do you test they agree? | OpenBao: FR-26 (no wider, no longer), FR-27 (unenforceable means not allowed), FR-28 (drift), NFR-5 row 10, SM-4, SM-7. K8s: dropped by the client | The mapping mechanics | **Partial:** "test that they agree" is only half specified. "No wider" is required, but nothing requires an agreement test in both directions over the decision table (see Gap 5). |
| **Delegation.** Stop widening; cascade a withdrawal | FR-14, FR-15, FR-18, FR-19, FR-17, NFR-5 rows 8 and 9, SM-3 | Cascade propagation design | Covered. §9 Q15 admits the parent-approval no-gain property is not yet proved (see Gap 1). |
| **Lists.** "What may this agent see?" How, and how fast? | FR-22 (same answer per item, all or nothing) | Algorithm | **Gap:** "how fast" has no target. NFR-3 leaves the list budget `[OPEN]`, there's no feed-size assumption, and SM-5 measures single checks only (see Gap 3). |
| **Time.** Expiry offline and under clock skew; replay a decision from last month | NFR-6, NFR-2, FR-3, FR-24, FR-29, SM-6 | Values for the margin, offset and horizons (§9 Q8, Q9b) | Covered. The open values are flagged for the spike. |
| **Scale.** 10,000 × 50,000: p99 latency and memory per check, measured | NFR-3, SM-5 | Measuring it | Covered. Memory has no target, only "measured", which matches the brief's wording. |

## 4. Deliverables and judging criteria (for context)

- Decision table, 25 checks with 5 attacks: SM-8, FR-13 attack case, FR-14 decision-table case, FR-12 "depends on Danny" case.
- Grant schema and fixed rules in the chosen engine, and architecture diagrams: architecture deliverables, correctly not in the PRD.
- AI_LOG and README with open questions: process deliverables. PRD §9 is a good source for the README's open questions.
- Judging "Security: fails closed, never widens, no self-approval" maps to NFR-1, FR-14/FR-15, FR-12 and NFR-5.

## 5. Real gaps (requirement level)

1. **The fixed-rule proof doesn't cover the relaxed "agents never approve".** NFR-5's table has no rows for: a parent hands down only within explicit "allowed to share" (FR-8); contained grants are never shared (FR-14); agents can never set markings, approval authority or sharing (FR-6, FR-7, FR-8); an agent creates service grants only within explicit authorisation and never gains anything new (FR-16). The brief says this rule must hold "by test or analysis", and §9 Q15 admits the no-gain property is unproved. *Location: NFR-5 table; §9 Q15.*
2. **No speed target for rule approval.** Requirement 03 says rules approve contained requests "in seconds". FR-10 has no time consequence, and no SM measures request-to-grant latency for contained actions (SM-5 covers checks, not approvals). *Location: FR-10; §8.*
3. **No speed target or scale assumption for list checks.** The Lists question asks "how fast". NFR-3 leaves the list-check budget `[OPEN]`, there is no assumed feed size, and SM-5 covers single checks only. *Location: NFR-3, FR-22, SM-5.*
4. **"Neither keeps its own copy of the other's rules" isn't testable.** NFR-4 says both products use one library and one grant model, but has no consequence such as "the same check from Quornex and Houston against the same store state gives the same decision" or "no product-local rules exist outside the permission store". *Location: NFR-4.*
5. **No agreement test between Countersign and OpenBao.** The Translation question asks how you test that they agree. FR-26 and FR-27 require "no wider" only. Nothing requires a differential test showing OpenBao allows exactly what Countersign allows over the decision table (narrower is fail-closed but still breaks the product). *Location: FR-26, FR-27, NFR-5 row 10.*

## 6. Minor editorial notes (not input gaps)

- FR-16 repeats bullets: "names an owner" twice and "audited" twice.
- FR-18 sits between FR-15 and FR-16, out of numeric order.
- The Argo CD, Harbor and Grafana deferral (§7.2) is the only scope cut against the PDF that isn't confirmed by the client. It is flagged in §9 Q6.
