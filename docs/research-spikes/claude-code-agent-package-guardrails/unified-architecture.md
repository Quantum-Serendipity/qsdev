# Unified Defense Architecture: Claude Code Agent Package Guardrails

This document synthesizes findings from seven research reports into a single, actionable specification for preventing Claude Code agents from installing compromised, vulnerable, or malicious packages. Every configuration snippet is copy-pasteable. Every layer is justified by mechanism, not assumption.

---

## 1. Architecture Overview

### The Five-Layer Defense Model

```
                        ┌──────────────────────────────────────────────────────┐
                        │                 CLAUDE CODE AGENT                    │
                        │  (generates tool calls based on user conversation)   │
                        └─────────────────────────┬────────────────────────────┘
                                                  │ Tool call generated
                                                  ▼
 ┌─────────────────────────────────────────────────────────────────────────────────────┐
 │  ADVISORY LAYER A: CLAUDE.md Instructions + Custom Skills                          │
 │  Nature: Probabilistic (shapes what the agent TRIES to do)                         │
 │  Effect: Routes most installs through approved paths voluntarily                   │
 │  Failure: Agent may ignore under context pressure, prompt injection, or subagent   │
 │           delegation. Not inherited by built-in or custom subagents.               │
 └─────────────────────────────────────────────┬───────────────────────────────────────┘
                                               │ Agent submits tool call
                                               ▼
 ┌─────────────────────────────────────────────────────────────────────────────────────┐
 │  ENFORCEMENT LAYER 1: PreToolUse Hooks (PRIMARY ENFORCEMENT)                       │
 │  Nature: Deterministic (fires on EVERY tool call, before permission check)         │
 │  Effect: Parses command → extracts package → queries APIs → blocks/rewrites/asks   │
 │  Failure: Pattern evasion (obfuscated commands, variable expansion). Timeout if     │
 │           API is down (design choice: fail-open or fail-closed).                   │
 │  Cannot be bypassed by: --dangerously-skip-permissions, bypassPermissions mode     │
 └─────────────────────────────────────────────┬───────────────────────────────────────┘
                                               │ If not blocked by hook
                                               ▼
 ┌─────────────────────────────────────────────────────────────────────────────────────┐
 │  ENFORCEMENT LAYER 2: Permission Deny Rules (FAST CATCH)                           │
 │  Nature: Deterministic (glob pattern matching on command strings)                  │
 │  Effect: Blocks commands matching known-dangerous patterns instantly               │
 │  Failure: Shell wrappers (bash -c), env prefix, variable expansion, subprocess     │
 │           spawning. Cannot inspect package names or query APIs.                    │
 └─────────────────────────────────────────────┬───────────────────────────────────────┘
                                               │ If not denied by rules
                                               ▼
 ┌─────────────────────────────────────────────────────────────────────────────────────┐
 │  ENFORCEMENT LAYER 3: OS/Environment Configuration (FAILSAFE)                      │
 │  Nature: Deterministic (package manager and OS-level settings)                     │
 │  Effect: .npmrc ignore-scripts, pip only-binary, nix.conf sandbox, etc.            │
 │  Failure: Agent could modify config files (mitigated by deny rules on config       │
 │           paths). Only covers configured package managers.                         │
 └─────────────────────────────────────────────┬───────────────────────────────────────┘
                                               │ Command executes
                                               ▼
                              ┌──────────────────────────────┐
                              │    Package Manager Executes   │
                              └──────────────────────────────┘
```

### The Critical Asymmetry

- **Hooks can block what rules allow**: A PreToolUse hook returning `deny` blocks execution even if a permission allow rule matches.
- **Rules can block what hooks allow**: A permission deny rule blocks execution even if a hook returns `allow`.
- **Neither can override the other's deny**: Both are one-way valves. The most restrictive answer always wins.
- **Advisory layers cannot override enforcement**: CLAUDE.md and skills shape intent; hooks and rules control execution. A CLAUDE.md instruction saying "allow this package" has no effect if a deny rule or hook blocks it.

### What Fires When

| Event | PreToolUse Hooks | Permission Rules | OS Config |
|-------|-----------------|------------------|-----------|
| Agent runs `npm install axios` | YES (parses command, queries APIs) | YES (matches glob patterns) | YES (`.npmrc` settings apply) |
| Agent runs `bash -c "npm install evil"` | YES (sees `bash -c ...` string) | PARTIAL (deny rules see literal string, may not match `npm install *`) | YES |
| Agent edits `package.json` then runs `npm install` | Hook on `npm install` (no pkg to validate); hook on Edit for manifest | Rule on `Bash(npm install)` (bare) | YES |
| Subagent runs install | YES (hooks fire for subagents) | YES (permissions inherited) | YES |
| Agent uses MCP `install_package` tool | YES (if hook matcher covers MCP tool) | YES (MCP tool permission rules) | N/A (MCP server handles install) |

---

## 2. Layer 1: PreToolUse Hook Configuration

### 2.1 Settings.json Hook Configuration

Place in `.claude/settings.json` (project-level, shared via git):

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
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/post-install-audit.sh",
            "timeout": 60,
            "statusMessage": "Running post-install audit..."
          }
        ]
      }
    ]
  }
}
```

For efficient filtering with the `if` field (v2.1.85+), use multiple hook entries:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "if": "Bash(npm *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking npm package safety..."
          },
          {
            "type": "command",
            "if": "Bash(yarn *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking yarn package safety..."
          },
          {
            "type": "command",
            "if": "Bash(pnpm *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking pnpm package safety..."
          },
          {
            "type": "command",
            "if": "Bash(pip *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking pip package safety..."
          },
          {
            "type": "command",
            "if": "Bash(pip3 *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking pip package safety..."
          },
          {
            "type": "command",
            "if": "Bash(uv *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking uv package safety..."
          },
          {
            "type": "command",
            "if": "Bash(cargo *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking cargo package safety..."
          },
          {
            "type": "command",
            "if": "Bash(go *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking go package safety..."
          },
          {
            "type": "command",
            "if": "Bash(gem *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking gem package safety..."
          },
          {
            "type": "command",
            "if": "Bash(nix-env *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking nix package safety..."
          },
          {
            "type": "command",
            "if": "Bash(nix profile *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking nix package safety..."
          },
          {
            "type": "command",
            "if": "Bash(brew *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Checking brew package safety..."
          }
        ]
      }
    ]
  }
}
```

Note: The `if` field also fires when commands are too complex to parse into subcommands (e.g., heavy pipe chains), which is desirable for security. The catch-all version (without `if`) is simpler and equally correct -- it just spawns the script for every Bash command, which exits immediately (exit 0) for non-install commands.

### 2.2 Hook Script Specification

The hook script must perform this algorithm:

```
INPUT: JSON on stdin with tool_input.command
OUTPUT: exit 0 (allow), exit 2 (deny), or JSON on stdout with permissionDecision

1. Parse the command string from tool_input.command
2. Pattern-match against all package install command syntaxes
3. If no match → exit 0 (allow, not a package install)
4. Extract: package manager, package name(s), version constraint(s)
5. For each package:
   a. Map package manager to ecosystem (npm→npm, pip→PyPI, cargo→crates.io, go→Go)
   b. Query OSV.dev /v1/query for known vulnerabilities (~120ms)
   c. Check publication age via registry API (npm registry, PyPI JSON API) (~200ms)
   d. Optionally: query Socket.dev MCP for supply chain score (~500ms)
   e. Optionally: query deps.dev for typosquatting signals (~200ms)
6. Apply decision matrix:
   - Critical CVE (CVSS >= 9.0) → BLOCK
   - High CVE (CVSS >= 7.0) → ASK (escalate to user)
   - Published < 3 days ago → BLOCK (age gate)
   - Supply chain score < 0.3 → BLOCK
   - Typosquat detected → BLOCK
   - Medium/Low CVE → WARN (add context, allow)
   - All clear → ALLOW, possibly with updatedInput rewrites
7. Return decision as exit code or structured JSON
```

