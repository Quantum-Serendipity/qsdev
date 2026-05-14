<!-- Source: https://simonwillison.net/2025/Apr/9/mcp-prompt-injection/ -->
<!-- Retrieved: 2026-05-14 -->

# Model Context Protocol has prompt injection security problems — Simon Willison

## Overview
Simon Willison's April 2025 analysis reveals critical security flaws in the Model Context Protocol when integrated with LLMs. The fundamental problem: mixing tools that perform actions with exposure to untrusted input creates opportunities for confused deputy attacks.

## Primary Attack Vectors

**Rug Pulls and Tool Shadowing**
MCP tools can alter their definitions post-installation. A seemingly safe tool on Day 1 could redirect credentials by Day 7. Additionally, malicious servers can override trusted tools when multiple servers connect to the same agent.

**Tool Poisoning Attacks**
Attackers embed malicious instructions within tool descriptions — visible to LLMs but hidden from users. Willison provides an example where an `add()` function's description instructs the model to read private configuration files and exfiltrate them via HTTP requests.

**WhatsApp MCP Exploitation**
Researchers demonstrated exfiltrating message history by creating a fake "get_fact_of_the_day()" tool that later tricks the system into sending private messages to attacker-controlled numbers, using whitespace obfuscation to hide malicious data during display.

## Core Problem
As Willison notes: "These vulnerabilities are not inherent to MCP itself — they're present any time we provide tools to an LLM exposed to untrusted inputs."

## Recommendations
- **Clients**: Display initial tool descriptions, alert users to changes, avoid hiding scrollbars
- **Servers**: Avoid unsafe practices like unescaped `os.system()` calls
- **Users**: Evaluate tool combinations carefully before installation

Willison concludes that despite two-plus years of prompt injection awareness, effective universal mitigations remain elusive.
