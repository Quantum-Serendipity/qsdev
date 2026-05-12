# Claude Code Hooks Mechanism: Deep Dive for Package Install Guardrails

## Executive Summary

Claude Code hooks are user-defined commands, HTTP endpoints, MCP tool calls, or LLM prompts that execute automatically at specific lifecycle points. The **PreToolUse** hook is the primary enforcement mechanism for package install guardrails: it fires before every tool call, receives the full command string as structured JSON, can pattern-match package install commands across all package managers, and can **block execution** via exit code 2 or structured JSON `permissionDecision: "deny"`. Critically, PreToolUse hooks fire *before* permission-mode checks and cannot be bypassed even by `--dangerously-skip-permissions`. Combined with the `updatedInput` capability (rewriting commands to safer versions), external API calls to vulnerability databases, and enterprise managed settings for mandatory enforcement, hooks provide a robust foundation for preventing AI agents from installing compromised packages.

---

## 1. How Claude Code Hooks Work

### 1.1 Lifecycle Events

Claude Code supports 27+ hook events organized into three cadences:

| Cadence | Events |
|---|---|
| Once per session | `SessionStart`, `SessionEnd`, `Setup` |
| Once per turn | `UserPromptSubmit`, `Stop`, `StopFailure`, `UserPromptExpansion` |
| Every tool call | **`PreToolUse`**, **`PostToolUse`**, `PostToolUseFailure`, `PermissionRequest`, `PermissionDenied`, `PostToolBatch` |
| Other | `SubagentStart/Stop`, `TaskCreated/Completed`, `ConfigChange`, `CwdChanged`, `FileChanged`, `PreCompact/PostCompact`, `Notification`, `WorktreeCreate/Remove`, `InstructionsLoaded`, `Elicitation/ElicitationResult`, `TeammateIdle` |

For package install guardrails, the critical events are:

- **`PreToolUse`** — fires before every tool call. Can **block** execution. This is the primary enforcement point.
- **`PostToolUse`** — fires after a tool call succeeds. Can add context but **cannot undo** the action. Useful for post-install auditing (e.g., running `npm audit` after install).
- **`PostToolBatch`** — fires after a batch of parallel tool calls. Can block before the next model turn.
- **`PermissionRequest`** — fires when a permission dialog appears. Can auto-allow or auto-deny.

### 1.2 Hook Handler Types

Five types of hook handlers are available:

| Type | Mechanism | Best For |
|---|---|---|
| `command` | Runs a shell command; receives JSON on stdin | Local validation scripts, regex matching |
| `http` | POSTs event JSON to a URL | External policy services, team-wide enforcement |
| `mcp_tool` | Calls a tool on a connected MCP server | Leveraging existing MCP security tooling |
| `prompt` | Single-turn LLM evaluation | Judgment calls that regex can't handle |
| `agent` | Spawns subagent with Read/Grep/Glob tools | Multi-step verification against codebase state |

For package install guardrails, `command` hooks are the most common and well-tested approach. `http` hooks are valuable for team-wide policy enforcement via a central service. `prompt` hooks could evaluate whether a package name looks suspicious but add latency and non-determinism.

### 1.3 Configuration Format

