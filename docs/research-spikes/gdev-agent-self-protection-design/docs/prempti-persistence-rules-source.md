<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/default/coding_agents_rules.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Extraction: Persistence Vectors domain rules and macros -->

# Prempti Persistence Vector Rules (from source)

## Referenced Lists

**env_file_names:** `.env, .envrc, .env.local, .env.development, .env.production, .env.staging, .env.test, .env.ci, .env.override`

**registry_config_files:** `.npmrc, .pypirc, pip.conf, .pnpmrc, .yarnrc.yml`

## Macros

**is_claude_settings_write**
```
tool.name in ("Write", "Edit")
and tool.real_file_path != ""
and (tool.real_file_path endswith "/.claude/settings.json"
     or tool.real_file_path endswith "/.claude/settings.local.json")
```

**is_settings_hooks_content**
```
tool.input contains "hooks"
```

**is_settings_mcp_content**
```
tool.input contains "mcpServers"
```

**is_git_hooks_write**
```
tool.name in ("Write", "Edit")
and tool.real_file_path != ""
and tool.real_file_path contains "/.git/hooks/"
```

**is_bash_git_hooks_write**
```
tool.name = "Bash"
and tool.input_command contains "/.git/hooks/"
```

**is_registry_config_write**
```
tool.name in ("Write", "Edit")
and tool.file_path != ""
and basename(tool.file_path) in (registry_config_files)
```

**is_registry_redirect_content**
```
tool.input contains "registry="
or tool.input contains "npmRegistryServer"
or tool.input contains "index-url"
or tool.input contains "extra-index-url"
or tool.input contains "index-server"
```

**is_bash_npm_registry_redirect**
```
tool.input_command contains "npm config set registry"
or tool.input_command contains "npm set registry"
```

**is_bash_pip_registry_redirect**
```
(tool.input_command contains "pip config set"
 or tool.input_command contains "pip3 config set")
and (tool.input_command contains "index-url"
     or tool.input_command contains "extra-index-url")
```

**is_bash_registry_redirect**
```
is_bash_npm_registry_redirect
or is_bash_pip_registry_redirect
```

**is_env_file_write**
```
tool.name in ("Write", "Edit")
and tool.file_path != ""
and basename(tool.file_path) in (env_file_names)
```

**is_api_base_url_override**
```
tool.input contains "ANTHROPIC_BASE_URL"
or tool.input contains "OPENAI_BASE_URL"
or tool.input contains "OPENAI_API_BASE"
```

**is_api_key_in_env**
```
tool.input contains "ANTHROPIC_API_KEY"
or tool.input contains "OPENAI_API_KEY"
or tool.input contains "GEMINI_API_KEY"
```

## Rules

**Ask before agent writing hooks into Claude Code settings**
- Priority: WARNING
- Condition: `is_claude_settings_write and is_settings_hooks_content`
- Tags: `[coding_agent_ask, AML.T0051_llm_prompt_injection, AML.T0043_craft_adversarial_data, mitre_t1546_event_triggered_execution]`
- Output: "Falco requires confirmation before %agent.name writes a hooks entry into Claude Code settings at %tool.real_file_path"

**Ask before agent registering MCP servers in Claude settings**
- Priority: WARNING
- Condition: `is_claude_settings_write and is_settings_mcp_content`
- Tags: `[coding_agent_ask, AML.T0048_llm_plugin_compromise, AML.T0051_llm_prompt_injection, mitre_t1059_command_and_scripting_interpreter]`
- Output: "Falco requires confirmation before %agent.name registers an MCP server in Claude Code settings at %tool.real_file_path"

**Ask before writing to git hooks directory**
- Priority: WARNING
- Condition: `is_git_hooks_write`
- Tags: `[coding_agent_ask, AML.T0043_craft_adversarial_data, mitre_t1546_event_triggered_execution]`
- Output: "Falco requires confirmation before writing a git hook at %tool.real_file_path"

**Ask before writing package registry redirect**
- Priority: WARNING
- Condition: `is_registry_config_write and is_registry_redirect_content`
- Tags: `[coding_agent_ask, AML.T0010_ml_supply_chain_compromise, mitre_t1195_supply_chain_compromise]`
- Output: "Falco requires confirmation before writing a package registry redirect to %tool.real_file_path"

**Ask before API base URL override in environment file**
- Priority: WARNING
- Condition: `is_env_file_write and is_api_base_url_override`
- Tags: `[coding_agent_ask, AML.T0051_llm_prompt_injection, AML.T0037_llm_adversarial_example, mitre_t1565_data_manipulation]`
- Output: "Falco requires confirmation before %agent.name overrides an AI API base URL in %tool.real_file_path"

**Ask before writing AI API key to environment file**
- Priority: WARNING
- Condition: `is_env_file_write and is_api_key_in_env`
- Tags: `[coding_agent_ask, AML.T0037_llm_adversarial_example, mitre_t1552_unsecured_credentials]`
- Output: "Falco requires confirmation before %agent.name writes an AI API key variable to %tool.real_file_path"

**Ask before Bash command accessing git hooks directory**
- Priority: WARNING
- Condition: `is_bash_git_hooks_write`
- Tags: `[coding_agent_ask, AML.T0043_craft_adversarial_data, mitre_t1546_event_triggered_execution]`
- Output: "Falco requires confirmation before a Bash command accesses the git hooks directory (%tool.input_command)"

**Ask before Bash command redirecting package registry**
- Priority: WARNING
- Condition: `tool.name = "Bash" and is_bash_registry_redirect`
- Tags: `[coding_agent_ask, AML.T0010_ml_supply_chain_compromise, mitre_t1195_supply_chain_compromise]`
- Output: "Falco requires confirmation before a Bash command redirects the package registry (%tool.input_command)"
