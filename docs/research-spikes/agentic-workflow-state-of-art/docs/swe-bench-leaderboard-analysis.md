# SWE-bench Leaderboard and Top Performer Analysis

- **Sources**:
  - https://epoch.ai/benchmarks/swe-bench-verified
  - https://www.swebench.com/
  - https://www.verdent.ai/blog/swe-bench-verified-technical-report
  - https://www.warp.dev/blog/swe-bench-verified
  - https://www.together.ai/blog/deepswe
- **Retrieved**: 2026-03-15

## Current Leaders (Early 2026)

Claude Opus 4.5 leads SWE-bench Verified at 80.9%. Other top performers include Gemini 3 Flash and MiniMax M2.5. Several Chinese models (GLM-5, Kimi K2.5, DeepSeek V3.2) in top ten.

## Key Systems and Their Approaches

### Verdent (76.1% pass@1, 81.2% pass@3)
- Multi-agent architecture with parallel agents
- Plan-code-verify development cycle
- Code review subagent adds ~0.5% to pass@3
- Powered by coordination of top-tier AI models
- No leaderboard tuning or test-time scaling claimed

### Warp (71% → 75.8%)
- Single-agent architecture (competitive for user-facing latency requirements)
- Focused toolset: grep/find/cat analogues, run_command, TODO list tool
- TODO list provides lightweight planning mechanism
- Recovery mechanisms: tool-call failure reporting, model fallback, tool use restrictions
- Key insight: single-attempt architectures competitive at coding tasks

### DeepSWE (59% with Qwen3-32B open-weight)
- Trained entirely through reinforcement learning
- 0/1 verifiable rewards from test suite execution
- Emergent behaviors: anticipates edge cases, runs regression tests proactively
- Dynamically adjusts "thinking effort" based on task difficulty
- 200 steps of RL training = ~20% improvement on SWE-Bench Verified
- Trained on 4,500 real-world tasks, 64 H100s, 6 days

### SWE-Search (23% relative improvement)
- Monte Carlo Tree Search (MCTS) adapted for software engineering
- Balances exploration and exploitation
- SWE-Agent for exploration, Value Agent for feedback, Discriminator Agent for debate
- Max 3 expansions per node, 100 total search iterations
- Performance scales with inference-time compute

### OpenHands (72.8% with Claude Sonnet 4.5)
- Docker-sandboxed runtime with bash, IPython, Chromium
- Multi-agent delegation via AgentDelegateAction
- Event-sourced state management for replay/recovery

## Pass@K Analysis

- **Pass@1**: Single-shot performance, correlates to real user experience
- **Pass@3**: Practical retry budget for "vibe coding" workflows
- **Pass@5**: Efficiency of search within limited iterations
- Consensus: pass@3 represents realistic retry budget for real-world usage
- Step limits: typically 100-150 steps per task

## Architectural Patterns Among Top Performers

1. **Multi-agent vs Single-agent**: Both competitive. Multi-agent adds review/verification quality. Single-agent has latency advantages.
2. **Plan-Code-Verify cycle**: Nearly universal among top performers
3. **Test execution for verification**: Core feedback mechanism
4. **Specialized tool interfaces (ACI)**: Better than raw shell access
5. **Error recovery mechanisms**: Essential for handling LLM variability