Hooks are configured in `settings.json` under the `hooks` key:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/validate-package-install.sh",
            "timeout": 30,
            "statusMessage": "Checking package safety..."
          }
        ]
      }
    ]
  }
}
```

Settings file locations and their scopes:

| File | Scope | Shareable | Overridable |
|---|---|---|---|
| `~/.claude/settings.json` | User (all projects) | No | By project/local |
| `.claude/settings.json` | Project (all collaborators) | Yes (git) | By local |
| `.claude/settings.local.json` | Local (you, this repo) | No (gitignored) | — |
| `/etc/claude-code/managed-settings.json` | Managed (all users) | Admin-deployed | **Cannot be overridden** |

---

## 2. What Context Hooks Receive

### 2.1 Common Input Fields (All Hooks)

```json
{
  "session_id": "abc123",
  "transcript_path": "/home/user/.claude/projects/.../transcript.jsonl",
  "cwd": "/home/user/my-project",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse",
  "effort": { "level": "medium" },
  "agent_id": "optional-subagent-id",
  "agent_type": "optional-agent-name"
}
```

### 2.2 PreToolUse-Specific Input (Bash Tool)

When Claude runs a Bash command, the PreToolUse hook receives:

```json
{
  "session_id": "abc123",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {
    "command": "npm install axios",
    "description": "Install axios HTTP library",
    "timeout": 120000,
    "run_in_background": false
  }
}
```

The `tool_input.command` field contains the **full command string** that Claude is about to execute. This is the field to pattern-match against for package install detection.

### 2.3 Environment Variables Available

- `CLAUDE_PROJECT_DIR` — project root (use for portable script paths)
- `CLAUDE_PLUGIN_ROOT` — plugin installation directory
- `CLAUDE_PLUGIN_DATA` — plugin persistent data directory
- `CLAUDE_EFFORT` — current effort level
- `CLAUDE_ENV_FILE` — (SessionStart/CwdChanged only) for persisting env vars

---

## 3. Pattern-Matching Package Install Commands

### 3.1 Package Manager Command Patterns

A comprehensive hook must match all common package install syntaxes:

```python
INSTALL_PATTERNS = [
    # npm / npx
    (r'\bnpm\s+(install|i|add|ci)\b', 'npm'),
    (r'\bnpx\s+', 'npx'),
    # yarn
    (r'\byarn\s+(add|install)\b', 'yarn'),
    # pnpm
    (r'\bpnpm\s+(add|install|i)\b', 'pnpm'),
    # bun
    (r'\bbun\s+(add|install|i)\b', 'bun'),
    # pip / pip3
    (r'\bpip3?\s+install\b', 'pip'),
    (r'\bpipx?\s+install\b', 'pipx'),
    (r'\buv\s+(pip\s+install|add)\b', 'uv'),
    # cargo
    (r'\bcargo\s+(add|install)\b', 'cargo'),
    # go
    (r'\bgo\s+(get|install)\b', 'go'),
    # gem
    (r'\bgem\s+install\b', 'gem'),
    # nix
    (r'\bnix-env\s+-i\b', 'nix'),
    (r'\bnix\s+profile\s+install\b', 'nix'),
    # composer
    (r'\bcomposer\s+require\b', 'composer'),
    # apt / system
    (r'\b(apt|apt-get)\s+install\b', 'apt'),
    (r'\bbrew\s+install\b', 'brew'),
]
```

### 3.2 Extracting Package Names

After detecting an install command, extract the package name(s) for validation:

```python
def extract_packages(command: str, manager: str) -> list[str]:
    """Extract package names from an install command."""
    # Strip flags (words starting with -)
    parts = command.split()
    packages = []
    skip_next = False
    for i, part in enumerate(parts):
        if skip_next:
            skip_next = False
            continue
        if part.startswith('-'):
            # Some flags take arguments: --save-dev, --registry URL
            if part in ('--registry', '--save-prefix', '-g'):
                skip_next = True
            continue
        # Skip the command prefix itself
        if part in ('npm', 'pip', 'pip3', 'cargo', 'yarn', 'pnpm', 'bun',
                     'go', 'gem', 'install', 'add', 'i', 'get', 'require'):
            continue
        packages.append(part)
    return packages
```

### 3.3 The `if` Field for Efficient Filtering (v2.1.85+)

The `if` field uses permission rule syntax to avoid spawning the hook process for non-install commands:

```json
{
  "type": "command",
  "if": "Bash(npm install *)",
  "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/check-npm-package.sh"
}
```

However, `if` only matches simple patterns. For comprehensive package manager coverage, you'd need multiple hook entries or a single catch-all that does regex internally. The `if` field also fires when the command is "too complex to parse into subcommands" (compound commands), providing a safety net.

### 3.4 Handling Compound Commands

Claude often chains commands: `npm install axios && npm test`. The `if` field evaluates each subcommand independently and fires the hook if any subcommand matches. For shell-form hooks without `if`, the hook receives the entire compound command string and must parse it.

---

## 4. Blocking, Modifying, and Warning

### 4.1 Exit Code 2: Block Execution

The simplest blocking mechanism — write reason to stderr, exit 2:

```bash
#!/bin/bash
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command')

if echo "$COMMAND" | grep -qE '\bnpm\s+install\b'; then
  PACKAGE=$(echo "$COMMAND" | sed 's/.*npm install //' | awk '{print $1}')
  echo "BLOCKED: Package install '$PACKAGE' requires security review" >&2
  exit 2
fi
exit 0
```

When exit code 2 is used, stderr text is fed back to Claude as error feedback. Claude will see the message and can adjust its approach.

### 4.2 Structured JSON: Deny with Reason

More control via JSON output on stdout with exit code 0:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Package 'evil-pkg' has known CVE-2026-1234. Use 'safe-alternative' instead."
  }
}
```

