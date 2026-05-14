# Vigil: LLM Security Scanner
- **Source**: https://github.com/deadbits/vigil-llm
- **Retrieved**: 2026-05-14

## Purpose

Python library and REST API for assessing LLM prompts and responses for prompt injections, jailbreaks, and other threats.

## Available Scanners (5 modules)

1. **Vector Database** - Text similarity matching with optional auto-updating when threats detected
2. **YARA Heuristics** - Pattern-based detection using YARA signatures for known attack techniques
3. **Transformer Model** - Uses deepset/deberta-v3-base-injection model for classification
4. **Prompt-Response Similarity** - Analyzes relationships between inputs and outputs
5. **Canary Tokens** - Embeds detectable markers for data leakage/goal hijacking detection

## Additional Features

Sentiment analysis, relevance checking (via LiteLLM), paraphrasing detection.

## Deployment Options

- Python library (direct import)
- REST API server (HTTP endpoints)
- Streamlit Web UI (interactive playground)

## API Endpoints

- POST /analyze/prompt - Assess single prompts
- POST /analyze/response - Evaluate prompt-response pairs
- POST /canary/add - Insert detection tokens
- POST /canary/check - Verify token presence
- POST /add/texts - Populate vector database
- GET /settings - View configuration

## Maintenance Status

Alpha state. 13 releases, latest v0.10.3-alpha from December 2023. 478 GitHub stars. Last release is 2.5 years old - likely unmaintained.
