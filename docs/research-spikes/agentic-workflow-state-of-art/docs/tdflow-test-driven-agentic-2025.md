# TDFlow: Agentic Workflows for Test Driven Software Engineering

- **Source URL**: https://arxiv.org/html/2510.23761v1
- **Retrieved**: 2026-03-15
- **Authors**: (Published October 2025, updated January 2026)

## Overview

TDFlow is a test-driven agentic workflow that achieves near-human-level test resolution by decomposing software engineering into four specialized sub-agents operating in an iterative test-feedback loop.

## Architecture: Four Sub-Agents

1. **Explore Files**: Proposes patches by analyzing repository structure and test failures. Read-only access (view files, search keywords, explore hierarchy). Receives previous failed patches and debugging reports.

2. **Revise Patch**: Fixes patches with incorrect context lines. Only adjusts surrounding context — does not modify inserted code.

3. **Debug One**: Uses restricted debugger (step, next, return, continue, breakpoints, variable inspection) to trace individual test failures. Produces diagnostic reports identifying root causes.

4. **Generate Tests**: Creates reproduction tests from issue descriptions. Validates tests before submission.

## Benchmark Results

### SWE-Bench Lite (Human-Written Tests)
| System | Pass Rate | Cost/Issue |
|--------|-----------|------------|
| OpenHands | 47.8% | $1.32 |
| ExpeRepair | 48.6% | $0.84 |
| SWE-Agent | 49.0% | $0.89 |
| Agentless | 61.0% | $0.53 |
| **TDFlow** | **88.8%** | **$1.51** |

### SWE-Bench Verified (500 instances)
- Human-written tests: **94.3%** pass rate
- LLM-generated tests: 68.0% pass rate
- LLM-only (0 Bad Test Rate): 93.3% pass rate

## Test-Driven Feedback Loop

1. Run failing tests to capture error messages
2. Explore Files proposes a global patch
3. Apply patch and re-run all tests
4. For each failing test, Debug One generates analysis
5. Aggregate reports fed back to Explore Files
6. Repeat until all tests pass or iteration limit reached

## Critical Finding: Test Quality as Bottleneck

When given correct tests (Bad Test Rate = 0), both human and LLM tests achieve ~93-94% pass rates. The 68% → 94.3% gap reveals that test generation — not test resolution — is the limiting factor.

## Cost Analysis

- Human-written tests: $1.51/issue
- Test generation: $2.83/issue additional (total $4.12)
- Test generation costs 2.8x more with lower success rates

## Scaling with Iterations

Performance improves with iterations but shows diminishing returns after 5-10 attempts. Cost-benefit suggests optimal stopping points exist.

## Advantages Over Monolithic Agents

1. Reduced cognitive load per sub-agent
2. Context efficiency — minimal sufficient context per task
3. Parallel scaling — Debug One agents run concurrently

## Limitations

- Fixed workflow cannot adapt to unconventional problems
- No early-stopping for unsolvable tests
- More complex infrastructure requirements
- Some SWE-Bench instances incompatible with per-test debuggability

## Significance

Strongest evidence to date that test-driven development is the single most effective quality technique for agentic coding. Human-level test resolution achieved; remaining frontier is test generation.
