# Mitigating the Risk of Prompt Injections in Browser Use

- **Source**: https://www.anthropic.com/research/prompt-injection-defenses
- **Retrieved**: 2026-05-14
- **Publisher**: Anthropic

---

## What is Prompt Injection?

Prompt injection occurs when attackers embed malicious instructions within web content to manipulate AI agent behavior. As Anthropic explains, "every webpage an agent visits is a potential vector for attack." These hidden directives can redirect agents to perform unintended actions while processing legitimate tasks.

## Defense Mechanisms

Anthropic has implemented three primary defensive approaches:

### 1. Adversarial Training
Claude undergoes reinforcement learning exposure to prompt injections embedded in simulated web environments. The model receives positive feedback when correctly identifying and refusing malicious instructions, even when designed to appear authoritative or urgent.

### 2. Content Classification Systems
Anthropic deployed improved classifiers that scan untrusted content entering the model's context window. These systems detect "adversarial commands embedded in various forms -- hidden text, manipulated images, deceptive UI elements" and adjust model behavior accordingly.

### 3. Human Red Team Testing
Security researchers continuously probe the browser agent for vulnerabilities. Anthropic also participates in external industry-wide challenge competitions to benchmark robustness against competitors.

## Effectiveness Data

Claude Opus 4.5 achieved a 1% attack success rate against an internal adaptive "Best-of-N" attacker using 100 attempts per environment -- representing substantial improvement over earlier versions. However, Anthropic acknowledges this progress doesn't solve the problem entirely, with ongoing research needed as attack techniques evolve.
