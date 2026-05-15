# Phase 32: Managed Hook Policy & Consulting Enforcement

## Goal

Deploy six production-ready consulting hook configurations with a 3-tier deployment strategy. Hooks are the only enforcement mechanism that operates *during* AI code generation — they provide in-flight correction that no other layer (pre-commit, CI/CD, code review) can match. This phase implements the rule: anything that would cause a client escalation if missed belongs in a hook, not in CLAUDE.md.

## Dependencies

Phase 4 complete (Claude Code addon — settings.json generation, hook deployment infrastructure, section markers, `.claude/` directory structure). Phase 13 complete (`.qsdev.yaml` config resolution, compliance levels, client profiles). Phase 15 complete (health/status reporting — SOC 2 compliance docs reference session log paths from this phase). Phase 30 complete (client profile activation — `qsdev client activate` sets `GDEV_CLIENT_PROFILE` environment variable consumed by the isolation hook).

## Phase Outputs

- 6 hook scripts in `internal/claudecode/hooks/` (embedded via `embed.FS`)
- 3-tier deployment via `qsdev enable hooks` / `qsdev disable hooks`
- `qsdev doctor` checks for Claude Code version regression range (v2.0.27-v2.0.31)
- Hook deployment matrix: managed-policy tier (credential scan, destructive prevention, SOC 2 log, client isolation), user tier (cost alerting), project tier (test enforcement)
- Append-only JSONL audit trail at `~/.qsdev/audit/sessions/<date>/<session-id>.jsonl`

---

### Unit 32.1: Destructive Command Prevention Hook

**Description:** Implement a PreToolUse hook that blocks destructive operations before they execute. The hook covers Bash tool uses targeting infrastructure deletion, database destruction, force-pushes to protected branches, and production resource removal. It reads the client profile to extend blocking patterns with client-specific production hostnames.

**Context:** Destructive commands are the highest-consequence failure mode in agentic AI development. A misdirected `terraform destroy` or `kubectl delete namespace production` can cause hours of outage and client escalation. No other layer catches these in-flight: pre-commit runs after the fact, CI/CD runs after push, and CLAUDE.md is advisory. The PreToolUse hook is the only mechanism that fires *before* the command executes and can return a non-zero exit to block it.

The hook targets the Bash tool only (not Write/Edit — those have separate protection via the credential scanner in Unit 32.2). Pattern matching on the `tool_input.command` field covers the primary attack surface. Client-profile integration extends the base patterns with client-specific hostnames and cluster names, so the hook blocks `kubectl ... --context client-prod-cluster` without the developer configuring anything.

Key constraint: `command` handler only (not `prompt` or `agent` type), target `Bash` tool, shell script implementation for portability, must complete in under 50ms.

**Desired Outcome:** `terraform destroy` (without `-target`) fired against any environment produces a hook block with a clear message explaining what was blocked and how to bypass if intentional. The hook never blocks legitimate operations like `rm -rf ./dist` (relative path, bounded scope).

**Steps:**

1. Create the hook script at `internal/claudecode/hooks/destructive-prevention.sh`:
   ```bash
   #!/usr/bin/env bash
   # gdev-managed hook: destructive-prevention
   # Blocks destructive commands before execution.
   # Type: PreToolUse | Tool: Bash | Performance budget: <50ms

   set -euo pipefail

   # Read tool input from stdin (Claude Code passes JSON to stdin for command hooks).
   INPUT=$(cat)
   COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')

   # Bail early if no command (non-Bash tool or empty input).
   if [ -z "$COMMAND" ]; then
       exit 0
   fi

   BLOCKED=""
   REASON=""

   # --- Pattern group 1: Filesystem destruction ---
   if echo "$COMMAND" | grep -qE 'rm\s+-[rf]{1,3}f?\s+/[^.]'; then
       BLOCKED=true
       REASON="Absolute path rm -rf detected. Use relative paths for local cleanup."
   fi

   # --- Pattern group 2: Database destruction ---
   if echo "$COMMAND" | grep -qiE '(DROP\s+DATABASE|DROP\s+SCHEMA\s+.*CASCADE|TRUNCATE\s+TABLE\s+.*CASCADE)'; then
       BLOCKED=true
       REASON="Destructive SQL operation detected. Confirm this is not targeting production data."
   fi

   # --- Pattern group 3: Kubernetes namespace/cluster deletion ---
   if echo "$COMMAND" | grep -qE 'kubectl\s+delete\s+namespace'; then
       BLOCKED=true
       REASON="kubectl delete namespace can destroy entire workloads. Verify target cluster and namespace."
   fi

   # --- Pattern group 4: Terraform destroy without target ---
   if echo "$COMMAND" | grep -qE 'terraform\s+destroy' && ! echo "$COMMAND" | grep -qE '\-target='; then
       BLOCKED=true
       REASON="terraform destroy without -target will destroy all managed resources. Add -target=<resource> or use a destroy plan."
   fi

   # --- Pattern group 5: Force push to protected branches ---
   if echo "$COMMAND" | grep -qE 'git\s+push\s+.*--force' && echo "$COMMAND" | grep -qE '\b(main|master|production|prod|release)\b'; then
       BLOCKED=true
       REASON="Force push to protected branch detected. Use --force-with-lease, or push to a feature branch."
   fi

   # --- Pattern group 6: Helm uninstall / helm delete ---
   if echo "$COMMAND" | grep -qE 'helm\s+(uninstall|delete)'; then
       # Check for production context indicators
       if echo "$COMMAND" | grep -qiE '(prod|production|prd)'; then
           BLOCKED=true
           REASON="helm uninstall in a production context detected. Verify target cluster."
       fi
   fi

   # --- Client-specific patterns (loaded from GDEV_CLIENT_PROFILE if set) ---
   if [ -n "${GDEV_CLIENT_PROFILE:-}" ] && [ -f "$GDEV_CLIENT_PROFILE" ]; then
       # Read production hostnames from client profile YAML.
       PROD_HOSTS=$(grep -oP '(?<=prod_hosts:\s)\S+' "$GDEV_CLIENT_PROFILE" 2>/dev/null || true)
       for HOST in $PROD_HOSTS; do
           if echo "$COMMAND" | grep -q "$HOST"; then
               BLOCKED=true
               REASON="Command references client production host ($HOST). Verify this is intentional."
               break
           fi
       done
   fi

   if [ -n "$BLOCKED" ]; then
       echo "BLOCKED by gdev destructive-prevention hook:" >&2
       echo "  $REASON" >&2
       echo "" >&2
       echo "  To proceed if intentional, add # gdev-allow-destructive to your command." >&2
       exit 1
   fi

   exit 0
   ```

2. Implement the bypass comment mechanism:
   - Any command containing `# gdev-allow-destructive` anywhere in the command string bypasses all pattern checks.
   - This is the intended escape hatch for legitimate destructive operations (e.g., tearing down a dev environment).
   - Add the bypass check at the start of the script, before any pattern groups:
     ```bash
     if echo "$COMMAND" | grep -q '# gdev-allow-destructive'; then
         exit 0
     fi
     ```

3. Define the hook configuration for `settings.json` deployment in `internal/claudecode/hooks/manifest.go`:
   ```go
   // DestructivePreventionHook is the settings.json entry for this hook.
   var DestructivePreventionHook = HookEntry{
       Matcher:   "Bash",
       HookType:  "PreToolUse",
       Command:   "~/.qsdev/hooks/destructive-prevention.sh",
       Tier:      TierManagedPolicy,
       ID:        "gdev-destructive-prevention",
   }
   ```

4. Write unit tests for pattern matching (test the script via `bash -c` invocations or extract patterns into a Go-testable function):
   - `rm -rf /var/data` → blocked.
   - `rm -rf ./dist` → allowed (relative path).
   - `terraform destroy` → blocked.
   - `terraform destroy -target=aws_instance.web` → allowed.
   - `git push origin main --force` → blocked.
   - `git push origin feature/my-branch --force` → allowed (non-protected branch).
   - `kubectl delete namespace staging` → blocked.
   - `kubectl delete pod my-pod` → allowed (pod, not namespace).
   - `DROP DATABASE testdb` → blocked.
   - `helm uninstall my-release-prod` → blocked.
   - `helm uninstall my-release-staging` → allowed.
   - Any blocked command with `# gdev-allow-destructive` comment → allowed.

