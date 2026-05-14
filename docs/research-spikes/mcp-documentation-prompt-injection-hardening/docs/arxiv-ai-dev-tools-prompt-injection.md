# Are AI-assisted Development Tools Immune to Prompt Injection?
- **Source**: https://arxiv.org/html/2603.21642v1
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized from arxiv HTML

## Overview
First empirical security analysis of prompt injection vulnerabilities across seven major Model Context Protocol (MCP) clients used in AI-assisted development.

## Four Attack Vectors Tested

### 1. Reading Sensitive Files Attack
The attack embeds hidden instructions in tool descriptions using `<IMPORTANT>` tags. A tool disguised as simple arithmetic includes commands to read configuration files (~/.cursor/mcp.json) and SSH credentials (~/.ssh/secret.txt), passing sensitive data as hidden parameters invisible to users.

**Vulnerable behavior:** LLM reads files before invocation; data passes silently through hidden parameters.

**Expected defense:** Block suspicious file paths; require explicit user permission; log activities.

### 2. Logging Tool Invocation Activities Attack
Tools claim "CRITICAL" priority status to execute before legitimate tools, establishing persistent surveillance. The attack creates log files recording all subsequent tool usage, user prompts, tool descriptions, and activity timelines.

**Attack mechanism:** Manipulates execution order through priority claims in descriptions; silently establishes monitoring infrastructure.

**Defense requirement:** Ignore priority claims; detect file write operations; require user permission for logging.

### 3. Creating Phishing Links Attack
Tool descriptions contain instructions to generate clickable markdown links where display text ("Click here") conceals actual URLs pointing to attacker-controlled domains. Account numbers get exposed in URL parameters.

**Exploitation:** Users see benign text but click malicious links; credentials transmitted to attackers.

**Prevention needed:** Display full URLs alongside text; warn about external connections; require confirmation.

### 4. Remote Execution of Scripts Attack
Hidden instructions within tool descriptions direct the LLM to download and execute remote shell scripts using `curl | bash` patterns. Scripts execute with user privileges, enabling malware installation and system compromise.

**Critical risk:** Full system compromise through privilege escalation.

## Key Findings by Client

**Most Secure:** Claude Desktop and Cline
- Claude Desktop: Strong ethical guidelines; content policy enforcement; consistent refusal of suspicious requests
- Cline: Pattern-based injection detection; explicit security warnings; proactive user education

**Most Vulnerable:** Cursor
- No tool description validation
- Absent parameter inspection
- No security warnings
- "Blindly trusts all server-provided metadata"
- All four attacks succeeded

**Partially Secure:** Continue, Gemini CLI, Claude Code, Langflow
- Inconsistent protection across attack types
- Context-dependent vulnerabilities
- Mixed success/failure patterns

## Security Features Analysis

| Feature | Finding |
|---------|---------|
| Static Validation | 5 of 7 clients implement none; 2 implement partial validation |
| Parameter Visibility | Most clients have low-to-partial visibility of tool parameters |
| Injection Detection | Varies from model-based (Claude) to pattern-based (Cline) to none |
| User Warnings | Inconsistent; absent in high-risk clients |
| Execution Sandboxing | Largely absent; most tools execute with full host privileges |
| Audit Logging | Minimal; most lack comprehensive logging for security review |

## Architectural Vulnerabilities

Common weaknesses across all tested clients:
- No validation of tool descriptions at registration
- Hidden parameters containing sensitive data
- Full host system privilege execution
- No behavioral monitoring for suspicious patterns
- Implicit trust in server-provided metadata without verification

## Recommendations

**For developers:** Implement static validation, enforce parameter visibility, deploy sandboxed execution, integrate behavioral monitoring.

**For organizations:** Conduct pre-deployment risk assessments; prioritize security in client selection; establish monitoring frameworks.

**For users:** Treat all tool output as untrusted; never enable auto-run modes; use `.cursorignore` to hide sensitive files; sandbox tools in Docker containers.
