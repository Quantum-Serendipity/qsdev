# Claude Code: Security Documentation

- **Source URL**: https://code.claude.com/docs/en/security
- **Retrieved**: 2026-05-14

## How We Approach Security

### Permission-based Architecture
Claude Code uses strict read-only permissions by default. When additional actions are needed (editing files, running tests, executing commands), Claude Code requests explicit permission. Users control whether to approve actions once or allow them automatically.

### Built-in Protections
- **Sandboxed bash tool**: Sandbox bash commands with filesystem and network isolation
- **Write access restriction**: Can only write to the folder where it was started and its subfolders
- **Prompt fatigue mitigation**: Support for allowlisting frequently used safe commands
- **Accept Edits mode**: Auto-approves file edits and fixed set of filesystem Bash commands

## Prompt Injection Protections

### Core Protections
- **Permission system**: Sensitive operations require explicit approval
- **Context-aware analysis**: Detects potentially harmful instructions by analyzing the full request
- **Input sanitization**: Prevents command injection by processing user inputs
- **Command blocklist**: Blocks risky commands that fetch arbitrary content from the web like `curl` and `wget` by default

### Additional Safeguards
- **Network request approval**: Tools that make network requests require user approval by default
- **Isolated context windows**: Web fetch uses a separate context window to avoid injecting potentially malicious prompts
- **Trust verification**: First-time codebase runs and new MCP servers require trust verification (disabled with -p flag)
- **Command injection detection**: Suspicious bash commands require manual approval even if previously allowlisted
- **Fail-closed matching**: Unmatched commands default to requiring manual approval
- **Natural language descriptions**: Complex bash commands include explanations for user understanding

### Best Practices for Working with Untrusted Content
1. Review suggested commands before approval
2. Avoid piping untrusted content directly to Claude
3. Verify proposed changes to critical files
4. Use virtual machines (VMs) to run scripts and make tool calls, especially when interacting with external web services
5. Report suspicious behavior with /feedback

**Warning**: "While these protections significantly reduce risk, no system is completely immune to all attacks."

## MCP Security
- Claude Code allows users to configure MCP servers
- Encourages writing own servers or using trusted providers
- MCP server permissions are configurable
- Anthropic reviews connectors against listing criteria before adding to directory
- **Anthropic does not security-audit or manage any MCP server**

## Cloud Execution Security
- Isolated virtual machines per session
- Network access controls (configurable, can be disabled or restricted)
- Credential protection through secure proxy
- Branch restrictions for git push
- Audit logging
- Automatic cleanup after session completion
