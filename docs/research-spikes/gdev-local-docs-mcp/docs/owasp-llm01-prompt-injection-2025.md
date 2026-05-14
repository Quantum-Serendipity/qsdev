# OWASP LLM01:2025 Prompt Injection

- **Source**: https://genai.owasp.org/llmrisk/llm01-prompt-injection/
- **Retrieved**: 2026-05-14
- **Publisher**: OWASP Gen AI Security Project

---

## Definition
"A Prompt Injection Vulnerability occurs when user prompts alter the LLM's behavior or output in unintended ways." These attacks don't require human-readable content -- models can process imperceptible injections.

## Two Primary Attack Categories

**Direct Prompt Injections**: User input directly modifies model behavior, either maliciously or accidentally.

**Indirect Prompt Injections**: LLMs accept input from external sources (websites, files) containing hidden instructions that alter behavior when processed.

## Potential Impact
Successful attacks can cause:
- Sensitive information disclosure
- System prompt exposure
- Biased or manipulated outputs
- Unauthorized function access
- Arbitrary command execution
- Compromised decision-making

## Seven Mitigation Strategies

1. **Constrain behavior** through specific system prompt instructions and context limits
2. **Define output formats** with clear specifications and deterministic validation
3. **Filter inputs/outputs** using semantic analysis and content rule-checking
4. **Enforce least privilege** by restricting API access and model permissions
5. **Require human approval** for high-risk operations
6. **Segregate external content** with clear untrusted source markers
7. **Conduct adversarial testing** via regular penetration testing

## Notable Attack Scenarios

- **Code injection** exploiting vulnerabilities in LLM-powered applications
- **Payload splitting** embedding malicious prompts across document sections
- **Multimodal injection** hiding instructions within images accompanying text
- **Obfuscation attacks** using Base64 encoding or multiple languages to evade filters

## Key Resources
Reference materials include Cornell University research on prompt injection attacks, MITRE ATLAS frameworks (AML.T0051.000/001, AML.T0054), and Kudelski Security's design-based mitigation approaches.
