# Consulting-Specific Claude Code Hook Configurations

## Executive Summary

This report presents six prototype hook configurations designed for consulting-firm use cases, informed by empirical analysis of 15+ community hook repositories, official Claude Code documentation, and SOC 2 compliance requirements. The configurations address the four gaps identified in community research: no cost/budget alerting, no CI/CD integration at the hook level, no enterprise compliance trail, and no client isolation enforcement. Each hook includes complete JSON configuration, a supporting script sketch, failure mode analysis, and deployment-level recommendation.

Key design principle: **anything that would cause client escalation if missed belongs in a hook, not CLAUDE.md.** Hooks are deterministic (exit code 2 = blocked, no exceptions), while CLAUDE.md instructions are probabilistic (compliance degrades as instruction count increases). Consulting firms need zero-tolerance enforcement for credential protection, test gates, and audit trails — exactly the use cases where hooks provide guarantees CLAUDE.md cannot.

---

## 1. Test Enforcement (Stop Hook)

### What It Enforces

When Claude Code finishes a response (the "Stop" event), this hook runs the project's test suite before accepting the result. If tests fail, it blocks the stop and sends Claude feedback explaining which tests failed, forcing Claude to fix the issue before completing.

### Why a Consulting Firm Needs This

Client-facing deliverables that ship with failing tests damage credibility and trigger escalations. CLAUDE.md can say "always run tests" but compliance is probabilistic — the community research found multiple bug reports of Claude ignoring testing instructions. A Stop hook guarantees tests run every time, no exceptions.

### Complete Configuration

**settings.json (project-level: `.claude/settings.json`)**:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/test-enforcement.sh",
            "timeout": 120,
            "statusMessage": "Running test suite..."
          }
        ]
      }
    ]
  }
}
```

**Supporting script: `.claude/hooks/test-enforcement.sh`**:

```bash
#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
STOP_HOOK_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false')

# Prevent infinite loops: if we're already in a stop-hook retry cycle, allow exit
if [ "$STOP_HOOK_ACTIVE" = "true" ]; then
  exit 0
fi

CWD=$(echo "$INPUT" | jq -r '.cwd')
cd "$CWD"

# Detect test runner based on project files
if [ -f "package.json" ]; then
  TEST_CMD="npm test -- --passWithNoTests 2>&1"
elif [ -f "pyproject.toml" ] || [ -f "setup.py" ]; then
  TEST_CMD="python -m pytest --tb=short -q 2>&1"
elif [ -f "go.mod" ]; then
  TEST_CMD="go test ./... 2>&1"
elif [ -f "Cargo.toml" ]; then
  TEST_CMD="cargo test 2>&1"
else
  # No test runner detected — allow stop, don't block on projects without tests
  exit 0
fi

# Run tests, capture output and exit code
TEST_OUTPUT=$(eval "$TEST_CMD") || TEST_EXIT=$?
TEST_EXIT=${TEST_EXIT:-0}

if [ "$TEST_EXIT" -ne 0 ]; then
  # Truncate output to avoid context bloat (last 50 lines)
  TRUNCATED=$(echo "$TEST_OUTPUT" | tail -50)

  # Block the stop — Claude will receive the reason and attempt to fix
  jq -n --arg reason "Tests failed. Fix these failures before completing:\n\n$TRUNCATED" \
    '{ "decision": "block", "reason": $reason }'
  exit 0
else
  exit 0
fi
```

### Edge Cases and Failure Modes

| Scenario | Behavior | Mitigation |
|----------|----------|------------|
| No tests exist | Script detects no test runner and exits 0 (allows stop) | `--passWithNoTests` flag for npm; pytest exits 0 with no tests by default |
| Long-running test suite (>120s) | Hook times out, action proceeds (non-blocking timeout) | Set timeout to match project's test duration; consider running only changed tests |
| Tests fail on issues Claude didn't cause | Claude enters fix loop on pre-existing failures | Keep test suite green before adopting hook; use `--findRelatedTests` variant for PostToolUse instead |
| Infinite retry loop | `stop_hook_active` flag prevents re-triggering | Always check this flag first |
| Test runner not installed | Command fails with non-zero exit (not 2), action proceeds | Non-blocking failure is the safe default |
| Hook script error | Exit code 1 = warning logged, stop proceeds | Acceptable — fail-open is better than locking developers out |

### Deployment Level

**Project-level** (`.claude/settings.json`, committed to repo). Each client project configures its own test runner. Not suitable for user-level because test commands differ per project.

### Implementation Complexity: **Moderate**

The core is simple, but handling test runner detection, output truncation, the `stop_hook_active` guard, and timeout tuning requires careful testing per project.

---

## 2. Credential/Secret Scanning (PreToolUse)

### What It Enforces

Before Claude writes content to any file (via Edit, Write, or MultiEdit tools), this hook scans the content being written for hardcoded credentials, API keys, private keys, and client-specific secrets. If a secret pattern is detected, the write is blocked and Claude receives feedback explaining what was caught.

### Why a Consulting Firm Needs This

Consultants work across multiple client environments with different credential sets. Claude sometimes helpfully includes example credentials, copies values from environment discussion, or generates realistic-looking but actual secrets. A single credential committed to a client repo triggers security incident response, SOC 2 findings, and potential contract penalties. This is the highest-consequence zero-tolerance category.

The community already has mintmcp/agent-security (regex-only, local-first) and karanb192's protect-secrets hook, confirming this is a solved pattern architecturally — the consulting-specific version adds client-specific secret patterns and stricter scope.

### Complete Configuration

**settings.json (user-level: `~/.claude/settings.json` for firm-wide enforcement)**:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/scan-secrets.sh",
            "timeout": 10,
            "statusMessage": "Scanning for credentials..."
          }
        ]
      }
    ]
  }
}
```

**Supporting script: `~/.claude/hooks/scan-secrets.sh`**:

