# RE-Bench: Evaluating Frontier AI R&D Capabilities Against Human Experts
- **Source**: https://arxiv.org/html/2411.15114v1 and https://metr.org/blog/2024-11-22-evaluating-r-d-capabilities-of-llms/
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted summary from METR/arxiv

## Overview

RE-Bench is a benchmark for measuring the performance of humans and frontier model agents on ML research engineering tasks. Contains 7 challenging, open-ended ML research engineering environments with data from 71 8-hour attempts by 61 distinct human experts.

## Key Results

### Short time horizons: AI wins
- Best AI agents achieve 4x higher score than human experts at 2-hour budget
- AI excels at rapid implementation and iteration within constrained timeframes

### Long time horizons: Humans win
- After 2 hours, AI models hit a plateau
- Humans continued to improve with more time
- With 8 hours, humans clearly superior

### Human baseline
- 82% of expert attempts achieved non-zero score
- 24% matched or exceeded strong reference solutions

## Models Tested
- Claude 3.5 Sonnet
- o1-preview
- Claude 3.7 Sonnet (preliminary evaluation)

## Significance

RE-Bench specifically measures the time horizon over which AI agents can productively work — a critical dimension for understanding agent autonomy limits. The plateau at 2 hours suggests current agents struggle with:
- Deep problem understanding requiring extended exploration
- Creative problem-solving that benefits from extended reflection
- Complex debugging requiring systematic investigation over time

Published at ICML 2025.
