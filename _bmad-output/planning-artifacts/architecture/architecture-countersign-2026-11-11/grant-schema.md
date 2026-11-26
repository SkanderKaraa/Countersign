---
title: "Grant schema and related objects"
created: 2026-11-17
status: reviewed by Skander, 2026-11-17. The Grant section is Skander's design (v2). The other sections were drafted by Claude from the PRD and corrected by Skander.
---

# Grant schema and related objects

These shapes are shared by every engine in the spike (cedar-go, OPA, and the own-Go reference) and by the store.

**General rules:**
- Records are **immutable**. Every change is a new event in the installation's log, and the current state is computed from the log.
- `log_position` is always set by the store, never by a caller.
- IDs are typed: a person's ID comes from Keycloak, an agent or service ID from Kubernetes. They are never free text.

## Grant (Skander's design, v2)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `id` | string | yes | Unique, never reused. |
| `principal` | identity ID | yes | A real authenticated ID (Keycloak or Kubernetes), never free text. |
| `action` | enum | yes | From a closed list. An unknown action is refused. |
| `resource` | canonical path | yes | No `..`, aliases resolved. Either an exact path or one subtree marker. |
| `kind` | enum | yes | `task` or `service` (FR-3). |
| `task` | task ID | if `kind = task` | Must refer to a real task. Empty for service grants. |
| `approver` | person ID or rule ID | yes, except on child grants | A person with approval authority, or the rule that approved it (FR-10). Never the principal. **Empty on a child grant:** `parent` and `handed_by` carry the chain, and the explanation walks up it to the first approval (FR-14). |
| `issued_at` | timestamp | yes | Set by the store, not the caller. |
| `ends_at` | timestamp | yes | After `issued_at`. Within the hard maximum for its kind, and within 4 h for a contained grant. On a child grant, no later than its parent's. |
| `parent` | grant ID | no | If set, the grant must be a subset of its parent grant (FR-14). |
| `handed_by` | identity ID | if `parent` is set | Who handed it down. |
| `depth` | small integer | human-only grants only | Default 0, which means no sharing. Always below the parent's (FR-18). A single field: there is no separate `may_share`, because two fields could contradict each other. Contained grants have no sharing fields; their sharing comes from the standing sharing limit (below). |
| `owner` | team or role | if `kind = service` | Never the creating agent (FR-16). |
| `created_by` | identity ID | if `kind = service` | Who created the service grant, so the owner can be alerted and the creation audited (FR-16). Never the owner. |
| `review_date` | timestamp | if `kind = service` | Before the service-grant hard maximum. `[PROPOSAL, to check with Danny]` |
| `log_position` | integer | yes | The event that created the grant. Set by the store. |

**No `withdrawn` or `status` field.** Withdrawal, narrowing and expiry are events or computed facts. The grant record never changes.

## Request (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `id` | string | yes | Unique. |
| `principal` | identity ID | yes | Authenticated. It is the requester. |
| `action` | enum | yes | Closed list. |
| `resource` | canonical path | yes | As for Grant. |
| `task` | task ID | yes | A real task. |
| `duration` | duration | yes | No longer than the hard maximum, or 4 h for a contained action. A longer request is refused, not shortened (FR-1, FR-10). |

**By construction, there is no approver field** (FR-1, NFR-5 rule 2). Any extra field is refused: the request is parsed strictly.

## Approval (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `request` | request ID | yes | An open request. |
| `approver` | person ID | yes | A person type, which agents can't produce (NFR-5 rule 5). Holds approval authority for that action and resource. Not the requester (FR-12). |
| `log_position` | integer | yes | Set by the store. |

## Marking (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `resource` | canonical path | yes | Exact or subtree. |
| `action` | enum | yes | Closed list. |
| `class` | enum | yes | `contained` or `human-only`. A floor-of-four action can never be marked `contained` (FR-6). |
| `set_by` | person ID | yes | A person with authority over the resource. Never an agent. |
| `log_position` | integer | yes | Set by the store. |

Unmarked, or unreadable, means human-only (FR-5).

## Approval authority (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `holder` | person ID | yes | Never an agent (FR-7). |
| `actions` | list of enum | yes | Closed list. |
| `resource` | canonical path | yes | No wider than the giver's own authority. |
| `given_by` | person ID | yes | An administrator, or a person with authority over the resource. |
| `ends_at` | timestamp | yes | Within the hard maximum for approval authority (target: 90 days `[tentative]`) (FR-7). |
| `log_position` | integer | yes | Set by the store. |

## Standing sharing limit, for contained grants (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `resource` | canonical path | yes | |
| `action` | enum | yes | Must be marked contained. |
| `depth` | small integer | yes | The maximum hand-down depth, within the system maximum of 3 `[tentative]` (FR-18). A hand-down from a contained grant for this resource and action may go no deeper than this limit. |
| `set_by` | person ID | yes | A person with authority over the resource. Never an agent. |
| `log_position` | integer | yes | Set by the store. |

`[ASSUMPTION]` See FR-8; to check with Danny.

## Service-grant authorisation, for an agent (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `agent` | agent ID | yes | |
| `actions` | list of enum | yes | |
| `resource` | canonical path | yes | |
| `given_by` | person ID | yes | A person with authority over the resource. |
| `ends_at` | timestamp | yes | Has a hard maximum, like approval authority (FR-16). |

A service grant created by the agent can be no wider than this authorisation, and no wider than what the agent holds at that moment (FR-16).
| `log_position` | integer | yes | Set by the store. |

## Administrator (drafted by Claude; review)

| Field | Type | Required | Validation rule |
|---|---|---|---|
| `person` | person ID | yes | |
| `added_by` | person ID or `installation` | yes | Another administrator. `installation` only for the first administrators set at install (FR-9). |
| `ends_at` | timestamp | yes | No two administrators lapse within 14 days `[tentative]` of each other (FR-9). |

An installation never has fewer than two administrators. This is checked when an administrator is removed (FR-9). The very first administrators are set once at install, outside the normal flow, and audited. How that is done is still open (PRD §9, question 12).
| `log_position` | integer | yes | Set by the store. |

## Events in the log (drafted by Claude; review)

The log holds one event type per change. Each carries its `log_position`, the store's time, and the identity that caused it:
- `grant_created` (an approval, or rules)
- `handed_down`
- `service_grant_created`
- `narrowed`: the grant ID with the **full effective resource and `ends_at` after the narrowing**, computed by the store, never a partial change (decision D3, 2026-11-17). Narrowing **is an event**. The grant record stays immutable, and the effective scope and end are those of the latest narrowing, or the grant's own.
- `withdrawn`
- `signal_accepted`: `task_closed` or `agent_ended`, with its source
- `marking_set` and `marking_removed`
- `authority_given`
- `sharing_limit_set`
- `service_authorisation_given`
- `administrator_added` and `administrator_removed`
- `policy_written` and `policy_removed`: the OpenBao reconciliation, for the audit
- `drift_detected`: hand-added access found in OpenBao, so the alert is in the audit (FR-28)