### 2.3 Complete Hook Script (Bash/jq)

```bash
#!/usr/bin/env bash
# .claude/hooks/package-guardrail.sh
# Dependencies: jq, curl
# Validates package installs against OSV.dev and registry publication age.
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
[ -z "$COMMAND" ] && exit 0

# --- Phase 1: Detect package install commands ---

detect_install() {
  local cmd="$1"

  # npm / npx
  if echo "$cmd" | grep -qE '\b(npm|npx)\s+(install|i|add|ci)\b'; then
    echo "npm"; return 0
  fi
  if echo "$cmd" | grep -qE '\bnpx\s+'; then
    echo "npx"; return 0
  fi
  # yarn
  if echo "$cmd" | grep -qE '\byarn\s+(add|install)\b'; then
    echo "yarn"; return 0
  fi
  # pnpm
  if echo "$cmd" | grep -qE '\bpnpm\s+(add|install|i)\b'; then
    echo "pnpm"; return 0
  fi
  # bun
  if echo "$cmd" | grep -qE '\bbun\s+(add|install|i)\b'; then
    echo "bun"; return 0
  fi
  # pip / pip3 / python -m pip
  if echo "$cmd" | grep -qE '\bpip3?\s+install\b'; then
    echo "pip"; return 0
  fi
  if echo "$cmd" | grep -qE '\bpython[23]?\s+-m\s+pip\s+install\b'; then
    echo "pip"; return 0
  fi
  # uv
  if echo "$cmd" | grep -qE '\buv\s+(pip\s+install|add)\b'; then
    echo "uv"; return 0
  fi
  # pipx
  if echo "$cmd" | grep -qE '\bpipx?\s+install\b'; then
    echo "pipx"; return 0
  fi
  # cargo
  if echo "$cmd" | grep -qE '\bcargo\s+(add|install)\b'; then
    echo "cargo"; return 0
  fi
  # go
  if echo "$cmd" | grep -qE '\bgo\s+(get|install)\b'; then
    echo "go"; return 0
  fi
  # gem
  if echo "$cmd" | grep -qE '\bgem\s+install\b'; then
    echo "gem"; return 0
  fi
  # bundle
  if echo "$cmd" | grep -qE '\bbundle\s+(install|add)\b'; then
    echo "bundle"; return 0
  fi
  # composer
  if echo "$cmd" | grep -qE '\bcomposer\s+require\b'; then
    echo "composer"; return 0
  fi
  # nix
  if echo "$cmd" | grep -qE '\bnix-env\s+-i\b'; then
    echo "nix"; return 0
  fi
  if echo "$cmd" | grep -qE '\bnix\s+profile\s+install\b'; then
    echo "nix"; return 0
  fi
  # apt
  if echo "$cmd" | grep -qE '\b(apt|apt-get)\s+install\b'; then
    echo "apt"; return 0
  fi
  # brew
  if echo "$cmd" | grep -qE '\bbrew\s+install\b'; then
    echo "brew"; return 0
  fi
  # pacman
  if echo "$cmd" | grep -qE '\bpacman\s+-S\b'; then
    echo "pacman"; return 0
  fi
  # pipe-to-shell
  if echo "$cmd" | grep -qE '(curl|wget).*\|.*(bash|sh|zsh)'; then
    echo "pipe-to-shell"; return 0
  fi

  return 1
}

MANAGER=$(detect_install "$COMMAND") || exit 0

# --- Phase 2: Block pipe-to-shell unconditionally ---

if [ "$MANAGER" = "pipe-to-shell" ]; then
  echo "BLOCKED: Pipe-to-shell execution is never allowed." >&2
  exit 2
fi

# --- Phase 3: Extract package names ---

extract_packages() {
  local cmd="$1"
  local mgr="$2"

  case "$mgr" in
    npm|yarn|pnpm|bun)
      # Extract tokens after the install verb, skip flags and flag arguments
      echo "$cmd" | grep -oP '(?:install|add|i)\s+\K.*' | tr ' ' '\n' | \
        grep -vE '^-' | grep -vE '^$' | head -20
      ;;
    pip|uv|pipx)
      echo "$cmd" | grep -oP 'install\s+\K.*' | tr ' ' '\n' | \
        grep -vE '^-' | grep -vE '^$' | grep -vE '^--' | head -20
      ;;
    cargo)
      echo "$cmd" | grep -oP '(add|install)\s+\K.*' | tr ' ' '\n' | \
        grep -vE '^-' | grep -vE '^$' | head -20
      ;;
    go)
      echo "$cmd" | grep -oP '(get|install)\s+\K.*' | tr ' ' '\n' | \
        grep -vE '^-' | grep -vE '^$' | head -20
      ;;
    gem|brew|apt|pacman)
      echo "$cmd" | grep -oP 'install\s+\K.*' | tr ' ' '\n' | \
        grep -vE '^-' | grep -vE '^$' | head -20
      ;;
    nix)
      echo "$cmd" | grep -oP '(?:install|-i)\s+\K.*' | tr ' ' '\n' | \
        grep -vE '^-' | grep -vE '^$' | head -20
      ;;
  esac
}

PACKAGES=$(extract_packages "$COMMAND" "$MANAGER")

# If no specific packages extracted (bare install from lockfile), allow with rewrite
if [ -z "$PACKAGES" ]; then
  # Bare npm install → rewrite to npm ci (lockfile-safe)
  if [ "$MANAGER" = "npm" ] && echo "$COMMAND" | grep -qE '^\s*npm\s+(install|i)\s*$'; then
    cat <<'JSONEOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "updatedInput": {
      "command": "npm ci"
    },
    "additionalContext": "Rewrote bare 'npm install' to 'npm ci' for lockfile integrity."
  }
}
JSONEOF
    exit 0
  fi
  exit 0
fi

# --- Phase 4: Map manager to OSV ecosystem ---

map_ecosystem() {
  case "$1" in
    npm|yarn|pnpm|bun|npx) echo "npm" ;;
    pip|uv|pipx)           echo "PyPI" ;;
    cargo)                 echo "crates.io" ;;
    go)                    echo "Go" ;;
    gem|bundle)            echo "RubyGems" ;;
    composer)              echo "Packagist" ;;
    *)                     echo "" ;;
  esac
}

ECOSYSTEM=$(map_ecosystem "$MANAGER")

# --- Phase 5: Validate each package ---

BLOCKED_PACKAGES=""
WARNED_PACKAGES=""

for PKG in $PACKAGES; do
  # Strip version specifier for API lookup
  PKG_NAME=$(echo "$PKG" | sed 's/@[^/]*$//' | sed 's/[>=<~^].*//')
  [ -z "$PKG_NAME" ] && continue

  # Skip if ecosystem is unknown (nix, apt, brew, pacman — no OSV ecosystem)
  if [ -z "$ECOSYSTEM" ]; then
    continue
  fi

  # 5a. Query OSV.dev for known vulnerabilities
  OSV_RESPONSE=$(curl -sf --max-time 5 \
    -X POST "https://api.osv.dev/v1/query" \
    -H "Content-Type: application/json" \
    -d "{\"package\":{\"name\":\"$PKG_NAME\",\"ecosystem\":\"$ECOSYSTEM\"}}" \
    2>/dev/null || echo '{}')

  VULN_COUNT=$(echo "$OSV_RESPONSE" | jq '.vulns | length // 0' 2>/dev/null || echo "0")

  if [ "$VULN_COUNT" -gt 0 ]; then
    # Check for critical/high severity
    CRITICAL_HIGH=$(echo "$OSV_RESPONSE" | jq '
      [.vulns[]? |
        (.severity[]? |
          select(.type == "CVSS_V3" or .type == "CVSS_V4") |
          .score | split("/")[0] | tonumber |
          select(. >= 7.0)
        )
      ] | length
    ' 2>/dev/null || echo "0")

    VULN_IDS=$(echo "$OSV_RESPONSE" | jq -r '[.vulns[].id] | .[0:3] | join(", ")' 2>/dev/null || echo "unknown")

    if [ "$CRITICAL_HIGH" -gt 0 ]; then
      BLOCKED_PACKAGES="${BLOCKED_PACKAGES}${PKG_NAME} (${VULN_COUNT} vulns: ${VULN_IDS})\n"
    else
      WARNED_PACKAGES="${WARNED_PACKAGES}${PKG_NAME} (${VULN_COUNT} low/medium vulns: ${VULN_IDS})\n"
    fi
  fi

  # 5b. Check publication age via npm registry (for npm ecosystem)
  if [ "$ECOSYSTEM" = "npm" ]; then
    PUB_TIME=$(curl -sf --max-time 3 \
      "https://registry.npmjs.org/${PKG_NAME}" \
      2>/dev/null | jq -r '.time // {} | to_entries | sort_by(.value) | last | .value // empty' 2>/dev/null || echo "")

    if [ -n "$PUB_TIME" ]; then
      PUB_EPOCH=$(date -d "$PUB_TIME" +%s 2>/dev/null || echo "0")
      NOW_EPOCH=$(date +%s)
      AGE_DAYS=$(( (NOW_EPOCH - PUB_EPOCH) / 86400 ))

      if [ "$AGE_DAYS" -lt 3 ]; then
        BLOCKED_PACKAGES="${BLOCKED_PACKAGES}${PKG_NAME} (published ${AGE_DAYS} days ago, minimum 3 days required)\n"
      fi
    fi
  fi
done

# --- Phase 6: Return decision ---

if [ -n "$BLOCKED_PACKAGES" ]; then
  {
    echo "BLOCKED: Package installation denied by security policy."
    echo ""
    echo "Blocked packages:"
    echo -e "$BLOCKED_PACKAGES"
    echo ""
    if [ -n "$WARNED_PACKAGES" ]; then
      echo "Additionally warned:"
      echo -e "$WARNED_PACKAGES"
    fi
    echo "Use the /safe-install skill or mcp__package_security__install_package tool for approved installations."
  } >&2
  exit 2
fi

if [ -n "$WARNED_PACKAGES" ]; then
  cat <<JSONEOF
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "ask",
    "permissionDecisionReason": "Packages have low/medium vulnerabilities:\n$(echo -e "$WARNED_PACKAGES" | tr '\n' ' ')\nApprove manually?"
  }
}
JSONEOF
  exit 0
fi

# --- Phase 7: Rewrite commands with safety flags ---

REWRITTEN_COMMAND="$COMMAND"
case "$MANAGER" in
  npm|yarn|pnpm|bun)
    if ! echo "$COMMAND" | grep -q '\-\-ignore-scripts'; then
      REWRITTEN_COMMAND="$COMMAND --ignore-scripts"
    fi
    ;;
  pip|uv)
    if ! echo "$COMMAND" | grep -q '\-\-only-binary'; then
      REWRITTEN_COMMAND="$COMMAND --only-binary :all:"
    fi
    ;;
  cargo)
    if ! echo "$COMMAND" | grep -q '\-\-locked'; then
      REWRITTEN_COMMAND="$COMMAND --locked"
    fi
    ;;
esac

if [ "$REWRITTEN_COMMAND" != "$COMMAND" ]; then
  cat <<JSONEOF
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "updatedInput": {
      "command": "$REWRITTEN_COMMAND"
    },
    "additionalContext": "Safety flags appended: $(diff <(echo "$COMMAND") <(echo "$REWRITTEN_COMMAND") | tail -1 || echo 'modified')"
  }
}
JSONEOF
  exit 0
fi

exit 0
```