### 4.3 Structured JSON: Escalate to User

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "ask",
    "permissionDecisionReason": "Package 'new-pkg' was published 2 days ago. Approve manually?"
  }
}
```

### 4.4 Rewriting Commands via `updatedInput`

The most powerful mechanism for package guardrails — rewrite the command to a safer version:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "updatedInput": {
      "command": "npm install axios@1.14.0"
    },
    "additionalContext": "Rewrote axios@1.14.1 (score: 40/100) to axios@1.14.0 (score: 71/100) per supply chain policy."
  }
}
```

**Warning**: When multiple PreToolUse hooks return `updatedInput`, the last one to finish wins, and order is non-deterministic (hooks run in parallel). Avoid having multiple hooks modify the same tool's input.

### 4.5 Decision Precedence

When multiple hooks match the same event, the most restrictive answer wins:
- `deny` overrides `ask`
- `ask` overrides `allow`
- `allow` overrides no-decision

This means a security hook can always block, even if other hooks allow.

---

## 5. External Validation: Calling Vulnerability APIs

### 5.1 Architecture

A PreToolUse hook script can call any external API before returning a decision. The hook timeout defaults to 600 seconds (10 minutes), configurable per hook. For package validation, a 30-second timeout is reasonable.

```python
#!/usr/bin/env python3
import json, sys, subprocess, urllib.request

input_data = json.load(sys.stdin)
command = input_data.get("tool_input", {}).get("command", "")

# Extract package name (simplified)
if "npm install" in command:
    pkg = command.split("npm install")[-1].strip().split()[0]
    
    # Call OSV.dev API
    req = urllib.request.Request(
        "https://api.osv.dev/v1/query",
        data=json.dumps({"package": {"name": pkg, "ecosystem": "npm"}}).encode(),
        headers={"Content-Type": "application/json"}
    )
    resp = urllib.request.urlopen(req, timeout=10)
    vulns = json.loads(resp.read())
    
    if vulns.get("vulns"):
        cve_ids = [v.get("id", "unknown") for v in vulns["vulns"][:3]]
        result = {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": f"Package '{pkg}' has known vulnerabilities: {', '.join(cve_ids)}"
            }
        }
        print(json.dumps(result))
        sys.exit(0)

sys.exit(0)  # Allow if no issues found
```

### 5.2 Available APIs and Tools

| Service | API | Latency | Coverage |
|---|---|---|---|
| **OSV.dev** | `api.osv.dev/v1/query` | ~100ms | Multi-ecosystem (npm, PyPI, crates.io, Go) |
| **Socket.dev** | `socket.dev/api` | ~200ms | Supply chain scores, malware detection, age gates |
| **npm audit** | Local CLI | ~1-2s | npm-specific, uses GitHub Advisory Database |
| **pip-audit** | Local CLI | ~1-2s | PyPI-specific, uses OSV |
| **cargo-audit** | Local CLI | ~1s | crates.io-specific |
| **Snyk** | `api.snyk.io` | ~300ms | Multi-ecosystem, commercial |
| **Deps.dev** | `deps.dev/api` | ~100ms | Google's dependency metadata (licenses, versions, advisories) |

### 5.3 The attach-guard Plugin (Reference Implementation)

The `attach-guard` Claude Code plugin is the most complete existing implementation:

- Intercepts package install commands via PreToolUse hooks
- Calls Socket.dev API for supply chain risk scoring
- Blocks packages scoring below 50/100
- Flags packages scoring 50-70/100
- **Rewrites commands** to safer versions via `updatedInput`
- Supports npm, pip, Go, and Cargo
- Age-gates packages published within 48 hours

Installation:
```
claude plugin marketplace add attach-dev/attach-guard
claude plugin install attach-guard@attach-dev
```

---

## 6. Hook Interaction with the Permission System

### 6.1 Execution Order

1. Claude generates tool call parameters
2. **PreToolUse hooks fire** (all matching hooks run in parallel)
3. Hook decisions merged (most restrictive wins)
4. If not denied by hooks: permission-mode check runs
5. If not denied by permissions: tool executes
6. **PostToolUse hooks fire**

### 6.2 Critical Security Property

**PreToolUse hooks fire before any permission-mode check.** A hook returning `permissionDecision: "deny"` blocks the tool call **even in `bypassPermissions` mode or with `--dangerously-skip-permissions`**.

