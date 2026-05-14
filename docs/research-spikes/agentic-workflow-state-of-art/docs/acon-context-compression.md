# ACON: Optimizing Context Compression for Long-Horizon LLM Agents

- **Source URLs**:
  - https://arxiv.org/abs/2510.00615
  - https://openreview.net/forum?id=7JbSwX6bNL
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Overview

ACON (Agent Context Optimization) is a unified framework that optimally compresses both environment observations and interaction histories into concise yet informative condensations for long-horizon agent tasks.

## The Problem

In long-horizon tasks, context length grows as agents accumulate histories of actions and observations. This raises costs and reduces efficiency. Prior work on context compression mostly focused on single-step tasks or narrow applications.

## How It Works

1. **Compression guideline optimization**: Given paired trajectories where full context succeeds but compressed context fails, capable LLMs analyze the causes of failure
2. **Natural language space**: Compression guidelines are expressed and optimized in natural language (not model parameters)
3. **Gradient-free**: Requires no parameter updates — directly usable with closed-source or production API models
4. **Distillation**: Optimized compressors can be distilled into smaller models for cost-efficient deployment

## Results

- **Memory reduction**: 26-54% peak token reduction
- **Task performance**: Largely preserved
- **Distillation quality**: Preserves over 95% of accuracy when distilled into smaller compressors
- **Smaller LM enhancement**: Up to 46% performance improvement for smaller LMs as long-horizon agents

## Tested On

AppWorld, OfficeBench, and Multi-objective QA benchmarks.

## Key Insight

The right compression strategy depends on the task — what's safe to discard varies by domain. ACON learns domain-specific compression guidelines by analyzing failure cases, making it adaptive rather than one-size-fits-all.