### 2.4 Exit Code Semantics

| Exit Code | Meaning | Effect |
|-----------|---------|--------|
| 0 | Allow (or JSON with decision on stdout) | Command proceeds (unless permission rules deny) |
| 2 | Deny (stderr message fed back to agent) | Command blocked. Agent sees stderr as error feedback. |
| Non-0, non-2 | Hook error | Depends on design: fail-open (allow) or fail-closed (deny). Default is fail-open. |

### 2.5 Structured JSON Decision Format

For richer control than exit codes:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Package 'evil-pkg' has CVE-2026-1234 (critical, CVSS 9.8). Use 'safe-alternative@2.1.0' instead."
  }
}
```

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "ask",
    "permissionDecisionReason": "Package 'new-pkg' was published 2 days ago. Manual approval required."
  }
}
```

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "updatedInput": {
      "command": "npm install axios@1.6.7 --ignore-scripts --save-exact"
    },
    "additionalContext": "Rewrote: pinned to latest safe version, added --ignore-scripts and --save-exact."
  }
}
```

### 2.6 Decision Precedence Across Multiple Hooks

When multiple PreToolUse hooks match the same event, they run in parallel and the most restrictive answer wins:

- `deny` overrides `ask`
- `ask` overrides `allow`
- `allow` overrides no-decision

**Warning**: When multiple hooks return `updatedInput`, the last to finish wins, and order is non-deterministic. Only ONE hook should rewrite the command.

### 2.7 Managed Settings (Enterprise Mandatory Enforcement)

For `/etc/claude-code/managed-settings.json` on Linux/NixOS:

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
            "command": "/opt/company/claude-hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Validating package safety (org policy)..."
          }
        ]
      }
    ]
  }
}
```

When `allowManagedHooksOnly` is `true`:
- Only managed hooks, SDK hooks, and force-enabled plugin hooks load
- User hooks: blocked
- Project hooks: blocked
- Plugin hooks (except force-enabled): blocked
- Users cannot set `disableAllHooks` to override

Drop-in directory for modular policy:

```
/etc/claude-code/
├── managed-settings.json
└── managed-settings.d/
    ├── 10-telemetry.json
    ├── 20-package-guardrails.json
    └── 30-file-protection.json
```

---

## 3. Layer 2: Permission Deny Rules

### 3.1 Comprehensive Deny Rule Set