This means:
- Security hooks cannot be bypassed by changing permission modes
- Hooks can tighten restrictions but **cannot loosen** them past what permission rules allow
- A hook returning `"allow"` does NOT bypass deny rules from settings — settings deny rules always win

### 6.3 Permission Deny Rules (Complementary Layer)

In addition to hooks, `settings.json` supports explicit deny rules:

```json
{
  "permissions": {
    "deny": [
      "Bash(curl * | bash)",
      "Bash(curl * | sh)",
      "Bash(wget * | bash)",
      "Bash(wget * | sh)"
    ]
  }
}
```

**Limitation**: Deny rules use glob patterns and are evaluated against subcommands. They cannot perform the nuanced package-name validation that hooks enable (API calls, version checking, score evaluation). Deny rules are best used as a broad safety net, not as the primary package guardrail.

**Critical limitation from dwarvesf/claude-guardrails**: "Deny rules only cover Claude's built-in tools, not bash." `Read ~/.ssh/id_rsa` is denied, but `bash cat ~/.ssh/id_rsa` is not. For Bash commands, hooks are the enforcement mechanism, not deny rules.

---

## 7. Making Hooks Mandatory (Enterprise Enforcement)

### 7.1 Project-Level Hooks

Committing `.claude/settings.json` with hooks to the repository applies them to all collaborators:

```json
// .claude/settings.json (committed to git)
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh"
          }
        ]
      }
    ]
  }
}
```

**Limitation**: Users can override with `.claude/settings.local.json` or `~/.claude/settings.json`. They can also set `"disableAllHooks": true` in their user settings.

### 7.2 Managed Settings (Non-Overridable)

Enterprise administrators can deploy hooks via managed settings that **cannot be overridden**:

**Linux/NixOS**: `/etc/claude-code/managed-settings.json`
```json
{
  "allowManagedHooksOnly": true,
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/company/claude-hooks/package-guardrail.sh"
          }
        ]
      }
    ]
  }
}
```

When `allowManagedHooksOnly` is `true`:
- Only managed hooks, SDK hooks, and force-enabled plugin hooks load
- User hooks: **blocked**
- Project hooks: **blocked**
- Plugin hooks (except force-enabled): **blocked**
- Users **cannot** set `disableAllHooks` to override managed hooks

Drop-in directory for modular policy:
```
/etc/claude-code/
├── managed-settings.json
└── managed-settings.d/
    ├── 10-telemetry.json
    ├── 20-package-guardrails.json
    └── 30-file-protection.json
```

### 7.3 HTTP Hooks for Centralized Policy