**Acceptance Criteria:**
- [ ] Hook is a shell script (`command` type, not `prompt`/`agent`) targeting Bash tool PreToolUse
- [ ] Blocks: `rm -rf` on absolute paths, `DROP DATABASE`, `kubectl delete namespace`, `terraform destroy` without `-target`, `git push --force` to main/master/production, `helm uninstall` in production context
- [ ] Does not block: relative-path `rm -rf ./dir`, `terraform destroy -target=<resource>`, force-push to feature branches
- [ ] `# gdev-allow-destructive` comment in command bypasses all checks
- [ ] Client-profile production hostnames extend blocking patterns when `GDEV_CLIENT_PROFILE` is set
- [ ] Hook completes in under 50ms (no network calls, no subprocess chains)
- [ ] Block message names the violated pattern and provides a remediation path
- [ ] Deployed at managed-policy tier (`~/.claude/settings.json`)

**Research Citations:**
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — destructive command patterns, bypass comment design, client-profile integration

**Status:** Not Started

---

### Unit 32.2: Credential & Secret Scanning Hook

**Description:** Implement a PreToolUse hook that scans file write operations for credentials and secrets before they are written to disk. The hook fires on Write and Edit tool uses only, applies 12+ regex patterns covering all major cloud provider credential formats, and blocks the write with a report identifying which pattern matched.

**Context:** Credential leakage through AI-generated code is a documented threat pattern: the model generates a config file with a placeholder, the developer replaces it with a real credential, and the model then writes both to disk. The PreToolUse hook fires before the write, giving gdev the last line of defense before a secret hits the filesystem (and potentially git history). This hook complements pre-commit hooks (gitleaks, ripsecrets) but operates earlier — before the file exists on disk.

The hook scans `tool_input.content` (for Write) and `tool_input.new_string` (for Edit) only. Reading existing files that contain secrets (Read tool) is intentionally not scanned — that would produce false positives on legitimately existing credential files. The hook does not scan filenames or paths, only content.

Performance constraint: regex compilation is the expensive part. The hook pre-compiles all patterns at startup (the `#!/usr/bin/env bash` shebang means the script starts fresh on each invocation, so patterns are compiled via Python or compiled-in Perl patterns). For reliable <100ms performance, implement as a small Python script that pre-compiles all patterns.

**Desired Outcome:** Writing `AWS_SECRET_ACCESS_KEY=AKIA...` to any file produces a hook block identifying the pattern, the approximate location in the content, and a suggestion to use environment variables or a secrets manager instead.

**Steps:**

1. Create the hook script at `internal/claudecode/hooks/credential-scan.py`:
   ```python
   #!/usr/bin/env python3
   """
   gdev-managed hook: credential-scan
   Scans Write and Edit tool content for credentials before writing to disk.
   Type: PreToolUse | Tool: Write, Edit | Performance budget: <100ms
   """
   import json
   import re
   import sys

   # Pre-compiled patterns (compilation happens once at script start).
   PATTERNS = [
       (r'AKIA[0-9A-Z]{16}', 'AWS Access Key ID'),
       (r'(?i)aws.{0,20}secret.{0,20}["\']?[A-Za-z0-9/+=]{40}["\']?', 'AWS Secret Access Key'),
       (r'"type":\s*"service_account"', 'GCP Service Account JSON'),
       (r'AIza[0-9A-Za-z\-_]{35}', 'Google API Key'),
       (r'(?i)azure.{0,30}client.?secret.{0,20}["\']?[A-Za-z0-9~._-]{32,}["\']?', 'Azure Client Secret'),
       (r'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----', 'Private Key (PEM)'),
       (r'ey[A-Za-z0-9-_=]{20,}\.ey[A-Za-z0-9-_=]{20,}', 'JWT Token'),
       (r'(?i)postgres(?:ql)?://[^:@\s]+:[^@\s]+@', 'PostgreSQL connection string with password'),
       (r'(?i)mysql://[^:@\s]+:[^@\s]+@', 'MySQL connection string with password'),
       (r'(?i)mongodb(?:\+srv)?://[^:@\s]+:[^@\s]+@', 'MongoDB connection string with password'),
       (r'(?i)redis://:?[^@\s]{8,}@', 'Redis connection string with password'),
       (r'(?i)(?:api[_-]?key|apikey|api[_-]?secret)\s*[=:]\s*["\']?[A-Za-z0-9_-]{20,}["\']?', 'Generic API key/secret'),
       (r'(?i)(?:password|passwd|pwd)\s*[=:]\s*["\'](?!.*\$\{)[^"\']{8,}["\']', 'Hardcoded password'),
       (r'ghp_[0-9a-zA-Z]{36}', 'GitHub Personal Access Token'),
       (r'github_pat_[0-9a-zA-Z_]{82}', 'GitHub PAT (fine-grained)'),
       (r'xox[baprs]-[0-9a-zA-Z-]+', 'Slack Token'),
   ]

   COMPILED = [(re.compile(pattern), label) for pattern, label in PATTERNS]

   def scan_content(content: str) -> list[tuple[str, str]]:
       """Return list of (label, excerpt) for all matched patterns."""
       findings = []
       for regex, label in COMPILED:
           match = regex.search(content)
           if match:
               # Excerpt: 40 chars around the match, redacted.
               start = max(0, match.start() - 20)
               end = min(len(content), match.end() + 20)
               excerpt = content[start:end].replace('\n', ' ')
               findings.append((label, excerpt))
       return findings

   def main():
       try:
           data = json.load(sys.stdin)
       except json.JSONDecodeError:
           sys.exit(0)  # Not JSON input, pass through.

       tool_name = data.get('tool_name', '')
       tool_input = data.get('tool_input', {})

       # Only scan Write and Edit tools.
       if tool_name == 'Write':
           content = tool_input.get('content', '')
       elif tool_name == 'Edit':
           content = tool_input.get('new_string', '')
       else:
           sys.exit(0)

       if not content:
           sys.exit(0)

       findings = scan_content(content)
       if not findings:
           sys.exit(0)

       print("BLOCKED by gdev credential-scan hook:", file=sys.stderr)
       print("", file=sys.stderr)
       for label, excerpt in findings:
           print(f"  Pattern matched: {label}", file=sys.stderr)
           print(f"  Near: ...{excerpt[:60]}...", file=sys.stderr)
           print("", file=sys.stderr)
       print("  Credentials should not be written to files.", file=sys.stderr)
       print("  Use environment variables, .env files (gitignored), or a secrets manager.", file=sys.stderr)
       print("  To override if this is intentional: add # gdev-allow-credential to the file content.", file=sys.stderr)
       sys.exit(1)

   if __name__ == '__main__':
       main()
   ```

2. Implement the bypass mechanism:
   - If the content being written contains the string `# gdev-allow-credential` (or `<!-- gdev-allow-credential -->` for HTML), skip all pattern checks.
   - This allows writing example/documentation files that contain fake credential strings.

3. Define the hook configuration:
   ```go
   var CredentialScanHook = HookEntry{
       Matcher:  "Write|Edit",
       HookType: "PreToolUse",
       Command:  "python3 ~/.qsdev/hooks/credential-scan.py",
       Tier:     TierManagedPolicy,
       ID:       "gdev-credential-scan",
   }
   ```

4. Write unit tests by calling the script via subprocess:
   - AWS Access Key ID pattern → blocked.
   - GCP Service Account JSON → blocked.
   - Private key PEM header → blocked.
   - JWT token → blocked.
   - PostgreSQL URL with password → blocked.
   - GitHub PAT → blocked.
   - File read (Read tool) → not scanned (pass-through).
   - Write tool with no secrets → passes.
   - Content with `# gdev-allow-credential` → bypasses all checks.
   - `password = "test"` (too short, < 8 chars) → not matched by hardcoded password pattern.

