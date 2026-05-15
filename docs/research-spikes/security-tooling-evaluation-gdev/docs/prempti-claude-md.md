<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/CLAUDE.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: This is the full architecture document from the Prempti repo. Extremely detailed. -->

**Prempti** is a policy and visibility layer for AI coding agents. It intercepts tool calls (shell commands, file writes, web requests, etc.) before execution, evaluates them against Falco rules, and produces allow/deny/ask verdicts in real time. It operates entirely in user space with no elevated privileges.

It is not a sandbox or OS-level security boundary: at the hook level it only sees what the agent declares, not the runtime side effects of the resulting commands. It is a cooperative policy layer -- the agent receives LLM-friendly feedback on blocked or flagged actions and adapts -- meant to complement containment techniques, not replace them.

The project targets **Claude Code** on **Linux, macOS, and Windows**. The architecture is designed to accommodate other coding agents (e.g., Codex) in the future.

## Architecture

The system flows through: Coding Agent -> Interceptor (hook) -> Falco (nodriver) with embedded Plugin -> Rule Engine.

**Pipeline flow:**

1. The coding agent's hook API fires before each tool call. The interceptor captures structured event data and pauses tool execution while awaiting a verdict.
2. The interceptor sends the event to the plugin's embedded broker via Unix domain socket.
3. The plugin feeds the event to Falco's rule engine via the source plugin API (`next_batch`). Falco evaluates all loaded rules.
4. Matching rules generate alerts. Falco delivers them back to the plugin's embedded broker via `http_output` (localhost).
5. The broker determines the verdict from rule tags (`deny`, `ask`, or allow-by-default) and responds to the interceptor.
6. The interceptor communicates the verdict to the coding agent using the standard hook response format.

## Components

| Component | Location | Language | Role |
|-----------|----------|----------|------|
| **Interceptor** | `hooks/claude-code/` | Rust | Thin passthrough: reads hook JSON from stdin, wraps in envelope, sends to broker, maps verdict to stdout. No content interpretation. |
| **Plugin** | `plugins/` | Rust (falco_plugin SDK) | Falco source+extract plugin with embedded broker. Parses events, extracts fields, feeds Falco, receives alerts, resolves verdicts. |
| **Supervisor** | `tools/premptictl/src/daemon/` | Rust | `ctl daemon` subcommand. Spawns Falco, captures and rotates its stdout/stderr into the log files, owns the Claude Code hook lifecycle, exposes a control socket for graceful shutdown. The init system (systemd / launchd / Windows Run key) starts the supervisor; the supervisor starts Falco. |
| **Rules** | `rules/` | YAML (Falco rule language) | Vendor and local security policies. |
| **Installer** | `installers/linux/`, `installers/macos/`, `installers/windows/` | Shell/PowerShell | Platform-specific packaging, installation, hook registration, mode switching. |
| **Skills** | `skills/` | Claude Code skill format | Coding agent skills for rule authoring, status, etc. |
| **Tests** | `tests/` | Rust | Cross-platform interceptor and E2E integration tests. |

## Key Design Decisions

### Broker embedded in plugin

The broker is part of the Falco plugin, not a separate process. This reduces moving parts: Falco is the only process the user needs to run (besides the stateless interceptor). The plugin spawns threads for the Unix socket server (accepting interceptor connections) and the HTTP server (receiving Falco alerts).

### Tags for verdict resolution

Rule verdicts are encoded in the `tags:` field of Falco rules, not in the `output:` string. The tag names are **configurable in the plugin configuration** and support multiple tags per verdict type. Defaults:

- `tags: [coding_agent_deny]` -- block the tool call
- `tags: [coding_agent_ask]` -- require user confirmation
- No deny/ask tag -- allow (no explicit allow tag needed)

There is no allow tag because the absence of a verdict IS the allow verdict. Rules only fire when their condition matches -- a tool call that doesn't match any deny or ask rule simply produces no deny/ask alert, and the broker resolves it as allow via batch-completion.

The broker parses the `tags` array from Falco's JSON alert output. Verdict escalation applies when multiple rules match: deny > ask > allow.

### Catch-all seen rule + HTTP verdict resolution

All verdict signals flow through Falco's `http_output` to the plugin's embedded HTTP server:

- Deny/ask alerts (from matching rules) resolve the pending request immediately.
- A **catch-all "seen" rule** (tagged `coding_agent_seen`) fires for every event. When the broker receives this alert, it knows rule evaluation is complete. If no deny/ask alert arrived for that correlation ID, the request is resolved as allow.

