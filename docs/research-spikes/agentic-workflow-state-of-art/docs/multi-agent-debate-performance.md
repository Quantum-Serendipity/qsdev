# Multi-Agent Debate: Performance, Efficiency, and Scaling Challenges
- **Sources**:
  - https://d2jud02ci9yv69.cloudfront.net/2025-04-28-mad-159/blog/mad/ (ICLR Blogposts 2025)
  - https://arxiv.org/html/2510.12697v1
  - https://arxiv.org/html/2512.20845
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from search results

## How Multi-Agent Debate Works

Multiple LLM agents independently generate initial answers in parallel. Over several rounds, agents review other agents' answers and incorporate collective feedback to refine their answers. Refined answers are aggregated to form the final answer.

## Judge Architecture

A judge agent manages the debate process:
- **Discriminative mode**: evaluating rounds for correctness
- **Extractive mode**: extracting the final solution
- **Adjudicating**: resolving persistent disagreement

## Agent Roles and Personas

Agents may be assigned distinct roles: "affirmative" and "negative", "angel" and "devil" personas, or domain-specific profiles. Role-playing stimulates critical thinking and divergent feedback.

## Key Performance Finding

Both multi-agent debate and self-consistency achieve significant improvements over standard prompting, though multi-agent debate significantly underperforms simple self-consistency using majority voting in many cases.

## Efficiency Concerns

- Deploying heterogeneous agents (different foundation models) yields higher accuracy on tasks like GSM-8K
- Sparse communication topologies limit which agents see each other's outputs to reduce token cost
- Dynamic debating graphs: each agent only interacts with most beneficial peers
- Conditional participation modules can cut token costs by up to 94.5%

## Multi-Agent Reflexion (MAR) — 2025

Structured multi-agent extension incorporating:
- Diverse reasoning personas
- Judge model synthesizing critiques into unified reflections
- Separation of acting, diagnosing, critiquing, and aggregating processes
- Reduces shared blind spots

## When Debate Helps vs. Hurts

- Helps: Tasks requiring diverse perspectives, creative problem-solving, fact-checking
- Hurts: Simple tasks where single-agent + self-consistency is cheaper and equally effective
- Key insight: debate's value comes from diversity of reasoning, not just redundancy
