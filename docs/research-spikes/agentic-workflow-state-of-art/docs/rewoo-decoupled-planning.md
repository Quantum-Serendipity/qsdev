# ReWOO: Decoupling Reasoning from Observations for Efficient Augmented Language Models
- **Sources**:
  - https://arxiv.org/abs/2305.18323
  - https://www.ibm.com/think/topics/rewoo
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from search results

## Core Innovation

ReWOO breaks from the think-act-observe pattern by decoupling reasoning from external observations. The model generates a complete reasoning plan before executing any tool interactions, fundamentally differing from ReAct which interleaves reasoning and tool execution.

## Three Modules

1. **Planner**: Uses predictable LLM reasoning to create a solution blueprint — sequential tuples (Plan, #E), where Plan is a descriptive message and #E is a token for storing evidence from the corresponding step
2. **Worker**: Interacts with environment through tool calls
3. **Solver**: Examines all plans and evidence to develop final solution

## Efficiency

- Instead of LLM call after each tool use (ReAct), ReWOO makes just 2 LLM calls (plan + integrate), regardless of number of tools
- Averaging over six public benchmarks: 64% token reduction with 4.4% absolute accuracy gain
- 5x token efficiency and 4% accuracy improvement on HotpotQA specifically

## Robustness

ReWOO demonstrates robustness under tool-failure scenarios — since the plan is generated upfront, individual tool failures don't cascade through the reasoning process as badly as in interleaved approaches.
