# OWASP LLM Prompt Injection Prevention Cheat Sheet
- **Source**: https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html
- **Retrieved**: 2026-05-14

## Primary Defense Techniques

**Input Validation & Sanitization**
- Pattern matching for dangerous keywords (e.g., "ignore previous instructions")
- Fuzzy matching for typoglycemia variants using Levenshtein distance (threshold 1-2)
- Normalization of whitespace and character repetition
- Length limits on input (10,000 characters recommended)
- Libraries: python-Levenshtein, rapidfuzz, apache-commons-text, agnivade/levenshtein

**Structured Prompt Architecture**
- Clear separation between system instructions and user data using delimited sections
- Explicit labeling of user input as DATA, not COMMANDS
- StruQ research as foundational approach

**Output Monitoring**
- Detection of system prompt leakage patterns
- Identification of API key exposure in responses
- Suspicious instruction disclosure monitoring
- Response length validation (5,000 character limit)

**Human-in-the-Loop (HITL)**
- Risk scoring based on keywords ("password," "api_key," "admin," "system")
- Escalation thresholds triggering human review
- Approval workflows for high-risk operations

## Additional Defense Layers

**Remote Content Sanitization**
- Filtering of code comments before analysis
- Validation/decoding of suspicious encoded content
- Markup sanitization in external documents

**Agent-Specific Defenses**
- Tool call validation against user permissions
- Parameter validation per tool type
- Anomaly detection in agent reasoning patterns
- Scope restrictions per least-privilege principle

**Model-Based Guardrails ("LLM-as-Judge")**
- Input screening for user prompts and retrieved content
- Output screening before user delivery
- Action screening for agent tool calls
- Dual-LLM pattern: privileged actor model + quarantined content model

**Guardrail Tools Recommended:**
- Llama Guard, ShieldGemma, IBM Granite Guardian, Prompt Guard, NVIDIA NeMo Guardrails

## Attack Pattern Detection Methods

- Direct Injection: Keywords like "ignore previous," "developer mode," "system override"
- Typoglycemia Variants: Scrambled middle characters
- Encoding Detection: Base64, Hex, Unicode smuggling, KaTeX/LaTeX invisible text
- Remote/Indirect Injection: Malicious instructions in code, commits, issues, web content
- HTML/Markdown Injection: Malicious links, hidden image tags
- RAG Poisoning: Detection of malicious content in vector databases

## Pipeline Architecture

Input validation -> HITL check -> Sanitization -> Structured prompting -> Generation -> Output validation

## Key Limitations

Hughes et al. demonstrate 89% BoN success rates on GPT-4o despite current defenses. Fundamental architectural innovation may be necessary for persistent attack resistance.
