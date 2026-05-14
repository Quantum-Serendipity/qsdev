<!-- Source: https://arxiv.org/html/2503.00061v2 -->
<!-- Retrieved: 2026-05-14 -->

# Adaptive Attacks Break Defenses Against Indirect Prompt Injection Attacks on LLM Agents

## Authors
Qiusi Zhan, Richard Fang, Henil Shalin Panchal, Daniel Kang (NAACL 2025 Findings)

## The Eight Defenses Tested

The paper evaluated three categories of defenses:

### Detection-based (3):
- Fine-tuned DeBERTaV3 detector
- LLM-based detector using simple yes/no classification
- Perplexity filtering to flag anomalous inputs

### Input-level (4):
- Instructional prevention (warning against external commands)
- Data prompt isolation (using delimiters around tool responses)
- Sandwich prevention (repeating user instruction after tool response)
- Paraphrasing (rewording external content to disrupt adversarial strings)

### Model-level (1):
- Adversarial finetuning on unsuccessful attack cases

## Attack Success Rates

All adaptive attacks exceeded 50% success rates across both tested agents:

- **Vicuna-7B agent**: Original baseline 56% ASR; defenses reduced this to 12-39%, but adaptive attacks achieved 50%+ ASR across all defenses
- **Llama3-8B agent**: Original baseline 9% ASR; despite stronger resilience, adaptive attacks still achieved 50%+ ASR

The paper notes that "multi-objective GCG" and "two-stage GCG" were particularly effective at bypassing defenses designed specifically against adversarial strings.

## Adaptive Attack Methodology

Attackers assumed white-box access and crafted adversarial strings embedded in external content. Three main techniques were employed:

1. **Greedy Coordinate Gradient (GCG)**: Optimizes adversarial token sequences to maximize probability of generating target outputs like "Action: {attacker_tool}"

2. **Multi-objective GCG**: Adds stealth constraints to simultaneously fool detection models while manipulating the agent

3. **Two-stage GCG**: Designed specifically for paraphrasing defenses, generating strings that survive paraphrasing transformations

## Why Defenses Failed

The research revealed several fundamental vulnerabilities:

- **Detection evasion**: Multi-objective training reduced detection rates "from 61% to 1%" for fine-tuned detectors
- **Prompt brittleness**: Simple instructional warnings proved ineffective; adversarial strings override explicit safety directives
- **Token-level vulnerability**: Even defenses targeting gibberish strings (perplexity filtering) were bypassed using semantically coherent adversarial strings
- **Long-term limitations**: Adaptive attacks focused only on first-step manipulation, leaving second-stage data-stealing attacks partially resilient

## Conclusions

The authors conclude that "testing defenses against adaptive attacks" is essential, as non-adaptive testing dramatically underestimates vulnerabilities.
