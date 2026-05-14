# Microsoft Taxonomy of Failure Modes in Agentic AI Systems
- **Source**: https://www.microsoft.com/en-us/security/blog/2025/04/24/new-whitepaper-outlines-the-taxonomy-of-failure-modes-in-ai-agents/
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted summary from Microsoft Security Blog and related coverage

## Overview

Microsoft AI Red Team mapped 27 safety & security failure modes in autonomous agent systems to help security professionals and ML engineers design safer AI systems.

## Failure Mode Categories

### Agent-Specific Failures
- **Agent compromise**: Agent subverted via modifying prompts, parameters, code for malicious behavior
- **Agent injection**: Injecting rogue agent into a system's agent network
- **Agent impersonation**: Malicious actor masquerades as a genuine agent

### Memory and Data Attacks
- **Memory poisoning/theft**: Corrupting persistent storage used by agents
- Particularly insidious: absence of robust semantic analysis allows malicious instructions to be stored, recalled, and executed

### Input and Control Failures
- **Cross-domain prompt injection (XPIA)**: External sources influence internal prompts
- **Human-in-the-loop bypass**: Circumventing human oversight mechanisms

### Permission and Isolation Issues
- **Incorrect permissions/insufficient isolation**: Agents having too much access or insufficient sandboxing

### Transparency and Alignment Issues
- **Misinterpretation of instructions**: Agents misunderstanding user goals
- **Insufficient transparency/accountability**: Users cannot understand or contest decisions
- **Parasocial relationships**: Users attributing false agency or intent

### Responsible AI and Multi-user Issues
- **Intra-agent RAI issues**: Internal conflicts in aligning with ethical constraints
- **Harms of allocation**: Multi-user scenarios with unfair resource allocation

### Other
- **Knowledge loss**: Organizational degradation from overreliance on agents

## Cascading Failure Example

When an inventory agent invents a nonexistent SKU and calls downstream APIs to price, stock, and ship it — the hallucinated fact triggers a multi-system incident affecting ordering, fulfillment, and customer communications.

## Tool Misuse Patterns

- Agents exceed intended permissions
- Call functions with incorrect parameters
- Execute capabilities in unintended ways creating security/operational risks
- Function calling hallucinations: inventing nonexistent functions or supplying inappropriate parameters

## Instruction Drift

Attention decays over extended interactions. Split-softmax addresses this by reweighting attention toward system prompt to mitigate drift-related hallucinations.

## Mitigation Approaches

### Observability and Monitoring
- LLM-based output audits add semantic understanding that traditional syntactic rules miss
- Behavioral tracking of execution traces to spot goal drift, prompt injection, or tool misuse

### Runtime Controls (QSAF Framework)
- Seven runtime controls monitoring agent subsystems in real time
- Proactive mitigation through fallback routing, starvation detection, memory integrity enforcement
