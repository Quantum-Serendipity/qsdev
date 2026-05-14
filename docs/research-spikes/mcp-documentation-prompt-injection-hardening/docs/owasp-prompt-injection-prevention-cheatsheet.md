# LLM Prompt Injection Prevention - OWASP Cheat Sheet Series
- **Source**: https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Attack Categories

### Direct Prompt Injection
Explicit malicious instructions in user input: "ignore all previous instructions" or "developer mode."

### Remote/Indirect Prompt Injection
Hidden in external content: code comments, commit messages, web pages, emails, documents.

### Obfuscation Techniques
- Encoding attacks: Base64, hex, Unicode invisible characters
- Typoglycemia variants: "ignroe" instead of "ignore" -- LLMs read scrambled middle letters
- Best-of-N jailbreaking: Numerous prompt variations until one bypasses safeguards
- HTML/Markdown injection: Hidden image tags or malicious links in rendered content

### Specialized Attack Vectors
- Jailbreaking: Role-playing and hypothetical scenario framing
- Multi-turn persistence: Attacks spanning multiple interactions using coded language
- System prompt extraction
- RAG poisoning
- Agent-specific attacks: Forging reasoning steps or manipulating tool parameters
- Multimodal injection: Hidden instructions in images or document metadata

## Primary Defense Mechanisms

### Input Validation & Sanitization
Pattern matching for dangerous keywords. Fuzzy matching for typoglycemia variants using Levenshtein distance (threshold 1-2).

### Structured Prompt Architecture
Separate system instructions from user data using explicit delimiters. Mark all user input as DATA, never INSTRUCTIONS.

### Output Monitoring
Validate responses for system prompt leakage, API key exposure, numbered instruction lists.

### Human-in-the-Loop Controls
Flag high-risk requests containing keywords like "password," "api_key," "admin."

### Model-Based Guardrails
Dual-LLM pattern: isolate untrusted content processing from tool-executing models.

## Implementation Framework

1. Detection: Identify injection attempts
2. HITL review: Route suspicious requests to humans
3. Sanitization: Clean and structure prompts
4. Generation: Call LLM with protected structure
5. Validation: Screen output before delivery

## Key Limitations

"Existing defenses exhibit power-law scaling vulnerabilities. Rate limiting, content filters, and safety training can eventually be defeated through sufficient variation attempts."

## Testing Strategy

Red team with: direct injection, encoded payloads, typoglycemia variants, character-spacing bypasses, indirect vectors in code comments and web content.
