# MCP Specification: Security Best Practices

- **Source URL**: https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices
- **Retrieved**: 2026-05-14

## Introduction

This document provides security considerations for the Model Context Protocol (MCP), complementing the MCP Authorization specification. It identifies security risks, attack vectors, and best practices specific to MCP implementations.

## Attacks and Mitigations

### Confused Deputy Problem
Attackers can exploit MCP proxy servers that connect to third-party APIs, creating "confused deputy" vulnerabilities. This attack allows malicious clients to obtain authorization codes without proper user consent by exploiting the combination of static client IDs, dynamic client registration, and consent cookies.

Mitigation: MCP proxy servers MUST implement per-client consent and proper security controls. Display clear consent dialog, implement CSRF protection, validate redirect URIs with exact string matching.

### Token Passthrough
"Token passthrough" is an anti-pattern where an MCP server accepts tokens from an MCP client without validating that the tokens were properly issued to the MCP server. This is explicitly forbidden in the authorization specification.

Risks include: security control circumvention, accountability/audit trail issues, trust boundary issues, future compatibility risk.

### Server-Side Request Forgery (SSRF)
During OAuth metadata discovery, MCP clients fetch URLs from sources that could be controlled by a malicious MCP server. Malicious servers can populate fields with URLs pointing to internal resources.

Attack patterns include: direct internal IP access, cloud metadata endpoints, localhost services, DNS rebinding, redirect chains.

Mitigation: Enforce HTTPS, block private IP ranges, validate redirect targets, use egress proxies, consider DNS resolution TOCTOU issues.

### Session Hijacking
When multiple stateful HTTP servers handle MCP requests, session hijacking becomes possible through:
- Session Hijack Prompt Injection: attacker sends malicious event to a different server using an obtained session ID
- Session Hijack Impersonation: attacker makes calls using a hijacked session ID

Mitigation: Verify all inbound requests, use secure non-deterministic session IDs, bind session IDs to user-specific information.

### Local MCP Server Compromise
Local MCP servers run on the user's machine and may have direct access to the user's system. Without proper sandboxing, malicious startup commands or payloads can be distributed.

Mitigation: Implement pre-configuration consent dialogs showing exact commands, highlight dangerous patterns, sandbox MCP server execution.

### Scope Minimization
Poor scope design increases token compromise impact. Implement progressive, least-privilege scope model with minimal initial scopes and incremental elevation.

## Notable Absence
The MCP security best practices document focuses heavily on OAuth/authorization security but does NOT contain specific guidance about:
- How tool result content should be treated as untrusted by the LLM
- Prompt injection defenses for content returned by tools
- Content sanitization requirements for tool outputs
- Trust hierarchy between system prompts and tool results
