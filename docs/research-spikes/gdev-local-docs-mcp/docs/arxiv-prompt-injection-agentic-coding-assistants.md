# Prompt Injection Attacks on Agentic Coding Assistants

- **Source**: https://arxiv.org/pdf/2601.17548
- **Retrieved**: 2026-05-14
- **Publisher**: arXiv (Maloyan & Namiot)

---

## Overview
This arXiv paper (2601.17548) examines vulnerabilities in AI-powered coding assistants when subjected to prompt injection attacks, focusing on how these attacks exploit skills, tools, and protocol ecosystems.

## Key Research Areas

**Scope of Study:**
The research systematically analyzes prompt injection vulnerabilities across agentic coding systems, examining attack vectors targeting different architectural layers including tool integration, skill execution, and inter-protocol communication.

**Systems Evaluated:**
The paper examines contemporary coding assistants including Claude, GitHub Copilot, Cursor, and OpenAI Codex, with particular focus on how Model Context Protocol (MCP) implementations handle untrusted inputs.

## Attack Categories Identified

The researchers categorized vulnerabilities into three primary ecosystems:
- **Skills layer**: Function execution and parameter handling
- **Tools layer**: External tool integration and API interactions
- **Protocol layer**: MCP and similar specification implementation flaws

## Defense Mechanisms Discussed

The paper references multiple defensive approaches:
- Input validation and sanitization strategies
- LLM-based content filtering (LlamaGuard)
- Rule-based guardrails (NeMo GuardRails)
- Multi-agent defense coordination
- Instruction hierarchy enforcement

## Methodology Notes

The research applies systematic evaluation frameworks similar to AgentDojo and other established security benchmarks for agent systems, with testing against multiple threat models including both direct and indirect injection vectors.

## Key Finding

Attack success rate for prompt injections in auto-execution mode ranged from 66.9% to 84.1% across tested coding assistants.