```json
{
  "permissions": {
    "deny": [
      "Bash(npm install *)",
      "Bash(npm install)",
      "Bash(npm i *)",
      "Bash(npm i)",
      "Bash(npm add *)",
      "Bash(npm ci *)",
      "Bash(npm ci)",
      "Bash(npx *)",
      "Bash(yarn add *)",
      "Bash(yarn install *)",
      "Bash(yarn install)",
      "Bash(pnpm add *)",
      "Bash(pnpm install *)",
      "Bash(pnpm install)",
      "Bash(pnpm i *)",
      "Bash(pnpm i)",
      "Bash(bun add *)",
      "Bash(bun install *)",
      "Bash(bun install)",
      "Bash(pip install *)",
      "Bash(pip3 install *)",
      "Bash(python -m pip install *)",
      "Bash(python3 -m pip install *)",
      "Bash(pipx install *)",
      "Bash(uv pip install *)",
      "Bash(uv add *)",
      "Bash(poetry add *)",
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(go get *)",
      "Bash(go install *)",
      "Bash(gem install *)",
      "Bash(bundle install *)",
      "Bash(bundle add *)",
      "Bash(composer require *)",
      "Bash(dotnet add package *)",
      "Bash(nix-env -i *)",
      "Bash(nix-env *)",
      "Bash(nix profile install *)",
      "Bash(cachix use *)",
      "Bash(apt install *)",
      "Bash(apt-get install *)",
      "Bash(brew install *)",
      "Bash(pacman -S *)",
      "Bash(curl * | bash *)",
      "Bash(curl * | sh *)",
      "Bash(wget * | bash *)",
      "Bash(wget * | sh *)"
    ]
  }
}
```

### 3.2 Allow Rules for Approved Pathways

```json
{
  "permissions": {
    "allow": [
      "Bash(${CLAUDE_PROJECT_DIR}/.claude/hooks/safe-install.sh *)",
      "Bash(${CLAUDE_PROJECT_DIR}/scripts/safe-install *)",
      "Bash(./scripts/safe-install *)",
      "Bash(./.claude/hooks/safe-install.sh *)",
      "Bash(npm run *)",
      "Bash(npm test *)",
      "Bash(npm run build *)",
      "Bash(npm audit *)",
      "Bash(pip-audit *)",
      "Bash(cargo audit *)",
      "Bash(nix build *)",
      "Bash(nix develop *)",
      "Bash(nix flake check *)",
      "mcp__package_security__install_package",
      "mcp__package_security__check_package",
      "mcp__socket__depscore"
    ]
  }
}
```

Allow rules work because they match a **different command** than the deny rules. `./scripts/safe-install npm axios` matches `Bash(./scripts/safe-install *)` but does NOT match `Bash(npm install *)`. The wrapper script internally calls the package manager; Claude Code's permission rules apply to the command Claude submits, not to subprocesses.

### 3.3 Known Bypass Patterns and Mitigations

| Bypass Pattern | Example | Deny Rule Catches It? | Hook Catches It? | Mitigation |
|---|---|---|---|---|
| Shell wrapper | `bash -c "npm install evil"` | NO (sees `bash -c ...`) | YES (regex on full string) | Hook parses for nested install commands |
| env prefix | `env npm install evil` | NO (`env` is not stripped) | YES (regex on full string) | Hook strips known wrappers before matching |
| command builtin | `command npm install evil` | NO | YES | Hook handles `command` prefix |
| Python subprocess | `python -c "import subprocess; ..."` | NO | PARTIAL (can regex for subprocess patterns) | OS sandbox restricts network |
| Node subprocess | `node -e "require('child_process')..."` | NO | PARTIAL | OS sandbox restricts network |
| Variable expansion | `CMD=npm; $CMD install evil` | NO (sees literal `$CMD`) | NO (sees literal `$CMD`) | OS-level config (`.npmrc`) still applies |
| Edit manifest + bare install | Edit `package.json`, run `npm install` | YES (catches bare `npm install`) | YES (hook rewrites to `npm ci`) | PostToolUse lockfile diff |
| Package aliasing | `npm install safe@npm:evil-pkg` | Catches `npm install *` pattern | YES (can parse alias syntax) | Socket.dev resolves actual identity |

### 3.4 Configuration at Each Settings Level

| Level | File | Use Case |
|---|---|---|
| **User** | `~/.claude/settings.json` | Personal safety net across all projects |
| **Project (shared)** | `.claude/settings.json` | Team-wide enforcement, committed to git |
| **Project (local)** | `.claude/settings.local.json` | Personal overrides (gitignored) |
| **Managed** | `/etc/claude-code/managed-settings.json` | Enterprise mandatory, cannot be overridden |

**Precedence for deny rules**: A deny at ANY level cannot be overridden by an allow at any other level. Managed deny > CLI args > Local > Project > User.

To make deny rules truly mandatory, use managed settings with:

```json
{
  "allowManagedPermissionRulesOnly": true,
  "permissions": {
    "deny": ["...all deny rules..."]
  }
}
```

This prevents users and projects from defining ANY `allow`, `ask`, or `deny` rules. Only managed rules apply.

---

## 4. Layer 3: OS/Environment Configuration

These settings persist at the package manager level and survive even if hooks and permissions are bypassed.

### 4.1 npm (.npmrc)

Create at project root (`.npmrc`) or user level (`~/.npmrc`):

```ini
# Block install-time script execution
ignore-scripts=true

# Require minimum package age (days) before install
min-release-age=3

# Pin exact versions (no range specifiers)
save-exact=true

# Set audit severity threshold
audit-level=high

# Enforce engine compatibility
engine-strict=true

# Use lockfile-only mode in CI
# (uncomment for CI: prefer-offline=true)
```

### 4.2 pnpm (pnpm-workspace.yaml / .npmrc)

```yaml
# pnpm-workspace.yaml
onlyBuiltDependencies:
  # Allowlist packages that need lifecycle scripts
  # - esbuild
  # - sharp

# .npmrc for pnpm
strict-dep-builds=true
```

Or via `.npmrc` for pnpm:

```ini
ignore-scripts=true
# pnpm auto-blocks lifecycle scripts in v10+ with strictDepBuilds
```

### 4.3 Yarn (.yarnrc.yml)

```yaml
# .yarnrc.yml
enableImmutableInstalls: true
npmMinimalAgeGate: 3d
```

### 4.4 pip / Python

Via `pip.conf` (project or user level):

```ini
[global]
# Refuse source distributions (eliminates setup.py execution)
only-binary = :all:

[install]
# Require hash verification
require-hashes = true
```

Via environment variables (set in `.envrc`, shell profile, or CI):

```bash
export PIP_ONLY_BINARY=":all:"
export PIP_REQUIRE_HASHES=1
```

For uv:

```bash
# Exclude packages newer than N days
export UV_EXCLUDE_NEWER="2026-05-09"  # 3 days before today
```

### 4.5 Cargo

Via `.cargo/config.toml` in project root:

```toml
[build]
# Require lockfile for all builds
locked = true

[net]
# Require cargo-audit results
# (no native flag -- enforce via CI and hooks)
```

### 4.6 Go

```bash
# Ensure sum database verification is not disabled
# (Go uses sumdb by default -- ensure it's not overridden)
unset GONOSUMDB
unset GONOSUMCHECK

# Use Go module proxy (default, provides checksum verification)
export GOPROXY="https://proxy.golang.org,direct"
export GONOSUMDB=""
export GONOSUMCHECK=""
```

### 4.7 Nix (nix.conf)

For NixOS, in `/etc/nix/nix.conf` or equivalent NixOS module:

```nix
# /etc/nix/nix.conf
sandbox = true
sandbox-fallback = false
require-sigs = true

# Restrict binary cache substituters
trusted-substituters = https://cache.nixos.org https://nix-community.cachix.org
trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs=

# Do NOT add users to trusted-users (it is root-equivalent)
# trusted-users = root  # Only root, never regular users

# Restrict allowed users for building
allowed-users = *
```

