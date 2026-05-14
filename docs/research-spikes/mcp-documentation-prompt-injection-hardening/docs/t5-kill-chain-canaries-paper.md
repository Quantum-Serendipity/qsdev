# Kill-Chain Canaries: Stage-Level Tracking of Prompt Injection
- **Source**: https://arxiv.org/html/2603.28013v2
- **Retrieved**: 2026-05-14

## How Kill-Chain Canaries Work

The methodology embeds a unique cryptographic token (SECRET-[A-F0-9]{8}) into each injection payload, then tracks its progression through the agent pipeline using a PropagationLogger.

## The Four Kill-Chain Stages

1. **EXPOSED**: The canary token appears in any tool result or context window -- confirming the model receives the injection
2. **PERSISTED**: The token survives into a write_memory call, indicating it passed the summarization stage
3. **RELAYED**: A downstream agent reads the contaminated memory record
4. **EXECUTED**: The canary appears in an outbound tool argument, confirming the injection drove harmful action

Key insight: "This gap between Exposed and Persisted identifies summarization-stage filtering; between Persisted and Executed, it identifies execution-stage refusal."

## Experimental Results Across LLMs (950 runs, 5 models)

| Model | ASR (Attack Success Rate) | Task Success |
|-------|---------------------------|--------------|
| Claude Haiku/Sonnet | 0% (0/164 runs) | 100% |
| GPT-5-mini | 3% | 94% |
| DeepSeek Chat | 25% (varies: 0-100% by surface) | 100% |
| GPT-4o-mini | 53% | 90% |

Critical finding: DeepSeek achieved 0% ASR on memory_poison but 100% on tool_poison -- identical model, opposite results depending on injection surface.

## Defense Conditions Tested (5 conditions)

- **None** (baseline, attacked control)
- **write_filter**: Keyword scanning before memory commit
- **pi_detector**: Secondary LLM classifying outgoing queries
- **spotlighting**: XML delimiters wrapping document content
- **all**: Combined three defenses

Result: All four defenses produced 100% ASR on GPT-4o-mini and DeepSeek for propagation and tool_poison scenarios (n=8 per cell).

## Key Detection Rate Findings

- Claude's defense eliminates injections during write_memory: 0/40 runs had canary survive into MemoryStore (95% CI: 0-8%)
- GPT-4o-mini exposure-to-execution lag: Median 1 step (immediate execution)
- DeepSeek execution lag: Bimodal, 2-3 steps typical, tail reaching 12 steps
- Objective drift as forensic signal: Post-hoc detection AUC collapsed to 0.39-0.57 (chance level)

## Practical Implications

1. **Write-node placement is highest-leverage safety decision**: Claude at write stage provides pipeline-wide decontamination
2. **Surface mismatch is structural, not fixable**: Safety gap exists downstream; evaluation coverage determines apparent security posture
3. **Parser-level trust is missing**: Invisible PDF injections equaled or exceeded visible-text injection rates
4. **Relay node identity matters more than downstream agents**: Claude relay blocks all downstream propagation regardless of which agent follows

Core conclusion: "prompt injection is not a model-capability problem, it is a pipeline-architecture problem."
