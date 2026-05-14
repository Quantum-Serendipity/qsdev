# Dissecting the SWE-Bench Leaderboards: Profiling Agent Architectures
- **Source**: https://arxiv.org/html/2506.17208v2
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv HTML page

## Architecture Taxonomy

Three critical dimensions:

### Workflow Authoring
- Human-authored workflows: predefined repair steps executed sequentially
- Emergent workflows: fully autonomous, driven by agents

### Control Flow Autonomy
1. Fixed Execution: deterministic, sequential
2. Scaffolded Execution: human-provided structure with local autonomy
3. Emergent Autonomy: agents determine execution flow from feedback

### Agent Configuration
- No agents, single agents, or multiple agents

## Performance

No single architecture consistently achieves SOTA. Multiple paradigms prove effective.

- SWE-Bench Verified: Median precision 46.9%, max 75.2%
- SWE-Bench Lite: Median 31.5%, max 60.0%

### Top-Performing Models
- Claude 3.5 Sonnet dominates multiple top entries
- Hybrid approaches combining Claude + Gemini 2.5 Pro
- Bytedance achieved 75.2% on Verified using multi-LLM combinations

### Submitter Performance
- Company submissions significantly outperform academia (p = 0.0001)
- Medium/large companies: 63.9-75.2%
- Academia median: ~31.5%

## Correlation with High Performance

1. Proprietary LLM access (Claude 3.5+, o1-series)
2. Multi-LLM ensembles
3. Scaffolded execution with rich context pipeline
4. Sophisticated multi-technique localization
5. Iterative refinement loops with feedback
6. Industry backing (resource advantages)

### Notable Outliers
- Agentless (academia): 50.8% through agent-free reasoning
- SWE-Agent (Princeton): 66%, influential open-source design
- Refact.ai (small company): 74.4%

## Temporal Evolution
- Initial (2023): ~20% resolution
- June 2024: 35-40%
- June 2025: 75.2% (Bytedance)
