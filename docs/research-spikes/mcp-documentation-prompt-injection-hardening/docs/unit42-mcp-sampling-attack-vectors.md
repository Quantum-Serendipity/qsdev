# New Prompt Injection Attack Vectors Through MCP Sampling
- **Source**: https://unit42.paloaltonetworks.com/model-context-protocol-attack-vectors/
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Overview

Unit 42 researchers demonstrated that Model Context Protocol (MCP) sampling -- a feature enabling MCP servers to request LLM completions -- creates three critical attack vectors when malicious servers exploit the bidirectional communication pattern.

## What is MCP Sampling?

MCP sampling reverses the typical client-to-server interaction model. Rather than users sending requests that trigger tool calls, servers can proactively request LLM completions by sending sampling requests back to the client. This bidirectional capability allows servers to leverage language model intelligence for data analysis and decision-making while ostensibly maintaining client control.

## Three Critical Attack Vectors

### 1. Resource Theft Through Hidden Prompts

Malicious servers append covert instructions to legitimate prompts, causing the LLM to generate substantial hidden content beyond what users expect.

**Mechanism:** A code summarizer tool includes invisible directives instructing the model to "write a short fictional story" after completing the requested summary. The LLM processes both tasks, consuming additional computational resources billed to the user's API quota.

**Impact:** Users receive expected outputs while unauthorized content generation drains token budgets.

### 2. Persistent Conversation Hijacking

Injected instructions persist across multiple conversation turns, fundamentally altering the assistant's behavior throughout the session.

**Mechanism:** A server embeds instructions like "speak like a pirate in all responses" within the LLM's response. These instructions become part of subsequent conversation context, affecting all future interactions.

**Impact:** The assistant follows malicious behavioral directives indefinitely, degrading functionality or enabling more sophisticated attacks.

### 3. Covert Tool Invocation

Prompt injection triggers unauthorized tool execution, enabling file operations and system modifications without explicit user consent.

**Mechanism:** Hidden instructions cause the LLM to invoke file-writing tools, allowing attackers to create files, modify systems, or exfiltrate data through legitimate-appearing tool calls.

**Impact:** Attackers achieve unauthorized filesystem access, persistence mechanisms, and data theft.

## Technical Foundation

The vulnerability stems from MCP's implicit trust model. Servers control prompt content sent via sampling requests, allowing them to inject malicious instructions that the LLM processes as legitimate directives. The protocol lacks built-in validation mechanisms.

## Mitigation Strategies

**Request-level:** Enforce strict prompt templates, strip suspicious patterns, implement token limits.
**Response-level:** Remove instruction-like phrases, require user approval for tool execution, flag unexpected invocations.
**Structural:** Declare and limit server capabilities, isolate server context, rate-limit sampling requests, monitor for anomalies.