**Acceptance Criteria:**
- [ ] Hook is a Python script (`command` type) targeting Write and Edit PreToolUse
- [ ] 12+ patterns covering: AWS access/secret keys, GCP service account JSON, Google API keys, Azure client secrets, PEM private keys, JWT tokens, PostgreSQL/MySQL/MongoDB/Redis connection strings with passwords, generic API keys, hardcoded passwords, GitHub PATs, Slack tokens
- [ ] Read tool uses are NOT scanned (no false positives from reading existing credential files)
- [ ] Block message identifies the matched pattern type and a redacted excerpt of the matching text
- [ ] `# gdev-allow-credential` comment in content bypasses all checks
- [ ] Performance: completes under 100ms (all pattern matching in-process, no subprocess chains)
- [ ] Deployed at managed-policy tier (`~/.claude/settings.json`)
- [ ] Remediation message suggests environment variables, .env files, and secrets managers

**Research Citations:**
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — credential pattern library, Write/Edit-only scanning rationale, bypass mechanism design

**Status:** Not Started

---

### Unit 32.3: Cost Alerting Hook

**Description:** Implement a PostToolUse + Stop hook combination that tracks cumulative token usage across a session and alerts at configurable cost thresholds. The Stop hook logs the session's total cost to an append-only JSONL file at `~/.qsdev/cost-log.jsonl`. This fills a gap in the Claude Code ecosystem — no existing tool provides session-level cost alerting via hooks.

**Context:** Claude Code API costs accumulate invisibly during long agentic sessions. A developer starting a "quick refactor" may end up with a $20 session without any warning. The thresholds ($5, $10, $25) are configurable in `~/.qsdev/cost-config.yaml` so developers can tune them to their preferences and billing sensitivity. The Stop hook fires at session end (when the model stops responding), making it the right place to write the final cost summary to the log.

Token counting uses two sources, in priority order: (1) tool result metadata, if Claude Code exposes token counts in the `tool_result` payload; (2) estimation from content length (input_tokens ≈ content_length / 4, output_tokens ≈ response_length / 4). The estimation is approximate and clearly labeled as such in the alert message. Cost calculation uses the current Claude Sonnet pricing (configurable in `cost-config.yaml` to stay accurate as pricing changes).

This hook is deployed at the **user tier** (personal preference), not managed-policy tier, because cost awareness is personal — one developer's $25 session is another developer's morning's work.

**Desired Outcome:** At a configurable threshold, Claude Code prints a cost alert to stderr and the developer knows to consider starting a new session. At session end, the cost is appended to `~/.qsdev/cost-log.jsonl` for monthly review.

**Steps:**

1. Create the cost-alerting PostToolUse hook at `internal/claudecode/hooks/cost-alert-post.sh`:
   ```bash
   #!/usr/bin/env bash
   # gdev-managed hook: cost-alert-post
   # Tracks token usage and alerts at configurable thresholds.
   # Type: PostToolUse | Tool: * (all tools) | Performance budget: <20ms

   set -euo pipefail

   COST_CONFIG="${HOME}/.qsdev/cost-config.yaml"
   COST_STATE="${HOME}/.qsdev/cost-state.json"

   # Read threshold from config (default: $10 per session).
   THRESHOLD_DOLLARS=$(grep -oP '(?<=alert_threshold_dollars:\s)\S+' "$COST_CONFIG" 2>/dev/null || echo "10")

   # Read tool result to extract token counts if available.
   INPUT=$(cat)
   INPUT_TOKENS=$(echo "$INPUT" | jq -r '.usage.input_tokens // 0' 2>/dev/null || echo 0)
   OUTPUT_TOKENS=$(echo "$INPUT" | jq -r '.usage.output_tokens // 0' 2>/dev/null || echo 0)

   # If no token metadata, estimate from content length.
   if [ "$INPUT_TOKENS" -eq 0 ] && [ "$OUTPUT_TOKENS" -eq 0 ]; then
       CONTENT=$(echo "$INPUT" | jq -r '.tool_result // ""' 2>/dev/null || echo "")
       OUTPUT_TOKENS=$(echo -n "$CONTENT" | wc -c | awk '{printf "%d", $1/4}')
   fi

   # Accumulate to session state file.
   SESSION_ID="${GDEV_SESSION_ID:-unknown}"
   CURRENT_INPUT=$(jq -r --arg sid "$SESSION_ID" '.[$sid].input_tokens // 0' "$COST_STATE" 2>/dev/null || echo 0)
   CURRENT_OUTPUT=$(jq -r --arg sid "$SESSION_ID" '.[$sid].output_tokens // 0' "$COST_STATE" 2>/dev/null || echo 0)

   NEW_INPUT=$((CURRENT_INPUT + INPUT_TOKENS))
   NEW_OUTPUT=$((CURRENT_OUTPUT + OUTPUT_TOKENS))

   # Update state.
   jq --arg sid "$SESSION_ID" \
      --argjson ni "$NEW_INPUT" \
      --argjson no "$NEW_OUTPUT" \
      '.[$sid] = {"input_tokens": $ni, "output_tokens": $no}' \
      "${COST_STATE:-/dev/null}" > /tmp/gdev-cost-state-tmp 2>/dev/null && \
      mv /tmp/gdev-cost-state-tmp "$COST_STATE" 2>/dev/null || true

   # Calculate estimated cost (Sonnet pricing: $3/M input, $15/M output).
   INPUT_PRICE=$(grep -oP '(?<=input_price_per_million:\s)\S+' "$COST_CONFIG" 2>/dev/null || echo "3")
   OUTPUT_PRICE=$(grep -oP '(?<=output_price_per_million:\s)\S+' "$COST_CONFIG" 2>/dev/null || echo "15")

   ESTIMATED_COST=$(awk "BEGIN {printf \"%.2f\", ($NEW_INPUT/1000000)*$INPUT_PRICE + ($NEW_OUTPUT/1000000)*$OUTPUT_PRICE}")

   # Alert if threshold exceeded (only alert once per threshold crossing).
   ALREADY_ALERTED=$(jq -r --arg sid "$SESSION_ID" '.[$sid].alerted_at // 0' "$COST_STATE" 2>/dev/null || echo 0)
   if awk "BEGIN {exit !($ESTIMATED_COST >= $THRESHOLD_DOLLARS && $ALREADY_ALERTED < $THRESHOLD_DOLLARS)}"; then
       echo "" >&2
       echo "  gdev cost alert: estimated session cost ~\$$ESTIMATED_COST (threshold: \$$THRESHOLD_DOLLARS)" >&2
       echo "  ~${NEW_INPUT} input tokens, ~${NEW_OUTPUT} output tokens (estimates may be approximate)" >&2
       echo "  Consider starting a new session to manage costs." >&2
       echo "" >&2
       # Mark threshold as alerted.
       jq --arg sid "$SESSION_ID" --argjson t "$THRESHOLD_DOLLARS" \
          '.[$sid].alerted_at = $t' "$COST_STATE" > /tmp/gdev-cost-state-tmp 2>/dev/null && \
          mv /tmp/gdev-cost-state-tmp "$COST_STATE" 2>/dev/null || true
   fi

   exit 0  # Cost alerting never blocks.
   ```

2. Create the session-end Stop hook at `internal/claudecode/hooks/cost-alert-stop.sh`:
   ```bash
   #!/usr/bin/env bash
   # gdev-managed hook: cost-alert-stop
   # Logs final session cost at session end.
   # Type: Stop | Performance budget: <50ms

   set -euo pipefail

   COST_STATE="${HOME}/.qsdev/cost-state.json"
   COST_LOG="${HOME}/.qsdev/cost-log.jsonl"
   SESSION_ID="${GDEV_SESSION_ID:-$(date +%s)}"

   # Read final token counts.
   FINAL_INPUT=$(jq -r --arg sid "$SESSION_ID" '.[$sid].input_tokens // 0' "$COST_STATE" 2>/dev/null || echo 0)
   FINAL_OUTPUT=$(jq -r --arg sid "$SESSION_ID" '.[$sid].output_tokens // 0' "$COST_STATE" 2>/dev/null || echo 0)

   # Write to cost log.
   mkdir -p "$(dirname "$COST_LOG")"
   jq -nc \
       --arg session_id "$SESSION_ID" \
       --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
       --argjson input "$FINAL_INPUT" \
       --argjson output "$FINAL_OUTPUT" \
       '{session_id: $session_id, timestamp: $timestamp, input_tokens: $input, output_tokens: $output}' \
       >> "$COST_LOG"

   # Clean up session state.
   jq --arg sid "$SESSION_ID" 'del(.[$sid])' "$COST_STATE" > /tmp/gdev-cost-cleanup 2>/dev/null && \
       mv /tmp/gdev-cost-cleanup "$COST_STATE" 2>/dev/null || true

   exit 0
   ```