```bash
#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name')

# Extract content being written based on tool type
if [ "$TOOL_NAME" = "Write" ]; then
  CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // ""')
  FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
elif [ "$TOOL_NAME" = "Edit" ] || [ "$TOOL_NAME" = "MultiEdit" ]; then
  CONTENT=$(echo "$INPUT" | jq -r '.tool_input.new_string // ""')
  FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
else
  exit 0
fi

# Skip scanning for certain file types (binary, images, etc.)
case "$FILE_PATH" in
  *.png|*.jpg|*.gif|*.ico|*.woff|*.woff2|*.ttf|*.eot) exit 0 ;;
esac

# --- Secret Detection Patterns ---
# Adapted from detect-secrets and truffleHog pattern sets

PATTERNS=(
  # AWS
  'AKIA[0-9A-Z]{16}'
  'aws[_-]?(secret[_-]?access[_-]?key|session[_-]?token)\s*[=:]\s*[A-Za-z0-9/+=]{20,}'

  # GitHub/GitLab tokens
  'gh[ps]_[A-Za-z0-9_]{36,}'
  'glpat-[A-Za-z0-9_-]{20,}'

  # Generic API keys
  '["\x27]?[Aa](pi|PI)[_-]?[Kk](ey|EY)["\x27]?\s*[=:]\s*["\x27][A-Za-z0-9_-]{20,}["\x27]'

  # Private keys
  '-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----'

  # JWT tokens
  'eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}'

  # Database connection strings
  '(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^\s"'"'"']{10,}'

  # Slack tokens
  'xox[bpras]-[A-Za-z0-9-]{10,}'

  # Stripe keys
  'sk_(live|test)_[A-Za-z0-9]{20,}'
  'pk_(live|test)_[A-Za-z0-9]{20,}'

  # SendGrid
  'SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}'

  # Twilio
  'AC[a-f0-9]{32}'

  # Generic secret assignment
  '(password|passwd|secret|token|credential)\s*[=:]\s*["\x27][^\s"'"'"']{8,}["\x27]'
)

# Check content against each pattern
for PATTERN in "${PATTERNS[@]}"; do
  if echo "$CONTENT" | grep -qPi "$PATTERN"; then
    MATCH=$(echo "$CONTENT" | grep -oPi "$PATTERN" | head -1)
    # Redact most of the match for the feedback message
    REDACTED="${MATCH:0:8}...REDACTED"

    jq -n --arg reason "BLOCKED: Potential secret detected in $FILE_PATH. Pattern match: '$REDACTED'. Use environment variables or a secrets manager instead of hardcoding credentials." \
      '{
        hookSpecificOutput: {
          hookEventName: "PreToolUse",
          permissionDecision: "deny",
          permissionDecisionReason: $reason
        }
      }'
    exit 0
  fi
done

# No secrets found — allow the write
exit 0
```

### Regex Patterns Covered

| Category | Pattern | Example Matches |
|----------|---------|-----------------|
| AWS Access Keys | `AKIA[0-9A-Z]{16}` | `AKIAIOSFODNN7EXAMPLE` |
| AWS Secrets | `aws_secret_access_key=...` | Config file assignments |
| GitHub Tokens | `ghp_[A-Za-z0-9_]{36,}` | Personal access tokens |
| GitLab Tokens | `glpat-...` | Personal access tokens |
| Private Keys | `-----BEGIN...PRIVATE KEY-----` | PEM-encoded keys |
| JWT Tokens | `eyJ...` three-part base64 | Bearer tokens |
| Database URIs | `mongodb://...`, `postgres://...` | Connection strings with embedded credentials |
| Slack Tokens | `xox[bpras]-...` | Bot/user tokens |
| Stripe Keys | `sk_live_...`, `sk_test_...` | API keys |
| Generic Secrets | `password = "..."` | Hardcoded password assignments |

### False Positive Handling

| False Positive Scenario | Mitigation |
|-------------------------|------------|
| Test fixtures with fake credentials | Use obviously fake values (`AKIAIOSFODNN7EXAMPLE`) — the regex matches all AKIA prefixes, so use env vars even in tests |
| Documentation examples | Write examples with placeholder syntax (`${YOUR_API_KEY}`) instead of realistic-looking strings |
| Base64 content that resembles JWT | Three-part JWT pattern is specific enough to avoid most false positives |
| Long random strings in generated code | Generic patterns require key/secret/password keywords as context anchors |
| `.env.example` files with placeholder values | If placeholders match patterns, accept the block — it's better to use `KEY=changeme` syntax |

**Escape hatch**: If a write is legitimately blocked, the developer can use `--dangerouslySkipPermissions` or add a project-level `.claude/settings.local.json` override. The user-level hook still fires, but the consulting firm can document the override policy.

### Deployment Level

**User-level** (`~/.claude/settings.json`) — firm-wide deployment via managed settings is ideal. Every consultant gets secret scanning regardless of project. Also add project-level hooks for client-specific patterns (e.g., client's internal API key format).

For managed policy (strongest enforcement):
```json
// /etc/claude-code/managed-settings.json (Linux)
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/consulting-hooks/scan-secrets.sh"
          }
        ]
      }
    ]
  }
}
```

### Implementation Complexity: **Moderate**

The hook structure is simple, but tuning regex patterns to minimize false positives while catching real secrets requires iteration. Consider starting with the mintmcp/agent-security package and extending with firm-specific patterns rather than building from scratch.

---

## 3. Destructive Command Prevention (PreToolUse)

### What It Enforces

Before Claude executes any Bash command, this hook checks against a blocklist of destructive patterns. Blocked categories: filesystem destruction, force pushes, hard resets, database drops, and — specific to consulting — commands that could affect other clients' environments (cross-project operations, production system access).

### Why a Consulting Firm Needs This

Community hooks (karanb192, multiple HN discussions) already block `rm -rf /` and `git push --force`. The consulting-specific version adds protections unique to multi-client environments: commands that reference other client directories, production hostnames, or deployment pipelines. A consultant accidentally running a deployment command for Client A while working on Client B's repo is a real-world incident category.

### Complete Configuration

**settings.json (user-level: `~/.claude/settings.json`)**:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/block-destructive.sh",
            "timeout": 5,
            "statusMessage": "Checking command safety..."
          }
        ]
      }
    ]
  }
}
```

**Supporting script: `~/.claude/hooks/block-destructive.sh`**:

```bash
#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')
CWD=$(echo "$INPUT" | jq -r '.cwd // ""')

