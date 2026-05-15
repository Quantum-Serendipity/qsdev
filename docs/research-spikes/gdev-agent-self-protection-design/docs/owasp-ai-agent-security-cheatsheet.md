<!-- Source: https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Content returned via WebFetch AI summary -->

# OWASP AI Agent Security Cheat Sheet Summary

## Core Threat Categories

The framework identifies **13 key risks** including prompt injection, tool abuse, data exfiltration, memory poisoning, goal hijacking, and "Denial of Wallet" (DoW)—attacks causing excessive API costs through unbounded loops. Additional threats include approval manipulation, cascading multi-agent failures, and supply chain compromises.

## Nine Defense Pillars

**1. Tool Security & Least Privilege**
Agents should receive only essential tools with scoped permissions. The guidance contrasts "dangerous: Agent has unrestricted shell access" against safer approaches using allowlists and blocked patterns like `*.env` and `*.key` files.

**2. Input Validation & Prompt Injection Defense**
"Treat all external data as untrusted (user messages, retrieved documents, API responses, emails)." Use clear delimiters between instructions and data, content filtering, and separate LLM calls to validate untrusted content.

**3. Memory & Context Security**
Implementation requires validation before persistence, user/session isolation, expiration limits, and integrity checks. The sheet demonstrates sanitizing injection attempts and redacting sensitive patterns (SSN, credit cards, API keys).

**4. Human-in-the-Loop Controls**
Actions receive risk classification (low/medium/high/critical). High-impact operations require "separate decision-making from execution" with parameter-bound approvals, step-up authentication, and short-lived authorization artifacts.

**5. Output Validation & Guardrails**
Structured outputs with schema validation, PII filtering, rate limiting, and detection of exfiltration attempts (e.g., base64-encoded data in URLs or oversized webhook calls).

**6. Monitoring & Observability**
"Log structured decision metadata for high-risk actions, including action classification, risk score...approval identifier, execution result, and policy version." Anomaly detection tracks approval drift, repeated bypass attempts, and privilege escalation patterns.

**7. Multi-Agent Security**
Signed inter-agent messages with expiration timestamps prevent replay attacks. Trust boundaries restrict message types and recipients per agent privilege level.

**8. Data Protection & Privacy**
Automatic classification (public/internal/confidential/restricted) triggers appropriate handling—full redaction for PII, partial masking for confidential data, normal handling for internal content.

**9. Adversarial Testing**
Maintain repeatable abuse-case test matrices covering prompt override, tool misuse, privilege escalation, memory poisoning, and data exfiltration. Block releases when high-risk policies change without updated tests.

## Critical "Do's and Don'ts"

**Essential practices:** Apply least privilege, validate external inputs, implement human oversight for high-risk actions, isolate user memory, monitor anomalies, use structured outputs, sign inter-agent communications, and perform adversarial testing before deployment.

**Critical prohibitions:** Never grant unrestricted tool access, trust external sources without validation, permit arbitrary code execution, store secrets in memory unencrypted, allow high-impact decisions without oversight, or skip testing after prompt/tool/provider changes.

## Agent Self-Modification Prevention

The framework emphasizes "AI Console Malicious Configuration" as a distinct threat—developer consoles can be compelled to ingest malicious instructions driving unauthorized LLM configuration changes. Defense relies on strict input validation, approval workflows, and audit logging of console activity.
