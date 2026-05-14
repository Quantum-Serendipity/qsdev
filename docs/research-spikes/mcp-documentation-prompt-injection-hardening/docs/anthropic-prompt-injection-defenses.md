# Anthropic: Mitigating the Risk of Prompt Injections in Browser Use

- **Source URL**: https://www.anthropic.com/research/prompt-injection-defenses
- **Retrieved**: 2026-05-14

## Overview
Anthropic addresses prompt injection risks—adversarial instructions hidden in web content—through a multi-layered defense approach centered on Claude Opus 4.5.

## Attack Success Rate
Claude Opus 4.5 achieved approximately "1% attack success rate" when evaluated against an internal adaptive attacker given 100 attempts per environment. Anthropic notes this represents "significant improvement" but acknowledges that "no browser agent is immune to prompt injection."

## Core Defense Mechanisms

### Reinforcement Learning Training
The model underwent specialized training using reinforcement learning to build robustness directly into capabilities. During development, Claude was "expose[d] to prompt injections embedded in simulated web content, and reward[ed]" when correctly identifying and refusing malicious instructions, even deceptive or urgent-appearing ones.

### Detection Classifiers
Anthropic implemented classifiers that "scan all untrusted content that enters the model's context window, and flag potential prompt injections." These systems detect "adversarial commands embedded in various forms—hidden text, manipulated images, deceptive UI elements." The organization improved these classifiers since the original Claude for Chrome research preview launch.

### Human Red Teaming
Internal security researchers continuously probe the browser agent, complemented by participation in external "Arena-style challenges" that benchmark robustness across the industry.

## Implementation Context
These defenses inform the expansion of Claude for Chrome from research preview to beta, now available to Max plan users.
