# Protecting Against Indirect Prompt Injection Attacks in MCP
- **Source**: https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Overview
Microsoft's Sarah Young and Den Delimarsky address security vulnerabilities in Model Context Protocol (MCP), an Anthropic-led open standard for connecting LLMs to external data sources and tools.

## Attack Types Described

**Indirect Prompt Injection (XPIA)**
Adversaries embed malicious directives in external materials like documents or emails. When AI systems process this content, they misinterpret embedded commands as legitimate user instructions, potentially causing data theft, harmful content generation, or interaction manipulation.

**Tool Poisoning**
A specific XPIA variant where attackers insert malicious instructions within MCP tool metadata (names, descriptions). Since LLMs rely on these descriptions to determine which tools to invoke, compromised metadata can coerce unintended tool executions while remaining invisible to users. Remote server scenarios enable "rug pull" scenarios where tool definitions change after user approval.

## Defensive Strategies

**AI Prompt Shields**
Microsoft's solution employs four mechanisms:
- Detection through machine learning and natural language processing
- "Spotlighting" to distinguish system instructions from untrusted inputs
- Delimiters and datamarking establishing clear boundaries between trusted and external data
- Continuous threat monitoring and updates

**Supply Chain Security**
Treating foundation models, embeddings services, and context providers with identical rigor as traditional code dependencies. Leveraging GitHub Advanced Security features and Azure DevOps integration for comprehensive scanning.

## Foundational Principle
"98% of reported breaches would be prevented by robust security hygiene," highlighting MFA, least privilege, and timely system updates as essential.
