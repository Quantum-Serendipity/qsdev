# Where Do LLMs Still Struggle? An In-Depth Analysis of Code Generation Benchmarks
- **Source**: https://arxiv.org/html/2511.04355v1
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv HTML page

## Benchmarks Analyzed

Four widely-adopted benchmarks, 865 tasks total:
- HumanEval: 164 tasks
- MBPP: 378 tasks
- LiveCodeBench: 175 tasks
- BigCodeBench-Hard: 148 tasks

Six models evaluated: Claude Sonnet-4, DeepSeek-V3, Qwen3-Coder, GPT-4o, Llama-3.3-70B, Mistral-3.2-24B

## Performance Gap Across Benchmarks

### HumanEval (nearly solved):
- 113 of 164 tasks solved by ALL models
- Claude Sonnet-4: only 2 failures
- Top models exceed 90% pass@1

### MBPP:
- 318 of 378 tasks solved by ALL models
- Qwen3-Coder and DeepSeek-V3 best performers

### LiveCodeBench (intermediate difficulty):
- 43 tasks universally solved, 35 never solved by any model
- Claude Sonnet-4: 54 failures (strongest)

### BigCodeBench-Hard (extremely challenging):
- Only 14 tasks solved by ALL models
- 76 tasks consistently FAILED across ALL models
- Failure rates: 68-77% across all models

## Four Recurring Failure Patterns

1. **Wrong Problem Mapping**: Models misclassify tasks into familiar problem categories (e.g., treating nesting validation as standard "balanced brackets")
2. **Flawed/Incomplete Algorithm Design**: Correct approach but lacks necessary components (e.g., missing handling for non-monotonic trends)
3. **Edge Case Mishandling**: Code fails on boundary scenarios (e.g., iterating only top-level files instead of recursive search)
4. **Formatting Mistakes**: Correct logic but wrong output format (e.g., unquoted digits when string literals expected)

## Key Observations

- Solution code complexity shows minimal correlation with failure rates (except LiveCodeBench)
- Many BigCodeBench failures stem from ambiguous prompts paired with overspecified tests
- Simpler models sometimes outperformed advanced ones by interpreting instructions literally
- Even "passing" code averages 1.45 static analysis issues per task (e.g., OpenCoder-8B)
