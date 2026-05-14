# Evaluation and Benchmarking of Agentic AI Systems

## Overview

This report surveys how agentic AI systems are evaluated and benchmarked as of early 2026, what the evaluation data reveals about quality drivers, and what the leading systems do differently. The focus is on extracting actionable insights for Claude Code — an AI coding assistant operating in the terminal.

The central finding across all benchmarks is that **the agent scaffold matters far more than the underlying model**. On SWE-Bench Pro, swapping the harness produces a 22-point swing in scores while swapping the model produces a 0.8-point swing. On Terminal-Bench, the gap between a good and bad scaffold is similarly decisive. This means that improvements to Claude Code's architecture, context engineering, tool use patterns, and verification strategies will yield far more quality improvement than waiting for a better base model.

---

## 1. SWE-bench and SWE-bench Verified

### The Benchmark

SWE-bench presents real GitHub issues from popular Python repositories and asks agents to generate patches that resolve the issue while passing existing tests. SWE-bench Verified is a human-validated subset of 500 instances created in collaboration with OpenAI, filtered for solvable problems with clear descriptions and correct test patches.

### Current Leaderboard (Early 2026)

| System | SWE-bench Verified Score |
|--------|-------------------------|
| Claude Opus 4.5 + scaffold | 80.9% |
| Claude Opus 4.5 + Live-SWE-agent | 79.2% |
| Gemini 3 Pro + Live-SWE-agent | 77.4% |
| ByteDance (top industrial entry) | 75.2% |
| Anthropic (Claude direct) | 73.2% |

The leaderboard now contains 77 evaluated models with an average score of 62.2%.

### Architecture Analysis of Top Performers

A systematic study of all SWE-bench submissions (arxiv 2506.17208) identified seven architectural groups:

- **G1-G3 (Fixed Execution)**: Human-authored, predetermined execution paths. Traditional "generate-and-validate." Characterizes earlier APR systems.
- **G4-G5 (Scaffolded Execution)**: Human-provided structural framework with local autonomy. System operates within defined stages (localization, generation, verification) but makes independent decisions within boundaries.
- **G6-G7 (Emergent Autonomy)**: Fully agentic systems with dynamic, reactive execution paths. No predefined control structures.

**Key findings**:
- No single architecture dominates. Multiple design paradigms can be effective.
- Claude 3.5 Sonnet is the most deployed model across top entries. Multiple LLM variants are frequently combined.
- Industry submissions significantly outperform academic ones (p < 0.02).
- Scaffolded execution (G4-G5) balances guidance with flexibility and increasingly dominates top positions.
- The most effective systems use retrieval-based localization combined with LLM reasoning, then iterate verification results back into generation loops.

### The Agentless Approach

Agentless challenged complexity trends with a simple three-phase approach: localization, repair, and patch validation — all in ~700 lines. Achieved competitive results at $0.34 per issue. Demonstrates that simplicity can rival complex agentic architectures.

### Moatless Tools: MCTS for Code Repair

Moatless uses Monte Carlo tree search with a custom reward function to explore different solution paths systematically. With Claude 3.5 Sonnet, achieves 39% solve rate at $0.14 per issue — demonstrating that search-based approaches can be cost-efficient alternatives to agentic iteration.

### Data Contamination Crisis

OpenAI's audit found that every frontier model (GPT-5.2, Claude Opus 4.5, Gemini 3 Flash) could reproduce verbatim gold patches for certain SWE-bench Verified tasks. OpenAI has stopped reporting Verified scores. This has driven the creation of:

- **SWE-bench Pro**: Every task requires 10+ lines, average 4.1 files changed. Top scores plummet to ~23%.
- **SWE-bench Live**: Monthly-refreshed tasks from post-training-cutoff issues. Best system achieves ~23% on non-SWE-bench instances.
- **SWE-Rebench**: Automated pipeline for contamination-free evaluation.

