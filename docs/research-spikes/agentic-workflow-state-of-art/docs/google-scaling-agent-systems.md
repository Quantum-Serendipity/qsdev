# Google Research: Towards a Science of Scaling Agent Systems

- **Source URL**: https://research.google/blog/towards-a-science-of-scaling-agent-systems-when-and-why-agent-systems-work/
- **Paper URL**: https://arxiv.org/abs/2512.08296
- **Retrieved**: 2026-03-15
- **Note**: Content reconstructed from multiple search results and summaries.

## Overview

Google Research and MIT's large-scale controlled evaluation of 180 agent configurations, providing the first quantitative scaling principles for AI agent systems. Published December 2025.

## Study Design

Evaluated five canonical agent architectures:
1. **Single-Agent** — baseline
2. **Independent Multi-Agent** — agents work separately, no coordination
3. **Centralized Multi-Agent** — one coordinator directs all agents
4. **Decentralized Multi-Agent** — agents communicate peer-to-peer
5. **Hybrid Multi-Agent** — combination of centralized and decentralized

Tested across three LLM families: OpenAI GPT, Google Gemini, Anthropic Claude.

## Key Findings

### Performance by Task Type
- **Parallelizable tasks** (e.g., financial reasoning): Centralized coordination improved performance by **+81%** over single agent
- **Sequential tasks** (e.g., PlanCraft): Multi-agent coordination **degraded** performance by **-70%**

### Error Amplification
- Independent agents amplify errors **17.2x**
- Centralized coordination contains error amplification to **4.4x**

### Saturation Effects
- Coordination yields diminishing or negative returns once single-agent baselines exceed ~45%
- More agents don't always mean better outcomes

### Predictive Model
A predictive model identifies the optimal architecture for **87%** of unseen tasks.

## Key Conclusions

1. Multi-agent coordination dramatically improves performance on parallelizable tasks but degrades it on sequential ones
2. The choice of architecture must match the task structure
3. "More agents" often hits a ceiling and can even degrade performance if not aligned with specific task properties
4. The research challenges the widespread assumption that more agents always lead to better performance

## Practical Implications

- Every additional agent means another LLM call — if agents wait on each other, response times grow quickly
- Latency and cost are red flags
- When multiple agents don't coordinate well, or when memory management and system structure aren't robust, problems cascade quickly — derailing 40-80% of implementations