For team-wide enforcement without per-machine deployment:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "http",
            "url": "https://security.company.com/hooks/validate-install",
            "headers": { "Authorization": "Bearer $GUARDRAIL_TOKEN" },
            "allowedEnvVars": ["GUARDRAIL_TOKEN"],
            "timeout": 15
          }
        ]
      }
    ]
  }
}
```

The Rulebricks approach exemplifies this: policy changes apply instantly across the team without git pull or restart.

---

## 8. Limitations and Bypass Vectors

### 8.1 What Hooks Cannot Do

1. **PostToolUse cannot undo actions.** The tool has already executed. PostToolUse can only log, warn, or add context for the next turn.
2. **Hooks communicate only via stdin/stdout/stderr and exit codes.** They cannot trigger `/` commands or tool calls.
3. **Shell profile pollution.** If `~/.bashrc` or `~/.zshrc` contains unconditional `echo` statements, they pollute stdout and break JSON parsing. Guard with `[[ $- == *i* ]]`.
4. **Non-deterministic `updatedInput` ordering.** When multiple hooks return `updatedInput`, the last to finish wins. Only one hook should modify input per tool call.
5. **No rollback mechanism.** If a package is installed and PostToolUse detects a problem, the package is already on disk.
6. **Timeout limitations.** Default 600s, configurable per hook. External API calls add latency. Network failures should be handled gracefully (fail-open vs fail-closed is a design choice).

### 8.2 Bypass Vectors

1. **Indirect installation via scripts.** Claude could write a shell script and then execute it, or use `make install`, or call a Makefile target that installs packages. The hook sees `bash make install`, not `npm install evil-pkg`.

2. **Obfuscated commands.** Claude could construct the install command via variable concatenation: `CMD="npm"; $CMD install evil-pkg`. Regex patterns may not catch this.

3. **Edit package.json directly.** Claude could use the `Write` or `Edit` tool to add a dependency to `package.json`/`requirements.txt`/`Cargo.toml` and then run the install command without specifying packages (e.g., `npm install` with no arguments). The PreToolUse hook on Bash sees only `npm install`, with no package name to validate.

4. **`disableAllHooks` (non-managed).** If hooks are only in project/user settings, a user can disable them. Only managed settings prevent this.

5. **Package aliasing.** `npm install safe-name@npm:evil-pkg` installs `evil-pkg` under the alias `safe-name`. The hook sees the alias, not the real package.

6. **Post-install scripts.** A seemingly safe package may execute malicious `postinstall` scripts. The hook validates the package name but cannot inspect what the package does after installation.

7. **Subagents.** Hooks fire on subagent tool calls too (verified: subagents inherit hooks), but if hooks are configured only for specific matchers, subagents using different patterns could slip through.

### 8.3 Mitigation Strategies

| Bypass Vector | Mitigation |
|---|---|
| Script execution | Also hook Write/Edit on package manifest files (`package.json`, `requirements.txt`, etc.) |
| Obfuscated commands | Use `prompt` or `agent` hook type for LLM-based analysis of complex commands |
| Direct manifest editing | PostToolUse hook on Edit/Write that triggers `npm audit` when manifest files change |
| `disableAllHooks` | Use managed settings with `allowManagedHooksOnly` |
| Package aliasing | Socket.dev API resolves aliases; OSV.dev checks by resolved name |
| Post-install scripts | OS-level sandboxing (bubblewrap on Linux) as defense-in-depth |
| Bare `npm install` after manifest edit | Stop hook that runs audit after each turn; `if` field matching `Bash(npm install)` (no args) |

### 8.4 The Project Hooks Attack Surface

A February 2026 security disclosure revealed that **malicious project files could define hooks that execute without user confirmation**. A cloned repository with a `.claude/settings.json` containing hooks could run arbitrary code. Mitigations:

- Scan cloned repos before opening: `find . -path "*/.claude/*" -o -name ".mcp.json" -o -name "CLAUDE.md"`
- Use managed settings with `allowManagedHooksOnly` to block project-level hooks entirely
- Review `.claude/settings.json` in any new repo before using Claude Code

---

## 9. Community Implementations

### 9.1 Existing Guardrail Projects

| Project | Approach | Package-Specific? |
|---|---|---|
| **attach-guard** | Plugin; PreToolUse + Socket.dev API | **Yes** — purpose-built for package install guardrails |
| **dwarvesf/claude-guardrails** | Settings + hooks; destructive command blocking | No — general security (rm -rf, git push, secrets) |
| **rulebricks/claude-code-guardrails** | PreToolUse + external policy API | No — general policy engine, but configurable for packages |
| **mafiaguy/claude-security-guardrails** | PreToolUse + PostToolUse; 60+ patterns + React dashboard | Partial — checks 16 known vulnerable packages by name |
| **Codacy integration** | MCP server; Trivy scanning | Partial — post-install dependency scanning, not pre-install blocking |

### 9.2 Reference Architecture for Package Guardrails

Based on the research, the optimal layered architecture is:

```
Layer 1: PreToolUse hook on Bash
  ├── Pattern-match install commands (npm/pip/cargo/go/nix/etc.)
  ├── Extract package name + version
  ├── Call vulnerability API (OSV.dev, Socket.dev)
  ├── Decision: deny / allow / rewrite to safe version / ask user
  └── Return structured JSON with permissionDecision

Layer 2: PreToolUse hook on Edit|Write
  ├── Detect changes to manifest files (package.json, requirements.txt, etc.)
  ├── Diff new vs old dependencies
  └── Validate new dependencies against same API

Layer 3: PostToolUse hook on Bash
  ├── Detect completed install commands
  ├── Run ecosystem audit tool (npm audit, pip-audit, cargo-audit)
  └── Add audit results as additionalContext for Claude

Layer 4: Stop hook
  ├── Run full dependency audit once per turn
  └── Block if critical vulnerabilities introduced

Layer 5: Managed settings (enterprise)
  ├── allowManagedHooksOnly: true
  ├── Deploy hooks via /etc/claude-code/managed-settings.json
  └── Cannot be overridden by users or projects
```

---

## 10. Practical Implementation Sketch

### 10.1 Minimal PreToolUse Hook (Bash/jq)

```bash
#!/usr/bin/env bash
# .claude/hooks/package-guardrail.sh
# Requires: jq, curl
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
[ -z "$COMMAND" ] && exit 0

