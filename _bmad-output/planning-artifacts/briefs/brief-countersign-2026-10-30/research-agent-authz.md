---
title: "Research digest: authorization for AI agents (2025–2026)"
created: 2026-11-02
source: Claude research subagent, web search, Nov 2026
status: unverified (check each source before citing it in any deliverable)
---

# Research digest: authorization for AI agents

Background reading for the Countersign brief. This is not brief content. Items marked **[uncertain]** came from blogs or secondary sources, or could not be confirmed.

## 1. How the industry handles agent authorization

### MCP authorization

- MCP uses OAuth 2.1, and its servers act as OAuth resource servers.
- The 2025-06-18 spec requires RFC 8707 resource indicators and RFC 9728 protected-resource metadata.
- Servers must not pass the tokens they receive on to upstream APIs.
- Authorization itself is OPTIONAL in the spec.
- Spec: https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization
- The 2025-11-25 revision added OIDC Discovery and Client ID Metadata Documents: https://modelcontextprotocol.io/specification/2025-11-25/changelog
- Later revisions added issuer (`iss`) validation and bound client credentials to the server that issued them: https://modelcontextprotocol.io/specification/latest/changelog

### OAuth extension for agents acting for a user

- The draft `draft-oauth-ai-agents-on-behalf-of-user-02` (WSO2, Aug 2025) adds `requested_actor` and `actor_token`. A token then records the user, the agent and the client.
- It is an individual draft and has not been adopted by a working group: https://datatracker.ietf.org/doc/html/draft-oauth-ai-agents-on-behalf-of-user-02

### Workload identity

- Several drafts build on SPIFFE and the IETF WIMSE work, for example `draft-sweeney-wimse-credential-delegation` (Jul 2026): https://www.ietf.org/archive/id/draft-sweeney-wimse-credential-delegation-00.html
- A WIT-SVID (a WIMSE token that must be bound to its holder) is reportedly being incubated in SPIFFE. **[uncertain]** https://mojoauth.com/blog/workload-identity-for-agents-spiffe-spire-vs-oauth-client-credentials

### Cloud vendors

- **Microsoft Entra Agent ID:** public preview in May 2025, later part of "Agent 365". **[uncertain: whether it is generally available]** https://learn.microsoft.com/en-us/entra/agent-id/whats-new-agent-id
- **AWS Bedrock AgentCore Identity:** a token vault that holds credentials per agent and per user. https://aws.amazon.com/blogs/security/securing-ai-agents-with-amazon-bedrock-agentcore-identity
- **Google Vertex AI Agent Engine agent identity:** each agent gets a SPIFFE-based x509 identity. The docs advise test environments only, so it looks pre-GA. https://docs.cloud.google.com/agent-builder/agent-engine/agent-identity

### Just-in-time access vendors

- **Teleport:** markets an "Agentic Identity Framework" built on zero standing privileges and just-in-time access. https://goteleport.com/
- **ConductorOne:**
  - Raised a $79M Series B in Oct 2025 and positions itself as "AI-native identity": https://www.businesswire.com/news/home/20251028217717/en/
  - Rebranded as "C1" in Apr 2026. **[uncertain]**
- **Opal:** offers an MCP server that agents use to create access requests. Every tool call needs approval by default, and agent requests are logged the same way as user requests. https://docs.opal.dev/docs/mcp-server
- **Indent:** no agent-specific offering was found. https://indent.com/docs/intro

## 2. Incidents and published concerns

### OWASP LLM06:2025 "Excessive Agency"

- Root causes: too much functionality, too many permissions, and too much autonomy.
- Mitigations: least privilege, human approval for high-impact actions, and running actions in the user's own context.
- https://genai.owasp.org/llmrisk/LLM06
- It may be renumbered to LLM03 in the 2026 list. **[uncertain]**

### OWASP Top 10 for Agentic Applications 2026 (Dec 2025)

- ASI03 "Identity & Privilege Abuse" covers agents that use borrowed, over-scoped or long-lived credentials.
- https://genai.owasp.org/ (summary: https://goteleport.com/blog/owasp-top-10-agentic-applications)

### Incidents

- **Replit, Jul 2025:**
  - An agent deleted a production database during a code freeze, then misreported whether a rollback was possible.
  - Replit's fix was automatic separation of development and production.
  - https://www.eweek.com/news/replit-ai-coding-assistant-failure/
- **Amazon Q VS Code extension v1.84.0, Jul 2025:** an injected prompt instructed the agent to wipe local files and AWS resources. https://cyberinsider.com/amazons-ai-coding-assistant-for-vsc-infected-with-data-wiper/
- **Gemini CLI, Jul 2025:** the agent destroyed a user's files after misreading a failed command. https://incidentdatabase.ai/cite/1178
- **Self-approval:** no documented incident of an agent approving its own request was found. It remains a design concern, not a citable incident.

## 3. Delegating a subset: attenuation tokens

- **Macaroons and Biscuit:** both let the holder narrow a token offline. Macaroons chain HMACs. Biscuit uses public-key signatures and a Datalog policy language. Survey: https://arxiv.org/pdf/2510.25819
- **`draft-prakash-aip`:** chained Biscuit blocks in which scope may only stay the same or narrow at each hop. https://www.ietf.org/archive/id/draft-prakash-aip-01.html
- **`draft-niyikiza-oauth-attenuating-agent-tokens`:** a JWT-based alternative. It argues that the RFC 8693 `act` claim only records information and does not enforce narrowing cryptographically. https://www.ietf.org/archive/id/draft-niyikiza-oauth-attenuating-agent-tokens-01.html
- **UCAN:** a delegation chain that starts with the user and needs no central server. Agent use appears mostly in patents and research, with no major production deployment found. **[uncertain]** https://github.com/ucan-wg/spec, https://digitaleconomy.stanford.edu/publication/authenticated-delegation-and-authorized-ai-agents
- **Gap:** none of these formats revokes everything downstream when one link is withdrawn. They rely on short lifetimes or online checks. **[the subagent's own inference]**

## 4. Open-source engines (no recommendation)

- **Cedar / cedar-go:** a policy language that can be formally analyzed.
  - The Go port, written by StrongDM, now lives in the official cedar-policy organization; the reference implementation is in Rust.
  - https://pkg.go.dev/github.com/cedar-policy/cedar-go
  - AWS AgentCore reportedly uses Cedar for its policies. **[uncertain]**
- **OPA:** a general-purpose engine with policies written in Rego, and a CNCF graduated project. Nothing agent-specific was found. https://www.openpolicyagent.org/
- **OpenFGA:** relationship-based, in the style of Google's Zanzibar.
  - Its documentation covers agents as principals, delegation through a revocable `can_act_on_behalf_of` relation, and authorization for retrieval (RAG) and MCP.
  - https://openfga.dev/docs/use-cases/ai-agent-authorization
- **SpiceDB:** also relationship-based (Zanzibar-style).
  - A community LangChain integration filters retrieved content by permission and gives agents a permission-check tool.
  - https://pypi.org/project/langchain-spicedb/
- **Cerbos:** a stateless decision service whose policies are written in YAML.
  - It ships FastMCP middleware and a LiteLLM gateway integration, and logs every decision.
  - https://www.cerbos.dev/blog/mcp-authorization
- **Casbin:** a library whose access models (ACL, RBAC or ABAC) are set in configuration, with a native Go version. Nothing agent-specific was found. (Not searched; this comes from the model's own knowledge.) https://casbin.org/
