<!-- Source: https://genai.owasp.org/llmrisk/llm01-prompt-injection/ -->
<!-- Retrieved: 2026-05-14 -->

# OWASP LLM01:2025 Prompt Injection

## Definition & Scope

Indirect prompt injections occur "when an LLM accepts input from external sources, such as websites or files." The malicious content "when interpreted by the model, alters the behavior of the model in unintended or unexpected ways."

## External Data Retrieval as Attack Vector

**Documentation & Web Content Risks:**
Indirect injections target systems pulling from external repositories. Scenario #2: "A user employs an LLM to summarize a webpage containing hidden instructions that cause the LLM to insert an image linking to a URL, leading to exfiltration of private conversation."

Scenario #4: An attacker "modifies a document in a repository used by a Retrieval-Augmented Generation (RAG) application. When a user's query returns the modified content, the malicious instructions alter the LLM's output."

## Mitigation Strategies for External Content

- **Input Filtering**: "Apply semantic filters and use string-checking to scan for non-allowed content."
- **Content Segregation**: "Separate and clearly denote untrusted content to limit its influence on user prompts."
- **RAG-Specific Validation**: Evaluate responses using the "RAG Triad: Assess context relevance, groundedness, and question/answer relevance."
- **Least Privilege**: Handle "functions in code rather than providing them to the model" to restrict capabilities.
