//! Proves Countersign's fixed rules about policies.cedar with SymCC and the
//! cvc5 solver, for every request and entity store the schema allows, not
//! for test inputs. Two negative controls must fail, so a vacuous "proof"
//! (for example, a schema that types every request away) would show up.
//!
//!     CVC5=/path/to/cvc5-1.3.1 cargo run --release
use cedar_policy::{PolicySet, RequestEnv, Schema, ValidationMode, Validator};
use cedar_policy_symcc::{solver::LocalSolver, CedarSymCompiler, CompiledPolicySet, CompiledSchema};
use std::{fmt::Write as _, str::FromStr, time::Instant};

enum Kind {
    /// No request of this shape is ever allowed.
    AlwaysDenies,
    /// Whenever policies.cedar allows a request of this shape, the property
    /// policy (a single permit) allows it too.
    Implies(&'static str),
}

struct Prop {
    rule: &'static str,
    claim: &'static str,
    principal: &'static str,
    action: &'static str,
    resource: &'static str,
    kind: Kind,
    /// Negative controls are false on purpose: SymCC must find a counterexample.
    control: bool,
}

const PROPS: &[Prop] = &[
    Prop { rule: "NFR-5 rule 5 (FR-11)", claim: "an agent never approves", principal: "Agent", action: "approve", resource: "Request", kind: Kind::AlwaysDenies, control: false },
    Prop { rule: "NFR-5 rule 6 (FR-12)", claim: "nobody approves their own request", principal: "Person", action: "approve", resource: "Request",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { principal != resource.requester };"#), control: false },
    Prop { rule: "FR-11, FR-21", claim: "an approval needs administrator rights or authority covering the target", principal: "Person", action: "approve", resource: "Request",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { context.approver_is_admin || resource.target in context.authority_scopes };"#), control: false },
    Prop { rule: "NFR-5 rule 12 (FR-6)", claim: "an agent never sets a marking", principal: "Agent", action: "set_marking", resource: "Path", kind: Kind::AlwaysDenies, control: false },
    Prop { rule: "NFR-5 rule 4 (FR-6)", claim: "a floor-of-four action is never marked contained", principal: "Person", action: "set_marking", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { !(context.class == "contained" && context.is_floor) };"#), control: false },
    Prop { rule: "FR-6", claim: "only the two known classes are ever set", principal: "Person", action: "set_marking", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { context.class == "human-only" || context.class == "contained" };"#), control: false },
    Prop { rule: "NFR-5 rule 8 (FR-14)", claim: "a hand-down is never wider or longer than its parent", principal: "Agent", action: "hand_down", resource: "Grant",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { principal == resource.principal && context.child_scope in resource.scope && context.child_ends_at <= resource.ends_at };"#), control: false },
    Prop { rule: "NFR-5 rule 11 (FR-8)", claim: "handing down always needs explicit sharing authority", principal: "Agent", action: "hand_down", resource: "Grant",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { (context.class == "human-only" && resource.depth > 0 && context.child_depth < resource.depth) || (context.class == "contained" && context.has_sharing_limit && context.chain_len <= context.sharing_limit) };"#), control: false },
    Prop { rule: "FR-16", claim: "a service grant is never handed down", principal: "Agent", action: "hand_down", resource: "Grant",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { resource.kind == "task" };"#), control: false },
    Prop { rule: "NFR-5 rule 1 (NFR-1)", claim: "a check is allowed only by a matching grant", principal: "Agent", action: "push", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { context.grant.principal == principal && context.grant.action == action && resource in context.grant.scope && context.now < context.grant.ends_at && !context.grant.withdrawn };"#), control: false },
    Prop { rule: "NFR-5 rules 8 and 9 (FR-14, FR-15)", claim: "a check is allowed only if the parent grant is live and covers the resource", principal: "Agent", action: "push", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { if context.grant has parent then resource in context.grant.parent.scope && !context.grant.parent.withdrawn && !context.grant.parent.task_closed && !context.grant.parent.agent_ended && context.now < context.grant.parent.ends_at else true };"#), control: false },
    Prop { rule: "FR-18", claim: "a chain deeper than the maximum depth is never allowed", principal: "Agent", action: "push", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { !(context.grant has parent && context.grant.parent has parent && context.grant.parent.parent has parent && context.grant.parent.parent.parent has parent) };"#), control: false },
    Prop { rule: "NFR-5 rule 13 (FR-16)", claim: "an agent proposes a service grant only if authorised and holding the access", principal: "Agent", action: "create_service_grant", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { context.creator_authorised && context.creator_holds };"#), control: false },
    Prop { rule: "FR-16", claim: "a service grant's owner is never empty or its creator", principal: "Person", action: "create_service_grant", resource: "Path",
           kind: Kind::Implies(r#"permit (principal, action, resource) when { context.owner != "" && !context.owner_is_creator };"#), control: false },
    // Negative controls: false on purpose.
    Prop { rule: "control", claim: "a person never approves (false: an authorised person does)", principal: "Person", action: "approve", resource: "Request", kind: Kind::AlwaysDenies, control: true },
    Prop { rule: "control", claim: "an agent's check is never allowed (false: a valid grant allows it)", principal: "Agent", action: "push", resource: "Path", kind: Kind::AlwaysDenies, control: true },
];

fn find_env(schema: &Schema, p: &Prop) -> RequestEnv {
    let action = format!(r#"Action::"{}""#, p.action);
    schema
        .request_envs()
        .find(|e| e.principal().to_string() == p.principal && e.action().to_string() == action && e.resource().to_string() == p.resource)
        .unwrap_or_else(|| panic!("no request environment {} {} {}", p.principal, action, p.resource))
}

#[tokio::main(flavor = "current_thread")]
async fn main() {
    let schema_text = std::fs::read_to_string("../cedarengine/policies.cedarschema").unwrap();
    let schema = Schema::from_cedarschema_str(&schema_text).expect("schema").0;
    let policies = PolicySet::from_str(&std::fs::read_to_string("../cedarengine/policies.cedar").unwrap()).expect("policies");
    let validation = Validator::new(schema.clone()).validate(&policies, ValidationMode::Strict);
    let errors = validation.validation_errors().count();
    assert_eq!(errors, 0, "policies.cedar must validate strictly against the schema");

    let compiled_schema = CompiledSchema::new(&schema).expect("compiled schema");
    let mut compiler = CedarSymCompiler::new(LocalSolver::cvc5().expect("cvc5 (set CVC5 to the cvc5 1.3.1 binary)")).unwrap();

    let mut out = String::new();
    writeln!(out, "# SymCC results\n\nPolicies `cedarengine/policies.cedar`, validated strictly against `cedarengine/policies.cedarschema` (0 errors). Analysed with cedar-policy-symcc 0.9.0 (cedar-policy 4.15.0) and cvc5 1.3.1, for every request and entity store the schema allows in the named request environment.\n").unwrap();
    writeln!(out, "| Rule | Claim | Request environment | Result | Time |\n|---|---|---|---|---|").unwrap();
    let mut proved = 0;
    let mut controls_ok = 0;
    let mut failed = 0;
    for p in PROPS {
        let env = find_env(&schema, p);
        let pset = CompiledPolicySet::compile(&policies, &env, &compiled_schema).expect("compile policies");
        let start = Instant::now();
        let counterexample = match p.kind {
            Kind::AlwaysDenies => compiler.check_always_denies_with_counterexample_opt(&pset).await.expect("solver"),
            Kind::Implies(text) => {
                let prop = PolicySet::from_str(text).expect("property policy");
                let prop = CompiledPolicySet::compile(&prop, &env, &compiled_schema).expect("compile property");
                compiler.check_implies_with_counterexample_opt(&pset, &prop).await.expect("solver")
            }
        };
        let took = start.elapsed();
        let result = match (&counterexample, p.control) {
            (None, false) => { proved += 1; "**proved**".to_string() }
            (Some(_), true) => { controls_ok += 1; "control: counterexample found, as it must be".to_string() }
            (Some(cex), false) => { failed += 1; format!("**NOT proved**: counterexample `{}`", cex.request) }
            (None, true) => { failed += 1; "**control FAILED**: proved a false claim, so the analysis is vacuous".to_string() }
        };
        writeln!(out, "| {} | {} | {}, {}, {} | {} | {:.0?} |", p.rule, p.claim, p.principal, p.action, p.resource, result, took).unwrap();
    }
    writeln!(out, "\n{proved} of {} properties proved; {controls_ok} of 2 negative controls behaved as they must; {failed} failures.", PROPS.len() - 2).unwrap();
    std::fs::create_dir_all("../results").unwrap();
    std::fs::write("../results/symcc.md", &out).unwrap();
    print!("{out}");
    if failed > 0 {
        std::process::exit(1);
    }
}
