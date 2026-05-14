---
name: gdev-status
description: Show gdev configuration status including enabled tools, ecosystems, and security posture.
allowed-tools: Bash(gdev *) Read Grep Glob
---

# gdev status

## Current Environment

!`gdev status --json 2>/dev/null || echo '{"tools": {}}'`

## Instructions

1. **Present enabled tools**: List all currently enabled tools from the status output, organized by category:
   - **Security**: Supply chain guards, SAST, secret scanning
   - **AI Agent**: Postmortem, version sentinel, MCP servers
   - **Developer Experience**: Changelog, linting, formatting
   - **Infrastructure**: Registry proxies, build caches

2. **Ecosystem summary**: Show which language ecosystems are configured and their versions.

3. **Security posture**: Summarize the security configuration:
   - Which security tools are active
   - Hook configuration (pre-commit, package guard)
   - Any security tools that are available but not enabled

4. **Warnings**: Highlight any configuration issues, outdated settings, or recommended improvements.