For devenv projects, in `devenv.yaml`:

```yaml
clean:
  enabled: true  # Reset shell hooks on each activation
```

### 4.8 How These Survive Bypass

| Scenario | OS Config Effect |
|---|---|
| Hook is disabled/bypassed | `.npmrc ignore-scripts=true` still blocks postinstall scripts |
| Deny rule is evaded via shell wrapper | `PIP_ONLY_BINARY=:all:` still refuses source distributions |
| Agent edits `.npmrc` to remove settings | Add `.npmrc` to deny rules: `Edit(.npmrc)` in deny list, or use CODEOWNERS |
| Agent uses subprocess to install | Package manager reads its own config regardless of how it was invoked |
| Nix evaluation runs unsandboxed code | `nix.conf sandbox-fallback=false` ensures builds are always sandboxed |

---

## 5. Advisory Layer: CLAUDE.md Instructions

### 5.1 Recommended CLAUDE.md Package Security Section

Add this to your project's `CLAUDE.md` or `.claude/rules/package-security.md`:

```markdown
## Package Installation Policy

### Mandatory Rules

1. **NEVER run raw package install commands.** Do not use `npm install`, `pip install`,
   `cargo add`, `go get`, `yarn add`, `pnpm add`, `bun add`, `gem install`, `nix-env -i`,
   or any other package manager install command directly.

2. **ALWAYS use the /safe-install skill** for package installation:
   `/safe-install npm axios` or `/safe-install pip requests`

3. **If /safe-install is unavailable**, use the `mcp__package_security__install_package`
   MCP tool, or ask the user to install manually.

4. **If a hook blocks your install**, do NOT attempt to work around it. Report the
   hook's error message to the user and ask for guidance. Blocked packages have known
   vulnerabilities or supply chain risks.

### Security Checks Before Adding Dependencies

Before proposing any new dependency:
- Verify the package exists and is actively maintained (>1 year history, recent commits)
- Check for known alternatives that are more established
- Prefer packages with Sigstore provenance attestations when available
- Never install packages published less than 3 days ago

### What the Hooks Do

This project uses PreToolUse hooks that automatically:
- Check all package installs against the OSV.dev vulnerability database
- Block packages with critical or high-severity CVEs
- Block packages published less than 3 days ago
- Append --ignore-scripts (npm) and --only-binary (pip) for safety
- Rewrite bare `npm install` to `npm ci` for lockfile integrity

### Nix-Specific Rules

- Never use `nix-env` -- it bypasses flake pinning
- Never run `nix flake update` without user approval
- Never add entries to `cachix.pull`, `extra-substituters`, or `trusted-public-keys`
- Never modify `trusted-users` in any Nix configuration
- When adding packages to devenv.nix, use only packages from the pinned nixpkgs input

### Lockfile Discipline

- Never modify lockfiles directly (package-lock.json, yarn.lock, Cargo.lock, etc.)
- In projects with existing lockfiles, prefer `npm ci` over `npm install`
- If a lockfile needs updating, explain why and get user approval first
```

### 5.2 How Instructions Complement Enforcement

CLAUDE.md reduces the frequency with which enforcement layers fire:

| Without CLAUDE.md | With CLAUDE.md |
|---|---|
| Agent tries `npm install axios` -> hook blocks -> agent retries -> hook blocks again | Agent uses `/safe-install npm axios` -> skill runs check -> installs safely |
| Agent tries 3 different shell wrappers to bypass -> all blocked | Agent reads "do NOT work around hooks" -> asks user for help |
| Agent installs via `package.json` edit + `npm install` -> hook catches bare install | Agent reads lockfile discipline -> uses approved path |

### 5.3 The Subagent Gap

**Critical limitation**: Built-in subagents (since v2.1.84) and custom subagents do NOT inherit CLAUDE.md instructions. Security instructions in CLAUDE.md are invisible to them.

**Mitigation**:
- Hooks and permission deny rules still apply to subagent tool calls (enforced by client, not model)
- For custom subagents (`.claude/agents/`), duplicate security instructions in the agent's system prompt
- For built-in subagents, rely entirely on hooks and permissions -- there is no way to inject CLAUDE.md content

| Subagent Type | CLAUDE.md Inherited? | Hooks Apply? | Permissions Apply? |
|---|---|---|---|
| Built-in (Explore, Plan) | NO (since v2.1.84) | YES | YES |
| Custom (`.claude/agents/`) | NO (own system prompt) | YES | YES |
| Forked | YES | YES | YES |

---

## 6. Advisory Layer: Custom Skills

### 6.1 /safe-install Skill Specification

Create `.claude/skills/safe-install/SKILL.md`:

```yaml
---
name: safe-install
description: >
  Securely install a package with pre-flight vulnerability and supply chain
  checks. Use whenever installing npm, pip, cargo, go, yarn, pnpm, bun,
  gem, or composer packages. Use when the user asks to "add a dependency",
  "install a library", or "add a package".
disable-model-invocation: false
allowed-tools: Bash(${CLAUDE_SKILL_DIR}/scripts/*) Read
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "${CLAUDE_SKILL_DIR}/scripts/check-package.sh"
  PostToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "${CLAUDE_SKILL_DIR}/scripts/post-install-audit.sh"
arguments: [pm, package]
---

# Secure Package Installation

Install the package `$package` using `$pm` with security validation.

## Current Project State
!`cat package.json 2>/dev/null | jq '{dependencies, devDependencies}' || echo "No package.json"`

## Pre-flight Checks

1. Run the security check script:
   ```bash
   ${CLAUDE_SKILL_DIR}/scripts/check-package.sh $pm $package
   ```

2. Interpret the results:
   - **PASS**: Proceed to installation
   - **WARN**: Report warnings to the user and ask for confirmation
   - **FAIL**: Do NOT install. Report the security findings and suggest alternatives.

## Installation

3. If checks pass, install with safety flags:
   - npm/yarn/pnpm/bun: `$pm install --save-exact --ignore-scripts $package`
   - pip: `pip install --only-binary :all: $package`
   - cargo: `cargo add $package`
   - go: `go get $package`

## Post-Install Audit

4. Run the ecosystem audit tool:
   - npm: `npm audit --json --audit-level=high`
   - pip: `pip-audit --format=json`
   - cargo: `cargo audit --json`

5. Report any new vulnerabilities introduced.

## Important

- NEVER install a package that fails the pre-flight check
- ALWAYS pin exact versions
- If the check script is unavailable, refuse to install and explain why
- If the user wants to override a security block, they must explicitly approve
```

### 6.2 How Skills + Deny Rules Create a Forced Path

```
Agent wants to install a package
       │
       ├──→ Tries raw `npm install axios`
       │    └── BLOCKED by deny rule Bash(npm install *)
       │        Agent receives: "Permission denied"
       │
       ├──→ Tries `bash -c "npm install axios"`
       │    └── BLOCKED by PreToolUse hook (parses nested command)
       │        Agent receives: "Use /safe-install instead"
       │
       ├──→ Uses /safe-install npm axios  ← THE APPROVED PATH
       │    ├── Skill loads, runs pre-flight check
       │    ├── Queries OSV.dev for CVEs
       │    ├── Checks publication age
       │    ├── If clean: installs with --ignore-scripts --save-exact
       │    └── Runs post-install audit
       │
       └──→ Uses mcp__package_security__install_package  ← ALSO APPROVED
            ├── MCP server queries OSV.dev + Socket.dev
            ├── If clean: executes install with safety flags
            └── Returns structured result
```