deny() {
  jq -n --arg reason "$1" \
    '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "deny",
        permissionDecisionReason: $reason
      }
    }'
  exit 0
}

# --- Category 1: Filesystem Destruction ---
# rm -rf with dangerous targets
if echo "$COMMAND" | grep -qP 'rm\s+(-[a-zA-Z]*r[a-zA-Z]*f|--recursive\s+--force|-[a-zA-Z]*f[a-zA-Z]*r)\s+(/|~|\$HOME|\.\./)'; then
  deny "BLOCKED: Destructive rm command targeting dangerous path. Use specific file paths instead."
fi

# Fork bombs
if echo "$COMMAND" | grep -qP ':\(\)\{|\.&\}|/dev/sd'; then
  deny "BLOCKED: Potentially destructive system command."
fi

# --- Category 2: Git Destructive Operations ---
if echo "$COMMAND" | grep -qPi 'git\s+push\s+.*(-f|--force)\s+(origin\s+)?(main|master|production|release)'; then
  deny "BLOCKED: Force push to protected branch. Use a feature branch and PR workflow."
fi

if echo "$COMMAND" | grep -qPi 'git\s+reset\s+--hard'; then
  deny "BLOCKED: git reset --hard destroys uncommitted work. Use git stash or create a backup branch first."
fi

if echo "$COMMAND" | grep -qPi 'git\s+clean\s+-[a-zA-Z]*f'; then
  deny "BLOCKED: git clean -f removes untracked files permanently. Review files first with git clean -n."
fi

# --- Category 3: Database Destruction ---
if echo "$COMMAND" | grep -qPi '(DROP\s+(TABLE|DATABASE|SCHEMA)|TRUNCATE\s+TABLE|DELETE\s+FROM\s+\S+\s*;?\s*$)'; then
  deny "BLOCKED: Destructive database command. Require explicit WHERE clause for DELETE, or use migrations for schema changes."
fi

# --- Category 4: Dangerous Downloads/Execution ---
if echo "$COMMAND" | grep -qP 'curl\s.*\|\s*(bash|sh|zsh)|wget\s.*\|\s*(bash|sh|zsh)'; then
  deny "BLOCKED: Piping remote content to shell. Download first, review, then execute."
fi

# --- Category 5: Consulting-Specific — Cross-Environment Protection ---
# Block commands referencing production systems
if echo "$COMMAND" | grep -qPi '(deploy|push|publish|release)\s.*(prod|production|staging|live)'; then
  deny "BLOCKED: Deployment to production/staging from local environment. Use CI/CD pipeline instead."
fi

# Block SSH/SCP to production hosts (customize hostnames per firm)
if echo "$COMMAND" | grep -qPi '(ssh|scp|rsync)\s.*\b(prod|production|live)\b'; then
  deny "BLOCKED: Direct access to production systems. Use bastion hosts or deployment pipelines."
fi

# Block operations outside the current project directory tree
# (prevents accidentally affecting another client's files)
if echo "$COMMAND" | grep -qP '(rm|mv|cp|cat\s*>)\s+(/home|/Users)/[^/]+/((?!'"$(basename "$CWD")"')[^/]+)'; then
  deny "BLOCKED: File operation appears to target a directory outside the current project. Verify you're working in the correct client's directory."
fi

# --- Category 6: Container/Infrastructure ---
if echo "$COMMAND" | grep -qPi 'docker\s+(system\s+prune|container\s+prune|volume\s+prune)\s+(-a|--all)'; then
  deny "BLOCKED: Docker prune -a removes ALL unused resources including other projects. Use targeted cleanup."
fi

if echo "$COMMAND" | grep -qPi '(terraform|tofu)\s+(destroy|apply)\s+(-auto-approve|--auto-approve)'; then
  deny "BLOCKED: Terraform auto-approve bypasses review. Run plan first and review changes."
fi

# No dangerous patterns found — allow command
exit 0
```

### What Gets Blocked

| Category | Patterns | Consulting Rationale |
|----------|----------|---------------------|
| Filesystem destruction | `rm -rf /`, `rm -rf ~/`, fork bombs | Universal safety |
| Git force operations | `git push --force main`, `git reset --hard`, `git clean -f` | Protect shared branches; client repos are collaborative |
| Database destruction | `DROP TABLE`, `TRUNCATE`, bare `DELETE FROM` | Client data is irreplaceable |
| Remote code execution | `curl \| bash` | Supply chain risk for client environments |
| Production deployment | `deploy prod`, `ssh prod*` | Must use CI/CD; accidental deployment is a critical incident |
| Cross-project operations | File ops targeting sibling directories | Multi-client isolation |
| Infrastructure destruction | `terraform destroy -auto-approve`, `docker system prune -a` | Shared infrastructure; must review changes |

### Failure Modes

| Scenario | Behavior | Mitigation |
|----------|----------|------------|
| False positive blocks legitimate command | Developer sees deny reason and can rephrase or use the command manually | Deny reasons explain *why* and suggest alternatives |
| Regex evasion (e.g., `g\it push --force`) | Pattern not matched, command proceeds | Defense in depth — also use git server-side branch protection |
| Hook script has a bug | Exit code 1 = warning only, command proceeds | Fail-open design; test suite for the hook itself (262 tests in karanb192's collection) |
| Command uses aliases or scripts | `./deploy.sh` won't match `deploy...prod` | Document that wrapper scripts should be reviewed separately |
| Multiline or chained commands | `&&` or `;` chains may only partially match | Grep checks the entire command string; consider splitting on `&&`/`;` for thorough checking |

### Deployment Level

**User-level** (`~/.claude/settings.json`) for firm-wide baseline. Augment with project-level hooks for client-specific protections (e.g., client's production hostnames, specific databases).

For managed policy deployment, the `allowManagedHooksOnly` flag ensures developers cannot disable this hook:

```json
// /etc/claude-code/managed-settings.json
{
  "allowManagedHooksOnly": true,
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/consulting-hooks/block-destructive.sh"
          }
        ]
      }
    ]
  }
}
```

### Implementation Complexity: **Trivial to Moderate**

Basic destructive command blocking (community "starter pack") is trivial — a few grep patterns. The consulting-specific cross-environment protection requires customization per firm (directory layout, production hostnames, deployment patterns). The karanb192/claude-code-hooks collection provides a well-tested starting point.

---

## 4. Cost Alerting (PostToolUse / Stop)

### What It Enforces

Tracks cumulative token usage across a session by reading the session's JSONL transcript file. At configurable thresholds, injects warnings into Claude's context (soft limit) or blocks the stop to force a checkpoint (hard limit). This is a **gap** in the community — no published implementation exists.

### Why a Consulting Firm Needs This

Consulting firms bill Claude Code API costs to engagements. Runaway sessions (e.g., Claude in a fix-retry loop, or exploring a large codebase without guidance) can accumulate thousands of tokens. Per Anthropic's cost data, the average is $6/developer/day but the top 10% spend more than $12/day — and automated/agentic use cases can far exceed this. Without alerting, a consultant may not realize they've burned through a day's budget on a single prompt chain. Budget awareness also helps with client billing transparency.

### Complete Configuration

**settings.json (user-level: `~/.claude/settings.json`)**:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/cost-tracker.sh",
            "timeout": 5,
            "async": true,
            "statusMessage": "Tracking token usage..."
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/cost-check-stop.sh",
            "timeout": 10,
            "statusMessage": "Checking session budget..."
          }
        ]
      }
    ]
  }
}
```