### Implications for Claude Code

1. **Scaffolded execution is the winning pattern** — provide structure (localization → generation → verification) while allowing autonomy within stages.
2. **Retrieval-based localization is critical** — the ability to find relevant code (file indexing, semantic search) directly determines patch quality.
3. **Verification loops matter** — top systems iterate test results back into generation, not just generate-once.
4. **The 22-point scaffold gap** means architecture improvements to Claude Code are the highest-leverage investment.

---

## 2. Code Generation Benchmarks

### Benchmark Landscape

| Benchmark | Tasks | Status | What It Measures |
|-----------|-------|--------|------------------|
| HumanEval | 164 | Nearly saturated | Basic function-level code generation |
| MBPP | 378 | Nearly saturated | Simple programming tasks |
| LiveCodeBench | 1,055+ | Active | Contamination-free competitive programming |
| BigCodeBench-Hard | 148 | Challenging | Complex, multi-library tasks |
| HumanEval Pro / MBPP Pro | Extended | Active | Self-invoking code, compositional reasoning |

### Performance Across Benchmarks

**HumanEval**: 113 of 164 tasks solved by ALL six models tested. Claude Sonnet-4 had only 2 failures. This benchmark is effectively solved — top models exceed 90% pass@1. It no longer differentiates frontier models.

**LiveCodeBench**: Intermediate difficulty. 43 tasks universally solved, 35 never solved. Claude Sonnet-4 strongest with 54 failures out of 175 tasks. Contamination-free by design (problems from competitive programming after training cutoff).

**BigCodeBench-Hard**: Only 14 of 148 tasks solved by all models. 76 tasks consistently failed by ALL models (68-77% failure rate). This is where the frontier actually lies.

### Four Recurring Failure Patterns

1. **Wrong problem mapping**: Models misclassify tasks into familiar problem categories rather than understanding the actual problem.
2. **Flawed algorithm design**: Correct general approach but missing necessary components (e.g., handling non-monotonic trends).
3. **Edge case mishandling**: Code fails on boundary scenarios (e.g., only iterating top-level files instead of recursive search).
4. **Formatting mistakes**: Correct logic but wrong output format — strict output requirements defeat otherwise-correct solutions.

### Key Insight

Solution code complexity shows minimal correlation with failure rates. The hard problems are hard because they require understanding subtle requirements, not because they require complex code. This means **better problem comprehension and specification parsing are more valuable than better code generation**.

### Implications for Claude Code

1. **Edge case awareness** should be explicitly prompted/checked — it's a consistent failure mode.
2. **Verification against tests** catches formatting and edge case failures that the model cannot self-detect.
3. **HumanEval-level tasks are not meaningful differentiators** — focus quality efforts on multi-file, multi-step tasks.
4. **Problem comprehension** (reading the issue carefully, understanding constraints) is more limiting than code generation ability.

---

## 3. GAIA Benchmark

### What It Tests

GAIA (General AI Assistants) contains 450 questions with unambiguous answers, requiring different levels of tooling and autonomy: reasoning, multi-modality, web browsing, and tool-use proficiency. Three difficulty levels test increasing complexity.

### Current Standings

| System | Score |
|--------|-------|
| H2O.ai h2oGPTe Agent | 75% (first C grade on test set) |
| Manus AI | ~75% |
| OpenAI Deep Research | ~48% |
| Writer Action Agent | 61% (Level 3) |
| GPT-5 Mini | 44.8% |
| Claude 3.7 Sonnet | 43.9% |

### What Top Performers Do Differently

- **Modular architecture**: Top systems use specialized components (planner, executor, memory) rather than monolithic approaches.
- **Cost efficiency through smart model selection**: KGoT architecture solves 57 GAIA tasks at ~$5 total with GPT-4o mini, vs. earlier systems solving 29 tasks at $187 with GPT-4o. A 2x improvement in task completion at 38x lower cost.
- **Robust web browsing and tool use**: GAIA heavily tests ability to research, retrieve documents, perform image recognition, and format answers precisely.

