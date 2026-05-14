# StruQ vs SecAlign: BAIR Blog Comparison
- **Source**: https://bair.berkeley.edu/blog/2025/04/11/prompt-injection-defense/
- **Retrieved**: 2026-05-14

## Core Approaches

**StruQ (Structured Instruction Tuning)**: Supervised fine-tuning on simulated injection attacks. Model learns to ignore injected instructions.

**SecAlign (Special Preference Optimization)**: Preference-based training with desirable (follow intended instructions) and undesirable (follow injected ones) labels. Creates larger probability gap between correct and incorrect outputs.

## Attack Success Rates

- StruQ: 45% ASR against optimization-based attacks
- SecAlign: 8% ASR (4x improvement over StruQ)
- Both: ~0% ASR against optimization-free attacks

## Shared Infrastructure

Both rely on a Secure Front-End:
- Reserves special delimiter tokens ([MARK], etc.)
- Filters delimiters from untrusted data
- Explicitly separates trusted prompts from external data

## Utility Preservation

- SecAlign maintains AlpacaEval2 performance on Llama3-8B-Instruct
- StruQ shows minimal degradation (4.5% decrease)

## Relationship to OpenAI's Instruction Hierarchy

Mentioned as separate defense operating under "more general multi-layer security policy" but no direct comparison provided.

## Practical Deployment Steps

1. Initialize with instruction-tuned LLM
2. Use training datasets (Cleaned Alpaca)
3. Format with special delimiters via string concatenation (no human labor)
4. Apply preference optimization (DPO or alternatives)
5. Deploy with secure front-end filter

## Key Limitation for MCP Context

Both require model-level changes (fine-tuning). Cannot be applied by an MCP server operator who doesn't control the model. Only applicable if the model provider (Anthropic) adopts similar approaches in Claude's training.
