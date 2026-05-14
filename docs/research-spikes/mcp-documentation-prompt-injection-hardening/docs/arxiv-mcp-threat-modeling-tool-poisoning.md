# MCP Threat Modeling and Tool Poisoning Analysis
- **Source**: https://arxiv.org/html/2603.22489v1
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized from arxiv HTML

## Overview
Comprehensive security analysis of the Model Context Protocol (MCP), focusing on client-side vulnerabilities to prompt injection attacks via tool poisoning. Identifies 57 distinct threats across MCP ecosystem components using STRIDE and DREAD frameworks.

## Key Threat Categories

**STRIDE Analysis Results:**
Threats spanning all STRIDE categories (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege) across five architectural components: MCP Host/Client, LLM, MCP Server, External Data Stores, and Authorization Servers.

**Critical Findings:**
- Tool poisoning ranked as the highest-severity client-side vulnerability with a DREAD score of 46.5/50 (Critical)
- LLM01 Prompt Injection scored 50/50 (maximum criticality)
- Five of seven tested MCP clients lacked static validation mechanisms for server-provided tool metadata

## Attack Vector: Tool Poisoning

**Definition:** "Malicious instructions embedded in tool metadata (descriptions, parameters, prompts) rather than user inputs" that manipulate LLM decision-making processes.

**Attack Flow:**
1. Attacker deploys malicious MCP server with poisoned tool descriptions
2. User connects unsuspectingly
3. Client passes unvalidated metadata to LLM
4. Poisoned descriptions manipulate model behavior
5. Sensitive data exfiltration or unintended actions execute

## Vulnerability Assessment

**Client-Side Weaknesses Identified:**
- Insufficient static validation: Most clients accept tool descriptions without rigorous analysis
- Limited user awareness: Critical parameter information remains obscured during execution
- Approval fatigue exploitation: Users habitually approve requests without careful inspection
- Novel attack surface: LLM intermediaries lack traditional input validation layers

## Defense-in-Depth Architecture

**Layer 1 - Registration and Validation:** JSON schema enforcement, digital signature verification, keyword scanning, permission anomaly detection.

**Layer 2 - Decision Path Analysis:** Decision Dependency Graph tracking to verify LLM selections align with user intent.

**Layer 3 - Runtime Monitoring:** Sandboxed execution, access control enforcement, behavioral monitoring, rate limiting.

**Layer 4 - User Transparency:** Full parameter disclosure, explicit confirmation for high-risk operations, comprehensive audit logging.

## Mitigation Strategies

**Protocol Hardening (Critical Priority):**
- Strict schema validation and immutable tool definitions
- OAuth 2.1 implementation with scoped token permissions
- Cryptographic signing of tool definitions
- Static scanning at registration

**Runtime Isolation (High Priority):**
- Container/VM-based sandboxing with restricted filesystem and network access
- Seccomp/AppArmor/SELinux policy enforcement
- Resource quotas and execution rate limiting

**Continuous Monitoring (Medium-High Priority):**
- Comprehensive logging of registrations and invocations
- Machine learning-based anomaly detection
- Periodic security review pipelines
- Real-time alert systems

## Empirical Testing Results

Evaluated seven major MCP clients against four attack types:
1. Reading sensitive files
2. Logging tool invocation activities
3. Creating phishing links
4. Remote script execution

Key findings documented client-specific vulnerability patterns, with detailed behavioral analysis explaining why attacks succeeded or failed on each implementation.