### Scaling Results on GAIA

From the test-time compute scaling paper:
- Baseline: 55.76%
- Best-of-N (BoN): 63.03% (+7.3 points — significant)
- Multi-model collaboration (GPT-4.1 + Claude + Gemini): 74.55% pass@4

BoN excels on simpler tasks; step-wise exploration (BoN-wise) is better for Level 3 (hardest) tasks.

### Implications for Claude Code

1. **Tool use proficiency is a differentiator** — GAIA rewards agents that can effectively use tools, not just reason.
2. **Best-of-N sampling produces large gains** when a verifier is available.
3. **Multi-model collaboration** achieves the best absolute scores, but is expensive.

---

## 4. WebArena / VisualWebArena

### Performance Trajectory

WebArena has seen dramatic improvement: from 14% to ~60% success rate in two years. The current best is IBM CUGA at 61.7% (Feb 2025) on the full benchmark. Gemini 2.5 Pro reaches 54.8%.

However, harder variants show the ceiling:
- WebChoreArena (realistic tedious tasks): 37.8%
- VisualWebArena (multimodal): significantly below human performance

### The "Standard Model" Architecture

The jump in WebArena performance came from convergence on a modular architecture:
1. **Planner**: Translates goals into step-by-step plans, selects tools, sets priorities
2. **Executor**: Performs each step, calls APIs, manages errors, reports back
3. **Memory**: Short-term (prompt/hidden state) and long-term (key-value scratchpad)

### WebArena Verified

The original WebArena had significant evaluation issues. WebArena Verified audited all 812 tasks, repaired misaligned evaluations, clarified ambiguous instructions, and replaced substring matching with type- and normalization-aware comparators. Reduced false-negative rate by 11.3 percentage points.

### Test-Time Scaling on WebArena (CATTS Paper)

| Approach | WebArena-Lite Score |
|----------|-------------------|
| N=1 (baseline) | 38.8% |
| N=10 (majority voting) | 43.2% |
| N=20 | 43.0% (no improvement) |
| CATTS (confidence-aware) | 47.9% (at 56% fewer tokens) |

**Critical insight**: Most steps have obvious correct actions. Sampling produces duplicates, not diversity. The key is identifying WHICH steps are uncertain and spending extra compute only there.

When arbitration overrides high-consensus votes (margin >0.7), tasks succeed only 35% vs. 46.9% without overrides — overthinking correct decisions actively hurts.

### Implications for Claude Code

1. **Planner-executor-memory is the proven architecture** for complex multi-step tasks.
2. **Confidence-aware compute allocation** — spend more thinking on uncertain decisions, not uniformly.
3. **Don't override high-confidence decisions** — overthinking degrades performance.
4. **Evaluation methodology matters** — WebArena Verified's 11.3pp false-negative reduction shows that benchmark design directly affects measured progress.

---

## 5. Real-World Coding Metrics

### Adoption Scale (2025)

- 24% of production code is now AI-written (29% US, 21% Europe)
- Code review agent adoption: 14.8% (Jan 2025) to 51.4% (Oct 2025)
- GitHub auto-reviewed 8M+ pull requests by April 2025

### Acceptance and Retention

- GitHub Copilot: 46% completion rate, ~30% acceptance rate
- 88% of accepted code retained in final submissions (production-ready, not just starting points)
- Copilot Chat code review: 70% comment acceptance rate

### Productivity Gains

- PRs per engineer: +113% (1.36 to 2.9)
- Cycle time: -24% (16.7 to 12.7 hours)
- PR turnaround: -75% (9.6 to 2.4 days)
- Task completion: 55% faster with Copilot (n=4,800)

### Quality Concerns

