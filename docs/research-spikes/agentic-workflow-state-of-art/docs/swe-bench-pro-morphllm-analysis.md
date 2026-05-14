# SWE-Bench Pro: Why 46% Beats 81% — Agent Scaffold Analysis
- **Source**: https://www.morphllm.com/swe-bench-pro and https://www.morphllm.com/best-ai-model-for-coding
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted summary from MorphLLM blog posts

## Key Finding: Scaffold > Model

The gap between a good and bad agent scaffold is 22 points on SWE-Bench Pro.
The gap between the top two coding models is 0.8 percentage points.
Architecture matters approximately 27x more than model choice at the frontier.

"A mid-tier model in a great harness beats a frontier model in a bad one."

## SWE-Bench Pro vs SWE-Bench Verified

### Why SWE-Bench Pro is harder:
- SWE-Bench Verified: 161 of 500 tasks require only 1-2 lines of change
- SWE-Bench Pro: Every task requires at least 10 lines, 100+ tasks need 100+ lines
- Average 4.1 files changed per task (cross-file coordination required)
- Built from GPL/copyleft repos and private codebases to reduce contamination

### Score comparison:
- SWE-Bench Verified top scores: >70% (multiple models)
- SWE-Bench Pro top scores: 23.3% (GPT-5), 23.1% (Claude Opus 4.1)

## Data Contamination Problem

OpenAI audit found every frontier model tested (GPT-5.2, Claude Opus 4.5, Gemini 3 Flash) could reproduce verbatim gold patches for certain SWE-Bench Verified tasks. OpenAI has stopped reporting Verified scores and recommends SWE-Bench Pro instead.

## Implications

Context retrieval, file indexing, and agent orchestration are the multiplier — not raw model intelligence. The harness, not the model, drives the remaining variance in coding agent quality.
