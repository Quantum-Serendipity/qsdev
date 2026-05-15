<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/default/coding_agents_rules.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Extraction: Self-Protection domain rules and macros -->

# Prempti Self-Protection Rules (from source)

## Macros

**is_premptictl_invocation**
```
condition: is_bash and tool.input_command contains "premptictl"
```

**is_service_stop_linux**
```
condition: agent.os = "linux" and is_bash and (
  tool.input_command contains "systemctl --user stop prempti"
  or tool.input_command contains "systemctl --user disable prempti"
  or tool.input_command contains "systemctl stop prempti"
  or tool.input_command contains "systemctl disable prempti"
  or tool.input_command contains "loginctl disable-linger"
  or tool.input_command contains "pkill falco"
  or tool.input_command contains "pkill -f falco"
  or tool.input_command contains "killall falco"
)
```

**is_service_stop_macos**
```
condition: agent.os = "macos" and is_bash and (
  tool.input_command contains "launchctl unload" and tool.input_command contains "prempti"
  or tool.input_command contains "launchctl bootout" and tool.input_command contains "prempti"
  or tool.input_command contains "launchctl remove dev.falcosecurity.prempti"
  or tool.input_command contains "pkill falco"
  or tool.input_command contains "pkill -f falco"
  or tool.input_command contains "killall falco"
)
```

**is_service_stop_windows**
```
condition: agent.os = "windows" and is_bash and (
  tool.input_command contains "taskkill" and tool.input_command contains "falco"
  or tool.input_command contains "Stop-Process" and tool.input_command contains "falco"
  or tool.input_command contains "Stop-Service" and tool.input_command contains "falco"
  or (tool.input_command contains "reg delete"
    and tool.input_command contains "Run"
    and tool.input_command contains "prempti")
)
```

**is_prempti_install_path**
```
condition: (agent.os in (linux, macos) and tool.real_file_path contains "/.prempti/")
or (agent.os = "windows" and tool.real_file_path icontains "/AppData/Local/prempti/")
```

**is_claude_settings_path**
```
condition: tool.real_file_path endswith "/.claude/settings.json"
or tool.real_file_path endswith "/.claude/settings.local.json"
```

## Rules

**Deny premptictl invocation**
- Priority: CRITICAL
- Condition: `is_premptictl_invocation`
- Tags: `[coding_agent_deny, prempti_self_protection]`
- Output: "Falco blocked an agent invocation of the premptictl CLI (%tool.input_command)"

**Deny service-stop alternatives targeting Prempti**
- Priority: CRITICAL
- Condition: `is_service_stop_linux or is_service_stop_macos or is_service_stop_windows`
- Tags: `[coding_agent_deny, prempti_self_protection]`
- Output: "Falco blocked a platform service-stop command targeting Prempti (%tool.input_command)"

**Deny writes under Prempti install prefix**
- Priority: CRITICAL
- Condition: `tool.name in ("Write", "Edit") and is_prempti_install_path`
- Tags: `[coding_agent_deny, prempti_self_protection]`
- Output: "Falco blocked writing to %tool.real_file_path under the Prempti install prefix"

**Deny writes to Claude Code settings file**
- Priority: CRITICAL
- Condition: `tool.name in ("Write", "Edit") and is_claude_settings_path`
- Tags: `[coding_agent_deny, prempti_self_protection]`
- Output: "Falco blocked writing to Claude Code settings at %tool.real_file_path"

**Deny writes to Claude Code policy limits file**
- Priority: CRITICAL
- Condition: `tool.name in ("Write", "Edit") and tool.real_file_path endswith "/.claude/policy-limits.json"`
- Tags: `[coding_agent_deny, prempti_self_protection]`
- Output: "Falco blocked writing to Claude Code policy-limits.json at %tool.real_file_path"

**Ask before reading Claude Code settings file**
- Priority: WARNING
- Condition: `tool.name = "Read" and is_claude_settings_path`
- Tags: `[coding_agent_ask, prempti_self_protection]`
- Output: "Falco requires confirmation to read Claude Code settings at %tool.real_file_path"
