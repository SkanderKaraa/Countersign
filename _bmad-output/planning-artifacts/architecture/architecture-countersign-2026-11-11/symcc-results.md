# SymCC results

Policies `cedarengine/policies.cedar`, validated strictly against `cedarengine/policies.cedarschema` (0 errors). Analysed with cedar-policy-symcc 0.9.0 (cedar-policy 4.15.0) and cvc5 1.3.1, for every request and entity store the schema allows in the named request environment.

| Rule | Claim | Request environment | Result | Time |
|---|---|---|---|---|
| NFR-5 rule 5 (FR-11) | an agent never approves | Agent, approve, Request | **proved** | 10µs |
| NFR-5 rule 6 (FR-12) | nobody approves their own request | Person, approve, Request | **proved** | 4ms |
| FR-11, FR-21 | an approval needs administrator rights or authority covering the target | Person, approve, Request | **proved** | 2ms |
| NFR-5 rule 12 (FR-6) | an agent never sets a marking | Agent, set_marking, Path | **proved** | 2µs |
| NFR-5 rule 4 (FR-6) | a floor-of-four action is never marked contained | Person, set_marking, Path | **proved** | 2ms |
| FR-6 | only the two known classes are ever set | Person, set_marking, Path | **proved** | 898µs |
| NFR-5 rule 8 (FR-14) | a hand-down is never wider or longer than its parent | Agent, hand_down, Grant | **proved** | 2ms |
| NFR-5 rule 11 (FR-8) | handing down always needs explicit sharing authority | Agent, hand_down, Grant | **proved** | 2ms |
| FR-16 | a service grant is never handed down | Agent, hand_down, Grant | **proved** | 2ms |
| NFR-5 rule 1 (NFR-1) | a check is allowed only by a matching grant | Agent, push, Path | **proved** | 12ms |
| NFR-5 rules 8 and 9 (FR-14, FR-15) | a check is allowed only if the parent grant is live and covers the resource | Agent, push, Path | **proved** | 24ms |
| FR-18 | a chain deeper than the maximum depth is never allowed | Agent, push, Path | **proved** | 31ms |
| NFR-5 rule 13 (FR-16) | an agent proposes a service grant only if authorised and holding the access | Agent, create_service_grant, Path | **proved** | 4ms |
| FR-16 | a service grant's owner is never empty or its creator | Person, create_service_grant, Path | **proved** | 1ms |
| control | a person never approves (false: an authorised person does) | Person, approve, Request | control: counterexample found, as it must be | 3ms |
| control | an agent's check is never allowed (false: a valid grant allows it) | Agent, push, Path | control: counterexample found, as it must be | 36ms |

14 of 14 properties proved; 2 of 2 negative controls behaved as they must; 0 failures.
