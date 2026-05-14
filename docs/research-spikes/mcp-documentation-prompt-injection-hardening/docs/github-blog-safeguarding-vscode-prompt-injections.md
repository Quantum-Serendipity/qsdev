# Safeguarding VS Code Against Prompt Injections
- **Source**: https://github.blog/security/vulnerability-research/safeguarding-vs-code-against-prompt-injections/
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Specific Security Mechanisms

VS Code implements a **multi-layered defense strategy**:

1. **User Confirmation Requirements**: Sensitive tools like `installExtension` mandate explicit user approval before execution. However, standard tools such as `read-files` execute automatically, creating an asymmetric trust model.

2. **URL Validation Overhaul**: The fetch_webpage tool originally used "regular expression comparison instead of properly parsing the URL," allowing domain-spoofing attacks. This has been corrected with decoupled URL validation from trusted domains.

3. **File Modification Safeguards**: The editFile tool now prevents modifications outside workspaces and "force user confirmation whenever sensitive files are edited, such as configuration files."

## Tool Output Trust Handling

The architecture treats tool outputs as data rather than instructions, but this separation proved insufficient:

> "VS Code properly separates tool output, user prompts, and system messages in JSON. However, on the backend side, all these messages are blended into a single text prompt for inference."

This blending allows LLMs to misinterpret data as directives -- a fundamental challenge that cannot be entirely eliminated through technical means alone.

## Trust Tiers for Tool Results

VS Code distinguishes between:
- **Auto-approved tools**: File reading, standard operations
- **MCP server tools**: Always require explicit user confirmation before execution
- **Extension-provided tools**: Subject to policy restrictions

## Sandboxing & Isolation

The article emphasizes environment-based containment:

> "Developer Containers allow developers to open and interact with code inside an isolated Docker container. In this case, Copilot runs tools inside a container rather than on your local machine."

Additionally, **GitHub Codespaces** provides cloud-based isolation, creating dedicated virtual machines for agent execution.

## Content Sanitization & Permission Models

- **Content Security Policy**: The Simple Browser tool uses sandbox directives to isolate rendered HTML
- **Workspace Trust**: Restricts task execution, limits settings modifications, and disables extensions in untrusted contexts
- **Policy Framework**: Organizations can "disallow specific capabilities (e.g. tools from extensions, MCP, or agent mode)"
