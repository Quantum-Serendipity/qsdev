# Defend Against Indirect Prompt Injection Attacks - Microsoft Learn

- **Source**: https://learn.microsoft.com/en-us/security/zero-trust/sfi/defend-indirect-prompt-injection
- **Retrieved**: 2026-05-14

## Defense-in-Depth Strategy

Implement a defense-in-depth strategy that layers multiple probabilistic and deterministic mitigations:

- **Prompt shields**: Analyze and sanitize incoming prompts to detect injection attempts.
- **Spotlighting**: Use data marking and metaprompting to isolate and neutralize external content within prompts.
- **Plan drift detection**: Monitor multi-step reasoning for deviations from intended task flows.
- **Critic agents**: Audit inputs and outputs in real time, especially in multi-agent systems.
- **Tool chain analysis**: Assess and block risky sequences of tool execution.
- **Security guardrails**: Embed final-layer protections within skills to catch bypassed threats.
- **Information flow control (IFC)**: Enforce policy-based isolation of untrusted content using metadata and quarantined inference environments.
- **Principle of least privilege**: Give agents only the minimal number of short-lived privileges needed to complete their tasks.
- **Human-in-the-loop**: The last line of defense against an attack is to verify risky actions with the user.

## Key Principles

- **Assume indirect prompt injection will happen**: Design systems with the expectation that some attacks will succeed.
- **Integrate early**: Apply indirect prompt injection risk evaluations during UX design, prompt engineering, and system architecture.
- **Use multiple layers**: No single solution is sufficient -- combine probabilistic and deterministic defenses.
- **Isolate untrusted content**: Use IFC to prevent untrusted data from influencing critical inference or planning.
- **Monitor continuously**: Implement runtime checks like plan drift detection and critic agents to catch anomalies.

## Trade-offs

- **Increased complexity**: Implementing and maintaining multiple layers requires coordination across engineering, security, and product teams.
- **Performance overhead**: Some mitigations, such as runtime monitoring or quarantined inference, may introduce latency or computational cost.
- **False positives**: Probabilistic defenses like prompt shields or plan drift detection may occasionally block legitimate actions, impacting user experience.
- **Resource investment**: Developing, testing, and tuning layered defenses demands ongoing investment in tooling, expertise, and infrastructure.
- **Customization required**: There is no universal solution -- each system must be evaluated to determine the appropriate combination of defenses.
