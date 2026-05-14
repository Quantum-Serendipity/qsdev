<!-- Source: https://www.anthropic.com/research/prompt-injection-defenses -->
<!-- Retrieved: 2026-05-14 -->

# Mitigating the Risk of Prompt Injections in Browser Use — Anthropic

## The Problem

Anthropic identifies prompt injection as a critical security challenge for browser-based AI agents. The threat emerges because "every webpage an agent visits is a potential vector for attack." Browser use amplifies risk through vast attack surfaces and the ability to take diverse actions like form-filling and file downloads.

## Defense Mechanisms

Anthropic employs three primary strategies:

### 1. Model Training Through Reinforcement Learning
The team builds robustness directly into Claude by exposing it to injections during training and rewarding correct identification of malicious instructions, even when deceptively framed.

### 2. Improved Classifiers
Anthropic scans untrusted content using classifiers that detect "adversarial commands embedded in various forms -- hidden text, manipulated images, deceptive UI elements." These classifiers have been strengthened since the initial browser extension preview.

### 3. Human Red Teaming
Security researchers continuously probe the system for vulnerabilities, noting that "human security researchers consistently outperform automated systems" at discovering creative attack vectors.

## Results and Limitations

Claude Opus 4.5 demonstrates improved robustness over previous models, achieving approximately 1% attack success rates against internal adaptive testing. However, Anthropic emphasizes this represents progress, not resolution: "A 1% attack success rate -- while a significant improvement -- still represents meaningful risk."

The organization commits to transparent progress reporting and ongoing research investment as attack techniques evolve.
