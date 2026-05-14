# Prompt Injection Attacks on Agentic Coding Assistants: A Systematic Analysis

- **Source URL**: https://arxiv.org/html/2601.17548v1
- **Retrieved**: 2026-05-14
- **Note**: This is an AI-summarized extraction; the full paper contains additional detail.

## Claude Code Attack Vectors

Claude Code skills face a critical vulnerability in their permission model. Skills defining "tool types but not tool targets" allows skills with Read access to read any file, not just project files. Malicious .cursorrules files can inject instructions that execute arbitrary shell commands when processed as trusted configuration.

## MCP-Related Vulnerabilities

- **Tool Poisoning**: Malicious instructions embedded in tool descriptions exploit implicit trust agents place in metadata
- **Protocol-Level Attacks**: D3.1 MCP attacks include tool squatting, rug pulls (post-approval behavior modification), and context contamination via shadowing
- **Transport Layer**: MITM attacks, DNS rebinding, and Server-Sent Events exploitation

MCP functions as "a semantic layer vulnerable to meaning-based manipulation," conflating data and instructions in problematic ways.

## Skill/Tool Ecosystem Attack Surface

Three vulnerability classes:
1. **Configuration Exploitation**: .cursorrules, .github/copilot-instructions.md processed without validation
2. **Marketplace Gaps**: Claude Code and Cursor lack marketplace review; Copilot Extensions require only cursory review
3. **Permission Granularity**: Tools cannot restrict file access scope; skills declare tool types without target restrictions

## Measured Attack Success Rates

- **MCPSecBench findings**: "85%+ of identified attacks successfully compromise at least one major platform"
- **Adaptive attacks**: Success rates exceed 90% when attackers optimize payloads
- **By objective**: Data exfiltration 84% success; persistence mechanisms 41%
- **Defense bypass**: All 12 evaluated defenses could be bypassed with "attack success rates exceeding 78% using adaptive optimization"

## Defense Mechanisms and Effectiveness

| Defense | Reported | Adaptive Attack Success |
|---------|----------|----------------------|
| Protect AI | <5% | 93% |
| PromptGuard | <3% | 91% |
| PIGuard | <5% | 89% |

More effective approaches (achieving <2-15% attack success):
- StruQ: Structured queries separating prompts and data
- SecAlign: Preference optimization reducing success from 96% to 2%
- Capability-scoping architectures (CaMeL, Progent)

"The space of possible injections is infinite while filters target finite pattern sets."

## Recommendations: Defense-in-Depth Framework

1. **Cryptographic Tool Identity**: Digital signing with immutable versioning (ETDI model)
2. **Capability Scoping**: Least privilege; Meta's "Rule of Two" limiting simultaneous access to untrusted inputs + sensitive data + external communication
3. **Runtime Intent Verification**: Multi-agent validation pipelines
4. **Sandboxed Execution**: Mandatory containerization with strict egress controls
5. **Provenance Tracking**: End-to-end source tracking
6. **Human-in-the-Loop Gates**: Tiered approval based on risk level

"No single mechanism provides adequate protection."
