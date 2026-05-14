# OpenAI o1/o3 Reasoning Models: Evidence for Extended Thinking

- **Sources**: OpenAI blog posts, benchmark comparisons, third-party evaluations
- **Retrieved**: 2026-03-15

## Architecture

o1 and o3 use "private chain-of-thought" — extended reasoning sequences not shown to users. o3's "simulated reasoning" allows the model to consider its own intermediate results and adjust reasoning as it progresses.

## Benchmark Improvements (o3 vs o1)

| Benchmark | o1 | o3 | Improvement |
|-----------|-----|-----|-------------|
| AIME 2024 (math) | 74.3% | 91.6% | +17.3 pts |
| GPQA Diamond (science) | 78% | 83.3% | +5.3 pts |
| ARC-AGI (low compute) | — | 76% | — |
| ARC-AGI (high compute) | — | 88% | Beyond human (85%) |

## Key Evidence

- o3 makes 20% fewer major errors than o1 on difficult real-world tasks
- Strongest gains in programming, business/consulting, creative ideation
- Performance scales with thinking budget — more tokens = better answers
- Extended reasoning + tool use combination is particularly powerful

## Thinking Budget Scaling

Claude's approach uses trigger words for budget:
- "think" < "think hard" < "think harder" < "ultrathink"
- Each level allocates progressively more thinking budget

## Implications

Extended thinking represents the clearest evidence of "compute for quality" trade-off:
- More thinking tokens → measurably better answers
- Effect is strongest for hard problems (math, complex coding, analysis)
- Simple problems don't benefit (and can be hurt by "overthinking")
- Budget should be adaptive — scale thinking to problem difficulty
