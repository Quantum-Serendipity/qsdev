# Introducing the Agent Governance Toolkit: Open-source runtime security for AI agents

- **Source URL**: https://opensource.microsoft.com/blog/2026/04/02/introducing-the-agent-governance-toolkit-open-source-runtime-security-for-ai-agents/
- **Retrieved**: 2026-05-15

## Failure Management Mechanisms

The toolkit implements several layers of failure protection:

**Circuit Breakers and SLO Enforcement**
The system incorporates "circuit breakers and SLO enforcement" as part of its Agent SRE package. This prevents cascading failures by monitoring system health and interrupting operations when thresholds are exceeded.

**Emergency Termination**
The architecture includes "a kill switch for emergency agent termination" within the Agent Runtime component, providing a fail-safe mechanism for immediate agent shutdown when policy violations or dangerous behaviors are detected.

**Cascading Failure Prevention**
The toolkit addresses OWASP's cascading failures risk through these SRE-inspired practices, ensuring one agent's failure doesn't trigger system-wide collapse.

## Policy Enforcement Architecture

**Sub-Millisecond Latency**
The policy engine operates at "sub-millisecond latency (<0.1ms p99)," intercepting every agent action before execution. This stateless design enables "horizontal scaling, containerized deployment, and auditability."

**Multiple Policy Languages**
The system supports "YAML rules, OPA Rego, and Cedar policy languages," providing flexibility in governance rule definition.

**Defense in Depth**
The architecture creates multiple independent security layers, each addressing different threat categories rather than relying on single-point enforcement.

## Error Handling in Security Checks

**Cross-Model Verification Kernel (CMVK)**
For the memory poisoning risk, the toolkit employs "majority voting," reducing susceptibility to individual verification failures.

## Deployment Resilience

The toolkit supports stateless deployment on Azure Kubernetes Service, Container Apps, and other platforms, enabling recovery and redundancy without persistent state that could complicate failure diagnosis.
