# Lasso Security: Detecting Indirect Prompt Injection in Claude Code

- **Source URL**: https://www.lasso.security/blog/the-hidden-backdoor-in-claude-coding-assistant
- **Retrieved**: 2026-05-14

## Attack Vectors Identified

Four primary categories of indirect prompt injection:

### 1. Instruction Override
Phrases like "Ignore all previous instructions" and "New system prompt:" attempting direct command replacement.

### 2. Role-Playing & Jailbreaks
Techniques such as DAN (Do Anything Now) persona adoption or claiming "developer mode" to bypass restrictions.

### 3. Encoding & Obfuscation
Malicious instructions hidden through:
- Base64 encoding
- Leetspeak variations
- Homoglyphs (Cyrillic characters substituted for Latin)
- Zero-width Unicode characters

### 4. Context Manipulation
Authority spoofing including "ADMIN MESSAGE FROM ANTHROPIC:" and hidden HTML comments.

## Attack Surface in Claude Code

Three specific scenarios demonstrated:
- **Poisoned repositories**: Malicious instructions in markdown files or code comments
- **Compromised documentation**: Web pages fetched during research containing injected directives
- **MCP Trojans**: Data from integrated services like Notion containing hidden instructions

## Detection Methodology

Solution implements 50+ regex patterns across attack categories:
- Instruction Override: 15 patterns
- Role-Playing/DAN: 12 patterns
- Encoding/Obfuscation: 12 patterns
- Context Manipulation: 15 patterns

Severity classifications: HIGH, MEDIUM, LOW. Warnings injected into Claude's context rather than blocking content.

## MCP-Specific Findings

"Every MCP connection is a trust boundary. Every trust boundary is an attack vector." MCP servers represent critical vulnerabilities, particularly when handling documentation or database access without content validation.

## Key Insight
The defender operates as a "PostToolUse hook" that intercepts tool results after execution. Processes untrusted content before Claude incorporates it into context window, acting as intermediary security layer.