**Supporting script: `~/.claude/hooks/cost-tracker.sh`** (PostToolUse — lightweight, async):

```bash
#!/usr/bin/env bash
# Runs async after each tool use to update a running cost tally.
# Writes to a sidecar file next to the session JSONL.
set -euo pipefail

INPUT=$(cat)
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path // ""')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')

if [ -z "$TRANSCRIPT_PATH" ] || [ ! -f "$TRANSCRIPT_PATH" ]; then
  exit 0
fi

# Cost tracking sidecar file
COST_FILE="${TRANSCRIPT_PATH%.jsonl}.cost.json"

# Calculate tokens from the JSONL
# Each line with "usage" contains: input_tokens, output_tokens, cache_creation_input_tokens, cache_read_input_tokens
TOKENS=$(grep -o '"usage":{[^}]*}' "$TRANSCRIPT_PATH" 2>/dev/null | \
  python3 -c "
import sys, json, re

total_input = 0
total_output = 0
total_cache_create = 0
total_cache_read = 0

for line in sys.stdin:
    # Extract the JSON object
    match = re.search(r'\{[^}]+\}', line)
    if match:
        try:
            usage = json.loads(match.group())
            total_input += usage.get('input_tokens', 0)
            total_output += usage.get('output_tokens', 0)
            total_cache_create += usage.get('cache_creation_input_tokens', 0)
            total_cache_read += usage.get('cache_read_input_tokens', 0)
        except json.JSONDecodeError:
            pass

# Approximate cost calculation (Sonnet 4.6 pricing)
# Input: \$3/MTok, Output: \$15/MTok, Cache write: \$3.75/MTok, Cache read: \$0.30/MTok
input_cost = (total_input / 1_000_000) * 3.0
output_cost = (total_output / 1_000_000) * 15.0
cache_create_cost = (total_cache_create / 1_000_000) * 3.75
cache_read_cost = (total_cache_read / 1_000_000) * 0.30
total_cost = input_cost + output_cost + cache_create_cost + cache_read_cost

result = {
    'session_id': '$SESSION_ID',
    'input_tokens': total_input,
    'output_tokens': total_output,
    'cache_creation_tokens': total_cache_create,
    'cache_read_tokens': total_cache_read,
    'estimated_cost_usd': round(total_cost, 4),
    'last_updated': '$(date -Iseconds)'
}
print(json.dumps(result))
" 2>/dev/null)

if [ -n "$TOKENS" ]; then
  echo "$TOKENS" > "$COST_FILE"
fi

exit 0
```

**Supporting script: `~/.claude/hooks/cost-check-stop.sh`** (Stop — budget gate):

```bash
#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path // ""')
STOP_HOOK_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false')

if [ "$STOP_HOOK_ACTIVE" = "true" ]; then
  exit 0
fi

COST_FILE="${TRANSCRIPT_PATH%.jsonl}.cost.json"

if [ ! -f "$COST_FILE" ]; then
  exit 0
fi

COST=$(jq -r '.estimated_cost_usd // 0' "$COST_FILE")
INPUT_TOKENS=$(jq -r '.input_tokens // 0' "$COST_FILE")
OUTPUT_TOKENS=$(jq -r '.output_tokens // 0' "$COST_FILE")

# --- Configurable Thresholds ---
SOFT_LIMIT=${CLAUDE_COST_SOFT_LIMIT:-1.00}   # Warn at $1.00
HARD_LIMIT=${CLAUDE_COST_HARD_LIMIT:-5.00}   # Block at $5.00

# Compare using bc for floating point
EXCEEDS_HARD=$(echo "$COST >= $HARD_LIMIT" | bc -l 2>/dev/null || echo "0")
EXCEEDS_SOFT=$(echo "$COST >= $SOFT_LIMIT" | bc -l 2>/dev/null || echo "0")

if [ "$EXCEEDS_HARD" = "1" ]; then
  jq -n \
    --arg reason "SESSION BUDGET EXCEEDED (\$$COST spent, limit \$$HARD_LIMIT). Input: ${INPUT_TOKENS} tokens, Output: ${OUTPUT_TOKENS} tokens. Consider: (1) /clear and start a focused new session, (2) /compact to reduce context size, (3) complete current work and stop." \
    '{ "decision": "block", "reason": $reason }'
  exit 0
elif [ "$EXCEEDS_SOFT" = "1" ]; then
  # Soft warning — inject context but don't block
  jq -n \
    --arg ctx "NOTE: Session cost is \$$COST (soft limit: \$$SOFT_LIMIT). Be cost-conscious: avoid unnecessary file reads, use targeted searches, consider /compact if context is large." \
    '{ "additionalContext": $ctx }'
  exit 0
fi

exit 0
```

