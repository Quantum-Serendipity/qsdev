# MCP-Guard: Defense Framework for Model Context Protocol Integrity
- **Source**: https://arxiv.org/html/2508.10991v1
- **Retrieved**: 2026-05-14

## What is MCP-Guard?

MCP-Guard is a security system designed to protect LLMs integrated with external tools via MCP. It addresses vulnerabilities like prompt injection, data exfiltration, and adversarial manipulation in LLM-tool ecosystems.

## Three-Stage Detection Pipeline

**Stage 1: Lightweight Static Scanning**
Pattern-based detectors identify obvious threats using regex matching and keyword filtering. Detectors target:
- SQL injection attacks
- Sensitive file access attempts
- Shell command injection
- Shadow hijack attacks
- Prompt injection attempts
- Cross-origin requests
- Malicious HTML tags

These operate in sub-millisecond timeframes, achieving approximately 95% accuracy while prioritizing precision to minimize false positives.

**Stage 2: Deep Neural Detection**
A fine-tuned E5 text embedding model addresses subtle, semantic-level attacks. This resource-intensive stage focuses on cases bypassing Stage 1, achieving "96.01% accuracy in identifying adversarial prompts."

**Stage 3: Intelligent Arbitration**
An LLM independently evaluates input safety using fixed prompts, classifying inputs as safe, unsafe, or uncertain. When uncertain, it defaults to Stage 2's neural detection probability against a threshold.

## Performance Results

The complete pipeline achieves:
- **89.63% accuracy** and **89.07% F1-score**
- **98.47% recall** (exceptionally low false negatives)
- **455.86 ms average latency** per request
- **12x speed improvement** over some competing systems

## Deployment Features

- Hot-updatable detectors enabling real-time threat adaptation
- Registry-free operation
- Scalable proxy-based architecture
- Low computational overhead

## MCP-AttackBench Dataset

The researchers created a benchmark containing "over 70,000 samples" across multiple attack categories, including jailbreak instructions, code-based attacks, prompt injection, data exfiltration, and tool-aware variants.

## Limitations

- Assumes MCP as the primary protocol
- Not tested for generalization beyond MCP ecosystems
- Network delays could affect reported latency metrics
- Open source status of framework itself unclear (dataset release confirmed)
