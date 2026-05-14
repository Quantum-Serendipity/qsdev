# Simon Willison: Model Context Protocol Has Prompt Injection Security Problems

- **Source URL**: https://simonwillison.net/2025/Apr/9/mcp-prompt-injection/
- **Retrieved**: 2026-05-14

## Overview
Simon Willison documents critical security flaws in MCP stemming from combining untrusted instructions with powerful tools—a fundamental architectural challenge rather than implementation error.

## Key Vulnerability Categories

### 1. Rug Pulls & Tool Shadowing
- **Silent Redefinition**: "MCP tools can mutate their own definitions after installation" allowing attackers to redirect credentials over time
- **Cross-Server Shadowing**: Malicious servers can intercept calls intended for trusted ones, exploiting the LLM's inability to distinguish between sources

### 2. Tool Poisoning Attacks
Attackers embed malicious instructions within tool descriptions—visible to the LLM but hidden from users. Invariant Labs demonstrated this with a seemingly innocent `add()` function containing hidden directives to exfiltrate private configuration files before performing its stated operation.

### 3. WhatsApp MCP Exploitation
Demonstration showing how attackers can:
- Establish a fake tool alongside legitimate WhatsApp integration
- Use whitespace obfuscation to hide data exfiltration in UI interfaces
- Redefine tool behavior mid-session to intercept sensitive messages
- Provide convincing but false instructions about "proxy numbers" to redirect communications

## Fundamental Problems

**Tool descriptions become attack vectors** when LLMs treat them as authoritative instructions rather than documentation. The system cannot reliably distinguish between legitimate guidance and malicious prompts.

**User interfaces mask malicious actions** through design choices like hidden horizontal scrollbars, preventing users from seeing stolen data being transmitted.

**Multiple installation vectors exist**—direct messages, malicious MCP servers, or untrusted input can all trigger unwanted tool invocations.

## Recommended Mitigations

Willison suggests treating MCP specification "SHOULDs" as mandatory:
- Display all exposed tools clearly to users
- Show visual confirmation when tools execute
- Alert users to tool description changes
- Require explicit approval for sensitive operations

## Conclusion
Despite two years of prompt injection awareness, "convincing mitigations" remain elusive. The core issue persists: LLMs inherently trust any inputs that parse as valid instructions, making tool-augmented systems vulnerable to confused deputy attacks whenever untrusted content appears alongside operational capabilities.