### How It Works

1. **PostToolUse** (async): After every tool call, reads the session JSONL, sums token usage fields, calculates approximate cost using model pricing, writes to a `.cost.json` sidecar file.
2. **Stop**: When Claude finishes responding, reads the sidecar file and checks against thresholds. Soft limit injects a cost-awareness note. Hard limit blocks the stop with a detailed budget message.

### Configuring Thresholds

Environment variables allow per-engagement customization without modifying the script:

```bash
export CLAUDE_COST_SOFT_LIMIT=2.00   # Client with generous budget
export CLAUDE_COST_HARD_LIMIT=10.00
```

Or set in SessionStart hook via `CLAUDE_ENV_FILE`:

```bash
#!/bin/bash
# .claude/hooks/set-budget.sh (SessionStart)
if [ -n "$CLAUDE_ENV_FILE" ]; then
  echo 'export CLAUDE_COST_SOFT_LIMIT=1.50' >> "$CLAUDE_ENV_FILE"
  echo 'export CLAUDE_COST_HARD_LIMIT=5.00' >> "$CLAUDE_ENV_FILE"
fi
```

### Failure Modes

| Scenario | Behavior | Mitigation |
|----------|----------|------------|
| JSONL format changes between Claude Code versions | Token calculation fails silently, cost file not updated | Script exits 0 on error; periodic manual verification with `/cost` |
| Cost sidecar file doesn't exist yet | Stop hook exits 0, no budget enforcement | PostToolUse hook creates it after first tool call |
| Pricing model changes | Cost estimate becomes inaccurate | Pricing constants in script, easy to update; always show token counts alongside dollar amounts |
| Async PostToolUse hasn't finished when Stop fires | Cost file may be stale | Acceptable — the Stop hook could also recalculate directly if needed |
| Session JSONL is very large (>100MB) | grep + python parsing is slow | Consider reading only the last N lines, or tracking incrementally (append-only sidecar) |
| Multiple Claude Code instances sharing a project | Cost files could collide | Session ID in filename prevents collisions |

### Deployment Level

**User-level** (`~/.claude/settings.json`) for firm-wide cost awareness. Thresholds can be customized per engagement via environment variables or SessionStart hooks.

### Implementation Complexity: **Complex**

This is the most complex hook in the set. It requires: (1) understanding the JSONL session format, (2) token-to-cost conversion with model-specific pricing, (3) sidecar file management, (4) environment variable configuration, (5) floating-point comparison in bash. Consider implementing in Python for robustness. Alternatively, use the ccusage CLI tool (ryoppippi/ccusage) as a pre-built foundation for token parsing and build the alerting layer on top.

---

## 5. Session Logging for Compliance (Multiple Events)

### What It Enforces

Logs structured session metadata to a local audit file at session boundaries and after tool use. Captures: session ID, start/end timestamps, project directory, model used, tools invoked, token counts, and session duration. Designed for SOC 2 audit trail requirements without transmitting session content (no prompts, no code, no responses).

### Why a Consulting Firm Needs This

SOC 2 Type II audits require evidence that AI coding tool usage is monitored and controlled. Per the 2026 Trust Services Criteria, organizations must document: who used AI tools, when, on which systems, and what controls were in place. The audit question is: "How do you know who used AI tools, when, and with what code?" This hook provides the "who, when, where" without the "what" — preserving developer privacy while satisfying audit requirements.

Additionally, consulting firms need to demonstrate to clients that AI tool usage on their engagements is tracked and governable. Session logs provide evidence for client security questionnaires and compliance certifications.

### Complete Configuration

**settings.json (managed policy for firm-wide enforcement)**:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/consulting-hooks/audit-log.sh session_start",
            "timeout": 5
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/opt/consulting-hooks/audit-log.sh tool_use",
            "timeout": 3,
            "async": true
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/opt/consulting-hooks/audit-log.sh session_checkpoint",
            "timeout": 5
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/opt/consulting-hooks/audit-log.sh session_end",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

**Supporting script: `/opt/consulting-hooks/audit-log.sh`**:

