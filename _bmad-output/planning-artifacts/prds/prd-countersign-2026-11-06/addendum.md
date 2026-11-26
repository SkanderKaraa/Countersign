---
title: "Addendum: Countersign PRD"
created: 2026-11-11
updated: 2026-11-11
---

# Addendum

Danny's (the client's) 2026-11-11 email, paraphrased, and decision-table seeds for the architecture.

## Danny's answer (email, 2026-11-11)

This is paraphrased, not quoted. Ask Danny before publishing the email itself.

The email answers PRD §9 questions 2 and 3. It is the source the PRD cites as "(Danny, 2026-11-11)".

- **Identity:** Keycloak is typically connected to an external identity provider, such as Google or Microsoft SSO. Keycloak says who someone is; Countersign decides what they may do.
- **Administrators:**
  - The first trusted administrators are established during installation, and they grant authority to others.
  - Having several administrators makes sense.
  - Keycloak's admin role should not be trusted automatically.
- **Sharing:**
  - Having access to something doesn't mean you can share it.
  - A parent agent can pass on only the permissions it has been explicitly authorised to share.
  - "Contained" means only that an action can be approved automatically within the existing rules. It gives no extra authority to share.
- **Service grants:** Authorised users **or agents** must be able to create independent permissions for services that need to outlive their creator.

## Decision-table seeds (for the architecture's 25-case table)

These cases appear in the FRs. Each one should become a row in the decision table (PRD SM-8), citing its FRs. They cover 8 of the 25 checks (in the decision-table sense: one test row each); Skander writes the remaining checks in the architecture. A ✓ in the Attack column marks a check that could be one of the 5 attacks in PRD SM-8.

| Seed | Case | Attack | Expected | FRs |
|---|---|---|---|---|
| 1 | A request names its own approver | ✓ | refuse | FR-1 |
| 2 | Sam starts `api-refactor`, which asks to deploy to production, and Sam approves the request | | open: PRD §9 question 4 (is it self-approval?) | FR-12 |
| 3 | `helper` writes ten contained config files, and the result is merged or synced into production | ✓ | refuse the check; the request returns "ask" (people only) | FR-13, §7.1 |
| 4 | A parent agent tries to hand down a contained grant without the "allowed to share" permission | ✓ | refuse; allow with a person's explicit authorisation | FR-8, FR-14 |
| 5 | A request for a child grant that is wider than, or lasts longer than, its parent grant | ✓ | refuse | FR-14 |
| 6 | A parent grant with three child grants and one grandchild grant is withdrawn | | refuse checks against all five grants within 10 s `[tentative]` | FR-15, FR-19 |
| 7 | A check is replayed from before a withdrawal, and another from after it | | allow, then refuse, citing the withdrawal | FR-24 |
| 8 | Access for an agent identity is added to OpenBao by hand | ✓ | detect and report | FR-28 |

**Note on seed 3:** it assumes the production gate exists; whether it does is `[OPEN, to check with Danny]` (PRD FR-13 and §9 question 6b).