3. Define cost configuration schema in `~/.qsdev/cost-config.yaml` template:
   ```yaml
   # gdev cost alerting configuration
   # Adjust thresholds and prices to match your billing preferences.

   alert_threshold_dollars: 10

   # Claude API pricing (update when Anthropic changes rates).
   # Current as of 2025: claude-sonnet-4-5
   input_price_per_million: 3
   output_price_per_million: 15
   ```
   - Generate this file on `qsdev enable hooks` if it does not exist.

4. Define the hook configurations:
   ```go
   var CostAlertPostHook = HookEntry{
       Matcher:  "",  // All tools (PostToolUse matcher is blank = all)
       HookType: "PostToolUse",
       Command:  "~/.qsdev/hooks/cost-alert-post.sh",
       Tier:     TierUser,
       ID:       "gdev-cost-alert-post",
   }

   var CostAlertStopHook = HookEntry{
       Matcher:  "",
       HookType: "Stop",
       Command:  "~/.qsdev/hooks/cost-alert-stop.sh",
       Tier:     TierUser,
       ID:       "gdev-cost-alert-stop",
   }
   ```

5. Implement `qsdev cost summary` as a bonus command in `cmd/cost.go`:
   - Reads `~/.qsdev/cost-log.jsonl`.
   - Groups sessions by calendar month.
   - Prints summary: sessions count, total tokens, estimated cost per month.
   - `--json` flag: raw JSONL passthrough.

6. Write unit tests:
   - PostToolUse hook correctly accumulates tokens across multiple calls.
   - Alert fires once when threshold is crossed (not on every subsequent call).
   - Stop hook writes a valid JSONL entry to `cost-log.jsonl`.
   - Stop hook cleans up session state.
   - Missing `cost-config.yaml` uses defaults (no error).
   - `qsdev cost summary` aggregates log entries by month.

**Acceptance Criteria:**
- [ ] PostToolUse hook accumulates token usage in a per-session state file at `~/.qsdev/cost-state.json`
- [ ] Alert fires when estimated cost exceeds configurable threshold; does not repeat for the same threshold crossing
- [ ] Alert message shows estimated cost, token counts, and suggests starting a new session
- [ ] Hook never blocks (always exits 0) — advisory only
- [ ] Stop hook appends a JSONL record to `~/.qsdev/cost-log.jsonl` at session end
- [ ] Session state cleaned up after Stop hook fires
- [ ] `~/.qsdev/cost-config.yaml` controls threshold and per-token pricing (generated on `qsdev enable hooks`)
- [ ] `qsdev cost summary` reads log and shows per-month cost breakdown
- [ ] Deployed at user tier (`~/.claude/settings.json`), not managed-policy tier

**Research Citations:**
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — cost alerting hook design, PostToolUse + Stop combination, community gap identification, token estimation approach

**Status:** Not Started

---

### Unit 32.4: SOC 2 Session Logging Hook

**Description:** Implement a metadata-only audit trail using hooks at four event types: SessionStart, PreToolUse, PostToolUse, and Stop. Each event writes a JSONL record capturing operation metadata (timestamps, tool names, file paths, durations) but never logging file contents, command output, prompt text, or conversation content. The audit trail supports SOC 2 CC6.1 (logical access) and CC7.2 (system monitoring) controls.

**Context:** Consulting engagements with SOC 2 clients require evidence of what AI tools did on client systems. The audit trail is the evidence artifact. It must be tamper-evident (append-only JSONL, no deletion API), privacy-preserving (metadata only — no customer data), and clearly scoped (per-session files in dated directories make it easy to produce evidence for a specific date range).

The four events form a complete session lifecycle: SessionStart captures the environment at session open; PreToolUse captures intent (what was about to happen); PostToolUse captures outcome (success/failure, duration); Stop captures session close. Together, they allow reconstructing a timeline of AI actions without any sensitive content.

**Code-Grounded Note:** Claude Code's `SessionStart` hook fires at the beginning of a new Claude Code conversation. There is a known bug (#10373) where `SessionStart` does not fire for new conversations in some Claude Code versions — the hook must log a fallback "first tool use" record when it detects no SessionStart was logged for the current session ID.

**Desired Outcome:** After a Claude Code session on a client project, `~/.qsdev/audit/sessions/2025-01-15/abc123.jsonl` contains a sequence of JSONL records that, together, provide a complete timeline of AI tool invocations without any file contents or conversation data. The log can be produced as evidence to a SOC 2 auditor.

**Steps:**

1. Define the JSONL record schema in `internal/claudecode/hooks/audit_schema.go`:
   ```go
   // AuditRecord is the schema for one line in the JSONL audit log.
   // All fields must be metadata only — no content, no output, no prompts.
   type AuditRecord struct {
       // Always present
       Timestamp string `json:"timestamp"` // RFC3339 UTC
       SessionID string `json:"session_id"`
       EventType string `json:"event_type"` // SessionStart|PreToolUse|PostToolUse|Stop

       // PreToolUse + PostToolUse fields
       ToolName  string   `json:"tool_name,omitempty"`
       FilePaths []string `json:"file_paths,omitempty"` // for Write/Edit/Read tools
       Command   string   `json:"command_summary,omitempty"` // first 80 chars of command, redacted for secrets

       // PostToolUse-only fields
       DurationMs int    `json:"duration_ms,omitempty"`
       Outcome    string `json:"outcome,omitempty"` // "success" | "error" | "blocked"

       // SessionStart fields
       WorkingDir  string `json:"working_dir,omitempty"`  // project root
       GdevProfile string `json:"gdev_profile,omitempty"` // active gdev profile

       // Stop fields
       TotalToolUses int `json:"total_tool_uses,omitempty"`
   }
   ```

2. Create the audit logging Python script at `internal/claudecode/hooks/audit-log.py`:
   ```python
   #!/usr/bin/env python3
   """
   gdev-managed hook: audit-log
   Writes metadata-only JSONL records for SOC 2 CC6.1/CC7.2 compliance.
   Type: SessionStart|PreToolUse|PostToolUse|Stop | Tool: * | Performance budget: <30ms
   IMPORTANT: Never logs file contents, command output, prompt text, or conversation data.
   """
   import json
   import os
   import sys
   import re
   from datetime import datetime, timezone
   from pathlib import Path

   def redact_secrets(command: str) -> str:
       """Remove potential secrets from command summaries."""
       # Redact AWS keys, tokens, passwords in command strings.
       command = re.sub(r'AKIA[0-9A-Z]{16}', '[REDACTED-AWS-KEY]', command)
       command = re.sub(r'(?i)(password|token|secret|key)\s*=\s*\S+', r'\1=[REDACTED]', command)
       return command[:80]  # Hard cap at 80 chars.

   def get_session_id() -> str:
       return os.environ.get('GDEV_SESSION_ID', 'unknown')

   def get_log_path(session_id: str) -> Path:
       date_str = datetime.now(timezone.utc).strftime('%Y-%m-%d')
       log_dir = Path.home() / '.qsdev' / 'audit' / 'sessions' / date_str
       log_dir.mkdir(parents=True, exist_ok=True)
       return log_dir / f'{session_id}.jsonl'

   def write_record(record: dict) -> None:
       session_id = record.get('session_id', 'unknown')
       log_path = get_log_path(session_id)
       # Append-only write.
       with open(log_path, 'a') as f:
           f.write(json.dumps(record, separators=(',', ':')) + '\n')

   def main():
       event_type = os.environ.get('GDEV_HOOK_EVENT', 'PreToolUse')
       session_id = get_session_id()
       timestamp = datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')

       try:
           data = json.load(sys.stdin)
       except (json.JSONDecodeError, ValueError):
           data = {}

       record = {
           'timestamp': timestamp,
           'session_id': session_id,
           'event_type': event_type,
       }

       if event_type in ('PreToolUse', 'PostToolUse'):
           tool_name = data.get('tool_name', '')
           record['tool_name'] = tool_name
           tool_input = data.get('tool_input', {})

           # Extract file paths (never contents).
           if tool_name in ('Write', 'Edit', 'Read'):
               record['file_paths'] = [tool_input.get('file_path', tool_input.get('path', ''))]
           elif tool_name == 'MultiEdit':
               record['file_paths'] = [e.get('file_path', '') for e in tool_input.get('edits', [])]

           # Extract command summary for Bash (redacted, 80 char max).
           if tool_name == 'Bash':
               record['command_summary'] = redact_secrets(tool_input.get('command', ''))

       if event_type == 'PostToolUse':
           record['outcome'] = 'success' if not data.get('is_error') else 'error'

       if event_type == 'SessionStart':
           record['working_dir'] = os.getcwd()
           record['gdev_profile'] = os.environ.get('GDEV_PROFILE', '')

       write_record(record)
       sys.exit(0)  # Audit logging never blocks.

   if __name__ == '__main__':
       main()
   ```

