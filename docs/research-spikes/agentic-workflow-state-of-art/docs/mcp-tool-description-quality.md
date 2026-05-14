# MCP Tool Description Quality and Optimization

- **Sources**:
  - https://arxiv.org/html/2602.14878 (MCP Tool Descriptions Are Smelly!)
  - https://arxiv.org/abs/2505.03275 (RAG-MCP)
  - https://modelcontextprotocol.io/specification/2025-11-25
  - https://agents.md/
- **Retrieved**: 2026-03-15

## Tool Description Quality Research

### The "Smelly Descriptions" Paper (arxiv:2602.14878)

Key findings from analyzing 856 tools across 103 major MCP servers:
- **97.1%** of tool descriptions contain at least one "smell" (quality issue)
- **56%** fail to state their purpose clearly
- No prior study had investigated optimization of tool descriptions

The paper designs a semi-automated augmentor combining rubric-based augmentation with a Foundation Model to generate refined, comprehensive, factually consistent tool descriptions.

### Tool Description Smells Identified

Tool descriptions serve as the essential linguistic guide directing AI behavior — conveying functionality, constraints, and usage cues. They shape:
- Tool selection (which tool to use)
- Parameterization (what arguments to pass)
- Multi-step orchestration (how to chain tools)

Poor descriptions lead to wrong tool selection, incorrect parameters, and failed multi-step plans.

## Tool Scalability Challenges

As the number of MCP tools increases, agents must rely on names and descriptions to identify the correct tool while adhering to input schemas. This has led to:
- **MCP gateways**: Routing layers that filter tools before presenting to agents
- **Discovery layers**: Help agents navigate large tool ecosystems
- **RAG-MCP** (arxiv:2505.03275): Uses retrieval-augmented generation to mitigate "prompt bloat" from too many tool descriptions
- **MCP-Zero**: Active tool discovery for autonomous agents

## AGENTS.md Convention

A complementary approach — provide structured instructions to coding agents via project files:
- Standard Markdown file without schema requirements
- Covers: commands, testing, project structure, code style, git workflow, boundaries
- Hierarchical: agents walk directory tree, closest file takes precedence
- Supported by OpenAI Codex, Cursor, Google Jules, Factory, and others
- Top-tier files cover six core areas matching what senior engineers carry as tribal knowledge

## MCP 2026 Roadmap

- Standard metadata format served via .well-known for discoverability without live connection
- Structured schema for tool discovery that enables AI systems to adapt behavior based on available tools
- More intelligent routing of user requests to appropriate resources