```bash
#!/usr/bin/env bash
set -euo pipefail

EVENT_TYPE="$1"
INPUT=$(cat)

# --- Configuration ---
AUDIT_DIR="${CLAUDE_AUDIT_DIR:-$HOME/.claude/audit}"
AUDIT_FILE="$AUDIT_DIR/claude-sessions-$(date +%Y-%m).jsonl"
mkdir -p "$AUDIT_DIR"

# --- Extract common fields ---
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')
CWD=$(echo "$INPUT" | jq -r '.cwd // "unknown"')
TIMESTAMP=$(date -Iseconds)
USERNAME=$(whoami)
HOSTNAME=$(hostname)

# --- Map cwd to client engagement (optional) ---
# Consulting firms typically organize: ~/clients/<client-name>/<project>/
CLIENT=$(echo "$CWD" | sed -n 's|.*/clients/\([^/]*\)/.*|\1|p')
CLIENT="${CLIENT:-unassigned}"

case "$EVENT_TYPE" in
  session_start)
    SOURCE=$(echo "$INPUT" | jq -r '.source // "unknown"')
    MODEL=$(echo "$INPUT" | jq -r '.model // "unknown"')

    jq -n \
      --arg event "session_start" \
      --arg session_id "$SESSION_ID" \
      --arg timestamp "$TIMESTAMP" \
      --arg user "$USERNAME" \
      --arg host "$HOSTNAME" \
      --arg cwd "$CWD" \
      --arg client "$CLIENT" \
      --arg source "$SOURCE" \
      --arg model "$MODEL" \
      '{
        event: $event,
        session_id: $session_id,
        timestamp: $timestamp,
        user: $user,
        hostname: $host,
        project_dir: $cwd,
        client_engagement: $client,
        start_source: $source,
        model: $model
      }' >> "$AUDIT_FILE"
    ;;

  tool_use)
    TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // "unknown"')

    jq -n \
      --arg event "tool_use" \
      --arg session_id "$SESSION_ID" \
      --arg timestamp "$TIMESTAMP" \
      --arg tool "$TOOL_NAME" \
      --arg client "$CLIENT" \
      '{
        event: $event,
        session_id: $session_id,
        timestamp: $timestamp,
        tool_name: $tool,
        client_engagement: $client
      }' >> "$AUDIT_FILE"
    ;;

  session_checkpoint)
    # Log a checkpoint when Claude completes a response cycle
    jq -n \
      --arg event "session_checkpoint" \
      --arg session_id "$SESSION_ID" \
      --arg timestamp "$TIMESTAMP" \
      --arg client "$CLIENT" \
      '{
        event: $event,
        session_id: $session_id,
        timestamp: $timestamp,
        client_engagement: $client
      }' >> "$AUDIT_FILE"
    ;;

  session_end)
    # Read cost sidecar if available (from Hook 4)
    TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path // ""')
    COST_FILE="${TRANSCRIPT_PATH%.jsonl}.cost.json"

    if [ -f "$COST_FILE" ]; then
      TOTAL_COST=$(jq -r '.estimated_cost_usd // 0' "$COST_FILE")
      INPUT_TOKENS=$(jq -r '.input_tokens // 0' "$COST_FILE")
      OUTPUT_TOKENS=$(jq -r '.output_tokens // 0' "$COST_FILE")
    else
      TOTAL_COST="unknown"
      INPUT_TOKENS="unknown"
      OUTPUT_TOKENS="unknown"
    fi

    jq -n \
      --arg event "session_end" \
      --arg session_id "$SESSION_ID" \
      --arg timestamp "$TIMESTAMP" \
      --arg user "$USERNAME" \
      --arg host "$HOSTNAME" \
      --arg cwd "$CWD" \
      --arg client "$CLIENT" \
      --arg cost "$TOTAL_COST" \
      --arg input_tok "$INPUT_TOKENS" \
      --arg output_tok "$OUTPUT_TOKENS" \
      '{
        event: $event,
        session_id: $session_id,
        timestamp: $timestamp,
        user: $user,
        hostname: $host,
        project_dir: $cwd,
        client_engagement: $client,
        estimated_cost_usd: $cost,
        input_tokens: $input_tok,
        output_tokens: $output_tok
      }' >> "$AUDIT_FILE"
    ;;
esac

exit 0
```

### Audit Log Format

Monthly JSONL files in `~/.claude/audit/claude-sessions-YYYY-MM.jsonl`:

```jsonl
{"event":"session_start","session_id":"abc-123","timestamp":"2026-03-27T14:30:00+00:00","user":"jsmith","hostname":"dev-laptop","project_dir":"/home/jsmith/clients/acme/web-app","client_engagement":"acme","start_source":"startup","model":"claude-sonnet-4-6"}
{"event":"tool_use","session_id":"abc-123","timestamp":"2026-03-27T14:30:05+00:00","tool_name":"Read","client_engagement":"acme"}
{"event":"tool_use","session_id":"abc-123","timestamp":"2026-03-27T14:30:12+00:00","tool_name":"Edit","client_engagement":"acme"}
{"event":"session_checkpoint","session_id":"abc-123","timestamp":"2026-03-27T14:30:30+00:00","client_engagement":"acme"}
{"event":"session_end","session_id":"abc-123","timestamp":"2026-03-27T15:45:00+00:00","user":"jsmith","hostname":"dev-laptop","project_dir":"/home/jsmith/clients/acme/web-app","client_engagement":"acme","estimated_cost_usd":"1.23","input_tokens":"45000","output_tokens":"12000"}
```

### SOC 2 Mapping

| SOC 2 Trust Services Criterion | What This Hook Provides |
|-------------------------------|------------------------|
| CC6.1 — Logical access controls | User identity, hostname, session timestamps |
| CC6.2 — Access restriction | Client engagement mapping (who accessed what) |
| CC7.2 — Monitoring activities | Tool usage frequency, session duration |
| CC7.3 — Change detection | Tool names logged (Edit, Write = code changes) |
| CC8.1 — Change management | Session correlation with git commits (via timestamps) |

### What Is Deliberately NOT Logged

- Prompt content (privacy, IP protection)
- Response content (client code confidentiality)
- File contents (data sensitivity)
- File paths beyond the project root (leaks project structure details)

This is metadata-only logging. Session content remains in the local JSONL transcript files, which are separate and can be managed under different retention policies.

### Failure Modes

| Scenario | Behavior | Mitigation |
|----------|----------|------------|
| Audit directory doesn't exist | Script creates it with `mkdir -p` | Handled in script |
| Disk full | jq write fails, script exits non-zero (not 2) | Action proceeds; set up disk space monitoring separately |
| Concurrent writes from multiple sessions | JSONL append is mostly atomic on Linux | Acceptable for audit purposes; each line is independent |
| SessionEnd not fired (crash, kill -9) | Missing end record | Correlation of start records without matching end records flags incomplete sessions |
| Clock skew between machines | Timestamps inconsistent | Use NTP; include hostname for disambiguation |

### Deployment Level

**Managed policy** (`/etc/claude-code/managed-settings.json`) — mandatory for all consultants, cannot be disabled. This is the strongest enforcement level, appropriate for compliance-critical controls.

### Implementation Complexity: **Moderate**

The script itself is straightforward (jq + append to file). Complexity is in the operational wrapper: log rotation, backup, access controls on the audit directory, and integration with the firm's SIEM or compliance platform for SOC 2 evidence collection.

---

## 6. Client Isolation Verification (SessionStart)

### What It Enforces

On session start, checks whether the current working directory maps to a known client engagement directory. If the directory is unrecognized, injects a warning into Claude's context. If it maps to a known client, injects client-specific context (project standards, sensitive directories to avoid, compliance requirements).

### Why a Consulting Firm Needs This

Consultants switch between client projects frequently. Opening Claude Code in the wrong directory — or in a personal/scratch directory without engagement controls — means none of the project-level hooks or CLAUDE.md settings apply. This hook provides a safety net: it doesn't block work, but it makes the consultant aware of their context and ensures client-specific rules are active.