### 6.3 Lifecycle-Scoped Hooks in Skill Frontmatter

The `hooks` field in SKILL.md frontmatter defines hooks that activate ONLY while the skill is running:

```yaml
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "${CLAUDE_SKILL_DIR}/scripts/check-package.sh"
```

These hooks are:
- Scoped to the skill's lifetime (deactivate when skill finishes)
- Additive with global hooks (both fire; most restrictive wins)
- Automatically cleaned up (no manual removal needed)
- For subagents spawned by the skill, `Stop` hooks convert to `SubagentStop`

**Limitation**: If the skill is never invoked (or compacted away), these hooks are inactive. Global hooks in settings.json remain the always-on safety net.

---

## 7. MCP Server Integration

### 7.1 Socket.dev MCP Server Setup

**Zero-auth public server (recommended for quick start):**

```bash
claude mcp add --transport http socket-mcp https://mcp.socket.dev/
```

**Local with API key (higher rate limits):**

```bash
claude mcp add socket-mcp -e SOCKET_API_KEY="sk-your-key" -- npx -y @socketsecurity/mcp@latest
```

**In `.mcp.json` (project-level, shared via git):**

```json
{
  "mcpServers": {
    "socket-mcp": {
      "type": "http",
      "url": "https://mcp.socket.dev/"
    }
  }
}
```

**Tool exposed**: `depscore` -- returns five scoring dimensions (0-1 scale): supply chain risk, code quality, maintenance, vulnerability, license compatibility.

**Usage by agent**:
```
mcp__socket-mcp__depscore({
  packages: [
    { ecosystem: "npm", depname: "axios", version: "1.7.0" }
  ]
})
```

### 7.2 Snyk MCP Setup

**Via Snyk CLI (bundled in v1.1298.0+):**

```bash
claude mcp add SnykMCP -e SNYK_TOKEN="${SNYK_TOKEN}" -- snyk mcp -t stdio
```

**Via npx:**

```bash
claude mcp add SnykMCP -e SNYK_TOKEN="${SNYK_TOKEN}" -- npx -y snyk@latest mcp -t stdio
```

**In `.mcp.json`:**

```json
{
  "mcpServers": {
    "SnykMCP": {
      "command": "snyk",
      "args": ["mcp", "-t", "stdio"],
      "env": { "SNYK_TOKEN": "${SNYK_TOKEN}" }
    }
  }
}
```

**Key tools**: `snyk_sca_scan` (dependency vulnerabilities), `snyk_code_scan` (first-party code), `snyk_container_scan` (container images).

**Note**: Snyk scans the project after the fact, not individual packages pre-install. Best used as a post-install validation complement, not a pre-install gate.

### 7.3 Custom MCP Server Architecture

A custom MCP server combining OSV.dev + Socket.dev + registry age checks:

**File structure:**

```
.claude/mcp-servers/package-security/
├── package.json
├── tsconfig.json
└── src/
    ├── index.ts           # Server entry point
    ├── tools/
    │   ├── check-package.ts    # Pre-flight validation
    │   └── install-package.ts  # Validated install
    ├── validators/
    │   ├── osv.ts         # OSV.dev API client
    │   ├── socket.ts      # Socket.dev API client
    │   ├── registry.ts    # npm/PyPI registry age check
    │   └── policy.ts      # Decision engine
    └── audit/
        └── logger.ts      # Structured audit logging
```

**Core server implementation:**

```typescript
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

const server = new McpServer({
  name: "package-security",
  version: "1.0.0"
});

server.tool(
  "check_package",
  "Check a package for vulnerabilities, supply chain risk, and age before installation",
  {
    name: z.string().describe("Package name (e.g., 'axios', 'requests')"),
    version: z.string().optional().describe("Specific version to check"),
    ecosystem: z.enum(["npm", "PyPI", "crates.io", "Go", "RubyGems", "Packagist"])
      .describe("Package ecosystem")
  },
  async ({ name, version, ecosystem }) => {
    // 1. Query OSV.dev for CVEs
    const osvResult = await queryOsv(name, version, ecosystem);
    // 2. Query Socket.dev for supply chain score
    const socketResult = await querySocket(name, version, ecosystem);
    // 3. Check publication age
    const ageResult = await checkAge(name, ecosystem);
    // 4. Apply policy
    const decision = applyPolicy(osvResult, socketResult, ageResult);

    return {
      content: [{
        type: "text",
        text: JSON.stringify(decision, null, 2)
      }]
    };
  }
);

server.tool(
  "install_package",
  "Install a package after security validation. Use this instead of raw npm/pip/cargo commands.",
  {
    name: z.string(),
    version: z.string().optional(),
    ecosystem: z.enum(["npm", "PyPI", "crates.io", "Go"]),
    devDependency: z.boolean().optional().default(false)
  },
  async ({ name, version, ecosystem, devDependency }) => {
    // 1. Run all checks
    const checks = await validatePackage(name, version, ecosystem);

    if (!checks.allowed) {
      return {
        content: [{
          type: "text",
          text: `BLOCKED: ${checks.reasons.join("; ")}`
        }],
        isError: true
      };
    }

    // 2. Build safe install command
    const cmd = buildSafeCommand(name, version, ecosystem, devDependency);

    // 3. Execute
    const result = await executeInstall(cmd);

    // 4. Post-install audit
    const audit = await runAudit(ecosystem);

    // 5. Log
    await logAttempt({ name, version, ecosystem, decision: "allowed", audit });

    return {
      content: [{
        type: "text",
        text: JSON.stringify({ installed: true, package: `${name}@${version}`, audit })
      }]
    };
  }
);

const transport = new StdioServerTransport();
await server.connect(transport);
```

**API endpoints used:**

| Service | Endpoint | Auth | Rate Limit | Latency |
|---------|----------|------|------------|---------|
| OSV.dev | `POST https://api.osv.dev/v1/query` | None | None | ~120ms |
| OSV.dev | `POST https://api.osv.dev/v1/querybatch` | None | None | ~600ms |
| Socket.dev | `https://mcp.socket.dev/` (MCP) | None (public) | Unknown | ~200-800ms |
| Socket.dev | `GET https://api.socket.dev/v0/npm/{pkg}/{ver}/score` | Bearer token | Quota-based | ~200-800ms |
| npm registry | `GET https://registry.npmjs.org/{pkg}` | None | Standard | ~200ms |
| PyPI | `GET https://pypi.org/pypi/{pkg}/json` | None | Standard | ~200ms |
| deps.dev | `GET https://api.deps.dev/v3alpha/systems/{sys}/packages/{name}:similarlyNamedPackages` | None | Undocumented | ~200ms |

### 7.4 MCP Tool Hook Configuration

