<!-- Source: https://github.com/tldrsec/prompt-injection-defenses -->
<!-- Retrieved: 2026-05-14 -->

# Prompt Injection Defenses: Comprehensive Overview

This GitHub repository by tldrsec catalogs practical and proposed defenses against prompt injection attacks on large language models.

## Primary Defense Categories

### Blast Radius Reduction
Assumes successful injection is inevitable and focuses on limiting damage through defensive architecture. Key principles include treating all LLM outputs as potentially malicious, implementing least-privilege access controls for API tokens, and restricting what actions the model can trigger. As NVIDIA's recommendations note: "all LLM productions be treated as potentially malicious, and that they be inspected and sanitized."

### Input Preprocessing
- *Paraphrasing*: Rephrasing user input to disrupt adversarial token sequences while preserving legitimate instructions
- *Retokenization*: Breaking tokens into smaller units to disrupt adversarial token combinations
- *Backtranslation*: Using models to infer original intent from responses, refusing prompts where backtranslated versions are refused
- *SmoothLLM*: Perturbing multiple input copies and aggregating predictions for detection

### Guardrails & Oversight Systems
Multi-layered filtering approaches including:
- Input guardrails screening for malicious content before LLM processing
- Output guardrails validating responses before reaching users
- Action guards gating high-risk operations like email sending
- Canary tokens embedded in prompts to detect leakage
- Tools like Llama Guard, NeMo Guardrails, and LLM Guard

### Taint Tracking
Monitoring untrusted data flow through systems, dynamically adjusting model permissions based on "taint levels" to restrict high-risk operations when untrusted input has been processed.

### Secure Threads / Dual LLM Pattern
Using multiple LLM instances with different permission levels:
- Privileged LLM handles trusted user input
- Quarantined LLM processes untrusted content without tool access
- Controller manages interactions and passes structured data between instances

### Ensemble Decisions
Multiple independent models cross-check outputs, making attacks less likely to compromise all models simultaneously, though at increased computational cost.

### Prompt Engineering Defenses
- *Post-prompting*: Placing user input after core instructions to prevent conflation
- *Instruction hierarchy*: Training models to prioritize privileged instructions over adversarial ones
- *Spotlighting*: Transforming inputs to signal their provenance
- *Structured queries*: Fine-tuning models to follow instructions only in designated prompt sections
- *Signed prompts*: Digitally signing trusted instructions for verification
- *Self-reminder*: Encapsulating queries in system prompts emphasizing responsible responses

### Robustness & Finetuning
Task-specific finetuning (Jatmo) and representation engineering approaches that modify model behavior at the activation level without retraining.

### Preflight Testing
Testing user input concatenation with non-deterministic prompts — erratic outputs suggest injection attempts.

## Notable Tools

- Rebuff: Detection via heuristics, LLM analysis, and vector databases
- NeMo-Guardrails: Programmable safety rails
- Guardrails: Input/output guards with risk quantification
- LLM Guard: Comprehensive filtering and monitoring

## Key Insights

The repository emphasizes that no single defense is foolproof. Multiple sources stress assuming injection will succeed and designing systems accordingly. Simon Willison advocates: "assume someone will successfully hijack your application. If they do, what access will they have?"

The most effective approaches combine multiple strategies — limiting permissions, monitoring both inputs and outputs, using structured data passing between models, and maintaining architectural separation between trusted and untrusted processing.
