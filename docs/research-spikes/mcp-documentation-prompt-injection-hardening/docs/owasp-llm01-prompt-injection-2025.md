<!-- Source: https://genai.owasp.org/llmrisk/llm01-prompt-injection/ -->
<!-- Retrieved: 2026-05-14 -->

# OWASP LLM01:2025 Prompt Injection Vulnerability

## Definition
"A Prompt Injection Vulnerability occurs when user prompts alter the LLM's behavior or output in unintended ways." These manipulations don't require human-readable formatting — the model simply needs to parse the content.

## Core Distinction
While prompt injection and jailbreaking are interconnected concepts, they differ meaningfully. Prompt injection involves altering model responses through specific inputs, whereas jailbreaking represents a specialized form where attackers cause models to disregard safety protocols entirely.

## Vulnerability Types

**Direct Prompt Injections:** User input directly and unexpectedly modifies model behavior, whether through intentional malicious crafting or unintentional user actions.

**Indirect Prompt Injections:** External sources (websites, files) contain data that, when interpreted by the model, unintentionally or deliberately alters behavior.

## Potential Impacts
- Sensitive information disclosure and system infrastructure exposure
- Content manipulation producing biased or incorrect outputs
- Unauthorized function access
- Arbitrary command execution in connected systems
- Compromise of critical decision-making processes

## Prevention & Mitigation Strategies

1. **Constrain Model Behavior:** Define specific roles, capabilities, and limitations within system prompts
2. **Define Output Formats:** Specify clear formats with reasoning and source citations
3. **Input/Output Filtering:** Apply semantic filters and validate using the RAG Triad framework
4. **Privilege Control:** Implement least-privilege access and handle functions in code
5. **Human Approval:** Require human oversight for high-risk operations
6. **Segregate External Content:** Clearly denote untrusted sources
7. **Adversarial Testing:** Conduct regular penetration testing and breach simulations

## Representative Attack Scenarios

- Direct injection into customer support chatbots accessing private data stores
- Indirect injection via hidden webpage instructions causing data exfiltration
- Unintentional injection through overlooked system instructions in documents
- RAG application manipulation via compromised repository documents
- Code injection exploiting LLM-powered email assistants
- Multimodal attacks embedding malicious prompts within images
- Adversarial suffixes: meaningless character strings influencing output maliciously
- Multilingual/obfuscated attacks using encoding techniques to evade filters

## References & Frameworks
The resource cites research from Arxiv, Cornell University, MITRE ATLAS, and NIST, covering both technical attacks and defense methodologies for LLM security.
