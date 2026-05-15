<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/default/coding_agents_rules.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch refused verbatim reproduction. This is a complete inventory of all rules and macros. -->

# Prempti Default Rules Inventory

## Rules (58 total)

| Rule Name | Priority | Tags | Description |
|-----------|----------|------|-------------|
| Monitor activity outside working directory | NOTICE | -- | Logs file access outside session working directory |
| Ask before writing outside working directory | WARNING | coding_agent_ask | Requires confirmation for writes outside cwd, excluding ~/.claude/ |
| Deny reading sensitive paths | CRITICAL | coding_agent_deny | Blocks reads from ~/.ssh/, cloud credentials, environment files |
| Deny writing to sensitive paths | CRITICAL | coding_agent_deny | Blocks writes to /etc/, ~/.ssh/, cloud credential directories |
| Ask before agent writing sandbox-disable configuration | WARNING | coding_agent_ask | Requires confirmation when modifying agent sandbox settings |
| Ask before Claude Code per-command sandbox escape | WARNING | coding_agent_ask | Requires confirmation for dangerouslyDisableSandbox:true Bash calls |
| Deny Bash command writing sandbox-disable content to agent settings file | CRITICAL | coding_agent_deny | Blocks Bash commands that disable sandbox via agent config file |
| Deny Codex CLI sandbox bypass flag | CRITICAL | coding_agent_deny | Blocks Codex with --dangerously-bypass-approvals-and-sandbox flags |
| Deny Gemini CLI sandbox disable via environment variable | CRITICAL | coding_agent_deny | Blocks GEMINI_SANDBOX set to disabling values (none, false, 0) |
| Deny Bash command replacing agent sandbox settings file | CRITICAL | coding_agent_deny | Blocks cp/mv/sed operations replacing sandbox configuration files |
| Deny credential file access via Bash | CRITICAL | coding_agent_deny | Blocks Bash referencing credential paths or secret environment variables |
| Deny credential file access via Read tool | CRITICAL | coding_agent_deny | Blocks Read tool accessing ~/.aws/credentials, ~/.ssh/ keys, etc. |
| Deny destructive system commands | CRITICAL | coding_agent_deny | Blocks dd, mkfs, direct device writes, init 0, sudo su operations |
| Ask before potentially destructive shell commands | WARNING | coding_agent_ask | Requires confirmation for rm -rf, sudo rm, shutdown/reboot/halt |
| Deny pipe to shell interpreter | CRITICAL | coding_agent_deny | Blocks curl\|bash, wget\|sh, bash <(...) patterns |
| Deny encoded payload execution | CRITICAL | coding_agent_deny | Blocks base64 decoding and inline interpreter one-liners |
| Deny curl wget exfiltration | CRITICAL | coding_agent_deny | Blocks curl/wget with POST, upload, or pipe-to-shell data transfer |
| Ask before writing to tmp staging paths | WARNING | coding_agent_ask | Requires confirmation for writes to /tmp, /var/tmp, /dev/shm |
| Deny reverse shell via Bash | CRITICAL | coding_agent_deny | Blocks /dev/tcp redirection, nc/ncat exec, socat, mkfifo shells |
| Deny cloud metadata service access via Bash | CRITICAL | coding_agent_deny | Blocks curl/wget to AWS IMDS, GCP, Azure metadata endpoints |
| Deny credential directory archive via Bash | CRITICAL | coding_agent_deny | Blocks tar/zip/gzip compressing credential directories |
| Deny SSH reverse tunnel and SOCKS proxy | CRITICAL | coding_agent_deny | Blocks ssh -R, -D, -w covert channel operations |
| Ask before cron and scheduled task manipulation | WARNING | coding_agent_ask | Requires confirmation before modifying crontab or /etc/cron |
| Deny audit trail destruction | CRITICAL | coding_agent_deny | Blocks history -c, HISTSIZE=0, .bash_history/.zsh_history removal |
| Deny package publish | CRITICAL | coding_agent_deny | Blocks npm publish, twine upload, cargo publish, gem push |
| Ask before modifying shell startup files | WARNING | coding_agent_ask | Requires confirmation before editing .bashrc, .zshrc, .profile |
| Ask before writing agent instruction files outside working directory | WARNING | coding_agent_ask | Requires confirmation for .cursorrules, AGENTS.md outside cwd |
| Deny cross-agent authentication file access | CRITICAL | coding_agent_deny | Blocks reading another agent's OAuth/session credential files |
| Deny MCP server or skill install from untrusted host | CRITICAL | coding_agent_deny | Blocks npm/pip install from known malicious code hosting domains |
| Deny MCP server execution from temporary directory | CRITICAL | coding_agent_deny | Blocks MCP with --stdio/--sse flags from /tmp paths |
| Ask before Glob with credential directory patterns | WARNING | coding_agent_ask | Requires confirmation for Glob patterns targeting credential dirs |
| Deny MCP config with command from temporary directory | CRITICAL | coding_agent_deny | Blocks .mcp.json with "command" field pointing to /tmp |
| Deny MCP config with IOC domain in server URL | CRITICAL | coding_agent_deny | Blocks .mcp.json with malicious hosting domain in server URL |
| Ask before MCP config with encoded server command | WARNING | coding_agent_ask | Requires confirmation for .mcp.json containing "base64" |
| Ask before agent self-registering MCP server | WARNING | coding_agent_ask | Requires confirmation for claude mcp add/install via Bash |
| Ask before writing to Claude slash command directory | WARNING | coding_agent_ask | Requires confirmation for writes to .claude/commands/ |
| Ask before writing CLAUDE.md outside working directory | WARNING | coding_agent_ask | Requires confirmation for CLAUDE.md outside cwd |
| Deny skill command file with IOC domain in content | CRITICAL | coding_agent_deny | Blocks skill files with malicious hosting domains |
| Deny skill command file with pipe-to-shell in content | CRITICAL | coding_agent_deny | Blocks skill files containing | bash or | sh patterns |
| Ask before npx auto-accept MCP or skill installation | WARNING | coding_agent_ask | Requires confirmation for npx -y/--yes MCP/skill package installs |
| Ask before Bash command writing to Claude slash command directory | WARNING | coding_agent_ask | Requires confirmation for Bash accessing .claude/commands/ |
| Ask before writing to Claude Code subagent definitions directory | WARNING | coding_agent_ask | Requires confirmation for writes to .claude/agents/ |
| Ask before writing to Claude Code skills directory | WARNING | coding_agent_ask | Requires confirmation for writes to .claude/skills/ |
| Ask before writing to Claude Code plugins directory | WARNING | coding_agent_ask | Requires confirmation for writes to .claude/plugins/ |
| Ask before writing to Claude Code settings backups directory | WARNING | coding_agent_ask | Requires confirmation for writes to .claude/backups/ |
| Ask before agent writing hooks into Claude Code settings | WARNING | coding_agent_ask | Requires confirmation when writing "hooks" to settings.json |
| Ask before agent registering MCP servers in Claude settings | WARNING | coding_agent_ask | Requires confirmation for "mcpServers" in settings.json |
| Ask before writing to git hooks directory | WARNING | coding_agent_ask | Requires confirmation for writes to .git/hooks/ |
| Ask before writing package registry redirect | WARNING | coding_agent_ask | Requires confirmation for registry= or npmRegistryServer in config |
| Ask before API base URL override in environment file | WARNING | coding_agent_ask | Requires confirmation for ANTHROPIC_BASE_URL in .env files |
| Ask before writing AI API key to environment file | WARNING | coding_agent_ask | Requires confirmation for ANTHROPIC_API_KEY in .env files |
| Ask before Bash command accessing git hooks directory | WARNING | coding_agent_ask | Requires confirmation for Bash referencing .git/hooks/ |
| Ask before Bash command redirecting package registry | WARNING | coding_agent_ask | Requires confirmation for npm config set registry via Bash |
| Deny premptictl invocation | CRITICAL | coding_agent_deny | Blocks any agent execution of the premptictl CLI tool |
| Deny service-stop alternatives targeting Prempti | CRITICAL | coding_agent_deny | Blocks systemctl/launchctl/taskkill operations against Falco/Prempti |
| Deny writes under Prempti install prefix | CRITICAL | coding_agent_deny | Blocks Write/Edit under ~/.prempti/ or Windows prempti install path |
| Deny writes to Claude Code settings file | CRITICAL | coding_agent_deny | Blocks Write/Edit on ~/.claude/settings.json |
| Deny writes to Claude Code policy limits file | CRITICAL | coding_agent_deny | Blocks Write/Edit on ~/.claude/policy-limits.json |
| Ask before reading Claude Code settings file | WARNING | coding_agent_ask | Requires confirmation to read ~/.claude/settings.json |

## Macros (79 total)

Categories: working directory/sensitive paths, IOC domains, sandbox disable (file paths, write/edit content, bash patterns), Codex/Gemini bypass, threat detection (bash, general), credential/exfiltration, cross-agent auth, MCP/supply chain, MCP configuration, skill/command files, persistence (settings/config), environment file attacks, self-protection.
