# OpenAI Guardrails Python - Prompt Injection Detection
- **Source**: https://openai.github.io/openai-guardrails-python/ref/checks/prompt_injection_detection/
- **Retrieved**: 2026-05-14

## How It Works

Operates at two critical checkpoints:
1. **Output Guardrail (Tool Call Validation)**: Validates that function calls align with user intent before execution
2. **Pre-flight Guardrail (Output Validation)**: Checks returned data post-execution for on-topic/data-leakage

"Detects prompt injection attempts in function calls and function call outputs using LLM-based analysis."

## Configuration Parameters

- **model**: The LLM for analysis (e.g., gpt-4.1-mini)
- **confidence_threshold**: Minimum confidence score (0.0-1.0) to trigger alerts
- **max_turns** (optional): Number of conversation turns to analyze; defaults to 10
- **include_reasoning** (optional): Boolean to include detailed observation/evidence fields

## Supported Models & Performance

| Model | ROC AUC | Notes |
|-------|---------|-------|
| gpt-5 | 0.993 | Highest accuracy |
| gpt-4.1 | - | - |
| gpt-4.1-mini | 0.987 | Default |
| gpt-5-mini | - | - |

**Latency**: gpt-4.1-mini achieves 1,481ms median, 2,563ms at p95.

## Detection Capabilities

Flags: Unrelated function calls, harmful operations, private data exposure, extraneous data.
Does NOT flag: Reasonable actions toward user goals, partial/ineffective responses.

## API Returns

GuardrailResult containing: flagged status, confidence score, user intent tracking, analyzed actions with reasoning fields.

## Key Observation

This is an LLM-based analysis guardrail, not a classifier. It uses another LLM call to assess whether tool calls and outputs align with user intent. This means ~1.5s latency per check and requires OpenAI API access. It's designed for tool call validation, not for screening content within tool results for hidden injections.
