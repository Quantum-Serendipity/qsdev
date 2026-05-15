<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/default/coding_agents_rules.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Extraction: MCP and Skill Content domain rules and macros -->

# Prempti MCP and Skill Content Rules (from source)

## Lists

**ioc_hosting_domains**: pastebin.com, transfer.sh, file.io, termbin.com, ix.io, glot.io, hastebin.com, ghostbin.co

## Macros

**is_mcp_config_write**
- Condition: Write/Edit tool targeting `.mcp.json` or `managed-mcp.json` files

**is_mcp_command_temp**
- Condition: Input contains `/tmp/`, `/dev/shm/`, `/var/tmp/`, `$TMPDIR`, `$TMP`, `$TEMP`, or `/run/user/`

**is_mcp_command_encoded**
- Condition: Input contains "base64"

**is_mcp_self_register**
- Condition: Bash commands using `claude mcp add`, `install`, `add-json`, `add-from-claude-desktop`, `plugin install`, or `skill add`

**is_skill_commands_path**
- Condition: Real file path contains `/.claude/commands/`

**is_skill_content_ioc**
- Condition: Skill commands path AND contains IOC domain

**is_skill_pipe_bash**
- Condition: Skill commands path AND contains patterns: `| bash`, `|bash`, `| sh`, `|sh`, `bash <(`, `sh <(`, absolute shell paths

**is_npx_auto_accept_mcp_skill**
- Condition: Package runner with `-y`/`--yes` flags (or `yarn dlx`/`pnpm dlx`) AND MCP/skill/plugin keywords

**is_bash_commands_write**
- Condition: Bash tool referencing `/.claude/commands/`

## Rules

**Deny MCP config with command from temporary directory**
- Priority: CRITICAL
- Condition: `is_mcp_config_write and is_mcp_command_temp`
- Tags: `[coding_agent_deny, AML.T0048_llm_plugin_compromise, AML.T0051_llm_prompt_injection, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco blocked writing an MCP server config with a command path in temporary storage at %tool.real_file_path"

**Deny MCP config with IOC domain in server URL**
- Priority: CRITICAL
- Condition: `is_mcp_config_write and contains_ioc_domain`
- Tags: `[coding_agent_deny, AML.T0048_llm_plugin_compromise, AML.T0010_ml_supply_chain_compromise, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco blocked writing an MCP server config with a malicious hosting domain in the server URL at %tool.real_file_path"

**Ask before MCP config with encoded server command**
- Priority: WARNING
- Condition: `is_mcp_config_write and is_mcp_command_encoded`
- Tags: `[coding_agent_ask, AML.T0048_llm_plugin_compromise, AML.T0057_llm_data_poisoning, mitre_t1027_obfuscated_files_or_information]`
- Output: "Falco requires confirmation before writing an MCP server config that references base64 at %tool.real_file_path"

**Ask before agent self-registering MCP server**
- Priority: WARNING
- Condition: `tool.name = "Bash" and is_mcp_self_register`
- Tags: `[coding_agent_ask, AML.T0048_llm_plugin_compromise, AML.T0054_llm_jailbreak, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco requires confirmation before the agent self-registers an MCP server or plugin (%tool.input_command)"

**Ask before writing to Claude slash command directory**
- Priority: WARNING
- Condition: Write/Edit to `/.claude/commands/` path
- Tags: `[coding_agent_ask, AML.T0051_llm_prompt_injection, AML.T0043_craft_adversarial_data, mitre_t1546_event_triggered_execution]`
- Output: "Falco requires confirmation before writing a Claude slash command at %tool.real_file_path"

**Ask before writing CLAUDE.md outside working directory**
- Priority: WARNING
- Condition: Write/Edit to `CLAUDE.md` outside current working directory
- Tags: `[coding_agent_ask, AML.T0051_llm_prompt_injection, AML.T0043_craft_adversarial_data, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco requires confirmation before writing CLAUDE.md at %tool.real_file_path outside working directory %agent.real_cwd"

**Deny skill command file with IOC domain in content**
- Priority: CRITICAL
- Condition: `tool.name in ("Write", "Edit") and is_skill_content_ioc`
- Tags: `[coding_agent_deny, AML.T0051_llm_prompt_injection, AML.T0010_ml_supply_chain_compromise, mitre_t1546_event_triggered_execution]`
- Output: "Falco blocked writing a skill file with a malicious hosting domain in its content at %tool.real_file_path"

**Deny skill command file with pipe-to-shell in content**
- Priority: CRITICAL
- Condition: `tool.name in ("Write", "Edit") and is_skill_pipe_bash`
- Tags: `[coding_agent_deny, AML.T0051_llm_prompt_injection, AML.T0043_craft_adversarial_data, mitre_t1546_event_triggered_execution]`
- Output: "Falco blocked writing a skill file with a pipe-to-shell pattern in its content at %tool.real_file_path"

**Deny MCP server or skill install from untrusted host**
- Priority: CRITICAL
- Condition: Bash with npm/pip install from IOC hosting domains
- Tags: `[coding_agent_deny, AML.T0048_llm_plugin_compromise, AML.T0010_ml_supply_chain_compromise, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco blocked package install from untrusted host"

**Deny MCP server execution from temporary directory**
- Priority: CRITICAL
- Condition: Bash with MCP --stdio/--sse flags from /tmp paths
- Tags: `[coding_agent_deny]`
- Output: "Falco blocked MCP server execution from temporary directory"

**Ask before npx auto-accept MCP or skill installation**
- Priority: WARNING
- Condition: `tool.name = "Bash" and is_npx_auto_accept_mcp_skill`
- Tags: `[coding_agent_ask, AML.T0048_llm_plugin_compromise, AML.T0010_ml_supply_chain_compromise, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco requires confirmation before auto-accepting package installation for an MCP server or skill (%tool.input_command)"

**Ask before Bash command writing to Claude slash command directory**
- Priority: WARNING
- Condition: `is_bash_commands_write`
- Tags: `[coding_agent_ask, AML.T0051_llm_prompt_injection, AML.T0043_craft_adversarial_data, mitre_t1546_event_triggered_execution]`
- Output: "Falco requires confirmation before a Bash command accesses the Claude slash command directory (%tool.input_command)"

**Ask before writing to Claude Code subagent definitions directory**
- Priority: WARNING
- Condition: Write/Edit to `/.claude/agents/` path
- Tags: `[coding_agent_ask]`
- Output: "Falco requires confirmation before writing a Claude Code subagent definition at %tool.real_file_path"

**Ask before writing to Claude Code skills directory**
- Priority: WARNING
- Condition: Write/Edit to `/.claude/skills/` path
- Tags: `[coding_agent_ask]`
- Output: "Falco requires confirmation before writing a Claude Code skill at %tool.real_file_path"

**Ask before writing to Claude Code plugins directory**
- Priority: WARNING
- Condition: Write/Edit to `/.claude/plugins/` path
- Tags: `[coding_agent_ask]`
- Output: "Falco requires confirmation before writing to Claude Code plugin storage at %tool.real_file_path"

**Ask before writing to Claude Code settings backups directory**
- Priority: WARNING
- Condition: Write/Edit to `/.claude/backups/` path
- Tags: `[coding_agent_ask, prempti_self_protection]`
- Output: "Falco requires confirmation before writing to Claude Code settings backups at %tool.real_file_path"
