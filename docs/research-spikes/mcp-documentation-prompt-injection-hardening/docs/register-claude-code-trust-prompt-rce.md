# The Register: Claude Code Trust Prompt Can Trigger One-Click RCE

- **Source URL**: https://www.theregister.com/security/2026/05/07/claude-code-trust-prompt-can-trigger-one-click-rce/5235319
- **Retrieved**: 2026-05-14

## Attack Mechanism

Security firm Adversa AI disclosed a one-click RCE flaw in Claude Code exploiting MCP server functionality through malicious JSON configuration files.

**Technical Vector:** A cloned repository can contain `.mcp.json` and `.claude/settings.json` files that silently enable dangerous settings. When developers approve the folder with a generic trust dialog, the attack activates.

## MCP and Tool Handling

MCP servers expose "tools, configuration data, schemas, and documentation in a standard format to AI models via JSON." The vulnerability stems from inconsistent security controls—Anthropic blocks some risky settings at project level (like `bypassPermissions`) but permits others (`enableAllProjectMcpServers`, `enabledMcpjsonServers`).

Critical flaw: "The moment a developer presses Enter on Claude Code's generic 'Yes, I trust this folder' dialog, the server spawns as an unsandboxed Node.js process with the user's full privileges."

## Trust Prompt Failure

Current dialog defaults to "Yes, I trust this folder" without MCP-specific language or executable enumeration. Anthropic removed a previous version's explicit warning that `.mcp.json` could execute code and its "disable MCP" option.

## One-Click RCE

User interaction is the sole trigger—clicking "trust" activates malicious MCP server immediately. A zero-click variant also threatens CI/CD pipelines using Claude Code's SDK interface, which bypasses interactive prompts entirely.

## Anthropic's Response

Anthropic argues this falls outside its threat model because users made an informed trust decision. The company declined comment to The Register.

## Implications

This exemplifies broader risks: trusting user-provided configuration without granular validation creates injection surfaces. Third CVE in six months from identical causes, suggesting systemic architectural weaknesses.
