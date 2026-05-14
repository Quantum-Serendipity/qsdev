# Anthropic API Docs: Mitigate Jailbreaks and Prompt Injections

- **Source URL**: https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks
- **Retrieved**: 2026-05-14

## Overview
Jailbreaking and prompt injections occur when users craft prompts to exploit model vulnerabilities. While Claude is "inherently resilient" to such attacks, additional steps can strengthen guardrails.

## Recommended Strategies

### Harmlessness Screens
Use a lightweight model like Claude Haiku 4.5 to pre-screen user inputs with structured outputs constraining response to boolean classification.

### Input Validation
Filter prompts for jailbreaking patterns. Can use an LLM to create generalized validation screens with known jailbreaking language as examples.

### Prompt Engineering
Craft prompts emphasizing ethical and legal boundaries. Example: system prompts with explicit value lists and refusal instructions.

### Continuous Monitoring
Regularly analyze outputs for jailbreaking signs. Consider throttling or banning users who repeatedly engage in abusive behavior.

### Advanced: Chain Safeguards
Combine strategies for robust protection. Example: multi-layered financial advisor chatbot with:
1. System prompt with compliance directives
2. Step-by-step instructions including harmlessness_screen tool usage
3. Structured output classification for compliance checking

## Notable Observations
- This document focuses on API-level defenses for application developers
- Does NOT specifically address the trust hierarchy (system > user > tool output) in explicit terms
- Does NOT provide specific XML tag framing recommendations for marking tool output as untrusted
- The advice is largely about defending against user-originated injection, not tool-result injection
- No specific guidance about MCP tool result handling
