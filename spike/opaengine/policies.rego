# Countersign's rules in Rego (OPA v1 syntax). They mirror policies.cedar
# rule for rule, with the same rule IDs, so the same tests apply.
#
# Each decision is one object: {"allow": bool, "rule": id}. Refusals (the
# "forbid" rules) are checked first in an else-chain, so a permit can never
# override them; with nothing matching, the answer is default-deny
# (NFR-5 rule 1). Times are Unix milliseconds.
package countersign

import rego.v1

deny(rule) := {"allow": false, "rule": rule}

allow(rule) := {"allow": true, "rule": rule}

valid_class(c) if c in {"human-only", "contained"}

# covers(scope, r): an exact path covers only itself; "<p>/**" covers p,
# everything below it, and any narrower subtree.
covers(scope, r) if scope == r

covers(scope, r) if {
	endswith(scope, "/**")
	p := trim_suffix(scope, "/**")
	rp := trim_suffix(r, "/**")
	rp == p
}

covers(scope, r) if {
	endswith(scope, "/**")
	p := trim_suffix(scope, "/**")
	rp := trim_suffix(r, "/**")
	startswith(rp, concat("", [p, "/"]))
}

# ---------------------------------------------------------------- check

live(n) if {
	not n.withdrawn
	not n.task_closed
	not n.agent_ended
	input.now_ms < n.ends_at_ms
}

# Unlike Cedar, Rego iterates, so the whole chain is checked whatever its
# length; the depth limit is a separate rule.
too_deep if count(input.chain) - 1 > input.max_depth

grant_allows if {
	first := input.chain[0]
	first.principal == input.principal
	first.action == input.action
	every n in input.chain {
		live(n)
		covers(n.scope, input.resource)
	}
}

check := deny("check.forbid-too-deep") if {
	too_deep
} else := allow("check.grant-allows") if {
	grant_allows
} else := deny("default-deny")

# ---------------------------------------------------------------- approve

authorised if input.approver_is_admin

authorised if {
	some s in input.authority_scopes
	covers(s, input.resource)
}

approve := deny("approve.forbid-agent") if {
	startswith(input.approver, "agent:")
} else := deny("approve.forbid-self") if {
	input.approver == input.requester
} else := deny("approve.forbid-unknown-class") if {
	not valid_class(input.class)
} else := allow("approve.authorised-person") if {
	startswith(input.approver, "person:")
	authorised
} else := deny("default-deny")

# ---------------------------------------------------------------- hand down

sharing_ok if {
	input.class == "human-only"
	input.parent_depth > 0
	input.child_depth < input.parent_depth
}

sharing_ok if {
	input.class == "contained"
	input.has_sharing_limit
	input.chain_len <= input.sharing_limit
}

hand_down := deny("hand_down.forbid-service-grant") if {
	input.parent_kind != "task"
} else := deny("hand_down.forbid-unknown-class") if {
	not valid_class(input.class)
} else := allow("hand_down.within-parent") if {
	input.holder == input.parent_principal
	covers(input.parent_scope, input.child_scope)
	input.child_ends_at_ms <= input.parent_ends_at_ms
	input.chain_len <= input.max_depth
	sharing_ok
} else := deny("default-deny")

# ---------------------------------------------------------------- markings

set_marking := deny("set_marking.forbid-agent") if {
	startswith(input.set_by, "agent:")
} else := deny("set_marking.forbid-unknown-class") if {
	not valid_class(input.class)
} else := deny("set_marking.forbid-floor-contained") if {
	input.class == "contained"
	input.is_floor
} else := allow("set_marking.administrator") if {
	startswith(input.set_by, "person:")
	input.set_by_is_admin
} else := deny("default-deny")

# ---------------------------------------------------------------- service grants

create_service_grant := deny("create_service_grant.forbid-owner-is-creator") if {
	input.owner == ""
} else := deny("create_service_grant.forbid-owner-is-creator") if {
	input.owner == input.creator
} else := allow("create_service_grant.administrator") if {
	startswith(input.creator, "person:")
	input.creator_is_admin
} else := allow("create_service_grant.authorised-agent") if {
	startswith(input.creator, "agent:")
	input.creator_authorised
	input.creator_holds
} else := deny("default-deny")

# ---------------------------------------------------------------- list check

# list_allows reads the candidate chains from data (known) and the item from
# input (unknown), so OPA can partially evaluate it once per list and
# evaluate only the residual per item (FR-22, row 23).
list_allows if {
	some chain in data.listctx.chains
	chain[0].principal == data.listctx.principal
	chain[0].action == data.listctx.action
	count(chain) - 1 <= data.listctx.max_depth
	every n in chain {
		not n.withdrawn
		not n.task_closed
		not n.agent_ended
		data.listctx.now_ms < n.ends_at_ms
		covers(n.scope, input.resource)
	}
}