3. Address the SessionStart bug (#10373):
   - In the PreToolUse event handler: check if the session's JSONL file exists.
   - If it does not exist (no SessionStart was logged): write a synthetic SessionStart record with a note: `"synthetic": true, "note": "SessionStart hook did not fire (known issue #10373)"`.
   - This ensures the first record in every session log captures the session start context.

4. Define the hook configurations (four hooks for four event types):
   ```go
   var AuditLogSessionStartHook = HookEntry{
       Matcher:  "",
       HookType: "SessionStart",
       Command:  "GDEV_HOOK_EVENT=SessionStart python3 ~/.qsdev/hooks/audit-log.py",
       Tier:     TierManagedPolicy,
       ID:       "gdev-audit-session-start",
   }
   // PreToolUse, PostToolUse, Stop hooks follow same pattern with GDEV_HOOK_EVENT= set.
   ```

5. Generate SOC 2 control mapping documentation:
   - The Phase 15 health/compliance report (`qsdev status --compliance`) should reference the audit log path.
   - In the generated compliance docs for `strict` compliance level: add a section mapping the audit log to CC6.1 (logical access controls) and CC7.2 (system monitoring and alerting).
   - `qsdev audit summary` command: reads session logs for a date range, reports total sessions, tools used, file paths touched (no contents).

6. Write unit tests:
   - SessionStart event writes correct record schema.
   - PreToolUse for Write tool includes file_path but no content.
   - PreToolUse for Bash tool includes redacted command_summary (≤80 chars).
   - PreToolUse for Read tool includes file_path.
   - PostToolUse includes outcome field.
   - Stop event writes final record.
   - AWS key in a Bash command is redacted in command_summary.
   - Synthetic SessionStart written when log file absent on first PreToolUse.
   - Log is append-only JSONL (each invocation appends one line).

**Acceptance Criteria:**
- [ ] Four hooks covering all event types: SessionStart, PreToolUse, PostToolUse, Stop
- [ ] Records written as append-only JSONL at `~/.qsdev/audit/sessions/<date>/<session-id>.jsonl`
- [ ] PreToolUse records include: timestamp, session_id, tool_name, file_paths (no contents), command_summary (Bash only, redacted, ≤80 chars)
- [ ] PostToolUse records include: duration_ms, outcome (success/error/blocked)
- [ ] NEVER logs: file contents, command output, prompt text, conversation data
- [ ] AWS keys and password-like strings in command_summary are redacted before logging
- [ ] Synthetic SessionStart record written on first PreToolUse when SessionStart hook did not fire (known bug #10373 workaround)
- [ ] All four hooks always exit 0 (audit logging never blocks operations)
- [ ] Deployed at managed-policy tier (`~/.claude/settings.json`)
- [ ] SOC 2 CC6.1 / CC7.2 mapping documented in generated compliance docs (Phase 15 integration)

**Research Citations:**
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — SOC 2 audit trail design, metadata-only logging rationale, CC6.1/CC7.2 mapping
- `research-spikes/gdev-health-reporting/` — compliance reporting integration points

**Status:** Not Started

---

### Unit 32.5: Test Enforcement Hook

**Description:** Implement a Stop hook that runs linting and type-checking before Claude Code reports task completion. The hook executes `devenv task lint` and `devenv task test` (if defined in the project) at session end, reporting failures as advisory warnings without blocking the stop event. This is the project-tier hook — it travels with the client repo, not the consultant.

**Context:** The agentic workflow state-of-art research identified external verification as the single highest-impact quality multiplier: TDFlow's test-driven feedback loop achieves 88.8% pass rate vs 49% for generation-only approaches. The Stop hook implements the verification side of this loop at the session level — Claude generates code, then the hook verifies it compiles, lints, and passes tests before the session ends. The developer sees the test results and can continue the session to fix failures or accept the output.

The hook is deliberately advisory (warns, does not block) because blocking the Stop event would prevent Claude Code from producing a response at all. The warning appears in Claude Code's output, where the developer can read it and decide to continue. This design intentionally avoids the failure mode where a noisy test suite means Claude Code can never complete a session.

The hook only fires when devenv tasks are defined — it checks for `devenv task lint` and `devenv task test` in the project's `devenv.nix` before running them. Projects without these tasks see a no-op.

**Desired Outcome:** At the end of a Claude Code session, the developer sees a summary of lint and test results. If lint fails, they know before they consider the task done. If tests pass, they have a confidence signal. The hook adds under 60 seconds of wall time for most projects.

**Steps:**

1. Create the hook script at `internal/claudecode/hooks/test-enforcement.sh`:
   ```bash
   #!/usr/bin/env bash
   # gdev-managed hook: test-enforcement
   # Runs lint and test at session end. Advisory only — never blocks.
   # Type: Stop | Performance budget: project test suite runtime (typically <60s)
   # Deployment tier: project (.claude/settings.json)

   set -uo pipefail

   # Check if devenv task lint is defined.
   HAS_LINT=false
   HAS_TEST=false

   if devenv tasks 2>/dev/null | grep -q '^lint\b'; then
       HAS_LINT=true
   fi

   if devenv tasks 2>/dev/null | grep -q '^test\b'; then
       HAS_TEST=true
   fi

   if [ "$HAS_LINT" = false ] && [ "$HAS_TEST" = false ]; then
       # No devenv tasks defined — no-op.
       exit 0
   fi

   echo "" >&2
   echo "  gdev test-enforcement: running verification before session end..." >&2

   LINT_STATUS="skipped"
   TEST_STATUS="skipped"

   if [ "$HAS_LINT" = true ]; then
       echo "  Running: devenv task lint" >&2
       if devenv task lint 2>&1 | tail -5 >&2; then
           LINT_STATUS="passed"
       else
           LINT_STATUS="failed"
       fi
   fi

   if [ "$HAS_TEST" = true ]; then
       echo "  Running: devenv task test" >&2
       if devenv task test 2>&1 | tail -10 >&2; then
           TEST_STATUS="passed"
       else
           TEST_STATUS="failed"
       fi
   fi

   echo "" >&2
   echo "  Lint: $LINT_STATUS  |  Tests: $TEST_STATUS" >&2

   if [ "$LINT_STATUS" = "failed" ] || [ "$TEST_STATUS" = "failed" ]; then
       echo "  ⚠ Verification failures detected. Consider continuing the session to fix them." >&2
       echo "" >&2
   else
       echo "  ✓ All checks passed." >&2
       echo "" >&2
   fi

   # Advisory only — always exit 0.
   exit 0
   ```

2. Define the hook configuration:
   ```go
   var TestEnforcementHook = HookEntry{
       Matcher:  "",
       HookType: "Stop",
       Command:  "~/.qsdev/hooks/test-enforcement.sh",
       Tier:     TierProject,
       ID:       "gdev-test-enforcement",
   }
   ```
   - Project-tier hooks are written to `.claude/settings.json` (not `~/.claude/settings.json`).
   - `qsdev enable hooks` writes this hook to the project `.claude/settings.json` using section markers from Phase 4 shared-file surgery.

3. Implement task detection:
   - `devenv tasks` lists all defined tasks; parse for `lint` and `test` entries.
   - If `devenv` binary is not in `$PATH`: no-op (the hook is in a project repo, devenv may not be installed globally in the CI environment where the hook fires).
   - Timeout protection: if `devenv task lint` runs longer than 120 seconds, kill it and report "timed out" in the summary.

4. Handle the timeout:
   ```bash
   if [ "$HAS_LINT" = true ]; then
       echo "  Running: devenv task lint" >&2
       if timeout 120 devenv task lint 2>&1 | tail -5 >&2; then
           LINT_STATUS="passed"
       else
           EXIT_CODE=$?
           if [ $EXIT_CODE -eq 124 ]; then
               LINT_STATUS="timed-out"
           else
               LINT_STATUS="failed"
           fi
       fi
   fi
   ```

5. Document the grounding in agentic research:
   - In the generated CLAUDE.md section for this hook, include: "This hook implements the external verification principle from agentic workflow research: TDFlow test-driven feedback achieves 88.8% pass rate vs 49% for generation-only. ([source: research-spikes/agentic-workflow-state-of-art/research.md])"

6. Write unit tests:
   - Project with no devenv tasks → hook exits 0 immediately (no-op).
   - Project with lint task → lint runs, status reported.
   - Project with both lint and test tasks → both run, aggregate status reported.
   - Lint failure → warning printed, exit 0 (advisory).
   - `devenv` not in PATH → hook exits 0 immediately (no-op).
   - Lint runs > 120 seconds → killed, "timed-out" status, exit 0.

**Acceptance Criteria:**
- [ ] Stop hook runs `devenv task lint` and `devenv task test` if those tasks are defined
- [ ] If neither task is defined, hook is a no-op (exits 0 immediately)
- [ ] If `devenv` binary absent, hook is a no-op (exits 0 immediately)
- [ ] Lint/test failures produce advisory warning in Claude Code output; hook still exits 0
- [ ] Tasks timeout after 120 seconds and report "timed-out" status
- [ ] Deployed at project tier (`.claude/settings.json`, travels with repo)
- [ ] CLAUDE.md section for this hook cites the agentic verification research finding (88.8% vs 49%)
- [ ] Separate lint and test status reported (not aggregated into one pass/fail)

**Research Citations:**
- `research-spikes/agentic-workflow-state-of-art/research.md` — external verification as quality multiplier, TDFlow 88.8% vs 49%, test-driven feedback loop design
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — Stop hook advisory pattern, project-tier deployment

**Status:** Not Started

---

### Unit 32.6: Client Isolation Verification Hook

**Description:** Implement a SessionStart hook (with PreToolUse fallback) that verifies environment isolation before any AI operations begin. The hook checks that the active AWS profile, kubeconfig, and git user match the expected client profile. Mismatches produce warnings to prevent accidental cross-client operations.

**Context:** Consulting developers routinely switch between client projects, each with its own cloud credentials and git identity. The most common cross-client contamination pattern: developer finishes work for Client A, opens a new Claude Code session for Client B without switching credentials, and AI tool uses the wrong AWS account. This is a compliance failure that is difficult to detect after the fact. The isolation hook catches it at session open.

The hook is warning-only (not blocking) because the developer may intentionally be working across clients in a migration or handoff scenario. Blocking would be too aggressive; warning is sufficient. The hook only fires when a client profile is active (i.e., `GDEV_CLIENT_PROFILE` is set by Phase 30's `qsdev client activate`). Projects without active client profiles skip all checks.

The `SessionStart` bug (#10373, documented) means this hook may not fire for all sessions. The PreToolUse fallback writes a "first tool use" isolation check that fires on the first Bash/Write/Edit use, ensuring at minimum one isolation check per session.

**Desired Outcome:** When a developer opens a Claude Code session in a Client B project while still authenticated to Client A's AWS account, the session start prints a warning identifying the mismatch and suggesting `qsdev client activate ClientB`.

**Steps:**

1. Create the hook script at `internal/claudecode/hooks/client-isolation.sh`:
   ```bash
   #!/usr/bin/env bash
   # gdev-managed hook: client-isolation
   # Verifies cloud credentials and git identity match the active client profile.
   # Type: SessionStart (with PreToolUse fallback) | Performance budget: <200ms

   set -uo pipefail

   # No-op if no client profile is active.
   if [ -z "${GDEV_CLIENT_PROFILE:-}" ]; then
       exit 0
   fi

   if [ ! -f "$GDEV_CLIENT_PROFILE" ]; then
       exit 0
   fi

   WARNINGS=()

   # --- Check 1: AWS profile ---
   EXPECTED_AWS_PROFILE=$(grep -oP '(?<=aws_profile:\s)\S+' "$GDEV_CLIENT_PROFILE" 2>/dev/null || true)
   if [ -n "$EXPECTED_AWS_PROFILE" ]; then
       CURRENT_AWS_PROFILE="${AWS_PROFILE:-default}"
       if [ "$CURRENT_AWS_PROFILE" != "$EXPECTED_AWS_PROFILE" ]; then
           WARNINGS+=("AWS_PROFILE mismatch: expected '$EXPECTED_AWS_PROFILE', got '$CURRENT_AWS_PROFILE'")
       fi
   fi

   # --- Check 2: kubeconfig isolation ---
   EXPECTED_KUBECONFIG_DIR=$(grep -oP '(?<=kubeconfig_dir:\s)\S+' "$GDEV_CLIENT_PROFILE" 2>/dev/null || true)
   if [ -n "$EXPECTED_KUBECONFIG_DIR" ]; then
       CURRENT_KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"
       if [[ "$CURRENT_KUBECONFIG" == "${HOME}/.kube/config" ]] || [[ "$CURRENT_KUBECONFIG" != *"$EXPECTED_KUBECONFIG_DIR"* ]]; then
           WARNINGS+=("KUBECONFIG is not project-local: '$CURRENT_KUBECONFIG' (expected dir: '$EXPECTED_KUBECONFIG_DIR')")
       fi
   fi

   # --- Check 3: git user identity ---
   EXPECTED_GIT_EMAIL=$(grep -oP '(?<=git_email:\s)\S+' "$GDEV_CLIENT_PROFILE" 2>/dev/null || true)
   if [ -n "$EXPECTED_GIT_EMAIL" ]; then
       CURRENT_GIT_EMAIL=$(git config user.email 2>/dev/null || true)
       if [ "$CURRENT_GIT_EMAIL" != "$EXPECTED_GIT_EMAIL" ]; then
           WARNINGS+=("git user.email mismatch: expected '$EXPECTED_GIT_EMAIL', got '$CURRENT_GIT_EMAIL'")
       fi
   fi

   # Report warnings (never block).
   if [ ${#WARNINGS[@]} -gt 0 ]; then
       CLIENT_NAME=$(grep -oP '(?<=name:\s).*' "$GDEV_CLIENT_PROFILE" | head -1 || echo "unknown")
       echo "" >&2
       echo "  gdev client-isolation warning (client: $CLIENT_NAME):" >&2
       for W in "${WARNINGS[@]}"; do
           echo "    ⚠ $W" >&2
       done
       echo "" >&2
       echo "  Fix: gdev client activate $CLIENT_NAME" >&2
       echo "  To suppress: GDEV_SKIP_ISOLATION_CHECK=1 before starting Claude Code." >&2
       echo "" >&2
   fi

   exit 0  # Isolation check never blocks.
   ```

2. Implement the PreToolUse fallback:
   - Create a sibling script `client-isolation-pre.sh`.
   - On first invocation in a session (tracked via a session state flag file at `~/.qsdev/isolation-checked/<session-id>`), run the same checks as the SessionStart hook.
   - Subsequent PreToolUse invocations in the same session skip the check (the flag file exists).
   - The flag file is written whether or not any warnings were produced.
   - This fallback compensates for the SessionStart bug (#10373).

3. Define the hook configurations:
   ```go
   var ClientIsolationSessionStartHook = HookEntry{
       Matcher:  "",
       HookType: "SessionStart",
       Command:  "~/.qsdev/hooks/client-isolation.sh",
       Tier:     TierManagedPolicy,
       ID:       "gdev-client-isolation-session-start",
   }

   var ClientIsolationPreToolUseHook = HookEntry{
       Matcher:  "Bash|Write|Edit",
       HookType: "PreToolUse",
       Command:  "~/.qsdev/hooks/client-isolation-pre.sh",
       Tier:     TierManagedPolicy,
       ID:       "gdev-client-isolation-pre",
   }
   ```

4. Document the SessionStart bug:
   - In the generated CLAUDE.md hook documentation: add a note about #10373 and explain the PreToolUse fallback.
   - In `qsdev doctor` output: check if Claude Code version is in the affected range and flag it.

5. Implement `GDEV_SKIP_ISOLATION_CHECK=1` bypass:
   - Both the SessionStart and PreToolUse scripts check this env var at the start.
   - If set (any non-empty value): exit 0 immediately.

6. Write unit tests:
   - No `GDEV_CLIENT_PROFILE` → no-op.
   - AWS profile mismatch → warning printed, exit 0.
   - AWS profile matches → no warning.
   - kubeconfig is `~/.kube/config` when project-local expected → warning.
   - git email mismatch → warning.
   - All checks pass → no output, exit 0.
   - `GDEV_SKIP_ISOLATION_CHECK=1` → immediate no-op.
   - PreToolUse fallback: first invocation runs checks, second skips (flag file present).

**Acceptance Criteria:**
- [ ] SessionStart hook verifies AWS_PROFILE, KUBECONFIG, and git user.email against active client profile
- [ ] No-op when `GDEV_CLIENT_PROFILE` is not set (projects without active client profile unaffected)
- [ ] All checks are warnings (advisory) — hook always exits 0
- [ ] PreToolUse fallback fires on first Bash/Write/Edit use when SessionStart did not fire (#10373 workaround)
- [ ] PreToolUse fallback runs at most once per session (flag file prevents repeated checks)
- [ ] `GDEV_SKIP_ISOLATION_CHECK=1` suppresses all checks
- [ ] Warning message names the mismatched check and suggests `qsdev client activate <name>`
- [ ] SessionStart bug (#10373) documented in generated CLAUDE.md hook section
- [ ] Deployed at managed-policy tier (`~/.claude/settings.json`)

**Research Citations:**
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — client isolation hook design, SessionStart bug #10373, PreToolUse fallback pattern

**Status:** Not Started

---

### Unit 32.7: Hook Deployment & Claude Code Version Pinning

**Description:** Implement the 3-tier hook deployment system — managed-policy, user-level, and project-level — via `qsdev enable hooks` and `qsdev disable hooks`. Implement Claude Code version regression detection in `qsdev doctor`: warn if the installed Claude Code version falls in the documented regression range (v2.0.27-v2.0.31) where hooks had documented breakage.

**Context:** The 3-tier deployment model ensures each hook reaches the right scope without redundancy. Managed-policy hooks (credential scanning, destructive prevention, SOC 2 logging, client isolation) must apply across all client projects and are therefore written to `~/.claude/settings.json`. User-level hooks (cost alerting) are also in `~/.claude/settings.json` but in a separate section, so they can be toggled independently. Project-level hooks (test enforcement) go in `.claude/settings.json` in the repo — they travel with the project and apply to all developers who run `qsdev enable hooks` in that repo.

The Phase 4 shared-file surgery infrastructure (section markers, safe append/remove) is the foundation for all hook deployment. `qsdev enable hooks` adds gdev-managed sections; `qsdev disable hooks` removes only gdev-managed sections, leaving any user-added hooks untouched.

Claude Code version pinning is a production requirement: versions v2.0.27-v2.0.31 had documented hook breakage (hooks silently not firing), affecting all hook-dependent security enforcement. `qsdev doctor` checks the installed version and warns before developers discover the problem through a security incident.

**Desired Outcome:** `qsdev enable hooks` deploys all applicable hooks to the correct settings.json files based on the current profile and project. `qsdev disable hooks` cleanly removes them. `qsdev doctor` catches the broken Claude Code version range before it causes a missed security event.

**Steps:**

1. Define the hook deployment tier types and entry schema in `internal/claudecode/hooks/manifest.go`:
   ```go
   type DeploymentTier int
   const (
       TierManagedPolicy DeploymentTier = iota // ~/.claude/settings.json [gdev-managed-policy]
       TierUser                                 // ~/.claude/settings.json [gdev-user]
       TierProject                              // .claude/settings.json  [gdev-project]
   )

   type HookEntry struct {
       // ID is the unique identifier for this hook (used for enable/disable tracking).
       ID string

       // Matcher is the tool name or pattern (empty = all tools).
       Matcher string

       // HookType is PreToolUse, PostToolUse, SessionStart, or Stop.
       HookType string

       // Command is the shell command to run.
       // Uses ~ for home directory (expanded at deployment time to absolute path).
       Command string

       // Tier determines which settings.json file receives this hook.
       Tier DeploymentTier

       // Description is shown in `qsdev status hooks`.
       Description string
   }

   // AllHooks is the canonical list of all gdev-managed hooks.
   var AllHooks = []HookEntry{
       // Managed-policy tier
       DestructivePreventionHook,
       CredentialScanHook,
       AuditLogSessionStartHook,
       AuditLogPreToolUseHook,
       AuditLogPostToolUseHook,
       AuditLogStopHook,
       ClientIsolationSessionStartHook,
       ClientIsolationPreToolUseHook,
       // User tier
       CostAlertPostHook,
       CostAlertStopHook,
       // Project tier
       TestEnforcementHook,
   }
   ```

2. Implement hook script installation in `internal/claudecode/hooks/install.go`:
   ```go
   // InstallHookScripts copies all hook scripts from the embedded FS to ~/.qsdev/hooks/.
   // Creates the directory if absent. Sets executable permissions on scripts.
   func InstallHookScripts(fs embed.FS, homeDir string) error {
       hooksDir := filepath.Join(homeDir, ".qsdev", "hooks")
       if err := os.MkdirAll(hooksDir, 0755); err != nil {
           return fmt.Errorf("cannot create hooks directory: %w", err)
       }

       scripts := []string{
           "destructive-prevention.sh",
           "credential-scan.py",
           "cost-alert-post.sh",
           "cost-alert-stop.sh",
           "audit-log.py",
           "client-isolation.sh",
           "client-isolation-pre.sh",
           "test-enforcement.sh",
       }

       for _, name := range scripts {
           content, err := fs.ReadFile("hooks/" + name)
           if err != nil {
               return fmt.Errorf("embedded hook %s not found: %w", name, err)
           }
           dest := filepath.Join(hooksDir, name)
           if err := os.WriteFile(dest, content, 0755); err != nil {
               return fmt.Errorf("cannot write hook %s: %w", name, err)
           }
       }
       return nil
   }
   ```

3. Implement `qsdev enable hooks` in `cmd/enable.go`:
   ```go
   func enableHooks(cmd *cobra.Command, args []string) error {
       // Step 1: Install hook scripts to ~/.qsdev/hooks/
       if err := hooks.InstallHookScripts(embeddedFS, homeDir); err != nil {
           return err
       }

       // Step 2: Deploy managed-policy + user hooks to ~/.claude/settings.json
       globalSettings := filepath.Join(homeDir, ".claude", "settings.json")
       if err := deployHooksToSettings(globalSettings, TierManagedPolicy, TierUser); err != nil {
           return err
       }

       // Step 3: Deploy project hooks to .claude/settings.json (if in a gdev project)
       if isGdevProject(projectRoot) {
           projectSettings := filepath.Join(projectRoot, ".claude", "settings.json")
           if err := deployHooksToSettings(projectSettings, TierProject); err != nil {
               return err
           }
       }

       // Step 4: Generate cost config if absent.
       costConfig := filepath.Join(homeDir, ".qsdev", "cost-config.yaml")
       if _, err := os.Stat(costConfig); os.IsNotExist(err) {
           if err := writeCostConfigTemplate(costConfig); err != nil {
               return err
           }
           fmt.Printf("Generated cost config: %s\n", costConfig)
       }

       fmt.Println("✓ Hooks enabled:")
       for _, h := range AllHooks {
           fmt.Printf("  [%s] %s (%s)\n", tierName(h.Tier), h.ID, h.HookType)
       }
       return nil
   }
   ```

4. Implement `deployHooksToSettings(settingsPath string, tiers ...DeploymentTier) error`:
   - Read existing `settings.json` (or start from `{}`).
   - Use Phase 4's section marker infrastructure to write hooks within `[gdev-managed-policy]`, `[gdev-user]`, or `[gdev-project]` sections.
   - Map each `HookEntry` to the `hooks` array format expected by Claude Code:
     ```json
     {
       "hooks": {
         "PreToolUse": [
           {
             "matcher": "Bash",
             "hooks": [{"type": "command", "command": "/home/user/.qsdev/hooks/destructive-prevention.sh"}]
           }
         ]
       }
     }
     ```
   - Expand `~` to absolute home directory path.
   - If a hook with the same ID is already present in the section: replace (idempotent).

5. Implement `qsdev disable hooks`:
   - Remove the `[gdev-managed-policy]` and `[gdev-user]` sections from `~/.claude/settings.json` using Phase 4's section removal.
   - Remove the `[gdev-project]` section from `.claude/settings.json` if present.
   - Do NOT remove hooks outside gdev-managed sections (user-added hooks are untouched).
   - Print a summary of what was removed.

6. Implement Claude Code version regression detection in `qsdev doctor`:
   ```go
   // checkClaudeCodeVersion checks for the documented hook regression range.
   func checkClaudeCodeVersion() *DoctorFinding {
       version, err := detectClaudeCodeVersion()
       if err != nil {
           return &DoctorFinding{
               Name:        "claude_code_version",
               Status:      "skip",
               Message:     "Claude Code not installed or version not detectable",
           }
       }

       // v2.0.27 through v2.0.31 have documented hook breakage.
       // Hooks silently don't fire in this range.
       regressionMin, _ := semver.Parse("2.0.27")
       regressionMax, _ := semver.Parse("2.0.31")
       current, err := semver.Parse(version)
       if err != nil {
           return nil // cannot parse, skip
       }

       if current.GTE(regressionMin) && current.LTE(regressionMax) {
           return &DoctorFinding{
               Name:    "claude_code_hook_regression",
               Status:  "fail",
               Severity: SeverityHigh,
               Message: fmt.Sprintf(
                   "Claude Code %s has documented hook regression (hooks may not fire).\n"+
                   "Affected range: v2.0.27-v2.0.31.\n"+
                   "Upgrade with: npm update -g @anthropic-ai/claude-code",
                   version,
               ),
           }
       }

       return &DoctorFinding{
           Name:    "claude_code_hook_regression",
           Status:  "pass",
           Message: fmt.Sprintf("Claude Code %s is outside the hook regression range.", version),
       }
   }
   ```

7. Implement `qsdev status hooks` (or `qsdev enable hooks --status`):
   - List all gdev-managed hooks across all three tiers.
   - For each hook: show ID, type, tier, and whether the hook script exists at the expected path.
   - Flags missing scripts (hook registered in settings.json but script absent — common after gdev upgrades).

8. Write integration tests:
   - `qsdev enable hooks` creates `~/.qsdev/hooks/` with executable scripts.
   - `qsdev enable hooks` adds correct sections to `~/.claude/settings.json`.
   - `qsdev enable hooks` is idempotent (re-run does not duplicate hooks).
   - `qsdev disable hooks` removes gdev sections but not user-added hooks.
   - `qsdev doctor` detects Claude Code in regression range.
   - `qsdev doctor` passes when Claude Code is outside regression range.
   - `qsdev status hooks` shows correct installed/missing status.

**Acceptance Criteria:**
- [ ] `qsdev enable hooks` installs all hook scripts to `~/.qsdev/hooks/` with executable permissions
- [ ] Managed-policy hooks deployed to `~/.claude/settings.json` in `[gdev-managed-policy]` section
- [ ] User hooks deployed to `~/.claude/settings.json` in `[gdev-user]` section
- [ ] Project hooks deployed to `.claude/settings.json` in `[gdev-project]` section
- [ ] `qsdev enable hooks` is idempotent (re-run replaces, does not duplicate)
- [ ] `qsdev disable hooks` removes only gdev-managed sections; user-added hooks are preserved
- [ ] `qsdev doctor` warns when Claude Code version is in documented regression range v2.0.27-v2.0.31
- [ ] `qsdev doctor` check identifies the upgrade command (`npm update -g @anthropic-ai/claude-code`)
- [ ] `qsdev status hooks` shows per-hook installed/missing status across all three tiers
- [ ] Hook script paths use absolute paths (not `~`) in deployed `settings.json`
- [ ] `~/.qsdev/cost-config.yaml` generated with defaults on first `qsdev enable hooks`

**Research Citations:**
- `research-spikes/claude-code-hooks-in-practice/consulting-hooks-research.md` — 3-tier deployment model, Claude Code version regression range (v2.0.27-v2.0.31), hooks format in settings.json, section marker approach
- `phases/04-claude-code-addon.md` — shared-file surgery infrastructure, section markers, `~/.claude/settings.json` modification patterns

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Hook Deployment Tiers

| Tier | settings.json location | Section marker | Hooks |
|------|----------------------|----------------|-------|
| Managed policy | `~/.claude/settings.json` | `[gdev-managed-policy]` | Destructive prevention, credential scan, SOC 2 audit log, client isolation |
| User | `~/.claude/settings.json` | `[gdev-user]` | Cost alerting |
| Project | `.claude/settings.json` | `[gdev-project]` | Test enforcement |

### Hook Script Locations

All hook scripts are embedded in the gdev binary via `embed.FS` at `internal/claudecode/hooks/`, and installed to `~/.qsdev/hooks/` on `qsdev enable hooks`. The `~/.qsdev/hooks/` directory is the runtime execution location.

### Claude Code Hook Format (settings.json)

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "/home/user/.qsdev/hooks/destructive-prevention.sh"}
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "/home/user/.qsdev/hooks/cost-alert-post.sh"}
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "/home/user/.qsdev/hooks/cost-alert-stop.sh"}
        ]
      }
    ]
  }
}
```

### Hook Type Constraints

Only `command` type is used across all hooks in this phase. `prompt` and `agent` hook types have 1-60s latency due to model invocation and are explicitly avoided. All hooks must complete within their performance budgets without network calls or model invocations.

### Known Issues

| Issue | Impact | Mitigation |
|-------|--------|-----------|
| SessionStart bug (#10373) | SessionStart hooks do not fire for new conversations in some versions | PreToolUse fallback in client-isolation and audit-log hooks; synthetic SessionStart record written on first PreToolUse |
| Claude Code version regression v2.0.27-v2.0.31 | Hooks silently do not fire | `qsdev doctor` warns and provides upgrade command |

### New Packages

| Package | Path | Purpose |
|---------|------|---------|
| `hooks` | `internal/claudecode/hooks/` | Hook scripts (embedded), manifest, install, deploy |

### New Commands

| Command | Notes |
|---------|-------|
| `qsdev enable hooks` | Deploys all hooks across 3 tiers |
| `qsdev disable hooks` | Removes gdev-managed hooks, preserves user hooks |
| `qsdev status hooks` | Shows per-hook installed/missing status |
| `qsdev cost summary` | Reads `~/.qsdev/cost-log.jsonl`, shows monthly breakdown |
| `qsdev audit summary` | Reads session audit logs, produces timeline summary |
| `qsdev doctor` | Extended with Claude Code version regression check |

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] All 6 hook scripts embedded in the gdev binary and installable to `~/.qsdev/hooks/`
- [ ] Destructive prevention hook blocks all 6 documented patterns and passes all negative control tests
- [ ] Credential scanner hook blocks all 12+ patterns on Write/Edit, never fires on Read
- [ ] Cost alerting hook accumulates tokens, alerts at threshold, logs to JSONL at session end
- [ ] SOC 2 audit log writes metadata-only JSONL across all 4 event types; never logs content
- [ ] Test enforcement hook runs devenv tasks at session end, warns on failure, never blocks
- [ ] Client isolation hook warns on credential/identity mismatch; no-op when no client profile active
- [ ] `qsdev enable hooks` deploys correct hooks to correct tier settings.json files
- [ ] `qsdev disable hooks` removes only gdev-managed sections
- [ ] `qsdev doctor` detects and warns on Claude Code v2.0.27-v2.0.31 regression range
- [ ] All hooks are `command` type (no `prompt`/`agent`); none block legitimate operations
- [ ] `# gdev-allow-destructive` and `# gdev-allow-credential` bypass comments documented in CLAUDE.md
- [ ] Known SessionStart bug (#10373) documented and mitigated via PreToolUse fallback
