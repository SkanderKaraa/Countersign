---
title: "Research digest: six authorization engines for an embedded Go library"
created: 2026-11-11
source: Claude research subagent, web (GitHub release pages, official docs), 2026-11-11
status: unverified, except the cedar-go and OPA versions, re-checked when pinning them (2026-11-17)
---

# Research digest: six authorization engines for an embedded Go library

This is input for choosing the two engines for the spike. It is not a decision.

## Overview

| Engine | Latest release | Embeds as a Go library | Explains a decision | Formal analysis | List filtering when embedded | Licence |
|---|---|---|---|---|---|---|
| cedar-go | v1.9.2 (pinned 2026-11-17; v1.9.1 when researched) | Yes (`github.com/cedar-policy/cedar-go`) | Policy IDs, through `Diagnostic.Reasons` | Rust only (SymCC plus the cvc5 solver), not in Go | Experimental (`x/exp/batch`); partial evaluation "not yet included" | Apache-2.0 |
| OPA | v1.22.4 (pinned 2026-11-17; v1.22.3 when researched) | Yes (`opa/v1/rego`, `/v1/sdk`) | Traces only | Tests and benchmarks only | Yes (`Rego.Partial`) | Apache-2.0 |
| OpenFGA | v1.22.1 (28 Oct 2026, year inferred) | As a whole server object (`pkg/server`) | Weak (uncertain); has Expand | None found | ListObjects | Apache-2.0 |
| SpiceDB | v1.57.1 (10 Nov 2026, year inferred) | Check only (`pkg/embedded`): no writes or lookup | Debug traces, gRPC only (unverified) | None found | Not when embedded | Apache-2.0 |
| Cerbos | v0.57.0 (23 Oct 2026) | **No**: its docs say it is "not a library that you can embed" | `includeMeta` | Policy tests | Server only | Apache-2.0 |
| Casbin (apache/casbin) | v3.11.0 (20 Aug 2026) | Yes | `EnforceEx` returns the matched rules | None | No (uncertain) | Apache-2.0 |

## Notes per engine

- **cedar-go:** supports `datetime` and `duration` types. Delegation chains have to be passed in as entity data. The schema validator is experimental. No performance numbers have been published for cedar-go; the Cedar paper (arXiv 2403.04651) measures the Rust version.
- **OPA:** Rego is very expressive, which makes it harder to analyse. The docs' sample role-based policy shows a p99 of 134 µs, on an unspecified machine. A Teleport and Doyensec post describes Rego as "error-prone".
- **OpenFGA:** relationship tuples with CEL conditions (which support timestamps and durations). Delegation fits naturally. Its weak points are explaining decisions and analysis.
- **SpiceDB:** supports expiring relationships and caveats, but the embedded form is too limited for this use.
- **Cerbos:** cannot run inside the process, which breaks the in-process requirement.
- **Casbin:** the vendor's own benchmark shows about 23.9 ms per check with 110k role-based rules, roughly 100× over the 250 µs target.

## The subagent's suggestion (not a decision)

- **cedar-go and OPA look strongest:**
  - **cedar-go:** explains its decisions and can be analysed, but only through Rust tooling, and its list filtering is experimental.
  - **OPA:** mature, with list filtering through partial evaluation, but its explanations are traces only, and it has no analysis.
- **OpenFGA is a credible third** if delegation chains matter most.
- **Nobody has published in-process numbers at 50,000 grants**, so the spike really does have to measure.

Sources: see the URLs in the subagent's report (cedar-go releases and README, pkg.go.dev, docs.rs cedar-policy-symcc, OPA docs, OpenFGA docs, SpiceDB pkg.go.dev, Cerbos docs, the Casbin benchmark page).