AI-generated code introduces significantly more issues than human code:
- **1.7x more total issues** per PR (10.83 vs 6.45)
- **1.4x more critical issues**
- **1.75x more logic/correctness errors**
- **1.64x more maintainability errors**
- **1.57x more security findings**

### Security Vulnerability Data

- 45% of AI code samples failed security tests (OWASP Top 10) — Veracode, 100+ models
- 62% contain design flaws or known vulnerabilities
- AI code blamed for 1 in 5 breaches (Aikido Security 2026)
- Specific weaknesses: 2.74x more XSS, 1.91x more insecure object references, 1.88x more improper password handling

### Developer Trust

- 46% actively distrust AI output accuracy
- Only 3% report high trust
- This trust deficit is rational given the data above

### AI Code Review Tool Performance

- Greptile: 82% bug catch rate (best)
- Cursor: 58%
- Traditional static analyzers: <20%
- AI review tools detect 42-48% of real-world runtime bugs — a significant leap over traditional tools

### Implications for Claude Code

1. **Verification is not optional** — AI code has measurably more defects. Every output needs checking.
2. **Security is the critical gap** — 45% failure rate on basic security tests is unacceptable for production code.
3. **Code review by AI is high-value** — 82% bug catch rate and 70% acceptance of AI review comments show this as a strong use case.
4. **The acceptance-retention signal** (88% of accepted code retained) suggests that when AI code is good, it's good enough to use directly.
5. **Productivity gains are real but come with quality debt** — faster cycle times mean more bugs shipped faster unless verification is built in.

---

## 6. Terminal/CLI Agent Benchmarks

### Terminal-Bench 2.0 (ICLR 2026)

The primary benchmark for terminal-based agents. 89 tasks in containerized environments covering software engineering, system configuration, research paper reimplementation, database migration, ML training, and kernel compilation.

**Scale**: 32,155 total trials across 16 models and 6 agents.

### Performance Results (March 2026)

