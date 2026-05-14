# The "Think" Tool: Enabling Claude to Stop and Think

- **Source URL**: https://www.anthropic.com/engineering/claude-think-tool
- **Retrieved**: 2026-03-15
- **Authors**: Anthropic Engineering
- **Published**: 2025

## Overview

The "think" tool gives Claude a dedicated space for structured thinking during complex agentic tasks. It creates an explicit thinking step as a tool call, allowing the model to pause and reason before acting.

## How It Works

The think tool is implemented as a standard tool call with a single "thought" parameter. When Claude encounters a complex decision point during a multi-step task, it can invoke the think tool to:
- Check if it has all necessary information
- Validate steps and reason about key facts
- Navigate policy-heavy environments
- Make sequential decisions where mistakes are costly

## Benchmark Results (tau-bench)

**Airline Domain**:
- Baseline: 0.370 pass rate
- Think tool + optimized prompt: **0.570 pass rate** (54% relative improvement)

**Retail Domain**:
- Baseline: 0.783 pass rate
- Think tool alone: **0.812 pass rate**

## When to Use

- Complex tool orchestration requiring careful analysis
- Policy-heavy environments with detailed guidelines
- Sequential decisions where each step builds on previous ones
- Long chains of tool calls needing consistency

## Relationship to Extended Thinking

The think tool differs from extended thinking mode:
- Extended thinking: happens before the first response token, scales reasoning budget
- Think tool: happens mid-response as a tool call, provides structured pause points during multi-step execution

Both can be used together — extended thinking for initial planning, think tool for mid-execution deliberation.

## Significance for Claude Code

Demonstrates that explicit thinking pauses during agentic execution measurably improve quality, especially for policy compliance and multi-step consistency. The think tool is trivially implementable (just a tool schema) with no infrastructure cost.