# Pattern-match package install commands
if echo "$COMMAND" | grep -qE '\b(npm|yarn|pnpm|bun)\s+(install|add|i)\b'; then
  MANAGER="node"
elif echo "$COMMAND" | grep -qE '\bpip3?\s+install\b'; then
  MANAGER="pip"
elif echo "$COMMAND" | grep -qE '\bcargo\s+(add|install)\b'; then
  MANAGER="cargo"
elif echo "$COMMAND" | grep -qE '\bgo\s+(get|install)\b'; then
  MANAGER="go"
else
  exit 0  # Not a package install command
fi

# Extract package names (simplified: first non-flag arg after install verb)
PACKAGES=$(echo "$COMMAND" | grep -oP '(?:install|add|get|i)\s+\K[^\s-][^\s]*')

for PKG in $PACKAGES; do
  # Strip version specifier for API lookup
  PKG_NAME=$(echo "$PKG" | sed 's/@.*//' | sed 's/[>=<].*//')
  [ -z "$PKG_NAME" ] && continue

  # Map manager to ecosystem
  case "$MANAGER" in
    node) ECOSYSTEM="npm" ;;
    pip)  ECOSYSTEM="PyPI" ;;
    cargo) ECOSYSTEM="crates.io" ;;
    go)   ECOSYSTEM="Go" ;;
  esac

  # Query OSV.dev
  RESPONSE=$(curl -sf --max-time 10 \
    -X POST "https://api.osv.dev/v1/query" \
    -H "Content-Type: application/json" \
    -d "{\"package\":{\"name\":\"$PKG_NAME\",\"ecosystem\":\"$ECOSYSTEM\"}}" \
    2>/dev/null || echo '{}')

  VULN_COUNT=$(echo "$RESPONSE" | jq '.vulns | length // 0')

  if [ "$VULN_COUNT" -gt 0 ]; then
    VULN_IDS=$(echo "$RESPONSE" | jq -r '[.vulns[].id] | .[0:3] | join(", ")')
    echo "BLOCKED: Package '$PKG_NAME' has $VULN_COUNT known vulnerabilities ($VULN_IDS)" >&2
    exit 2
  fi
done

exit 0
```

### 10.2 Settings Configuration

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Validating package safety..."
          }
        ]
      }
    ]
  }
}
```

---

## 11. Comparison: Hooks vs. Alternative Approaches

| Approach | Deterministic? | Pre-Install Block? | Bypass Resistance | Maintenance |
|---|---|---|---|---|
| **PreToolUse hooks** | Yes | Yes | High (fires before permissions) | Medium (maintain regex + API calls) |
| **CLAUDE.md instructions** | No (LLM may ignore) | No (advisory only) | Low | Low |
| **Permission deny rules** | Yes | Yes (for simple patterns) | Medium | Low |
| **MCP server (Codacy-style)** | No (agent must call tool) | No | Low | Medium |
| **Custom skills** | No (agent must use skill) | No | Low | Medium |
| **OS sandboxing (bubblewrap)** | Yes | No (blocks network/filesystem) | High | High |

**Conclusion**: PreToolUse hooks are the only mechanism that is both deterministic (always fires), pre-execution (blocks before install), and high bypass-resistance (fires before permission checks, cannot be disabled by agent). All other approaches are either advisory or post-execution.

---

## Sources

- [Claude Code Hooks Reference (Official)](docs/claude-code-hooks-reference-official.md)
- [Claude Code Hooks Guide (Official)](docs/claude-code-hooks-guide-official.md)
- [Claude Code Settings Reference (Official)](docs/claude-code-settings-reference.md)
- [Anthropic Bash Command Validator Example](docs/anthropic-bash-command-validator-example.md)
- [attach-guard Plugin (Package-Specific)](docs/attach-guard-plugin.md)
- [dwarvesf/claude-guardrails](docs/dwarvesf-claude-guardrails.md)
- [rulebricks/claude-code-guardrails](docs/rulebricks-claude-code-guardrails.md)
- [mafiaguy/claude-security-guardrails](docs/mafiaguy-claude-security-guardrails.md)
- [Codacy Guardrails (MCP Approach)](docs/codacy-claude-code-guardrails.md)
- [paddo.dev Guardrails Article](docs/paddo-dev-hooks-guardrails.md)
- [disler/claude-code-hooks-mastery](docs/disler-hooks-mastery.md)
