# AlphaCode: Competition-Level Code Generation

- **Source URL**: https://deepmind.google/blog/competitive-programming-with-alphacode/
- **Retrieved**: 2026-03-15
- **Authors**: DeepMind
- **Published**: Science, 2022

## Approach: Massive Sampling + Filtering + Clustering

1. **Generation**: Generate millions of diverse program candidates per problem (orders of magnitude more than prior work)
2. **Filtering**: Execute on example tests from problem description, removing ~99% of samples that fail
3. **Clustering**: Execute remaining programs on generated test inputs, group by output equivalence classes
4. **Selection**: Pick one sample from each of the 10 largest clusters for submission

## Results

- Achieved top 54.3% average ranking on Codeforces competitions
- Approximately median competitor level
- First AI system to reach competitive programming performance

## Key Insight: Ensemble at Scale

AlphaCode demonstrates that brute-force sampling + intelligent filtering can achieve quality far beyond any single generation. The filtering step (execution against tests) serves as the critical quality gate — without it, the millions of samples would be useless.

## Cost

Extremely compute-intensive: millions of samples per problem. Not practical for real-time coding assistance, but establishes the theoretical power of sampling + verification.

## Significance

Proved that the combination of (1) massive parallel sampling, (2) execution-based filtering, and (3) clustering for diversity produces quality gains that no single-shot approach can match. The principle — generate many, filter by execution — scales down to practical best-of-N approaches in coding agents.
