---
title: "Research digest: OpenBao, for Countersign enforcement"
created: 2026-11-17
source: Claude research subagent; openbao.org 2.7.x docs and GitHub releases, checked 2026-11-17
status: verified against primary sources (the one inference is marked)
---

# Research digest: OpenBao

- **Latest release:** v2.7.2, 2026-10-26 (https://github.com/openbao/openbao/releases/tag/v2.7.2). v2.6.5 was patched the same day.
- **Policies on identities are evaluated at request time.**
  - Policies attached to an identity **entity or group** are evaluated on every request. Changing them "control[s] the behavior of already issued tokens" (https://openbao.org/docs/concepts/identity/).
  - Policies on the **token itself** are fixed at issue: "There is no way to modify the policies associated with a token once the token has been issued" (https://openbao.org/docs/concepts/policies/).
  - **So:** Countersign's grants must live on entities or groups, never in a role's `token_policies`.
  - Removing a policy should take effect on the next request (an inference; the docs don't say it outright).
  - An external group's membership syncs only at login or renewal.
- **Kubernetes auth:** a pod logs in with `role` and its service-account JWT.
  - The alias defaults to the service account's **UID** (`alias_name_source=serviceaccount_uid`). The `serviceaccount_name` option carries a re-creation risk.
  - Lifetime controls: `token_ttl`, `token_max_ttl`, and `token_explicit_max_ttl` (a hard cap).
  - Sources: https://openbao.org/docs/api/auth/kubernetes/ and https://openbao.org/docs/auth/kubernetes/
- **Built-in expiry:**
  - Each policy path can carry an `expiration` (RFC3339) that "automatically remove[s] access at a point in time".
  - A whole policy can carry an `expiration` or `ttl`.
  - Source: https://openbao.org/docs/api/system/policies/
- **Not available:**
  - No recurring time windows, and no use counts per path. `token_num_uses` counts per token only.
  - CEL applies only to PKI and JWT role validation.
  - Sentinel is (very probably) absent.
- **Listing, for comparison and drift:**
  - Policies: `LIST /sys/policies/acl[/:prefix]` and `/sys/policies/detailed/acl`.
  - Entities and groups: list by ID, then read each one's `policies` and `metadata`.
  - Policies have no metadata field, so mark them with a name prefix (`countersign-…`). Entities can carry metadata.
- **Limits:**
  - One policy can be up to 1 MiB, and the number of policies is unlimited.
  - About 28,000 policies per entity.
  - Large numbers of policies raise response time (`vault.core.fetch_acl_and_token`), so avoid thousands of policies on one entity.
  - Source: https://openbao.org/docs/internals/limits/