**Critical config**: `rule_matching: all` must be set in `falco.yaml`. The default (`first`) only fires one rule per event -- this would prevent both a deny rule and the seen rule from firing on the same event.

**Rule load ordering**: The seen rule must be loaded as the last rule file so that deny/ask rules fire first and their alerts are enqueued before the seen alert.

**HTTP handler constraints**: The handler must respond fast (Falco's output worker thread is shared across all output channels -- a slow handler blocks everything). The HTTP server must be ready before events flow (Falco does not retry on connection failure -- alerts are silently dropped).

The plugin requires two capabilities: **sourcing** (event generation) and **extraction** (field extraction for rules).

### Single data source, generic event fields

One Falco data source: **`coding_agent`**. Two field namespaces:

| Field | Type | Description |
|-------|------|-------------|
| `correlation.id` | u64 | Broker-assigned unique ID for this event (monotonic counter, always > 0) |
| `agent.name` | string | Coding agent identifier (e.g., `claude_code`) |
| `agent.os` | string | Host OS -- `linux`, `macos`, `windows`, or `unknown` |
| `agent.pid` | u64 | PID of the agent process that invoked the hook |
| `agent.hook_event_name` | string | Lifecycle hook type (e.g., `PreToolUse`) |
| `agent.session_id` | string | Session identifier |
| `agent.cwd` | string | Working directory, raw from Claude Code JSON |
| `agent.real_cwd` | string | Working directory, resolved to absolute canonical path |
| `tool.use_id` | string | Tool call identifier from Claude Code |
| `tool.name` | string | Tool name (e.g., `Bash`, `Write`, `Edit`) |
| `tool.input` | string | Full tool input as JSON |
| `tool.input_command` | string | Shell command (Bash tool calls) |
| `tool.file_path` | string | Target file path, raw |
| `tool.real_file_path` | string | Target file path, resolved to absolute canonical path |
| `agent.permission_mode` | string | Session permission mode reported by the agent |
| `agent.transcript_path` | string | Session transcript file path |

### Rule output convention

The rule `output:` field is an LLM-friendly sentence explaining what happened and why. It must start with "Falco" to attribute the verdict. Use resolved field values to make the message informative.

```yaml
output: >
  Falco blocked writing to %tool.real_file_path because it is a sensitive path
```

### Operational modes

Two plugin modes, switchable without reinstallation via `premptictl mode <guardrails|monitor>`:
- **Guardrails** (default) -- verdicts enforced (deny/ask/allow).
- **Monitor** -- rules evaluated and logged, but all verdicts resolve to allow.

### Fail-safety

- **Fail-closed**: if the plugin/Falco is unreachable, tool calls are denied.
- When the hook is registered and the service is stopped or restarting, ALL Claude Code tool calls are blocked. This is by design.

### Installation directory structure

All components are installed under `~/.prempti/`:

```
~/.prempti/
├── bin/                    # Executables: falco, claude-interceptor, premptictl
├── config/
│   ├── falco.yaml          # Base Falco config
│   ├── falco.coding_agents_plugin.yaml  # Plugin config
│   └── supervisor.yaml     # Supervisor config (preserved on upgrade)
├── log/                    # Falco logs (rotated by supervisor)
├── run/                    # Runtime: broker.sock, supervisor.sock
├── share/                  # Shared libraries
└── rules/
    ├── default/
    │   └── coding_agents_rules.yaml  # Default ruleset (overwritten on upgrade)
    ├── user/               # User custom rules (preserved on upgrade)
    └── seen.yaml           # Catch-all seen rule (loaded last)
```

### Falco configuration isolation

Falco runs with a fully isolated configuration -- no default files from `/etc/falco/`:
- `engine.kind: nodriver` -- no kernel driver needed
- `--disable-source syscall` removes syscall source entirely
- Config split into base settings and plugin fragment

## Technology Stack

- **Falco 0.43** -- rule engine, running in `nodriver` mode
- **Rust** -- interceptor and plugin (using `falco_plugin` crate v0.5.0)
- **Platforms** -- Linux (official Falco builds), macOS (Falco built from source), Windows (Falco built from source)

## Build & Development

Cargo workspace with three crates: hooks/claude-code, plugins/coding-agents-plugin, tools/premptictl.
Version 0.3.0 (Apache-2.0).
Requires latest stable Rust.
