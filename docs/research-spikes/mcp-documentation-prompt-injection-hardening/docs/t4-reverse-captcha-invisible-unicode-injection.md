# Reverse CAPTCHA: Evaluating LLM Susceptibility to Invisible Unicode Instruction Injection

- **Source**: https://arxiv.org/html/2603.00164v1
- **Retrieved**: 2026-05-14

## Unicode Character Types Tested

The research evaluated two encoding schemes:

1. **Zero-Width Binary (ZW)**: Uses zero-width space (U+200B) for 0 and zero-width non-joiner (U+200C) for 1, encoding each ASCII character as 8 binary digits.

2. **Unicode Tags**: Maps ASCII characters to U+E0000 plus their codepoint (e.g., 'R' = U+E0052), producing one invisible character per ASCII character.

## Models Evaluated

- **OpenAI**: GPT-5.2, GPT-4o-mini
- **Anthropic**: Claude Opus 4, Claude Sonnet 4, Claude Haiku 4.5

## Key Success Rates

**Tool Use Impact (most significant finding):**
- Claude Haiku: 0.8% -> 49.2% compliance (Cohen's h = 1.37)
- Claude Opus: 6.7% -> 51.1% compliance
- Claude Sonnet: 16.9% -> 71.2% compliance
- GPT-4o-mini: 0.1% -> 1.6% compliance
- GPT-5.2: 0.1% -> 20.6% compliance

**Provider-Specific Encoding Preferences (tools enabled):**
- GPT-5.2: 69-70% on zero-width binary; near-zero on Tags
- Claude Opus: 100% on Tags; 48-68% on zero-width binary
- Claude Sonnet: "highly susceptible to both encodings with tools"

## Reverse CAPTCHA Methodology

The framework uses "270 test cases spanning two encoding schemes, four hint levels, and two payload types." The hint gradient includes:
- Unhinted (no indication of hidden content)
- Codepoint hints (identifies specific Unicode codepoints)
- Full hints (complete encoding rules provided)
- Full hints + adversarial injection

## Defense Recommendations

1. **Input sanitization** targeting Unicode Tags block (U+E0000-E+E007F) and suspicious zero-width sequences
2. **Tool-use guardrails** flagging programmatic Unicode decoding patterns
3. **Tokenizer-level filtering** preventing model perception of hidden content
4. **Training-time hardening** against following decoded instructions

The authors note that "naive stripping of all zero-width characters risks breaking legitimate uses," recommending targeted pattern-based approaches instead.
