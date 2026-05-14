# Let Me Speak Freely? Format Restrictions on LLM Performance

- **Source URL**: https://arxiv.org/abs/2408.02442
- **Retrieved**: 2026-03-15
- **Authors**: Zhi Rui Tam, Cheng-Kuang Wu, et al.
- **Published**: EMNLP 2024 Industry Track

## Study Design

Three structured generation approaches tested with increasing flexibility:
1. **Constrained Decoding (JSON-mode)**: Restricts token space during generation
2. **Format-Restricting Instructions (FRI)**: Directs output in JSON/XML/YAML without token constraints
3. **NL-to-Format**: Two-step — generate in natural language first, then convert

## Models Tested

GPT-3.5-turbo, Claude-3-haiku, Gemini-1.5-flash, LLaMA-3-8B-Instruct, Gemma-2-9B-Instruct

## Key Results: Reasoning Tasks (Degradation)

**GSM8K (Math)**:
- GPT-3.5-turbo: Text 76.6% → JSON 49.25% (**-27.35 points**)
- Claude-3-haiku: Text 86.51% → JSON 23.44% (**-63.07 points**)
- LLaMA-3-8B: Text 74.73% → JSON 48.9% (**-25.83 points**)

**Last Letter Concatenation**:
- GPT-3.5-turbo: Text 56.74% → JSON 25.2% (**-31.54 points**)
- LLaMA-3-8B: Text 70.07% → JSON 28% (**-42.07 points**)

## Key Results: Classification Tasks (Often Improved)

**DDXPlus Medical Diagnosis**:
- Gemini-1.5-flash: Text 41.59% → JSON 60.36% (**+18.77 points**)

Sports Understanding, Stereotype Classification, Financial Classification: relatively stable.

## Critical Finding: Schema vs No-Schema

| Model | Text | JSON (no schema) | JSON (with schema) |
|-------|------|------------------|-------------------|
| GPT-3.5-turbo | 75.99% | 74.70% | 49.25% |
| Claude-3-haiku | 86.51% | 86.99% | 23.44% |

Schema constraints cause the most damage — removing explicit schema requirements preserves most performance.

## Root Cause

Parsing failures are NOT the primary cause. Gemini-1.5-flash had 0% JSON parsing errors yet substantial performance variance. The degradation comes from forcing structured format adherence during reasoning, particularly the misordering of reasoning steps and answer keys.

## Practical Implications

- Use structured output for classification/extraction tasks (can improve performance)
- Avoid structured output during reasoning-heavy tasks
- Two-step approach (reason in NL, then convert) preserves reasoning quality
- Schema strictness inversely correlates with reasoning quality
