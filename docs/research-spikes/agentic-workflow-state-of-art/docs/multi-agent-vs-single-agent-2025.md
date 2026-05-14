# Multi-Agent vs Single-Agent Effectiveness: Benchmark Studies (2025)
- **Sources**:
  - https://arxiv.org/abs/2505.18286 (Single-agent or Multi-agent? Why Not Both?)
  - https://research.google/blog/towards-a-science-of-scaling-agent-systems-when-and-why-agent-systems-work/
  - https://arxiv.org/html/2509.10769v1 (AgentArch benchmark)
- **Retrieved**: 2026-03-15
- **Note**: Composite summary from multiple 2025 research papers

## Key Finding: Benefits Diminish with Model Capability

"The benefits of MAS over SAS diminish as LLM capabilities improve." — Hybrid approach improved accuracy by 1.1-12% while reducing deployment costs by up to 20%.

## Google Research: Scaling Agent Systems

Tested five canonical architectures (1 single-agent, 4 multi-agent variants) across four benchmarks.

### Key findings:
- Parallelizable tasks (financial reasoning): centralized coordination improved performance by 80.9% over single agent
- Sequential reasoning tasks: every multi-agent variant degraded performance by 39-70%
- Empirical threshold: ~45% single-agent accuracy — above this, adding more agents yields diminishing or negative returns
- Effective team sizes limited to ~3-4 agents
- Communication overhead grows super-linearly (exponent 1.724)

## AgentArch Benchmark (September 2025)

Tested 18 configurations: single vs multi-agent, ReAct vs function calling, thinking tools, memory management styles.

Finding: Even state-of-the-art LLMs struggle to maintain reliable performance on complex enterprise workflows in ANY setup.

## When Multi-Agent Works

- Highly parallelizable tasks where independent sub-problems can be solved concurrently
- Tasks requiring diverse expertise that benefits from specialized agent roles
- When base model capability is below the ~45% threshold

## When Single-Agent is Better

- Sequential reasoning tasks
- When base model is highly capable
- When communication overhead exceeds parallelism benefits
- When task requires coherent long-range planning
