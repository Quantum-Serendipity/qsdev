# Windsurf Security: How to Use AI Coding Safely
- **Source**: https://www.mintmcp.com/blog/windsurf-security
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Permission Model & Access Control

Windsurf implements role-based access controls (RBAC) through its Enterprise tier, enabling organizations to "Connect identity provider (Okta, Azure AD, Google) through SAML" and configure feature toggles for MCP servers and command auto-execution. The platform supports granular permissions where administrators can "Enable read-only operations for analysts while restricting write tools to senior developers."

## Sandboxing & Execution Controls

The article emphasizes disabling dangerous default behaviors rather than describing robust sandboxing. Key controls include:
- Disabling auto-execution of terminal commands via Admin Portal settings
- Requiring human approval for infrastructure-modifying commands
- Using `.codeiumignore` to exclude sensitive directories

## Tool Output Handling & MCP Security

Windsurf addresses MCP risks through allowlisting: "Whitelist approved MCP servers explicitly rather than allowing unrestricted tool access." "MCP Server Auto-Invocation" and "Lack of security controls for tool execution" represent primary vulnerabilities.

## Prompt Injection Defenses

Identified Threats:
- "Prompt Injection via Filenames: Malicious filenames can manipulate AI behavior"

Mitigation Strategies:
- Implementing `.windsurfrules` files with "NEVER/ALWAYS security flags"
- Security rules files that "enforce secure coding patterns"
- Regular security audits of AI-generated code

No technical architectural details about how Windsurf prevents malicious inputs from compromising agent behavior.

## Critical Gap

"Organizations cannot see what agents access or control their actions without proper monitoring infrastructure" -- suggesting native visibility limitations require external tools for real-time defense.