Delegate validation from a PreToolUse hook to the MCP server:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "mcp_tool",
            "server": "package_security",
            "tool": "validate_bash_command",
            "input": { "command": "${tool_input.command}" }
          }
        ]
      }
    ]
  }
}
```

**Critical caveat**: MCP tool hooks are **non-blocking on failure**. If the MCP server is down or returns an error, the command is allowed through. For security-critical enforcement, use a `command` type hook that fails-closed, with the MCP tool as a supplementary signal.

---

## 8. Deployment Profiles

### 8.1 Individual Developer (Minimal Setup)

**Effort**: 15-30 minutes. **Files**: 3 files, user-level settings.

**Step 1: User settings** (`~/.claude/settings.json`):

```json
{
  "permissions": {
    "deny": [
      "Bash(npm install *)", "Bash(npm install)",
      "Bash(npm i *)", "Bash(npm i)",
      "Bash(npm add *)", "Bash(npm ci *)", "Bash(npm ci)",
      "Bash(npx *)",
      "Bash(yarn add *)", "Bash(yarn install *)", "Bash(yarn install)",
      "Bash(pnpm add *)", "Bash(pnpm install *)", "Bash(pnpm install)",
      "Bash(bun add *)", "Bash(bun install *)", "Bash(bun install)",
      "Bash(pip install *)", "Bash(pip3 install *)",
      "Bash(python -m pip install *)", "Bash(python3 -m pip install *)",
      "Bash(uv pip install *)", "Bash(uv add *)",
      "Bash(cargo add *)", "Bash(cargo install *)",
      "Bash(go get *)", "Bash(go install *)",
      "Bash(gem install *)", "Bash(brew install *)",
      "Bash(nix-env *)", "Bash(nix profile install *)", "Bash(cachix use *)",
      "Bash(curl * | bash *)", "Bash(curl * | sh *)",
      "Bash(wget * | bash *)", "Bash(wget * | sh *)"
    ],
    "allow": [
      "Bash(npm run *)", "Bash(npm test *)", "Bash(npm audit *)",
      "Bash(pip-audit *)", "Bash(cargo audit *)",
      "Bash(nix build *)", "Bash(nix develop *)", "Bash(nix flake check *)"
    ]
  }
}
```

**Step 2: Connect Socket.dev MCP** (one command):

```bash
claude mcp add --transport http socket-mcp https://mcp.socket.dev/
```

**Step 3: Add CLAUDE.md package policy** (append to project CLAUDE.md):

```markdown
## Package Security

