---
title: "Review: versions and technology reality check of ARCHITECTURE-SPINE.md"
reviewed: ARCHITECTURE-SPINE.md (updated 2026-11-17)
date: 2026-11-17
reviewer: independent review subagent
scope: Stack table versions; technical claims in AD-6, AD-8, AD-13
method: Go module proxy (@latest, @v/list, .mod), go.dev/dl JSON and release history, crates.io API, GitHub release redirects, OpenBao v2.7.2 source tarball (codeload.github.com), openbao.org docs, local module and cargo caches, spike go.mod / Cargo.toml / Cargo.lock
---

# Review: versions and reality check

**Verdict:** The stack is very nearly current, and the OpenBao and SymCC claims hold up against source and docs. One toolchain pin is a security patch behind. The rest are wording gaps and fit risks, not wrong technology.

## Findings

### V1 — high — Go 1.27.3
- **Finding:** Go 1.27.4 came out on 2026-11-10, a week before the spine was updated, and it includes **security fixes** to cgo, the go command, the compiler, the linker and net/http. The pin matches the local toolchain (`go version go1.27.3`), which suggests it was read from the machine and not checked on the web. Shipping a security product built with a toolchain that has known security bugs (net/http is on the store's gRPC/HTTP path) is avoidable.
- **Evidence:** https://go.dev/dl/?mode=json (first entry `go1.27.4`); https://go.dev/doc/devel/release ("go1.27.4 (released 2026-11-10) includes security fixes to cgo, the go command, the compiler, the linker, and the net/http package"); /home/skander/countersign/spike/go.mod:3 (`go 1.27`, no `toolchain` line).
- **Suggested fix:** Pin Go 1.27.4 (or say "latest 1.27.x patch, minimum 1.27.4"), and add a `toolchain go1.27.4` line to the product go.mod.

### V2 — medium — AD-13: one OpenBao policy per grant vs the AD-9 worst case
- **Finding:** AD-9 plans for 5,000 grants for one principal. With one policy per grant attached to the agent's entity, that means about 5,000 policies on one entity. OpenBao's limits page warns that many policies on one entity raise per-request ACL build time (`vault.core.fetch_acl_and_token`), and the research digest itself says "avoid thousands of policies on one entity". The spine doesn't reconcile the two.
- **Evidence:** research-openbao.md:33-37; https://openbao.org/docs/internals/limits/; ARCHITECTURE-SPINE.md:134 and :180. In source, entity and group policies are gathered and the ACL is built on every request (openbao v2.7.2 `internal/vault/request_handling.go:142-190, 331-350`).
- **Suggested fix:** State the expected OpenBao policy count per entity. Either cap it, or merge an agent's grants into a few policies whose paths each carry their own per-path `expiration` (supported: `internal/vault/policy/policy.go:165`). Mark it as needing measurement before v1.

### V3 — medium — AD-13: policy `expiration` uses OpenBao's own clock
- **Finding:** Confirmed: policy `expiration` exists, and OpenBao checks it at fetch time against `time.Now()` (cached entries too). So it's enforced on time, not only "on next load". But the clock is OpenBao's own wall clock, not AD-4's "later of two clocks". A slow OpenBao clock lengthens access past the grant's end, and nothing in the spine bounds that. Also, the API refuses an `expiration` in the past, and OpenBao deletes the policy once it expires, so the reconciler must treat "ended grant, policy missing" as in line with the log, and must never try to write it again.
- **Evidence:** https://openbao.org/docs/api/system/policies/ ("will no longer be valid and will be removed on next load. Cannot be set in conjunction with ttl and cannot be in the past"); openbao v2.7.2 `internal/vault/policy/policy_store.go:390-475`.
- **Suggested fix:** In AD-13, say that OpenBao's `expiration` is a backstop on OpenBao's clock, and that the reconciler's withdrawal at the grant's end (by the store's clock) is the main mechanism. Add the "expired policy is absent, not drift" rule, and a clock-skew requirement for OpenBao hosts.

### V4 — medium — AD-8 / AD-6: the proofs run on Rust Cedar, production runs cedar-go
- **Finding:** SymCC proves properties of `policies.cedar` under the Rust `cedar-policy` 4.15.0 semantics, but production decisions come from cedar-go v1.9.2. The spine's differential test compares cedar-go, OPA and the Go reference, not cedar-go and Rust Cedar. So the proofs carry over to production only on trust in cedar-go's conformance. cedar-go does ship the Rust project's integration corpus (`corpus-tests*.tar.gz`), which is supporting evidence but isn't run against these policies.
- **Evidence:** ARCHITECTURE-SPINE.md:116, :219, :253; ~/go/pkg/mod/github.com/cedar-policy/cedar-go@v1.9.2/README.md:31-47 and its corpus files; /home/skander/countersign/spike/symcc/Cargo.lock:142-209.
- **Suggested fix:** State the assumption in AD-8. Ideally, add a CI check that replays the SymCC counterexamples and a sample of requests through both Rust `Authorizer` and cedar-go and requires identical decisions.

### V5 — low — AD-6: "cedar-go has no list method"
- **Finding:** That's imprecise. cedar-go v1.9.2's core lacks partial evaluation (README: "does not yet include … partial evaluation"), but it ships an experimental batch authorization API (`x/exp/batch`) built on a form of partial evaluation. The pre-registration says correctly that it exists and isn't used, but the spine reads as if it doesn't exist.
- **Evidence:** ~/go/pkg/mod/github.com/cedar-policy/cedar-go@v1.9.2/README.md:46, :127; x/exp/batch/batch.go package doc; spike-preregistration.md:76; ARCHITECTURE-SPINE.md:104.
- **Suggested fix:** Change it to: "cedar-go's only list facility is the experimental `x/exp/batch` (outside semver), so it isn't used. The engine decides each item by one check."

### V6 — low — Stack: pgx v5.13.0, gRPC-Go v1.85.0 and OpenBao v2.7.2 are current but unexercised and uncited
- **Finding:** All three are the current latest (checked below), but none appears in any go.mod and no research digest covers pgx or gRPC. So the versions are right today but rest on nothing in the project. Also, the store will import the OpenBao Go client `github.com/openbao/openbao/api/v2`, whose latest is **v2.7.1**, versioned separately from the server's v2.7.2. The Stack table doesn't list it.
- **Evidence:** https://proxy.golang.org/github.com/jackc/pgx/v5/@latest (v5.13.0, 2026-11-11); https://proxy.golang.org/google.golang.org/grpc/@latest (v1.85.0, 2026-10-27); https://proxy.golang.org/github.com/openbao/openbao/api/v2/@latest (v2.7.1, 2026-10-01); /home/skander/countersign/spike/go.mod (no pgx, gRPC or openbao entries).
- **Suggested fix:** Add a "source / checked on" column to the Stack table, and add a row for the OpenBao Go client (api/v2 v2.7.1).

### V7 — low — PostgreSQL unpinned
- **Finding:** Leaving it unpinned is reasonable, but pgx v5.13.0 supports only PostgreSQL 14 and higher. "Verify per installation" gives no threshold to verify against.
- **Evidence:** pgx v5.13.0 README, "Supported Go and PostgreSQL Versions" ("PostgreSQL 14 and higher").
- **Suggested fix:** Write "minimum PostgreSQL 14 (pgx v5.13 support floor); verify per installation".

### V8 — low — cvc5 1.3.1 is not the latest cvc5
- **Finding:** cvc5 1.4.2 is the latest release (2026-10-08). 1.3.1 is still the **right** pin, because cedar-policy-symcc 0.9.0's README prescribes exactly cvc5 1.3.1. But the spine doesn't say why, so someone might "upgrade" it.
- **Evidence:** https://github.com/cvc5/cvc5/releases/latest → cvc5-1.4.2; ~/.cargo/registry/src/index.crates.io-*/cedar-policy-symcc-0.9.0/README.md:23-26.
- **Suggested fix:** Annotate the row: "1.3.1 (the version cedar-policy-symcc 0.9.0 requires; don't bump separately)".

### V9 — low — cedar-policy-symcc 0.9.0 is 0.x and one day old
- **Finding:** 0.9.0 and cedar-policy 4.15.0 were both published 2026-11-16 (current latest, confirmed). SymCC is pre-1.0 and has had a minor release roughly monthly (0.6.0 Jul, 0.7.0 Sep, 0.8.0 Oct, 0.8.1 and 0.9.0 Nov), so its API may break between minors. Cargo.toml correctly pins with `=`.
- **Evidence:** https://crates.io/api/v1/crates/cedar-policy-symcc; /home/skander/countersign/spike/symcc/Cargo.toml:8-9.
- **Suggested fix:** Keep the exact pins, and add to AD-8 that upgrades of SymCC, cedar-policy and cvc5 move together and re-run the full suite. Optionally also add SymCC's `check_never_errors` property, which backs AD-7's "a policy error refuses".

### V10 — low — research-engines.md is still marked unverified, with older versions
- **Finding:** The digest lists cedar-go v1.9.1 and OPA v1.22.3 with `status: unverified`. The spine pins v1.9.2 and v1.22.4, which match the spike's go.mod and are the current latest. The spine is right, but its companion is stale.
- **Evidence:** research-engines.md:5, :16-17; /home/skander/countersign/spike/go.mod:6-7.
- **Suggested fix:** Add a note to research-engines.md that the versions were superseded by the spike's go.mod (checked 2026-11-17).

## Confirmed correct
- **cedar-go v1.9.2** is the latest (proxy @latest, 2026-11-13) and is in spike go.mod:6. Its go.mod needs Go 1.26 or later.
- **OPA v1.22.4** is the latest (proxy @latest, 2026-11-12) and is in spike go.mod:7.
- **pgx v5.13.0** is the latest (2026-11-11) and needs Go 1.25 or later.
- **gRPC-Go v1.85.0** is the latest stable (2026-10-27; only 1.86.0-dev is newer). It needs Go 1.26 or later.
- **OpenBao v2.7.2** is the latest server release (GitHub releases/latest redirects to v2.7.2), matching research-openbao.md.
- **cedar-policy-symcc 0.9.0 and cedar-policy 4.15.0** are the latest on crates.io (2026-11-16), not yanked, and locked in Cargo.lock:142 and :206.
- **cvc5 1.3.1** is the version cedar-policy-symcc 0.9.0 requires.
- **AD-13, `expiration`:** it exists, both per policy (API) and per path (HCL `expiration`), and is enforced at fetch time (policy_store.go:390-475).
- **AD-13, entity and group policies:** they're re-read from the identity store on every request (`fetchEntityAndDerivedPolicies`, request_handling.go:142-190). So removing a policy takes effect on the next request, which upgrades the digest's "inference" to confirmed from source. One caveat: a token created with `no_identity_policies` skips them, which fails closed. Token policies are fixed at issue.
- **AD-13, listing by prefix:** `LIST /sys/policies/acl/:prefix` does a plain string-prefix match (`ListPoliciesWithPrefix` → `logical.CollectKeysWithPrefix`, `strings.HasPrefix`; sdk/logical/storage.go:202-216). So `countersign-` works. It scans every policy key, and names are lowercased (`SanitizeName`). It's still untested at runtime, as the spine says.
- **AD-6:** cedar-go has no partial evaluation in its core (README:46). OPA's `Rego.Partial` exists.
- **AD-8:** SymCC's capabilities (always-allows/denies, implies, equivalence, disjointness, never-errors, with counterexamples, over all requests and entity stores the schema allows per request environment) match the README. The spike ran it with the exact pinned versions (symcc-results.md, src/main.rs:80).
