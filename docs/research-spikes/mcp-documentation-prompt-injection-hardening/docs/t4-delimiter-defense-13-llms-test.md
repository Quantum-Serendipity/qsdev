# I Tested Delimiter-Based Prompt Injection Defense Across 13 LLMs

- **Source**: https://dev.to/whetlan/i-tested-delimiter-based-prompt-injection-defense-across-13-llms-50mn
- **Retrieved**: 2026-05-14

## Models Tested
The study evaluated 13 LLMs across two categories:
- **API models**: Claude (Sonnet, Haiku 3.5), Grok 3-mini-fast, Gemini 2.5 Flash, DeepSeek V4 Pro/Flash/V3, GPT-4o, GPT-5.4 Mini, Qwen Turbo, Kimi
- **Local models**: Via Ollama (11 API + local variants)

## Experimental Setup
- **Test cases**: ~5,500 total evaluations
- **Payload format**: Attack strings wrapped in random 128-character hexadecimal delimiters
- **Context**: Embedded within ~1,000-word documents
- **Task**: Model summarization with temperature 0.0
- **Detection method**: Canary string presence in output indicates successful injection

## Overall Defense Effectiveness

| Configuration | Defense Rate |
|---|---|
| With delimiters | 89.7% |
| Without delimiters | 60.7% |
| **Improvement** | **+29 percentage points** |

## Per-Model Performance

**Perfect Defense (100%)**
- Claude Sonnet/Haiku 3.5
- GPT-5.4 Mini
- DeepSeek V4 Pro

**Strong Defense (94-98%)**
- DeepSeek V4 Flash: 94%
- GPT-4o: 97.8%

**Significant Improvement Shown**
- Grok 3-mini-fast: 32% → 100% (+68 points)
- Gemini 2.5 Flash: 36.6% → 100%
- DeepSeek V4 Pro: 43% → 100%

**Weak Defense (59-79%)**
- Qwen Turbo: 59% (even with delimiters)
- Kimi: 73.9%
- DeepSeek V3: 79%

## Attack Vectors Tested
1. Direct override commands
2. Role-switching with fake `[SYSTEM]` tags
3. Authority claims ("PRIORITY SYSTEM UPDATE")
4. Gradual drift (legitimate content evolving into injection)
5. Delimiter mimicry (payload includes closing delimiter)
6. Subtle blending (canary as "validation token")
7. Repetition flood (25+ variations of same injection)

## Defense Template Comparison

**Strict Template Performance**: 96.3% success rate
- Minimal instruction: boundaries established, content treated as data only
- Defines zone for data vs. instructions

**Contextual Template Performance**: 89.1% success rate
- Includes threat model explanation
- Describes untrusted source origin
- Counterintuitive finding: explaining threat model reduced effectiveness on some models (Kimi: 97.8% strict vs. 50% contextual)

## Attack Resilience by Type

| Attack Type | Defense Success with Delimiters |
|---|---|
| Role switching | 100% |
| Delimiter mimicry | 89.3% |
| Gradual drift | 88.8% |
| Direct override | 86.3% |

## Generational Improvement Patterns

DeepSeek progression:
- V3 (older): 79%
- V4 Flash: 94%
- V4 Pro: 100%

OpenAI progression:
- GPT-4o: 97.8%
- GPT-5.4 Mini: 100%

## Critical Limitations

The study acknowledges important scope constraints:
- **Single task**: Document summarization only; tool calls and RAG pipelines untested
- **Temperature constraint**: Testing at 0.0; production systems typically use higher values
- **Language scope**: English payloads only; cross-language injection vectors unexplored
- **Behavioral detection**: Canary-based approach misses subtle behavior changes without explicit output

## Practical Implementation Recommendations

1. **Delimiters are worthwhile**: Provides substantial defense improvement for most current models
2. **Expect model variance**: Performance ranges from 59% to 100%—delimiter defense isn't universal
3. **Use strict templates**: Prefer terse boundary declarations over threat explanation
4. **Treat as layered defense**: Not a complete solution; combine with additional safeguards
5. **Avoid false confidence**: Strong models show inherent resilience; weaker models remain vulnerable

## Resource

The author published the complete test harness and dataset (5,500+ records) as "DataBoundary" on GitHub and HuggingFace, enabling community extension with additional models, attack payloads, and defense variations.