### Complete Configuration

**settings.json (user-level: `~/.claude/settings.json`)**:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/client-isolation.sh",
            "timeout": 5,
            "statusMessage": "Verifying client engagement..."
          }
        ]
      }
    ]
  }
}
```

**Supporting script: `~/.claude/hooks/client-isolation.sh`**:

```bash
#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
CWD=$(echo "$INPUT" | jq -r '.cwd // ""')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')

# --- Client Registry ---
# Maps directory patterns to client names and their specific context
# Customize this per firm. Could also load from a YAML/JSON config file.
CLIENTS_DIR="${HOME}/clients"
CLIENT_REGISTRY="${HOME}/.claude/client-registry.json"

# Detect client from directory path
# Expected structure: ~/clients/<client-name>/<project>/...
CLIENT=""
if [[ "$CWD" == "$CLIENTS_DIR"/* ]]; then
  # Extract client name from path
  RELATIVE="${CWD#$CLIENTS_DIR/}"
  CLIENT="${RELATIVE%%/*}"
fi

if [ -z "$CLIENT" ]; then
  # Not in a recognized client directory
  jq -n --arg ctx "WARNING: You are not in a recognized client engagement directory. Current directory: $CWD. Client projects should be under $CLIENTS_DIR/<client-name>/. Project-level hooks and CLAUDE.md may not be active. If this is intentional (personal project, internal work), proceed with caution." \
    '{ "additionalContext": $ctx }'
  exit 0
fi

# In a recognized client directory — inject client-specific context
CONTEXT="Client engagement: $CLIENT. "

# Load client-specific rules if registry exists
if [ -f "$CLIENT_REGISTRY" ]; then
  CLIENT_RULES=$(jq -r --arg client "$CLIENT" '.[$client] // empty' "$CLIENT_REGISTRY")
  if [ -n "$CLIENT_RULES" ]; then
    SENSITIVE_DIRS=$(echo "$CLIENT_RULES" | jq -r '.sensitive_dirs // [] | join(", ")')
    COMPLIANCE=$(echo "$CLIENT_RULES" | jq -r '.compliance // "standard"')
    NOTES=$(echo "$CLIENT_RULES" | jq -r '.notes // ""')

    CONTEXT+="Compliance level: $COMPLIANCE. "
    if [ -n "$SENSITIVE_DIRS" ]; then
      CONTEXT+="Sensitive directories (do not read/modify without explicit approval): $SENSITIVE_DIRS. "
    fi
    if [ -n "$NOTES" ]; then
      CONTEXT+="Client notes: $NOTES"
    fi
  fi
fi

jq -n --arg ctx "$CONTEXT" '{ "additionalContext": $ctx }'
exit 0
```

**Client registry file: `~/.claude/client-registry.json`**:

```json
{
  "acme-corp": {
    "compliance": "SOC2+HIPAA",
    "sensitive_dirs": ["infrastructure/secrets", "database/migrations", "config/production"],
    "notes": "All database changes require migration files. No direct SQL execution. PHI data in /data/ directory — never read or reference."
  },
  "globex": {
    "compliance": "standard",
    "sensitive_dirs": ["deploy/"],
    "notes": "Uses monorepo — only modify packages/ assigned to our team."
  },
  "initech": {
    "compliance": "SOC2",
    "sensitive_dirs": ["secrets/", ".env*"],
    "notes": "Legacy codebase. Do not refactor outside current ticket scope."
  }
}
```

### What Gets Injected

| Directory | Behavior |
|-----------|----------|
| `~/clients/acme-corp/web-app` | Injects: "Client engagement: acme-corp. Compliance level: SOC2+HIPAA. Sensitive directories: infrastructure/secrets, database/migrations, config/production. Client notes: All database changes require migration files..." |
| `~/clients/globex/api` | Injects: "Client engagement: globex. Compliance level: standard. Sensitive directories: deploy/. Client notes: Uses monorepo..." |
| `~/personal-projects/experiment` | Injects: "WARNING: You are not in a recognized client engagement directory..." |
| `/tmp/scratch` | Injects warning about unrecognized directory |

### Failure Modes

| Scenario | Behavior | Mitigation |
|----------|----------|------------|
| Client registry file doesn't exist | Client is detected from directory but no rules injected | Graceful degradation — still shows client name |
| Directory structure doesn't match convention | Client not detected, warning injected | Warning is informational, not blocking — doesn't break workflow |
| Registry has stale entries | Old client context injected | Review registry periodically; could auto-expire entries |
| Consultant works in nested directory far from client root | Client still detected from path ancestry | Works as designed — regex extracts first component after clients/ |
| SessionStart doesn't fire on resume | Use `startup\|resume` matcher to cover both cases | Configuration shown uses `startup` only; extend to `resume` if needed |

### Deployment Level

**User-level** (`~/.claude/settings.json`) — each consultant maintains their own client registry. The hook script itself can be firm-managed (shared via managed settings), but the registry is personal since each consultant works on different clients.

### Implementation Complexity: **Trivial**

The simplest hook in the set. Directory path matching and JSON context injection. The client registry is a simple JSON file. No external dependencies beyond jq.

---

## Deployment Strategy Summary

### Hook Precedence and Layering

```
┌─────────────────────────────────────────────────┐
│  Managed Policy (/etc/claude-code/)             │  Cannot be overridden
│  - Secret scanning (Hook 2)                      │  IT/DevOps deploys
│  - Destructive command prevention (Hook 3)       │
│  - Compliance logging (Hook 5)                   │
├─────────────────────────────────────────────────┤
│  User-Level (~/.claude/settings.json)           │  Consultant configures
│  - Cost alerting (Hook 4)                        │
│  - Client isolation (Hook 6)                     │
├─────────────────────────────────────────────────┤
│  Project-Level (.claude/settings.json)          │  Per-client project
│  - Test enforcement (Hook 1)                     │  Committed to repo
│  - Client-specific secret patterns               │
│  - Client-specific destructive command patterns   │
└─────────────────────────────────────────────────┘
```

### Implementation Priority

| Priority | Hook | Rationale |
|----------|------|-----------|
| 1 (Do First) | Destructive Command Prevention (#3) | Lowest complexity, highest immediate value, community-proven patterns |
| 2 | Credential Scanning (#2) | High consequence if missed, existing tools (mintmcp/agent-security) provide starting point |
| 3 | Test Enforcement (#1) | Moderate complexity, project-specific configuration needed |
| 4 | Client Isolation (#6) | Trivial implementation, useful orientation for consultants |
| 5 | Session Logging (#5) | Moderate complexity, needed before SOC 2 audit cycle |
| 6 (Do Last) | Cost Alerting (#4) | Highest complexity, most novel (no community precedent), requires JSONL parsing |

### Supporting Script Requirements

| Hook | External Dependencies | Language |
|------|----------------------|----------|
| Test Enforcement | Project test runner (npm, pytest, etc.) | Bash |
| Credential Scanning | grep with PCRE (-P flag) | Bash (or Python for complex patterns) |
| Destructive Commands | grep with PCRE (-P flag) | Bash |
| Cost Alerting | python3, bc, jq | Bash + Python |
| Session Logging | jq | Bash |
| Client Isolation | jq | Bash |

All hooks require **jq** for JSON parsing of stdin input. This is the single universal dependency.

---

## Interaction Between Hooks

Several hooks are designed to work together:

1. **Cost Alerting (#4) feeds Session Logging (#5)**: The cost sidecar file created by the PostToolUse cost tracker is read by the SessionEnd audit logger to record final session cost.

2. **Client Isolation (#6) feeds Session Logging (#5)**: The client engagement name extracted from the directory path appears in audit log entries.

3. **Test Enforcement (#1) and Cost Alerting (#4) both use Stop**: When both fire on the same Stop event, they execute sequentially. If tests fail (blocking), the cost check doesn't matter yet. If tests pass, cost check runs next.

4. **Credential Scanning (#2) and Destructive Commands (#3) both use PreToolUse**: Different matchers (Write|Edit vs. Bash) so they don't conflict. If both need to fire (e.g., a Bash command that writes a file), Claude Code runs all matching hooks.

---

## Comparison to Alternatives

| Requirement | Claude Code Hook | Pre-commit Hook | CI/CD Pipeline |
|-------------|-----------------|-----------------|----------------|
| Test enforcement | Runs during development, immediate feedback | Runs at commit time | Runs after push, delayed feedback |
| Secret scanning | Blocks before content is written | Catches at commit | Catches after push (too late for some secrets) |
| Destructive commands | Blocks before execution | N/A | N/A |
| Cost tracking | Real-time during session | N/A | N/A |
| Audit logging | Every tool use, real-time | Commit-level only | Build-level only |
| Client isolation | Session start | N/A | Repo-level only |

**Key advantage of hooks**: They intervene *before* damage is done. Pre-commit hooks catch issues at commit time (after code is written). CI/CD catches issues after push (after code is shared). Hooks catch issues at the moment of generation — before the content ever reaches the filesystem, git, or a remote.

**Hooks are not a replacement for CI/CD** — they're an additional layer. The consulting recommendation is defense-in-depth: hooks for immediate prevention, pre-commit for commit-time validation, CI/CD for comprehensive testing.

---

## Open Questions and Future Work

1. **Hook performance at scale**: How do 6+ hooks (some on every PostToolUse) affect Claude Code latency? The async flag helps for non-blocking hooks, but the Stop hook pipeline (tests + cost check) could add 30-120 seconds.

2. **Managed policy deployment tooling**: The documentation describes managed-settings.json at system paths, but the actual MDM/deployment story for Linux (where most developers work) is underspecified. NixOS makes this easy (declarative config), but other Linux distributions may need Ansible/Puppet recipes.

3. **JSONL format stability**: The cost alerting hook depends on the internal structure of Claude Code's session JSONL files. These are not documented as a stable API. Version upgrades could break the parser.

4. **Multi-instance sessions**: When using agent teams (7x token cost), the cost tracker would need to aggregate across teammate transcripts. The current design tracks only the main session.

5. **Hook testing framework**: Community collections like karanb192's have 262 tests. A consulting firm's hook suite should be tested similarly — but there's no standard testing framework for Claude Code hooks yet.

---

## Sources

### Web Sources (saved to docs/)
- `docs/claude-code-costs-official.md` — Official cost management documentation
- `docs/mintmcp-agent-security.md` — Agent-security secrets scanning implementation
- `docs/claude-code-hooks-reference-full.md` — Complete hooks reference documentation

### Existing Spike Sources
- `community-hooks-research.md` — Community hook configurations survey (15+ repos)
- `claudemd-patterns-research.md` — CLAUDE.md patterns and enforcement spectrum analysis
- `docs/github-karanb192-claude-code-hooks.md` — Safety levels and block-dangerous-commands implementation
- `docs/github-chriswiles-claude-code-showcase.md` — Full CI-in-editor pipeline with hooks

### Web Search Sources (not separately fetched)
- [SOC 2 and AI Coding Tools Compliance](https://www.getprobo.com/hub/ai-coding-tools-soc2-compliance) — Audit trail requirements for AI assistants
- [Best Ways to Monitor Claude Code Token Usage](https://dev.to/kuldeep_paul/best-ways-to-monitor-claude-code-token-usage-and-costs-in-2026-5j3) — JSONL file format and token tracking
- [Shipyard: Track Claude Code Usage](https://shipyard.build/blog/claude-code-track-usage/) — Session JSONL paths and structure
- [ccusage](https://github.com/ryoppippi/ccusage) — CLI tool for JSONL usage analysis
- [Block API Keys & Secrets with Claude Code Hooks](https://www.aitmpl.com/blog/security-hooks-secrets/) — Secret detection patterns
- [Managed Settings Guide](https://managed-settings.com/) — Enterprise managed-settings.json reference
- [Claude Code Enterprise Deployment](https://smartscope.blog/en/generative-ai/claude/claude-code-enterprise-deployment/) — Enterprise deployment patterns
