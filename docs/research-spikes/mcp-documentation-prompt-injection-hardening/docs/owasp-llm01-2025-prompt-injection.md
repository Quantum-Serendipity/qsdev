# LLM01:2025 Prompt Injection - OWASP Top 10 for LLM Applications
- **Source**: https://genai.owasp.org/llmrisk/llm01-prompt-injection/
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Definition
"A Prompt Injection Vulnerability occurs when user prompts alter the LLM's behavior or output in unintended ways." These attacks can affect models even through imperceptible inputs, and don't require human-readable content as long as the model parses it.

## Vulnerability Types

**Direct Prompt Injections:** User inputs directly alter model behavior, either intentionally (malicious) or unintentionally.

**Indirect Prompt Injections:** External sources (websites, files) contain data that, when interpreted by the model, causes unintended behavioral changes.

## Impact Areas
- Sensitive information disclosure
- System prompt/infrastructure revelation
- Content manipulation and bias
- Unauthorized function access
- Arbitrary command execution
- Critical decision manipulation

## Prevention & Mitigation Strategies

1. **Constrain behavior** - Define role, capabilities, and limitations in system prompts
2. **Validate output formats** - Specify clear formats with detailed reasoning and citations
3. **Filter inputs/outputs** - Apply semantic filters and content scanning
4. **Enforce privilege control** - Use least-privilege access principles
5. **Require human approval** - Implement controls for high-risk actions
6. **Segregate external content** - Clearly denote untrusted sources
7. **Adversarial testing** - Conduct regular penetration testing and simulations

## Attack Scenarios Overview
The document includes nine scenarios spanning: direct injections into chatbots, indirect webpage-based attacks, unintentional triggers, RAG manipulation, code injection vulnerabilities, payload splitting in resumes, multimodal image-based attacks, adversarial suffixes, and multilingual obfuscation techniques.