| System | Score |
|--------|-------|
| Gemini 3.1 Pro (best scaffold) | 78.4% |
| Codex CLI + GPT-5.2 | 77.3% (62.9% on paper's evaluation) |
| Claude Opus 4.6 | 74.7% (up from 65.4% in January) |
| Claude Sonnet 4.5 | 50.0% |
| Claude Opus 4.1 | 46.5% |
| GPT-5 (base) | 43.8% |

**Key finding**: Performance nearly doubled in eight months. Current benchmark may saturate within ~1 year.

### Failure Mode Analysis

Three primary failure classes:

1. **Execution errors** (dominant for frontier models): Instruction adherence failures, tool misuse, incorrect parameters. This is the main bottleneck for the best models.
2. **Coherence errors** (dominant for weaker models): Step repetition, context loss during long trajectories, premature termination, reasoning-action mismatches.
3. **Verification errors**: Insufficient outcome verification, assuming success without checking.

**Command-level failures**: 24.1% due to executable not installed/not in PATH. 9.6% running executable failures.

### Cost and Efficiency

- Most tasks: <20 minutes, <25 model calls
- Extreme cases: up to 2 hours, ~100M tokens
- **No correlation between turns per trial and success** — more turns does not help
- **Higher token counts don't improve performance** — quality of reasoning matters, not volume

### tau-bench / tau2-bench

Evaluates conversational customer service agents across domains. Tests policy adherence, multi-turn conversation, and tool use in realistic scenarios with a dual-control environment.

### RE-Bench (METR)

Tests AI R&D capabilities on 7 open-ended ML research engineering environments.

**Key finding**: AI agents achieve 4x higher scores than human experts at 2-hour budgets but hit a plateau. Humans continue improving and clearly outperform at 8 hours. This reveals a fundamental limitation in current agent time horizons — agents are fast sprinters but poor marathon runners.

### Implications for Claude Code

1. **Instruction adherence is the #1 improvement area** for frontier models on terminal tasks. Not capability — compliance.
2. **More turns and tokens don't help** — spending more compute on the same approach yields no gains. Better to get the approach right.
3. **Verification before termination** is a direct quality lever — agents that check their work outperform those that don't.
4. **The 2-hour plateau** (RE-Bench) suggests Claude Code should be designed for effective handoffs and checkpointing rather than unlimited autonomous operation.
5. **PATH and environment issues** (24% of command failures) suggest explicit environment verification early in task execution.

---

## 7. LLM-as-Judge Evaluation

### Methodology Overview

- **MT-Bench**: 80 multi-turn questions, GPT-4 as judge, 1-10 scoring
- **Chatbot Arena**: Anonymous head-to-head battles, crowdsourced, Bradley-Terry model, Elo-like scores
- **AlpacaEval 2.0**: Length-controlled win rates, 0.98 Spearman correlation with Chatbot Arena, costs <$10

### Reliability Data

**Strengths**:
- GPT-4 as judge achieves >80% agreement with human preferences (matches inter-human agreement)
- Length-controlled AlpacaEval correlation with Chatbot Arena: 0.98 (very high)
- Cost-effective: AlpacaEval runs in <3 minutes for <$10

**Serious concerns (2025 research)**:
- A null model (constant irrelevant response) can secure high win rates on AlpacaEval 2.0 and MT-Bench
- Short adversarial phrases can dramatically inflate scores
- LLM judges show overconfidence — minimal variation in confidence between correct and incorrect assessments
- Casual users penalize abstention more than subtle inaccuracy

### Self-Evaluation Limitations

- LLMs are systematically overconfident. Training optimizes for fluency, not truthfulness.
- Models not trained to say "I don't know" — uncertainty expression is penalized during training.
- SFT yields reasonably calibrated confidence; RLHF (PPO, GRPO) and DPO induce overconfidence.
- Self-critique methods are heuristic and still produce overconfident responses.

### Implications for Claude Code

1. **LLM self-evaluation is useful but unreliable** — good for catching obvious errors, bad for assessing novel or complex reasoning.
2. **External verification is strictly necessary** — test execution, linting, type checking are more reliable than self-assessment.
3. **Overconfidence is a training artifact** — be skeptical of confident-sounding self-assessments. The model's confidence tone is not a signal of correctness.
4. **Length bias exists** — longer outputs are not better outputs. Length-controlled evaluation should be the standard.

---

## 8. Quality Decomposition

### Ten Critical Dimensions for AI Code Quality

Based on industry frameworks and benchmark analysis:

1. **Functional Correctness**: Does the code produce correct output? Measured by pass@k against test suites.
2. **Completeness**: Does it handle all requirements, including edge cases? Common failure — partial implementations.
3. **Security**: Free from vulnerabilities? 45% failure rate on OWASP Top 10 makes this critical.
4. **Maintainability**: Readable, well-structured, follows conventions? AI code has 1.64x more maintainability errors.
5. **Efficiency**: Time/space complexity appropriate? Rarely benchmarked but matters in production.
6. **Style Consistency**: Matches project conventions? Important for code review acceptance.
7. **Error Handling**: Graceful failure, informative errors? Often omitted by AI code.
8. **Documentation**: Comments, docstrings, README updates? Varies widely by model and prompt.
9. **Test Coverage**: Does the change include tests? Rarely produced unprompted.
10. **Integration Correctness**: Works with existing codebase? Multi-file benchmarks (SWE-bench Pro, BigCodeBench) test this.

### How Benchmarks Weight These Dimensions

| Benchmark | Correctness | Completeness | Security | Maintainability | Integration |
|-----------|:-:|:-:|:-:|:-:|:-:|
| HumanEval | Primary | Partial | No | No | No |
| SWE-bench | Primary | Partial | No | Implicit | Yes |
| SWE-bench Pro | Primary | Yes | No | Implicit | Yes (critical) |
| BigCodeBench | Primary | Yes | No | Partial | Yes |
| Terminal-Bench | Primary | Yes | No | No | Yes |
| Veracode Report | No | No | Primary | No | No |

**The gap**: No single benchmark comprehensively measures all dimensions. Security and maintainability are the most under-tested despite having the highest real-world defect rates.

### AI vs Human Code Quality (Detailed)

From the CodeRabbit AI-vs-Human report:
- AI-generated PRs: 10.83 issues each (vs 6.45 human)
- 1.7x more total issues
- 1.4x more critical issues
- Even passing AI code: average 1.45 static analysis issues per successful task

The pass@k metric (does it pass tests?) masks significant quality debt in maintainability, security, and style.

### Implications for Claude Code

1. **Multi-dimensional quality checks** should be standard — correctness is necessary but not sufficient.
2. **Security scanning** should be integrated into every code generation workflow.
3. **Static analysis** catches defects that test suites miss — AI code passing tests still has 1.45 issues per task.
4. **Style matching to project conventions** directly affects acceptance rates.
5. **Test generation** should be prompted/included as a standard part of code changes.

---

## 9. Scaling Laws for Agents

### Test-Time Compute: The New Scaling Paradigm

Test-time compute is replacing "train a bigger model" as the primary improvement lever. Models like o1, o3, and DeepSeek R1 demonstrate that thinking longer at inference outperforms adding parameters.

### Key Scaling Results

**Best-of-N (BoN) on GAIA**:
- Baseline: 55.76% → BoN: 63.03% (+7.3 points)
- BoN excels on simpler tasks where repeated attempts improve performance
- Step-wise BoN better for hardest tasks (+11.5 points on Level 3)

**Multi-Model Collaboration**:
- GPT-4.1 + 3 other SOTA models: 74.55% pass@4 (surpasses open-source SOTA)
- Heterogeneous models yield higher results than single-model sampling

**WebArena CATTS**:
- Confidence-aware scaling: 47.9% at 56% fewer tokens than uniform scaling (43.2%)
- N=10 to N=20: no improvement — pure diminishing returns

**Terminal-Bench**: No correlation between turns per trial and success rates.

### Diminishing Returns Patterns

1. **Reflection at every step hurts**: Direct reflection decreased performance (55.15% vs 55.76% baseline). Threshold-triggered reflection (<2 threshold) achieved 56.36%. Selective reflection outperforms constant application.
2. **Increasing aggregation rounds minimal**: Going from T=2 to T=4 improves only 0-1.6% while doubling compute.
3. **Over-refinement degrades**: Correct answers may be mistakenly discarded. Adaptive early termination reduces cost to 49% while preserving/improving performance.
4. **On hard problems, iteration doesn't help**: "Revising one idea usually does not help, because the high level plan is wrong. You only see gains when you explore qualitatively different strategies in parallel."

### The 45% Threshold

Google Research found an empirical threshold of ~45% single-agent accuracy. Above this, adding more agents typically yields diminishing or negative returns. Below it, multi-agent coordination can improve performance by up to 80.9%.

### Compute-Optimal Agent Strategies

1. **Allocate compute non-uniformly**: Spend more on uncertain decisions, less on obvious ones (CATTS principle).
2. **Parallel diverse attempts > sequential refinement**: Best-of-N with diverse strategies beats iterative self-correction.
3. **Use heterogeneous models**: Different models make different errors; combining them outperforms same-model sampling.
4. **Know when to stop**: Adaptive early termination preserves quality at 49% of compute cost.
5. **Verify, don't refine**: List-wise verification outperforms scoring methods by ~3 points consistently.

### Implications for Claude Code

1. **Selective reflection, not constant**: Only re-examine work when specific signals indicate problems (test failure, linting errors, uncertainty).
2. **Parallel exploration for hard problems**: When the first approach fails, try a qualitatively different strategy rather than refining the failed one.
3. **Early termination**: Build in signals to stop when additional work won't help.
4. **Heterogeneous sub-agents** could improve quality on complex tasks (using different models for different sub-tasks).
5. **Don't scale naively**: More tokens, more turns, and more iterations are not the answer. Better strategy selection is.

---

## 10. Failure Mode Taxonomy

### Microsoft's 27 Failure Modes (2025)

Microsoft's AI Red Team cataloged 27 safety and security failure modes organized into categories:

**Agent-Specific**: Compromise, injection, impersonation
**Memory/Data**: Poisoning, theft
**Input/Control**: Cross-domain prompt injection (XPIA), human-in-the-loop bypass
**Permission/Isolation**: Excessive access, insufficient sandboxing
**Alignment**: Instruction misinterpretation, insufficient transparency, parasocial relationships
**Multi-user**: Allocation harms, intra-agent responsible AI conflicts

### Practical Failure Mode Taxonomy for Coding Agents

From Terminal-Bench failure analysis, Galileo's guide, and the QSAF framework, a practical taxonomy for coding agents:

#### 1. Hallucination (Functions, APIs, Paths)
- Inventing nonexistent functions or APIs
- Referencing files/paths that don't exist
- Making up package names or configuration options
- **Cascading effect**: Hallucinated facts trigger multi-system incidents
- **Mitigation**: Tool-based verification (does this file exist? does this API exist?), grounding in codebase search results

#### 2. Instruction Drift
- Attention decays over extended interactions
- Agent gradually diverges from original task
- System prompt influence weakens over time
- **Mitigation**: Split-softmax (reweight attention to system prompt), periodic task re-anchoring, shorter autonomous runs

#### 3. Tool Misuse
- Exceeding intended permissions
- Calling functions with incorrect parameters
- Executing capabilities in unintended ways
- 24.1% of Terminal-Bench command failures: executable not found/not in PATH
- **Mitigation**: Parameter validation, permission scoping, environment verification before tool use

#### 4. Infinite Loops / Repetition
- Step repetition without progress
- Retrying the same failed approach
- Context loss causing circular behavior
- **Mitigation**: Trajectory monitoring, repetition detection, forced strategy switches after N failed attempts

#### 5. Partial Completion
- Solving the easy part, leaving the hard part
- Implementing the happy path without error handling
- Writing code but not tests
- **Mitigation**: Explicit completeness checklists, multi-dimensional verification

#### 6. Premature Termination
- Declaring success without verification
- Stopping at first plausible answer
- Insufficient outcome checking
- **Mitigation**: Mandatory verification step before completion, test execution, outcome validation

#### 7. Over-Refinement
- Continuing to modify correct solutions
- Correct answers discarded during self-critique
- Excessive iteration degrading quality
- **Mitigation**: Adaptive early termination, confidence-based stopping criteria

#### 8. Context Loss / Memory Failures
- Losing track of previous steps in long trajectories
- Forgetting constraints mentioned earlier
- Reasoning-action mismatches from context overflow
- **Mitigation**: External memory (scratchpads, task files), context compression, working memory management

#### 9. Security Violations
- Generating code with known vulnerabilities
- Ignoring input validation
- Improper credential handling
- **Mitigation**: Security scanning integration, vulnerability-aware code generation, explicit security review step

#### 10. Cognitive Degradation (QSAF)
- Compromised memory and logic accumulate
- Agent overrides original role or constraints
- Behavior becomes unpredictable as task alignment is lost
- **Mitigation**: Seven runtime controls monitoring agent subsystems, fallback routing, starvation detection, memory integrity enforcement

### How the Best Systems Mitigate Failures

1. **Verification discipline**: Agents that rigorously validate outcomes before terminating consistently outperform (Terminal-Bench finding).
2. **Structured execution**: Scaffolded approaches (SWE-bench G4-G5) prevent drift better than fully emergent approaches.
3. **Error recovery with strategy switching**: Superior agents distinguish recoverable from terminal errors and adapt strategy.
4. **Observability**: LLM-based output audits catching hallucinations, policy violations, and reasoning errors that traditional rules miss.
5. **Selective reflection**: Reflecting only when uncertainty is high, not at every step.

### Implications for Claude Code

1. **Hallucination grounding**: Every file path, function name, and API call should be verifiable against actual codebase state.
2. **Instruction anchoring**: For long tasks, periodically re-read the original task description to prevent drift.
3. **Mandatory verification before completion**: Never declare success without running tests or checking outcomes.
4. **Strategy switching after repeated failure**: If the same approach fails N times, try something qualitatively different.
5. **Environment verification early**: Check that tools, executables, and paths exist before attempting to use them.
6. **Explicit stopping criteria**: Define when additional work is unlikely to help and stop.

---

## Cross-Cutting Themes

### Theme 1: Scaffold > Model

The most consistent finding across all benchmarks is that the agent architecture matters more than the model. SWE-bench Pro shows a 22-point scaffold swing vs. 0.8-point model swing. Terminal-Bench shows similar patterns. A mid-tier model in a great harness beats a frontier model in a bad one.

**For Claude Code**: Invest in better context engineering, file indexing, verification pipelines, and tool use patterns rather than waiting for model improvements.

### Theme 2: Verification is the Quality Multiplier

Across SWE-bench (verification loops), Terminal-Bench (verification errors as primary failure mode), real-world metrics (1.7x more defects), and security data (45% failure rate), the pattern is clear: verification is where quality is won or lost.

**For Claude Code**: Build verification into every workflow — run tests, run linters, run security scans, check that files exist before editing them.

### Theme 3: Selective Compute Allocation

Test-time compute scaling research consistently shows that uniform scaling produces diminishing returns. CATTS saves 56% of tokens while improving by 4.7 points. Reflection at every step hurts performance. The key is identifying which decisions are uncertain and spending more compute there.

**For Claude Code**: Don't iterate uniformly. Identify specific failure points (test failures, unclear requirements, uncertain approach) and focus compute on those.

### Theme 4: The Time Horizon Limitation

RE-Bench shows agents plateau at 2 hours. Terminal-Bench shows no correlation between turns and success. Over-refinement degrades quality. Current agents are sprinters, not marathon runners.

**For Claude Code**: Design for effective checkpointing and handoffs. Break large tasks into bounded sub-tasks rather than attempting unbounded autonomous operation.

### Theme 5: Multi-Dimensional Quality

Benchmarks primarily measure correctness via test passage, but real-world quality includes security (45% failure), maintainability (1.64x more errors), and integration correctness. Passing tests is necessary but far from sufficient.

**For Claude Code**: Implement multi-dimensional quality checks: correctness (tests), security (scanning), style (linting), maintainability (static analysis), completeness (edge case coverage).

---

## Depth Checklist Review

- [x] **Underlying mechanisms**: Covered how each benchmark works, how agents are scored, and what architectures succeed
- [x] **Key tradeoffs and limitations**: Covered data contamination, benchmark limitations, self-evaluation unreliability, scaling diminishing returns
- [x] **Comparisons**: Compared benchmarks against each other, scaffolded vs emergent architectures, single vs multi-agent, model vs scaffold impact
- [x] **Failure modes and edge cases**: Comprehensive 10-category failure taxonomy with mitigations, Terminal-Bench failure analysis, security vulnerability data
- [x] **Concrete examples**: Specific numbers (22-point scaffold gap, 45% security failure, 4x RE-Bench speedup, CATTS 56% token savings), architecture descriptions (Agentless, Moatless, CATTS), real systems (GitHub Copilot, Greptile, Terminal-Bench agents)
- [x] **Decision-ready**: Specific Claude Code implications for each section, cross-cutting themes with actionable recommendations
