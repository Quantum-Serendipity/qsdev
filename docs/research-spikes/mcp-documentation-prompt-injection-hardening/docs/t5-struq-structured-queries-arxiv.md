# StruQ: Structured Queries for Prompt Injection Defense
- **Source**: https://arxiv.org/html/2402.06363v2
- **Retrieved**: 2026-05-14

## How StruQ Works

StruQ operates through two integrated components: a secure front-end and a specially trained LLM. The system separates prompts (instructions) from user data into distinct channels, preventing attackers from injecting malicious instructions into data fields.

## Structured Query Format

The front-end encodes queries using reserved tokens as delimiters:
- `[MARK]` replaces `###`
- `[INST]`, `[INPT]`, `[RESP]` replace "instruction," "input," and "response"
- `[COLN]` replaces the colon

The front-end encodes the query into a special format, based on a hard-coded template that separates instructions from data portions.

## Prompt-Data Separation

The secure front-end filters user data to eliminate delimiter strings, ensuring attackers cannot forge structural markers. The system "filter[s] out any instances of those delimiters in the user data, so that these reserved tokens cannot be spoofed by an attacker."

## Training Approach: Structured Instruction Tuning

Rather than standard instruction tuning, StruQ uses a modified approach combining:
- 50% clean samples from standard datasets
- 25% samples with naive attacks injected into data
- 25% samples with completion attacks using fake delimiters

This teaches models to follow instructions in prompt sections exclusively.

## Attack Success Rates

Against Llama-7B, StruQ achieved:
- Manual attacks: <2% success rate
- Tree-of-Attacks: 97% -> 9%
- Greedy Coordinate Gradient: 97% -> 58%

## Key Limitations

The system faces notable constraints: it "only protects programmatic applications that use an API" and remains vulnerable to sophisticated optimization-based attacks. Additionally, "GCG attacks achieve a non-trivial attack success rate," indicating incomplete robustness.

## Applicability Beyond Chat

StruQ explicitly targets LLM-integrated applications using structured APIs rather than conversational chatbots, making it unsuitable for open-ended multi-turn interactions where users dynamically contribute both instructions and data.
