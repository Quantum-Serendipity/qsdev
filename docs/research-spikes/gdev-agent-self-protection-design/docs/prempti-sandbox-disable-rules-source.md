<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/default/coding_agents_rules.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Extraction: Sandbox Disable domain rules and macros -->

# Prempti Sandbox Disable Rules (from source)

## Macros

**is_agent_sandbox_config**
```
tool.real_file_path endswith "/.claude/settings.json"
or tool.real_file_path endswith "/.claude/settings.local.json"
or tool.real_file_path endswith "/.codex/config.toml"
or tool.real_file_path endswith "/.gemini/settings.json"
```

**is_sandbox_disable_content** (composite)
```
is_sandbox_disable_codex or is_sandbox_disable_enabled_false
or is_sandbox_toolsandboxing_false or is_sandbox_allow_unsandboxed
or is_sandbox_allow_unsandboxed_numeric or is_sandbox_disable_value_change
or is_sandbox_disable_value_zero or is_sandbox_disable_value_null
or is_sandbox_disable_gemini_none or is_sandbox_disable_gemini_disabled
```

Sub-macros:
- `is_sandbox_disable_codex`: tool.input contains "danger-full-access"
- `is_sandbox_disable_enabled_false`: tool.input contains "sandbox" and "false"
- `is_sandbox_toolsandboxing_false`: tool.input contains "toolSandboxing" and "false"
- `is_sandbox_allow_unsandboxed`: tool.input contains "allowUnsandboxedCommands" and "true"
- `is_sandbox_allow_unsandboxed_numeric`: tool.input contains "allowUnsandboxedCommands" and ":1" or ": 1"
- `is_sandbox_disable_value_change`: tool.input contains "enabled" and "false"
- `is_sandbox_disable_value_zero`: tool.input contains "enabled" and ":0" or ": 0"
- `is_sandbox_disable_value_null`: tool.input contains "enabled" and "null"
- `is_sandbox_disable_gemini_none`: tool.input contains "sandbox" and "none"
- `is_sandbox_disable_gemini_disabled`: tool.input contains "sandbox" and "disabled"

**is_bash_sandbox_settings_path**
```
tool.input_command contains ".claude/settings.json"
or tool.input_command contains ".claude/settings.local.json"
or tool.input_command contains ".codex/config.toml"
or tool.input_command contains ".gemini/settings.json"
```

**is_bash_sandbox_disable_cmd** (composite)
```
is_bash_disable_codex or is_bash_disable_sandbox_false
or is_bash_disable_sandbox_false_pyfalse or is_bash_disable_sandbox_null
or is_bash_disable_enabled_zero or is_bash_disable_toolsandboxing_false
or is_bash_disable_allow_unsandboxed or is_bash_disable_allow_unsandboxed_numeric
or is_bash_disable_sandbox_none or is_bash_disable_sandbox_disabled
```

**is_bash_sandbox_settings_replace**
```
is_bash_settings_sed_write or is_bash_settings_cp or is_bash_settings_mv
```
Where:
- `is_bash_settings_sed_write`: command contains "sed" and "-i" and settings path
- `is_bash_settings_cp`: command contains "cp " and settings path
- `is_bash_settings_mv`: command contains "mv " and settings path

**is_codex_sandbox_bypass**
```
is_codex_bypass_flag or is_codex_bypass_flag_underscore or is_codex_danger_flag
```
Where:
- `is_codex_bypass_flag`: command contains "dangerously-bypass-approvals-and-sandbox"
- `is_codex_bypass_flag_underscore`: command contains "dangerously_bypass_approvals_and_sandbox"
- `is_codex_danger_flag`: command contains "codex" and "danger-full-access"

**is_gemini_sandbox_env_bypass**
```
is_gemini_env_none or is_gemini_env_false or is_gemini_env_disabled or is_gemini_env_zero
```

## Rules

**Ask before agent writing sandbox-disable configuration**
- Priority: WARNING
- Condition: `tool.name in ("Write", "Edit") and tool.real_file_path != "" and is_agent_sandbox_config and is_sandbox_disable_content`
- Tags: `[coding_agent_ask, AML.T0054, AML.T0051, mitre_t1562.001]`
- Output: "Falco requires confirmation before %agent.name modifies its sandbox configuration at %tool.real_file_path"

**Ask before Claude Code per-command sandbox escape**
- Priority: WARNING
- Condition: `tool.name = "Bash" and tool.input contains "dangerouslyDisableSandbox" and tool.input contains "true"`
- Tags: `[coding_agent_ask, AML.T0054, AML.T0051, mitre_t1562.001]`
- Output: "Falco requires confirmation before Claude Code runs an unsandboxed Bash command (%tool.input_command)"

**Deny Bash command writing sandbox-disable content to agent settings file**
- Priority: CRITICAL
- Condition: `tool.name = "Bash" and is_bash_sandbox_settings_path and is_bash_sandbox_disable_cmd`
- Tags: `[coding_agent_deny, AML.T0054, AML.T0051, mitre_t1562.001]`
- Output: "Falco blocked %agent.name from disabling its sandbox via Bash command: %tool.input_command"

**Deny Codex CLI sandbox bypass flag**
- Priority: CRITICAL
- Condition: `tool.name = "Bash" and is_codex_sandbox_bypass`
- Tags: `[coding_agent_deny, AML.T0054, AML.T0051, mitre_t1562.001]`
- Output: "Falco blocked %agent.name from starting Codex with a sandbox bypass flag (%tool.input_command)"

**Deny Gemini CLI sandbox disable via environment variable**
- Priority: CRITICAL
- Condition: `tool.name = "Bash" and is_gemini_sandbox_env_bypass`
- Tags: `[coding_agent_deny, AML.T0054, AML.T0051, mitre_t1562.001]`
- Output: "Falco blocked %agent.name from disabling Gemini sandbox via environment variable (%tool.input_command)"

**Deny Bash command replacing agent sandbox settings file**
- Priority: CRITICAL
- Condition: `tool.name = "Bash" and is_bash_sandbox_settings_replace`
- Tags: `[coding_agent_deny, AML.T0054, AML.T0051, mitre_t1562.001]`
- Output: "Falco blocked %agent.name from replacing an agent sandbox settings file (%tool.input_command)"
