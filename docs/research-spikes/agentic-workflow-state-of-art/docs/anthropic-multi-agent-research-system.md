# Anthropic: How We Built Our Multi-Agent Research System

- **Source URL**: https://www.anthropic.com/engineering/multi-agent-research-system
- **Retrieved**: 2026-03-15
- **Note**: Content reconstructed from multiple search results and summaries; not a direct page capture.

## Overview

Anthropic's engineering blog post (June 2025) describing how they built a multi-agent research system that outperformed single-agent Claude Opus 4 by 90.2% on internal research evaluation tasks.

## Architecture: Orchestrator-Worker Pattern

The system uses a lead agent (Claude Opus 4) coordinating specialized subagents (Claude Sonnet 4).

### Lead Researcher Agent
When a user submits a query, the Lead Researcher:
1. Analyzes the query
2. Develops a strategy
3. Records the plan in memory
4. Spawns subagents to explore different aspects simultaneously

### Subagent Design
Each subagent needs:
- An objective
- An output format
- Guidance on tools and sources to use
- Clear task boundaries

### Parallelization
- Lead agent spins up 3-5 subagents in parallel (not serially)
- Subagents use 3+ tools in parallel
- These changes cut research time by up to 90% for complex queries

### Extended and Interleaved Thinking
- Extended thinking allows the Lead Researcher to write out their reasoning before acting
- Subagents plan their steps, then after receiving tool outputs, use interleaved thinking to evaluate results, spot gaps, and refine next queries

## Performance Results

### BrowseComp Evaluation
Three factors explained 95% of the performance variance:
1. **Token usage** (80% of variance)
2. Number of tool calls
3. Model choice

### Token Economics
- Agents typically use ~4x more tokens than chat interactions
- Multi-agent systems consume ~15x more tokens than chats
- Multi-agent systems work mainly because they help spend enough tokens to solve the problem

### 90.2% Improvement
When asked complex tasks (e.g., identify all board members of Information Technology S&P 500 companies), the multi-agent system found correct answers by decomposing tasks for subagents, while single-agent failed with slow, sequential searches.

## Key Lessons

### Prompt Engineering
Studied how skilled humans approach research tasks and encoded these strategies in prompts:
- Decomposing difficult questions into smaller tasks
- Carefully evaluating the quality of sources
- Adjusting search approaches based on new information
- Recognizing when to focus on depth versus breadth

### Tool Design
Agent-tool interfaces require the same careful design attention as human-computer interfaces:
- Poor tool descriptions can send agents down completely wrong paths
- Each tool needs distinct purposes and clear descriptions
- Tools help agents select appropriate approaches for specific tasks

### Evaluation
Used an LLM judge that evaluated each output against criteria in a rubric:
- Factual accuracy
- Citation accuracy
- Completeness
- Source quality
- Tool efficiency

## Subagent Benefits
- Facilitate compression by operating in parallel with their own context windows
- Explore different aspects simultaneously
- Condense most important tokens for the lead research agent
- Provide separation of concerns — distinct tools, prompts, and exploration trajectories
- Reduce path dependency
- Enable thorough, independent investigations
