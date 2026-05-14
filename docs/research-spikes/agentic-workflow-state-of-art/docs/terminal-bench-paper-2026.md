# Terminal-Bench: Benchmarking Agents on Hard, Realistic Tasks in Command Line Interfaces
- **Source**: https://arxiv.org/html/2601.11868v1
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv HTML page. Published as ICLR 2026 conference paper.

## Overview

Terminal-Bench 2.0 comprises 89 carefully curated tasks in containerized terminal environments. Each task includes: (1) a containerized environment initialized with relevant packages and files, (2) an instruction describing the task to be completed, (3) a set of tests to verify completion, and (4) a reference solution manually written to solve this task.

The framework emphasizes outcome-driven evaluation rather than process validation. Tasks test final container states through automated verification rather than scrutinizing specific command sequences.

## Task Categories

Tasks span diverse domains reflecting real professional workflows:
- Software engineering (largest category)
- System configuration and legacy system management
- Research paper reimplementation
- Database migration and optimization
- Machine learning model training
- Kernel compilation and low-level programming

Distribution: 48.6% of tasks estimated at under one hour for domain experts, while 71.6% require 1-24 hours for junior engineers.

## Evaluation Scale

- 16 models tested (closed-source and open-weight)
- 6 agents evaluated: Claude Code, Codex CLI, Gemini CLI, OpenHands, Mini-SWE-Agent, Terminus 2
- 32,155 total trials (minimum 5 runs per model-agent combination)
- Terminus 2 created as a neutral testbed using only Bash commands and headless terminal

## Key Performance Results

### Top performers (March 2026):
- Codex CLI + GPT-5.2: 62.9%
- Gemini 3.1 Pro: 78.4% (on updated leaderboard)
- Terminus 2 + Claude Opus 4.5: 57.8%
- Terminus 2 + Gemini 3 Pro: 56.9%
- Claude Opus 4.6: 74.7% (up from 65.4% in January)
- Claude Sonnet 4.5: 50.0%

### Key finding: Agent scaffold matters enormously
"Codex CLI resolution rate increases by 52% when using GPT-5.2 instead of GPT-5-Nano, while Gemini-2.5-Pro sees a 17% increase in resolution rate when paired with Terminus 2 instead of OpenHands"

## Failure Analysis

Three primary failure classes (using LLM-as-judge annotation with 90% agreement against human labels):

### Execution errors (dominant for frontier models):
- Strict instruction adherence failures
- Tool misuse or incorrect parameter selection

### Coherence errors (balanced for weaker models):
- Step repetition
- Context loss during long trajectories
- Unaware of termination conditions
- Premature termination
- Reasoning-action mismatches

### Verification errors:
- Insufficient or absent outcome verification
- Weak verification approaches

## Command-Level Error Analysis

Primary failure categories (across 3,800 sampled failures):
- Executable not installed/not in PATH: 24.1%
- Running executable failures: 9.6%
- Incorrect file/syntax handling
- Environment variable and dependency issues

Error rates ranged from 9.2% (Grok 4) to 26.7% (GPT-OSS-120B).

## Cost & Efficiency

- Task completion costs: $1-$100 per task depending on model
- Most attempts: <20 minutes, <25 model calls
- Extreme cases: up to 2 hours, ~100M tokens
- Critical finding: "there is essentially no correlation between the number of average turns per trial and model success rates"
- Higher token counts don't necessarily improve performance

## Temporal Trend

Within eight months, state-of-the-art performance nearly doubled. Current benchmark may saturate within ~1 year.
