# Little Canary: Prompt Injection Detection via Sacrificial Model Probes
- **Source**: https://github.com/hermes-labs-ai/little-canary
- **Retrieved**: 2026-05-14

## What It Is

Little Canary is an open-source Python package that detects prompt injection attacks before they reach your main LLM application. It uses a two-stage detection approach with sacrificial canary-model probes.

## Core Detection Mechanism

Uses sacrificial canary-model probes -- small, deliberately vulnerable LLMs exposed to incoming user input to observe how attacks compromise them behaviorally. "A compromised canary is a strong signal." By analyzing what happens to this expendable model, the system identifies attacks that string-based rules might miss.

## Two-Layer Architecture

**Layer 1: Structural Filter (~1ms)**
Regex-based detection of known attack patterns, plus decoding-then-rechecking for obfuscated payloads (base64, hex, ROT13, reverse encoding).

**Layer 2: Canary Probe (~250ms)**
Raw input feeds into a small sacrificial LLM (Qwen 2.5 1.5B by default) at temperature=0. System analyzes canary's response for compromise signals: persona adoption, instruction compliance, system prompt leakage, or refusal collapse.

## Deployment Modes

- **Block mode**: Hard-blocks detected attacks
- **Advisory mode**: Never blocks; flags for the production LLM
- **Full mode**: Blocks obvious attacks, flags ambiguous ones

## Performance Metrics

- 99.0% detection on TensorTrust (400 real attacks with Claude Opus)
- 94.8% detection with a 3B local model
- ~250ms latency per check
- 0% false positives on 40 realistic chatbot prompts

## Key Limitations

- Intentionally fails open -- if canary unavailable, inputs pass through unscreened
- Should NOT be used as sole control plane for autonomous agents
- Designed as an inbound risk sensor paired with outbound runtime controls
- Small local model may not catch sophisticated attacks that only affect larger models
