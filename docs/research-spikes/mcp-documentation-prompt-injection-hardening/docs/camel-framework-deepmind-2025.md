<!-- Source: https://afine.com/llm-security-prompt-injection-camel -->
<!-- Retrieved: 2026-05-14 -->

# CaMeL Framework: LLM Security Against Prompt Injection — Google DeepMind

## Overview

The CaMeL (Capabilities for Machine Learning) framework, released by Google's DeepMind team, represents an advancement in defending against prompt injection attacks. It builds upon Simon Willison's earlier Dual LLM pattern while addressing its architectural limitations.

## Core Architecture

CaMeL maintains the foundational separation of the Dual LLM approach through three key components:

1. **Privileged LLM (P-LLM)**: Processes trusted user input and handles tool access, including sensitive operations like sending emails or booking flights
2. **Quarantined LLM (Q-LLM)**: Processes untrusted data sources without access to tools
3. **Controller/CaMeL Interpreter**: Regular software managing interactions between components and enforcing security policies

## Key Innovation: Control and Data Flow Protection

The framework addresses a critical vulnerability in the original Dual LLM design. While the predecessor "secures only the control flow which is handled by the Privileged LLM," CaMeL protects both dimensions:

- **Control Flow**: The P-LLM translates user queries into pseudo-Python code determining which actions occur
- **Data Flow**: The interpreter tracks how data is processed and transformed across operations

## Capability-Based Security

CaMeL implements a metadata system where each value carries "capability" information including:

- **Origin tracking**: Where the value originates and which function created it
- **Permission controls**: Who can read or modify the value

The interpreter enforces security policies based on these capabilities. For example, even if an attacker manipulates the Q-LLM to modify an email recipient, the system would prevent sending to unauthorized addresses by checking capabilities against defined policies.

## Acknowledged Limitations

The framework explicitly cannot defend against text-to-text attacks, such as prompt injections instructing the Q-LLM to incorrectly summarize content. The authors suggest mitigating this by presenting users with data flow transparency to identify suspicious outputs.

## Additional Benefits

- **Cost optimization**: The planning phase requires sophisticated, expensive models, but routine tasks can use less powerful local models
- **Privacy enhancement**: Untrusted data can be processed locally via the Q-LLM, preventing exposure to external LLM providers who only receive user queries

The framework represents "a big step in a right direction in the LLM security field" by applying traditional security engineering principles to AI systems.