Never run package install commands directly. Always ask the user before
adding any dependency. Check the Socket.dev MCP tool (depscore) for supply
chain scores before recommending any package.
```

### 8.2 Team/Project (Shared Configuration)

**Effort**: 1-2 hours. **Files**: 5+ files, committed to git.

**Step 1: Project settings** (`.claude/settings.json`):

```json
{
  "permissions": {
    "defaultMode": "default",
    "disableBypassPermissionsMode": "disable",
    "deny": [
      "Bash(npm install *)", "Bash(npm install)",
      "Bash(npm i *)", "Bash(npm i)",
      "Bash(npm add *)", "Bash(npm ci *)", "Bash(npm ci)",
      "Bash(npx *)",
      "Bash(yarn add *)", "Bash(yarn install *)", "Bash(yarn install)",
      "Bash(pnpm add *)", "Bash(pnpm install *)", "Bash(pnpm install)",
      "Bash(bun add *)", "Bash(bun install *)", "Bash(bun install)",
      "Bash(pip install *)", "Bash(pip3 install *)",
      "Bash(python -m pip install *)", "Bash(python3 -m pip install *)",
      "Bash(uv pip install *)", "Bash(uv add *)",
      "Bash(cargo add *)", "Bash(cargo install *)",
      "Bash(go get *)", "Bash(go install *)",
      "Bash(gem install *)", "Bash(bundle add *)", "Bash(bundle install *)",
      "Bash(composer require *)", "Bash(dotnet add package *)",
      "Bash(nix-env *)", "Bash(nix profile install *)", "Bash(cachix use *)",
      "Bash(apt install *)", "Bash(apt-get install *)",
      "Bash(brew install *)", "Bash(pacman -S *)",
      "Bash(curl * | bash *)", "Bash(curl * | sh *)",
      "Bash(wget * | bash *)", "Bash(wget * | sh *)"
    ],
    "allow": [
      "Bash(./scripts/safe-install *)",
      "Bash(./.claude/hooks/safe-install.sh *)",
      "Bash(npm run *)", "Bash(npm test *)", "Bash(npm audit *)",
      "Bash(pip-audit *)", "Bash(cargo audit *)",
      "Bash(nix build *)", "Bash(nix develop *)",
      "mcp__package_security__install_package",
      "mcp__package_security__check_package",
      "mcp__socket-mcp__depscore"
    ]
  },
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
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/post-install-audit.sh",
            "timeout": 60,
            "statusMessage": "Running post-install audit..."
          }
        ]
      }
    ]
  }
}
```

**Step 2: Project MCP configuration** (`.mcp.json`):

```json
{
  "mcpServers": {
    "socket-mcp": {
      "type": "http",
      "url": "https://mcp.socket.dev/"
    }
  }
}
```

**Step 3: Hook scripts** (`.claude/hooks/package-guardrail.sh`) -- use the complete script from Section 2.3.

**Step 4: OS config files** (`.npmrc`, `pip.conf`, etc.) -- use settings from Section 4.

**Step 5: CLAUDE.md + /safe-install skill** -- use content from Sections 5.1 and 6.1.

**Step 6: Add to `.gitignore`**:

```
.claude/settings.local.json
```

### 8.3 Enterprise (Managed, Non-Overridable)

**Effort**: 4-8 hours initial, plus ongoing maintenance. **Files**: Managed settings deployed via config management (NixOS module, Ansible, MDM).

**Step 1: Managed settings** (`/etc/claude-code/managed-settings.json`):

```json
{
  "allowManagedHooksOnly": true,
  "allowManagedPermissionRulesOnly": true,
  "allowManagedMcpServersOnly": true,
  "disableBypassPermissionsMode": "disable",
  "disableAutoMode": "disable",
  "permissions": {
    "deny": [
      "Bash(npm install *)", "Bash(npm install)",
      "Bash(npm i *)", "Bash(npm i)",
      "Bash(npm add *)", "Bash(npm ci *)", "Bash(npm ci)",
      "Bash(npx *)",
      "Bash(yarn add *)", "Bash(yarn install *)", "Bash(yarn install)",
      "Bash(pnpm add *)", "Bash(pnpm install *)", "Bash(pnpm install)",
      "Bash(bun add *)", "Bash(bun install *)", "Bash(bun install)",
      "Bash(pip install *)", "Bash(pip3 install *)",
      "Bash(python -m pip install *)", "Bash(python3 -m pip install *)",
      "Bash(uv pip install *)", "Bash(uv add *)",
      "Bash(cargo add *)", "Bash(cargo install *)",
      "Bash(go get *)", "Bash(go install *)",
      "Bash(gem install *)", "Bash(bundle add *)", "Bash(bundle install *)",
      "Bash(composer require *)", "Bash(dotnet add package *)",
      "Bash(nix-env *)", "Bash(nix profile install *)", "Bash(cachix use *)",
      "Bash(apt install *)", "Bash(apt-get install *)",
      "Bash(brew install *)", "Bash(pacman -S *)",
      "Bash(curl * | bash *)", "Bash(curl * | sh *)",
      "Bash(wget * | bash *)", "Bash(wget * | sh *)"
    ],
    "allow": [
      "Bash(/opt/company/scripts/safe-install *)",
      "Bash(npm run *)", "Bash(npm test *)", "Bash(npm audit *)",
      "Bash(pip-audit *)", "Bash(cargo audit *)",
      "mcp__company_package_security__install_package",
      "mcp__company_package_security__check_package"
    ]
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/company/claude-hooks/package-guardrail.sh",
            "timeout": 30,
            "statusMessage": "Org security policy: validating package..."
          }
        ]
      }
    ]
  },
  "mcpServers": {
    "company_package_security": {
      "command": "/opt/company/mcp-servers/package-security/bin/server",
      "env": {
        "SOCKET_API_KEY": "${COMPANY_SOCKET_API_KEY}",
        "AUDIT_LOG_PATH": "/var/log/claude-code/package-audit.jsonl"
      },
      "alwaysLoad": true
    }
  },
  "claudeMd": "## Company Policy\n\nAll package installations must go through the company's package security MCP server. Direct install commands are blocked by organizational policy."
}
```

**Step 2: HTTP hooks for centralized policy** (alternative to local scripts):

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

HTTP hooks provide centralized policy that updates instantly across all developers without git pull or restart.

**Step 3: NixOS module** (deploy managed settings via NixOS):

```nix
# /etc/nixos/claude-code-guardrails.nix
{ pkgs, ... }:
{
  environment.etc."claude-code/managed-settings.json" = {
    text = builtins.toJSON {
      allowManagedHooksOnly = true;
      allowManagedPermissionRulesOnly = true;
      # ... full managed settings object
    };
    mode = "0444";  # Read-only
  };
}
```

---

## 9. Known Limitations and Gaps

### 9.1 Bypass Vector Inventory

| # | Bypass Vector | Severity | Current Mitigation | Residual Risk |
|---|---|---|---|---|
| 1 | **Shell wrapper**: `bash -c "npm install evil"` | Medium | Hook regex parses for nested install patterns; deny rules miss this | Hook regex can be evaded with encoding/quoting |
| 2 | **Variable expansion**: `CMD=npm; $CMD install evil` | Medium | Not catchable by hooks (sees literal `$CMD`); OS config still applies | `.npmrc ignore-scripts` blocks postinstall; package itself still installs |
| 3 | **Subprocess spawning**: `python -c "subprocess.run(['pip','install','evil'])"` | Low | Hard to pattern-match; OS sandbox restricts network | Sandbox may not be enabled in all environments |
| 4 | **Manifest edit + bare install**: Edit `package.json`, run `npm install` | Medium | Hook rewrites bare `npm install` to `npm ci`; PostToolUse diffs lockfile | If lockfile does not exist yet, `npm ci` fails and agent falls back |
| 5 | **Package aliasing**: `npm install safe@npm:evil-pkg` | Low | Socket.dev resolves actual identity; hook can parse `@npm:` syntax | Requires Socket.dev integration to catch |
| 6 | **Obfuscated commands**: `eval $(echo "bnBtIGluc3RhbGw=" \| base64 -d)` | Low | Not catchable by pattern matching; LLM-based `prompt` hook could detect | High latency and non-determinism with prompt hooks |
| 7 | **`disableAllHooks` setting** | High | Managed settings with `allowManagedHooksOnly: true` prevent this | Only works with enterprise managed settings deployment |
| 8 | **Post-install script execution** | High | `.npmrc ignore-scripts=true`; `PIP_ONLY_BINARY=:all:`; hook rewrites to append safety flags | Some packages require install scripts to function; allowlist needed |

### 9.2 Nix Vulnerability Tooling Gap

**No Nix ecosystem exists in OSV.dev.** The tools that exist (vulnix, nix-security-tracker) lack API surfaces suitable for real-time hook integration.

**Current best approach:**
1. Map nixpkgs attribute names to upstream package names (e.g., `python3Packages.requests` -> `requests` on PyPI) and query OSV.dev
2. Run vulnix periodically as a system-level audit (not in hooks)
3. Block `nix-env -i` via deny rules (imperative installs bypass flake pinning)
4. Rely on Nix's inherent protections: sandboxed builds, content-addressed store, flake lock pinning

**What would fix this**: A machine-readable mapping from nixpkgs attribute names to upstream ecosystems, or nixpkgs appearing as an OSV ecosystem.

### 9.3 Subagent CLAUDE.md Regression

Since v2.1.84, built-in subagents have `omitClaudeMd: true`. CLAUDE.md security instructions are invisible to Explore, Plan, and general-purpose subagents. This is a documented regression (issue #40459) with no fix as of this writing.

**Impact**: If a built-in subagent performs a package install, CLAUDE.md behavioral guidance does not apply.

**Mitigation**: Hooks and permission deny rules still fire for all subagent tool calls. The enforcement layers are unaffected; only the advisory layer is degraded.

### 9.4 Provenance Verification Immaturity

No major ecosystem allows consumers to **require** provenance at install time:
- npm provenance: ~7% adoption. `npm audit signatures` is project-level, not per-package pre-install.
- PyPI PEP 740: ~20,000 attestations. pip does not verify attestations automatically.
- Go: checksum database is the only default-enforced integrity mechanism.
- Cargo: cargo-vet provides human-review attestation but is opt-in.

**Recommendation**: Use provenance as a **positive signal** (prefer packages with attestations) but do not block on absence. Revisit when adoption crosses ~50%.

### 9.5 False Positive Considerations

| Source | False Positive Risk | Mitigation |
|---|---|---|
| OSV.dev | Low -- reports only confirmed CVEs with version ranges | Check if installed version is actually in affected range |
| Socket.dev | Medium -- scoring algorithm may penalize legitimate low-maintenance packages | Use as warning signal, not hard block, for scores 0.3-0.5 |
| Age gate (3 days) | Medium -- blocks legitimate new packages and security patches | Auto-exempt security updates; allow user override via `ask` decision |
| Typosquat detection | Medium -- common short names may trigger false positives | Only block when similarity score is high AND target package has low downloads |
| npm audit | Low-Medium -- may flag vulnerabilities in devDependencies that are not exploitable | Use `--omit=dev` for production audit; `--audit-level=high` to filter noise |

### 9.6 Historical Enforcement Bugs

These bugs demonstrate why defense-in-depth matters:

| Bug | Version | Impact | Status |
|---|---|---|---|
| 50-subcommand bypass | Pre-v2.1.90 | Commands with 50+ subcommands silently skipped deny rule enforcement | Patched (v2.1.90) |
| Deny rules non-functional | v1.0.93 | ALL deny rules in settings.json were completely ignored | Patched |
| Local settings deny bypass | Issue #8961 | Deny rules in `.claude/settings.local.json` were ignored | Status unclear |

Each of these bugs would have been caught by the hook layer (which fires independently of permission rules) or the OS config layer (which fires independently of Claude Code entirely).

---

## Sources

This specification synthesizes findings from these research reports:

| Report | Key Contribution |
|--------|-----------------|
| `hooks-research.md` | PreToolUse mechanics, handler types, exit code semantics, `updatedInput`, bypass vectors, execution order, managed settings |
| `permissions-research.md` | Deny/allow/ask rules, glob syntax, compound command handling, settings hierarchy, permission modes, 50-subcommand bypass |
| `mcp-server-research.md` | Socket.dev + Snyk MCP setup, custom MCP server architecture, MCP tool hooks, enforcement patterns |
| `claude-md-guardrails-research.md` | CLAUDE.md loading and processing, effectiveness measurements, failure modes, subagent gap, compaction behavior |
| `custom-skills-research.md` | Skill mechanics, /safe-install pattern, lifecycle-scoped hooks, skills vs hooks vs MCP comparison, indirect forcing pattern |
| `vulnerability-apis-research.md` | OSV.dev deep dive (latency, coverage, query format), Socket.dev scoring algorithm, deps.dev typosquatting, GHSA malware, Nix tooling gap, provenance immaturity |
| `sibling-spike-cross-reference.md` | Age gates, install script sandboxing, lockfile enforcement, private registries, Nix-specific protections, implementation priority tiers |
