# Microsoft Agent Governance Toolkit: Policy Enforcement Model

- **Source URL**: https://github.com/microsoft/agent-governance-toolkit
- **Retrieved**: 2026-05-15

## Core Policy Decision Model

The toolkit implements a **deterministic binary verdict system**:

- **ALLOW**: Action proceeds
- **DENY**: Action is blocked
- **Implicit default**: Configurable via `PolicyDefaults(action=PolicyAction.ALLOW)` or similar

No "ASK" outcome; decisions are binary and pre-execution.

## Policy Evaluation Pipeline

```
Agent Action --> Policy Check --> Allow / Deny --> Audit Log    (< 0.1 ms)
```

Evaluations occur before action execution at the application layer.

## Rule Conflict Resolution

Priority-based rule ordering:
```python
PolicyRule(
    name="block-dangerous-tools",
    condition=PolicyCondition(field="tool_name", operator=PolicyOperator.IN, value=[...]),
    action=PolicyAction.DENY, 
    priority=100,  # Higher priority rules evaluated first
)
```

## Policy Languages Supported

- YAML
- OPA/Rego
- Cedar

## Key Design Principle

The toolkit explicitly contrasts itself from prompt guardrails: "This is not a prompt guardrail or content moderation tool. It governs agent actions, not LLM inputs/outputs."

## gdev Relevance

Microsoft's toolkit is binary (allow/deny) without an "ask" outcome. gdev's ternary model (allow/deny/ask) is more nuanced, adding the human-in-the-loop escalation that Microsoft's toolkit leaves to the agent framework. The priority-based rule ordering pattern is directly applicable to gdev's multi-rule evaluation.
