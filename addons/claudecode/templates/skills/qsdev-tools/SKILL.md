---
name: qsdev-tools
description: List all available qsdev tools organized by category with enabled/disabled state.
allowed-tools: Bash(qsdev *) Read Grep Glob
---

# qsdev tools

## Current Environment

!`qsdev list --json 2>/dev/null || echo '{"available": []}'`

## Instructions

1. **Present tools by category**: Organize the available tools into categories and present them clearly:

   - **Security**: Tools for supply chain security, static analysis, secret detection, container scanning, license compliance
   - **AI Agent**: Tools for Claude Code integration, MCP servers, semantic search, verification protocols
   - **Developer Experience**: Tools for changelog generation, commit linting, formatting, code quality
   - **Infrastructure**: Tools for registry proxies, build caches, CI integration

2. **Show state**: For each tool, indicate whether it is:
   - Enabled (currently active in the project)
   - Disabled (available but not active)
   - Always-on (cannot be disabled)

3. **Provide descriptions**: Include a brief description of what each tool does.

4. **Enable instructions**: For disabled tools the user might want to enable, mention they can use `/qsdev-enable <tool-name>` to activate them.
