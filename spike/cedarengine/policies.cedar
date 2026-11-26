// Countersign's rules in Cedar. Cedar denies by default: with no permit,
// the answer is "refuse" (NFR-5 rule 1). A forbid always wins over a permit,
// so each fixed rule is written as a forbid that no permit can override.
//
// Times are Long values in Unix milliseconds. Paths are entities whose
// parents are every wider subtree ("repo/api/x" is in "repo/api/**" and
// "repo/**"), so "covers" is Cedar's `in`.

// ---------------------------------------------------------------- check

// A grant allows a check when it belongs to the principal, is for the
// action, and it and every ancestor are live and cover the resource
// (NFR-5 rules 8 and 9). Cedar has no recursion, so the three ancestor
// levels the system maximum depth allows are written out. Each level is an
// if-then-else on `has parent`, so Cedar's strict validator (and SymCC) can
// prove `parent` exists wherever it is read.
@id("check.grant-allows")
permit (principal, action, resource is Path)
when {
  context has grant &&
  context.grant.principal == principal &&
  context.grant.action == action &&
  resource in context.grant.scope &&
  !context.grant.withdrawn && !context.grant.task_closed && !context.grant.agent_ended &&
  context.now < context.grant.ends_at
}
when {
  if context.grant has parent then
    resource in context.grant.parent.scope &&
    !context.grant.parent.withdrawn && !context.grant.parent.task_closed && !context.grant.parent.agent_ended &&
    context.now < context.grant.parent.ends_at
  else true
}
when {
  if context.grant has parent && context.grant.parent has parent then
    resource in context.grant.parent.parent.scope &&
    !context.grant.parent.parent.withdrawn && !context.grant.parent.parent.task_closed && !context.grant.parent.parent.agent_ended &&
    context.now < context.grant.parent.parent.ends_at
  else true
}
when {
  if context.grant has parent && context.grant.parent has parent && context.grant.parent.parent has parent then
    resource in context.grant.parent.parent.parent.scope &&
    !context.grant.parent.parent.parent.withdrawn && !context.grant.parent.parent.parent.task_closed && !context.grant.parent.parent.parent.agent_ended &&
    context.now < context.grant.parent.parent.parent.ends_at
  else true
};

// A chain deeper than the system maximum is never allowed.
@id("check.forbid-too-deep")
forbid (principal, action, resource is Path)
when {
  context has grant &&
  context.grant has parent && context.grant.parent has parent &&
  context.grant.parent.parent has parent && context.grant.parent.parent.parent has parent
};

// ---------------------------------------------------------------- approve

// An administrator, or a person whose approval authority for this action
// covers the target resource.
@id("approve.authorised-person")
permit (principal is Person, action == Action::"approve", resource is Request)
when { context.approver_is_admin || resource.target in context.authority_scopes };

@id("approve.forbid-unknown-class")
forbid (principal, action == Action::"approve", resource is Request)
when { resource.class != "human-only" && resource.class != "contained" };

// NFR-5 rule 5: an agent never approves, however the approval is worded.
@id("approve.forbid-agent")
forbid (principal is Agent, action == Action::"approve", resource);

// NFR-5 rule 6: nobody approves their own request.
@id("approve.forbid-self")
forbid (principal, action == Action::"approve", resource is Request)
when { principal == resource.requester };

// ---------------------------------------------------------------- hand down

// A grant's holder may hand part of it down: never wider or longer
// (rule 8), and only with explicit sharing authority, for every grant
// (rule 11): the parent's depth for a human-only grant, the standing sharing
// limit for a contained one.
@id("hand_down.within-parent")
permit (principal, action == Action::"hand_down", resource is Grant)
when {
  principal == resource.principal &&
  context.child_scope in resource.scope &&
  context.child_ends_at <= resource.ends_at &&
  context.chain_len <= context.max_depth
}
when {
  (context.class == "human-only" && resource.depth > 0 && context.child_depth < resource.depth) ||
  (context.class == "contained" && context.has_sharing_limit && context.chain_len <= context.sharing_limit)
};

// A service grant is never handed down: a helper that needs long-lived access
// gets its own service grant (Skander, 2026-11-17). Anything that isn't a
// task grant refuses, so an unknown kind can't slip through.
@id("hand_down.forbid-service-grant")
forbid (principal, action == Action::"hand_down", resource is Grant)
when { resource.kind != "task" };

// Any class other than the two known ones refuses.
@id("hand_down.forbid-unknown-class")
forbid (principal, action == Action::"hand_down", resource)
when { context.class != "human-only" && context.class != "contained" };

// ---------------------------------------------------------------- markings

@id("set_marking.administrator")
permit (principal is Person, action == Action::"set_marking", resource is Path)
when { context.set_by_is_admin };

@id("set_marking.forbid-unknown-class")
forbid (principal, action == Action::"set_marking", resource)
when { context.class != "human-only" && context.class != "contained" };

// NFR-5 rule 12: an agent never sets a marking.
@id("set_marking.forbid-agent")
forbid (principal is Agent, action == Action::"set_marking", resource);

// NFR-5 rule 4: a floor-of-four action is never marked contained.
@id("set_marking.forbid-floor-contained")
forbid (principal, action == Action::"set_marking", resource)
when { context.class == "contained" && context.is_floor };

// ---------------------------------------------------------------- service grants

@id("create_service_grant.administrator")
permit (principal is Person, action == Action::"create_service_grant", resource is Path)
when { context.creator_is_admin };

// NFR-5 rule 13: an agent needs explicit authorisation and must hold the
// access itself, so creating a service grant never gives anything new.
@id("create_service_grant.authorised-agent")
permit (principal is Agent, action == Action::"create_service_grant", resource is Path)
when { context.creator_authorised && context.creator_holds };

// The owner is a team or role, never the creator.
@id("create_service_grant.forbid-owner-is-creator")
forbid (principal, action == Action::"create_service_grant", resource)
when { context.owner == "" || context.owner_is_creator };
