<!-- Source: https://arxiv.org/html/2601.17549v1 -->
<!-- Retrieved: 2026-05-14 -->

# Breaking the Protocol: Security Analysis of the Model Context Protocol Specification and Prompt Injection Vulnerabilities in Tool-Integrated LLM Agents

## Overview
First rigorous security analysis of the Model Context Protocol (MCP) v1.0 specification, identifying architectural vulnerabilities that amplify prompt injection attacks in tool-integrated LLM agents.

## Threat Model
The researchers consider adversaries who control or compromise individual MCP servers within multi-server deployments. Key attack vectors include:

- **Server Discovery Vulnerabilities**: Typosquatting (34%), supply chain compromise (28%), social engineering (23%), and marketplace poisoning (15%)
- **Black-box Access**: Attackers cannot modify LLM weights but can inject malicious content into data sources and server responses

## Three Critical Protocol Vulnerabilities

### 1. Least Privilege Violation
Servers self-assert capabilities during initialization without cryptographic verification. The specification permits capability escalation post-initialization — a server claiming only resource access can later invoke sampling functions without authorization checks.

### 2. Sampling Without Origin Authentication
"Servers can request LLM completions from clients via sampling/createMessage, allowing servers to inject prompts" without distinction from legitimate user input. Analysis of three major hosts (Claude Desktop, Cursor, Continue) found none provided visual indicators distinguishing server-injected from user-originated messages.

### 3. Implicit Trust Propagation
Multi-server deployments lack isolation boundaries. Tool responses from one server influence subsequent invocations on others, enabling cross-server data exfiltration and persistence attacks. The specification prioritizes composability over isolation by default.

## Experimental Findings (ProtoAmp Framework)

847 attack scenarios comparing MCP-integrated agents against non-MCP baselines:

| Attack Type | Baseline | MCP | Amplification |
|---|---|---|---|
| Indirect Injection | 31.2% | 47.8% | +16.6% |
| Tool Response Manipulation | 28.4% | 52.1% | +23.7% |
| Cross-Server Propagation | 19.7% | 61.3% | +41.6% |
| Sampling Injection | N/A | 67.2% | -- |
| **Overall** | **26.4%** | **52.8%** | **+26.4%** |

Attack success rates increased 23-41% when identical attacks were executed through MCP architecture, with largest amplification in cross-server scenarios.

## Proposed Defense: AttestMCP

A backward-compatible protocol extension addressing identified weaknesses:

**Core Mechanisms:**
- **Capability Attestation**: Cryptographic certificates from capability authorities verify server permissions
- **Message Authentication**: HMAC-SHA256 signatures bind JSON-RPC messages to authenticated server identity
- **Origin Tagging**: Sampling requests tagged to distinguish server vs. user-originated prompts
- **Isolation Enforcement**: Cross-server data flow requires explicit user authorization
- **Replay Protection**: Timestamp plus nonce validation

**Results:**
AttestMCP reduced overall attack success from 52.8% to 12.4% (76.5% reduction), with 85.8% improvement against cross-server attacks. Median performance overhead: 8.3ms per message (negligible vs. typical 500-2000ms LLM inference latency).

## Trust Model Architecture

Federated capability authorities:
- Commercial servers verified via domain ownership
- Open-source servers verified through package registry account binding
- Cross-signing agreements enable ecosystem interoperability
- Certificate revocation within 4-24 hours depending on urgency

## Key Limitations

AttestMCP cannot address:
- Attacks within legitimately-authorized servers
- Social engineering pressuring users to authorize malicious capabilities
- "First-contact attacks" where malicious servers bypass historical pinning
- Ecosystem adoption barriers if most servers remain unsigned

Residual 12.4% attack success "primarily reflects indirect injection through legitimately-retrieved content — a fundamental limitation shared with all LLM systems."

## Recommendations

1. Incorporate mandatory capability attestation into MCP v2.0
2. Require origin tagging for all sampling requests at protocol level
3. Establish explicit isolation boundaries with user-prompted cross-server authorization
