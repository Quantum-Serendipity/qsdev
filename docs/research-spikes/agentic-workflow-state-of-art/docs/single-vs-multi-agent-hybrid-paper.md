# Single-agent or Multi-agent Systems? Why Not Both?

- **Source URL**: https://arxiv.org/abs/2505.18286
- **Retrieved**: 2026-03-15
- **Note**: Content extracted from arxiv abstract page.

## Abstract

The paper examines whether multi-agent systems (MAS) remain superior to single-agent systems (SAS) as LLM capabilities advance. Key finding: "the benefits of MAS over SAS diminish as LLM capabilities improve."

## Key Findings

### Performance Trade-offs
- MAS traditionally excelled through task decomposition and error correction via specialized agents
- Advanced LLMs (OpenAI o3, Gemini 2.5-Pro) now handle long-context reasoning, memory retention, and tool usage better, reducing MAS advantages
- The gap between approaches narrows significantly with frontier models

### Proposed Solution: Request Cascading
A hybrid paradigm using request cascading between MAS and SAS that balances efficiency and accuracy across various agentic applications.

### Results
- Accuracy improvements: 1.1-12%
- Cost reduction: up to 20%
- Addresses both complexity and runtime costs of traditional MAS deployments

## Main Conclusion

Rather than choosing between approaches, organizations should consider hybrid architectures that strategically combine single and multi-agent systems based on task requirements and model capabilities, improving overall system economics.
