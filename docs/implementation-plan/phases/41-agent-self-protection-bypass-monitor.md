# Phase 41: Agent Self-Protection -- Bypass Policy & Monitor Mode

## Goal

Deploy the interactive bypass system, per-rule monitor mode, Tier 2 security rules, session state management, and the PreToolUse hook for Write/Edit tools. This phase enables real-world calibration of self-protection rules before full enforcement and provides the developer escape hatches needed for legitimate operations that Tier 1 rules would otherwise block (editing devenv.nix, modifying .pre-commit-config.yaml, registering MCP servers). The bypass system is designed to be agent-resistant: the `gdev hook bypass-next` CLI command is itself blocked by Tier 1 rule SP-09, forcing the developer to run it in their own terminal outside Claude Code's control.

## Dependencies

Phase 40 complete (rule definition schema, path canonicalization, fail-closed harness, Tier 1 rules, `gdev-hook pre-bash` binary). Phase 32 complete (hook deployment infrastructure, `gdev enable hooks`). Phase 15 (health/status reporting) for `gdev status --self-protection` integration.

## Phase Outputs

- Three-tier bypass system with token generation, consumption, and audit logging
- Per-rule monitor mode with `enforce_always` exception for Tier 1 rules
- 11 Tier 2 interactive-bypass rules (SP-15, SP-16, CFG-01 through CFG-07, MCP-05, MCP-06)
- 3 SessionStart integrity checks (INT-01, INT-02, INT-03)
- Session state management with auto-cleanup
- `gdev-hook pre-write` subcommand for Write/Edit tool rules
- `gdev hook monitor/enforce/off/audit/bypass-next` CLI commands
- Audit trail integration for verdict records and bypass events

---

### Unit 41.1: Three-Tier Bypass System

**Description:** Implement the three-tier bypass architecture that provides escape hatches proportional to risk. Tier 1 (absolute deny) has no bypass mechanism -- the developer must edit files outside Claude Code. Tier 2 (interactive bypass) requires the developer to run `gdev hook bypass-next <rule-id>` in their own terminal, which creates a single-use token consumed by the next hook invocation. Tier 3 (standard bypass) accepts magic comments like `# gdev-allow-destructive` in the command or file content. Tier 2's agent resistance comes from chained protection: SP-09 (a Tier 1 rule) blocks `gdev hook bypass-next` when invoked through Claude Code's Bash tool, forcing the developer to act outside the agent's reach.

**Context:** The escape hatch research analyzed 6 bypass mechanism types (magic comments, CLI command, environment variable, interactive prompt, time-limited token, out-of-band channel) for human-accessibility versus agent-exploitability. The key finding was that Claude Code's `permissionDecision: "ask"` has two critical bugs (#39344 and #52822), making the JSON ask verdict unreliable. The recommended alternative converts all "ask" rules to Tier 2 interactive-bypass denials that block with exit code 2 and instruct the developer to use the separate-terminal workflow.

The chained protection pattern works as follows: (1) Agent attempts a protected operation, hook blocks with exit 2 and a message saying "run `gdev hook bypass-next <rule>` in your terminal"; (2) Developer switches to their terminal and runs the bypass command; (3) Developer returns to Claude Code and tells the agent to retry; (4) On retry, the hook finds the valid token, consumes it, allows the operation, and logs the bypass. The agent cannot execute step 2 because SP-09 blocks it, and the developer's terminal is outside the agent's control.

**Desired Outcome:** When an agent tries to edit devenv.nix to add `sandbox = false`, the hook blocks with: "BLOCKED by gdev self-protection [CFG-01]: Edit to devenv.nix weakens security settings. Run `gdev hook bypass-next CFG-01 --reason 'description'` in your terminal to authorize." The developer runs the bypass command in their terminal, then tells the agent to retry. The retry succeeds. The bypass is logged to the audit trail.

**Steps:**

1. Implement the bypass token system in `internal/selfprotect/bypass.go`:
   ```go
   package selfprotect

   import (
       "crypto/rand"
       "encoding/hex"
       "encoding/json"
       "os"
       "path/filepath"
       "time"
   )

   const (
       DefaultTokenTTL = 5 * time.Minute
       TokenDir        = ".qsdev/self-protection/bypass-tokens"
   )

   // BypassToken is a single-use authorization token for a Tier 2 rule.
   type BypassToken struct {
       TokenID    string    `json:"token_id"`
       RuleID     string    `json:"rule_id"`
       Reason     string    `json:"reason"`
       CreatedAt  time.Time `json:"created_at"`
       ExpiresAt  time.Time `json:"expires_at"`
       CreatorPID int       `json:"creator_pid"`
       Consumed   bool      `json:"consumed"`
   }

   // CreateBypassToken generates a single-use token for the specified rule.
   func CreateBypassToken(ruleID, reason string, ttl time.Duration) (*BypassToken, error) {
       if ttl == 0 {
           ttl = DefaultTokenTTL
       }
       tokenBytes := make([]byte, 8)
       if _, err := rand.Read(tokenBytes); err != nil {
           return nil, err
       }
       token := &BypassToken{
           TokenID:    "tok_" + hex.EncodeToString(tokenBytes),
           RuleID:     ruleID,
           Reason:     reason,
           CreatedAt:  time.Now().UTC(),
           ExpiresAt:  time.Now().UTC().Add(ttl),
           CreatorPID: os.Getpid(),
       }

       home, _ := os.UserHomeDir()
       dir := filepath.Join(home, TokenDir)
       os.MkdirAll(dir, 0700)

       path := filepath.Join(dir, ruleID+"-"+token.TokenID+".json")
       data, _ := json.MarshalIndent(token, "", "  ")
       return token, os.WriteFile(path, data, 0600)
   }

   // ConsumeBypassToken checks for and consumes a valid token for the rule.
   // Returns the token if valid, nil if no valid token exists.
   func ConsumeBypassToken(ruleID string) *BypassToken {
       home, _ := os.UserHomeDir()
       dir := filepath.Join(home, TokenDir)
       entries, err := os.ReadDir(dir)
       if err != nil {
           return nil
       }
       for _, entry := range entries {
           if !strings.HasPrefix(entry.Name(), ruleID+"-") {
               continue
           }
           path := filepath.Join(dir, entry.Name())
           data, err := os.ReadFile(path)
           if err != nil {
               continue
           }
           var token BypassToken
           if err := json.Unmarshal(data, &token); err != nil {
               continue
           }
           if token.Consumed || time.Now().After(token.ExpiresAt) {
               os.Remove(path) // clean up expired/consumed tokens
               continue
           }
           // Valid token found -- consume it
           token.Consumed = true
           data, _ = json.MarshalIndent(token, "", "  ")
           os.WriteFile(path, data, 0600)
           // Schedule cleanup
           defer os.Remove(path)
           return &token
       }
       return nil
   }
   ```

2. Implement the `gdev hook bypass-next` CLI command in `cmd/hook_bypass.go`:
   ```go
   func hookBypassNext(cmd *cobra.Command, args []string) error {
       ruleID := args[0]
       reason, _ := cmd.Flags().GetString("reason")
       ttl, _ := cmd.Flags().GetDuration("ttl")

       // Validate the rule exists and is Tier 2
       rules := selfprotect.LoadAllRules()
       rule, found := selfprotect.FindRule(rules, ruleID)
       if !found {
           return fmt.Errorf("unknown rule: %s", ruleID)
       }
       if rule.BypassTier == 1 {
           return fmt.Errorf("rule %s is Tier 1 (absolute deny) and cannot be bypassed", ruleID)
       }
       if rule.BypassTier == 3 {
           fmt.Printf("Rule %s is Tier 3 -- use the magic comment `%s` instead.\n",
               ruleID, rule.BypassComment)
           return nil
       }

       token, err := selfprotect.CreateBypassToken(ruleID, reason, ttl)
       if err != nil {
           return err
       }

       fmt.Printf("Bypass token created for rule %s:\n", ruleID)
       fmt.Printf("  Token ID: %s\n", token.TokenID)
       fmt.Printf("  Expires:  %s (%v from now)\n",
           token.ExpiresAt.Format(time.RFC3339), ttl)
       fmt.Printf("  Reason:   %s\n", reason)
       fmt.Println("\nReturn to Claude Code and retry the operation.")
       return nil
   }
   ```

3. Integrate token checking into the rule evaluation pipeline:
   - In `EvaluateRule()`, after a Tier 2 rule matches and would deny:
   - Call `ConsumeBypassToken(rule.ID)`
   - If a valid token is returned: allow the operation, log the bypass
   - If no token: block with exit 2 and the bypass instruction message
   - Bypass audit record written to both the session log and `bypasses.jsonl`

4. Implement Tier 3 magic comment bypass checking:
   - Phase 32 already defines `# gdev-allow-destructive` and `# gdev-allow-credential`
   - Extend the pattern: each Tier 3 rule has a `bypass_comment` field
   - The Bash hook checks `command` for the comment string
   - The Write/Edit hook checks `content`/`new_string` for the comment string

5. Implement bypass attempt logging for Tier 1 rules:
   - If a Tier 1 rule fires, log a `tier1_bypass_attempt` record
   - The record captures: rule_id, tool_name, command/path summary
   - This is a CRITICAL-level audit event (potential attack indicator)

6. Write tests:
   - `gdev hook bypass-next CFG-01` creates a token file in `~/.qsdev/self-protection/bypass-tokens/`
   - Token is consumed on next matching rule evaluation
   - Consumed token is deleted
   - Expired token (past TTL) is ignored and cleaned up
   - `gdev hook bypass-next SP-04` fails with "Tier 1, cannot be bypassed"
   - `gdev hook bypass-next` via Bash tool is blocked by SP-09
   - Tier 3 magic comment `# gdev-allow-destructive` bypasses the corresponding rule
   - Bypass events are logged to the audit trail

**Acceptance Criteria:**
- [ ] `gdev hook bypass-next <rule-id> --reason <text>` creates a single-use token with 5-minute default TTL
- [ ] Token is stored at `~/.qsdev/self-protection/bypass-tokens/<rule-id>-<token-id>.json`
- [ ] Token is consumed (deleted) on first use by the matching rule's hook
- [ ] Expired tokens are cleaned up during token lookup
- [ ] Tier 1 rules reject bypass attempts with a clear error message
- [ ] Tier 2 rules check for valid tokens before blocking; allow if token is valid
- [ ] Tier 3 rules check for magic comments in command/content
- [ ] SP-09 blocks `gdev hook bypass-next` when invoked via Claude Code's Bash tool (chained protection)
- [ ] All bypass events (consumed tokens, magic comments, Tier 1 attempts) are logged to the audit trail
- [ ] Bypass token TTL is configurable via `--ttl` flag (default 5m, max 30m)

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/escape-hatch-research.md` Section 2 -- Three-tier bypass architecture
- `research-spikes/gdev-agent-self-protection-design/escape-hatch-research.md` Section 5.2 -- Chained protection (two-step approval pattern)
- `research-spikes/gdev-agent-self-protection-design/escape-hatch-research.md` Section 3 -- Mandatory audit logging schema
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 1.1 -- The "ask" verdict paradox resolution

**Status:** Not Started

---

### Unit 41.2: Per-Rule Monitor Mode

**Description:** Implement per-rule monitor mode where each rule can independently be in `enforce`, `monitor`, or `off` mode. Monitor mode evaluates the rule's full logic but overrides the verdict to allow, logging what would have been blocked. The `enforce_always` flag (from Phase 40's Tier 1 rules) prevents monitor mode from overriding critical rules -- these 18 rules enforce even when their category is in monitor mode. The design follows the universal pattern from 6 surveyed security systems: evaluate fully, log the would-block decision, allow the operation to proceed.

**Context:** The monitor mode research surveyed SELinux permissive, AppArmor complain, Windows Defender ASR audit, AWS WAF count, seccomp SECCOMP_RET_LOG, and Kubernetes ValidatingAdmissionPolicy warn/audit modes. All 6 systems share the same core pattern: full evaluation with verdict override. Key findings adopted by gdev: per-rule granularity (not global-only), same log stream with a distinguishing field (`mode`), no auto-expiration (soft reminders instead), and AppArmor's pattern where deny rules enforce even in complain mode (`enforce_always`).

New rules deploy in monitor mode by default (except `enforce_always` rules). After a 5-day calibration period, `gdev hook audit` reviews would-block events and `gdev hook enforce --clean` promotes rules with zero events. This incremental enforcement avoids the "deploy and discover false positives in production" failure mode.

**Desired Outcome:** After deploying self-protection hooks, the developer runs `gdev hook audit` after 5 days and sees: "4 rules with 0 events -- safe to promote. 2 rules with events -- review needed." They run `gdev hook enforce --clean` to promote the clean rules. The rules with events are reviewed individually and either refined (pattern adjustment) or promoted after confirming the events are true positives.

**Steps:**

1. Implement mode state storage in `internal/selfprotect/state.go`:
   ```go
   // HookState stores per-rule mode and statistics.
   // Persisted to ~/.qsdev/self-protection/hook-state.yaml.
   type HookState struct {
       Rules map[string]RuleState `yaml:"rules"`
   }

   type RuleState struct {
       Mode          string    `yaml:"mode"` // enforce, monitor, off
       ModeSince     time.Time `yaml:"mode_since"`
       MonitorEvents int       `yaml:"monitor_events"`
   }
   ```

2. Implement mode override in the evaluation pipeline:
   ```go
   func ApplyMode(rule Rule, state RuleState, evaluatedVerdict string) string {
       // enforce_always rules never have their verdict overridden
       if rule.EnforceAlways && evaluatedVerdict == "deny" {
           return "deny"
       }
       // Monitor mode: override to allow, log would-have-blocked
       if state.Mode == "monitor" {
           if evaluatedVerdict == "deny" {
               // Log advisory warning to stderr
               fmt.Fprintf(os.Stderr,
                   "[MONITOR] rule %s would have blocked: %s\n",
                   rule.ID, rule.Message)
           }
           return "allow"
       }
       // Off mode: skip evaluation entirely (handled upstream)
       // Enforce mode: return evaluated verdict
       return evaluatedVerdict
   }
   ```

3. Implement monitor mode JSONL logging:
   ```go
   type MonitorLogEntry struct {
       Timestamp        string `json:"timestamp"`
       EventType        string `json:"event_type"` // "hook_evaluation"
       SessionID        string `json:"session_id"`
       Mode             string `json:"mode"` // "monitor" or "enforce"
       RuleID           string `json:"rule_id"`
       RuleCategory     string `json:"rule_category"`
       HookEvent        string `json:"hook_event"`
       ToolName         string `json:"tool_name"`
       EvaluatedVerdict string `json:"evaluated_verdict"`
       EffectiveVerdict string `json:"effective_verdict"`
       Reason           string `json:"reason"`
       MatchedPattern   string `json:"matched_pattern,omitempty"`
       EvalTimeMs       int    `json:"evaluation_time_ms"`
   }
   ```
   - Logged to `~/.qsdev/audit/sessions/<date>/<session-id>.jsonl` (same stream as SOC 2 audit)
   - Distinguished by `mode: "monitor"` and `effective_verdict: "allow"` (even though evaluated was "deny")

4. Implement CLI commands for mode management in `cmd/hook.go`:
   ```
   gdev hook monitor <rule-id>            Set one rule to monitor mode
   gdev hook monitor --category <cat>     Set all rules in category
   gdev hook monitor --all                Set all rules (respects enforce_always)
   gdev hook enforce <rule-id>            Promote one rule to enforce
   gdev hook enforce --clean              Promote all rules with 0 monitor events
   gdev hook enforce --category <cat>     Promote all rules in category
   gdev hook enforce --all                Promote all rules
   gdev hook off <rule-id>               Disable one rule (warning for Tier 1)
   gdev hook status                       Show all rules with modes and event counts
   ```

5. Implement `gdev hook audit` for reviewing monitor-mode events:
   - Read monitor-mode entries from the audit trail
   - Group by rule ID
   - Show event count per rule
   - For rules with 0 events: "Clean -- safe to promote"
   - For rules with events: show summary of each event
   - `--detail <rule-id>`: show full details of each event for a specific rule
   - `--since <date>`: filter events by date
   - `--json`: export events as JSON for scripting

6. Implement calibration period reminders:
   - At SessionStart: check if any rules have been in monitor mode for > 5 days
   - If so, print once per session: "N rules in monitor mode for >5 days. Run `gdev hook audit`."
   - After 10 days: escalate to warning tone
   - After 20 days: `gdev doctor` flags as a health issue
   - Never auto-promote (universal pattern across all 6 surveyed systems)

7. Write tests:
   - Rule in monitor mode: evaluates fully, logs would-deny, returns allow (exit 0)
   - Rule with `enforce_always`: evaluates, returns deny (exit 2) even in monitor mode
   - `gdev hook enforce --clean` promotes only rules with 0 events
   - Mode change persists in `hook-state.yaml` across invocations
   - Monitor events increment counter in `hook-state.yaml`
   - SessionStart reminder fires after 5 days in monitor mode

**Acceptance Criteria:**
- [ ] Each rule can independently be set to `enforce`, `monitor`, or `off` mode
- [ ] Monitor mode evaluates full rule logic but overrides verdict to allow
- [ ] `enforce_always` rules enforce even in monitor mode (exit 2 on deny)
- [ ] Monitor-mode would-deny events produce `[MONITOR]` advisory on stderr
- [ ] All monitor events logged to same JSONL audit trail with `mode: "monitor"` field
- [ ] `gdev hook status` shows per-rule mode, event counts, and time in current mode
- [ ] `gdev hook audit` summarizes monitor events grouped by rule
- [ ] `gdev hook enforce --clean` promotes rules with 0 monitor events
- [ ] Calibration reminders at 5/10/20 days (never auto-promotes)
- [ ] Mode state persisted in `~/.qsdev/self-protection/hook-state.yaml`
- [ ] Attempting to set a Tier 1 rule to `off` produces a warning (allowed but flagged)

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/monitor-mode-research.md` -- Complete monitor mode design
- `research-spikes/gdev-agent-self-protection-design/monitor-mode-research.md` Section 2 -- 6-system prior art survey
- `research-spikes/gdev-agent-self-protection-design/monitor-mode-research.md` Section 3.4 -- AppArmor deny-in-complain pattern (enforce_always)
- `research-spikes/gdev-agent-self-protection-design/monitor-mode-research.md` Section 5 -- JSONL log format design
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 1.3 -- enforce_always vs Tier 1 reconciliation

**Status:** Not Started

---

### Unit 41.3: Tier 2 Security Rules

**Description:** Implement the 11 Tier 2 interactive-bypass rules that protect configuration files the developer legitimately needs to modify through Claude Code. These rules block by default (exit code 2) but can be bypassed with a `gdev hook bypass-next` token from Unit 41.1. They support monitor mode for initial calibration. Several rules require content inspection (checking what is being written, not just where) to reduce false positives -- for example, CFG-01 only fires when devenv.nix edits contain `sandbox = false`, not on any devenv.nix edit.

**Context:** The verdict model research originally assigned "ask" verdicts to these 10+ rules. The escape hatch research converted them to Tier 2 interactive-bypass denials because Claude Code's `permissionDecision: "ask"` has bugs #39344 and #52822. The synthesis research confirmed: all security-relevant rules default to deny with exit code 2. The rules are:

- **SP-15**: Screen Task tool prompts for mutation verbs + protected path references
- **SP-16**: Block Read of `~/.claude/settings.json` (reconnaissance prevention)
- **CFG-01**: Block devenv.nix edits that weaken security settings
- **CFG-02**: Block edits to .pre-commit-config.yaml
- **CFG-03**: Block edits to .gdev.yaml that change compliance level or disable tools
- **CFG-04**: Block edits to CLAUDE.md that remove gdev-managed sections
- **CFG-05**: Block writes to `.claude/commands/`, `.claude/rules/`, `.claude/agents/`
- **CFG-06**: Block edits to `.mcp.json`
- **CFG-07**: Block .npmrc changes setting `ignore-scripts=false`
- **MCP-05**: Block MCP config with base64-encoded commands
- **MCP-06**: Block `claude mcp add` / `claude mcp install` via Bash

**Desired Outcome:** An agent editing devenv.nix to add a new package sees no block (content does not weaken security). An agent editing devenv.nix to add `sandbox = false` is blocked with a bypass instruction. The developer creates a bypass token, and the retry succeeds.

**Steps:**

1. Implement content-inspection rules in `internal/selfprotect/content_rules.go`:
   ```go
   // EvaluateContentRule checks tool_input.content or tool_input.new_string
   // against the rule's content match patterns.
   func EvaluateContentRule(rule Rule, toolInput ToolInput) *RuleViolation {
       content := toolInput.Content
       if content == "" {
           content = toolInput.NewString
       }
       if content == "" || rule.ContentMatch == nil {
           return nil
       }
       for _, pattern := range rule.ContentMatch.Contains {
           if strings.Contains(content, pattern) {
               return &RuleViolation{
                   RuleID:  rule.ID,
                   Reason:  rule.Message,
                   Pattern: pattern,
               }
           }
       }
       for _, pattern := range rule.ContentMatch.Regex {
           re := getCompiledRegex(pattern)
           if re.MatchString(content) {
               return &RuleViolation{
                   RuleID:  rule.ID,
                   Reason:  rule.Message,
                   Pattern: pattern,
               }
           }
       }
       return nil
   }
   ```

2. Define each Tier 2 rule in `defaults.yaml`:

   **CFG-01** (devenv.nix security weakening):
   ```yaml
   - id: CFG-01
     name: "Block devenv.nix security weakening"
     category: config-guard
     severity: high
     hook_event: PreToolUse
     matcher_tool: "Write|Edit"
     verdict: deny
     bypass_tier: 2
     enforce_always: false
     fail_policy: fail-closed
     description: "Block edits to devenv.nix that weaken security settings"
     message: "Edit to devenv.nix weakens security settings. Run `gdev hook bypass-next CFG-01 --reason 'description'` in your terminal to authorize."
     path_match:
       canonicalize: true
       exact_paths: []
     content_match:
       regex:
         - 'sandbox\s*=\s*false'
         - 'restrict-eval\s*=\s*false'
   ```
   Note: path matching is done by filename (`devenv.nix`) in the project root, combined with content inspection. The rule only fires when BOTH the file is `devenv.nix` AND the content contains security-weakening patterns.

   **CFG-04** (CLAUDE.md gdev section removal):
   ```yaml
   - id: CFG-04
     name: "Block CLAUDE.md gdev section removal"
     category: config-guard
     severity: high
     hook_event: PreToolUse
     matcher_tool: "Edit"
     verdict: deny
     bypass_tier: 2
     description: "Block edits to CLAUDE.md that remove gdev-managed sections"
     message: "This edit removes gdev-managed sections from CLAUDE.md. Run `gdev hook bypass-next CFG-04` in your terminal to authorize."
     content_match:
       contains:
         - "<!-- gdev-managed -->"
   ```
   Note: this rule fires on Edit tool when `old_string` contains `<!-- gdev-managed -->` but `new_string` does not. Requires special evaluation logic.

   **SP-15** (subagent prompt screen):
   ```yaml
   - id: SP-15
     name: "Screen Task tool prompts for protected path references"
     category: self-protection
     severity: high
     hook_event: PreToolUse
     matcher_tool: "Task"
     verdict: deny
     bypass_tier: 2
     description: "Screen subagent prompts for mutation verbs paired with protected path references"
     message: "Subagent prompt references security infrastructure modification. Run `gdev hook bypass-next SP-15` in your terminal to verify this is intentional."
     command_match:
       regex:
         - '(edit|write|modify|remove|delete|replace|overwrite|truncate|disable|bypass|kill|stop|uninstall).{0,100}(settings\.json|\.claude/|\.qsdev/|hooks/|gdev|security|deny.?rules?|hook|guard|protect)'
   ```

3. Implement the CLAUDE.md gdev-section-removal detection:
   ```go
   // For Edit tool with CFG-04: check if old_string contains gdev markers
   // but new_string does not (indicates section removal)
   func checkGdevSectionRemoval(toolInput ToolInput) bool {
       if toolInput.OldString == "" || toolInput.NewString == "" {
           return false
       }
       oldHasMarker := strings.Contains(toolInput.OldString, "<!-- gdev-managed -->")
       newHasMarker := strings.Contains(toolInput.NewString, "<!-- gdev-managed -->")
       return oldHasMarker && !newHasMarker
   }
   ```

4. Implement SP-16 (Read tool protection for settings.json):
   - Register a PreToolUse hook for the Read matcher
   - Check if `tool_input.file_path` (canonicalized) matches `~/.claude/settings.json`
   - Block with Tier 2 bypass instruction
   - This rule can be disabled by user override (acknowledged as potentially noisy)

5. Integrate all Tier 2 rules into the evaluation pipeline:
   - Tier 2 rules evaluate after Tier 1 rules
   - If a Tier 1 rule has already denied, Tier 2 evaluation is skipped (short-circuit)
   - If a Tier 2 rule denies, check for a valid bypass token before blocking
   - If a valid token exists, consume it, log the bypass, allow the operation
   - If no token, block with exit 2 and bypass instruction

6. Write tests for each Tier 2 rule:
   - CFG-01: `devenv.nix` with `sandbox = false` -> blocked; `devenv.nix` adding a package -> allowed
   - CFG-02: any Write/Edit to `.pre-commit-config.yaml` -> blocked
   - CFG-03: `.gdev.yaml` changing `compliance_level: relaxed` -> blocked; adding a new key -> allowed
   - CFG-04: Edit removing `<!-- gdev-managed -->` from CLAUDE.md -> blocked; adding content -> allowed
   - CFG-05: Write to `.claude/commands/evil.md` -> blocked
   - CFG-06: Write to `.mcp.json` -> blocked
   - CFG-07: `.npmrc` with `ignore-scripts=false` -> blocked; `.npmrc` with `registry=...` -> allowed
   - SP-15: Task prompt "edit ~/.claude/settings.json and remove hooks" -> blocked; "research the topic of security" -> allowed
   - SP-16: Read `~/.claude/settings.json` -> blocked; Read `~/.claude/CLAUDE.md` -> allowed
   - MCP-05: `.mcp.json` content with `base64` -> blocked
   - MCP-06: `claude mcp add` in Bash -> blocked
   - All Tier 2 rules: bypass token present -> allowed with audit log

**Acceptance Criteria:**
- [ ] All 11 Tier 2 rules implemented with correct matching logic
- [ ] Content inspection rules (CFG-01, CFG-04, CFG-07, MCP-05) fire only on matching content, not on any write to the file
- [ ] SP-15 screens Task tool prompts with mutation-verb + protected-path regex
- [ ] SP-16 blocks Read of `~/.claude/settings.json` (can be disabled via user override)
- [ ] CFG-04 detects gdev section marker removal in Edit tool (old_string has marker, new_string does not)
- [ ] All Tier 2 rules check for bypass tokens before blocking
- [ ] Bypass instruction message names the rule ID and the exact `gdev hook bypass-next` command to run
- [ ] All Tier 2 rules support monitor mode (log would-deny, allow operation)
- [ ] False positive rates validated: common devenv.nix edits (adding packages, changing versions) do not trigger CFG-01
- [ ] Tier 2 rules coordinate with Phase 32 credential scanner (CFG-05/06 do not duplicate credential patterns)

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/verdict-model-research.md` Section 5.2 -- Per-rule verdict assignment table
- `research-spikes/gdev-agent-self-protection-design/prempti-patterns-research.md` Section 3 -- gdev-specific rules GSP-1 through GSP-6
- `research-spikes/gdev-agent-self-protection-design/threat-model-research.md` Section 4 Rule Sets C and E -- Subagent prompt screen and configuration poisoning guard
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 2.2 -- Complete rule catalog with Tier assignments

**Status:** Not Started

---

### Unit 41.4: Session State Management

**Description:** Implement session state tracking for self-protection hooks. Each Claude Code session gets a state file that tracks: bypassed rules with timestamps and reasons, monitor mode violation counts, total rule evaluations, and calibration period progress. Session state files are stored at `~/.qsdev/self-protection/sessions/<session-id>.json` and auto-cleaned after 7 days.

**Context:** Session state serves three purposes: (1) tracking bypass tokens consumed within a session for audit trail correlation, (2) accumulating monitor-mode event counts for the `gdev hook audit` review workflow, and (3) providing data for `gdev status --self-protection` which shows the current session's self-protection activity. The state file is under `~/.qsdev/` and is therefore protected by SP-03/SP-07 (the agent cannot modify it).

**Desired Outcome:** `gdev status --self-protection` shows: "Session abc123: 47 rules evaluated, 2 Tier 2 bypasses consumed, 3 monitor-mode would-deny events. 7 rules in monitor mode (5 days remaining in calibration)."

**Steps:**

1. Implement session state in `internal/selfprotect/session.go`:
   ```go
   type SessionState struct {
       SessionID       string            `json:"session_id"`
       StartedAt       time.Time         `json:"started_at"`
       RuleEvaluations int               `json:"rule_evaluations"`
       Denials         int               `json:"denials"`
       MonitorEvents   int               `json:"monitor_events"`
       BypassesUsed    []BypassRecord    `json:"bypasses_used"`
       RuleCounts      map[string]int    `json:"rule_counts"` // per-rule evaluation count
   }

   type BypassRecord struct {
       Timestamp  time.Time `json:"timestamp"`
       RuleID     string    `json:"rule_id"`
       TokenID    string    `json:"token_id"`
       Reason     string    `json:"reason"`
   }
   ```

2. Implement session ID detection:
   - Check `CLAUDE_SESSION_ID` environment variable (set by Claude Code)
   - Fallback to `GDEV_SESSION_ID` (set by gdev)
   - Last resort: generate from PID + timestamp

3. Implement auto-cleanup:
   - On every `gdev` invocation (not every hook call -- too expensive)
   - Scan `~/.qsdev/self-protection/sessions/`
   - Remove files older than 7 days
   - Log cleanup count to debug output

4. Implement `gdev status --self-protection`:
   ```
   $ gdev status --self-protection
   Self-Protection Status:
     Rules:    32 total (18 Tier 1, 11 Tier 2, 3 integrity)
     Mode:     18 enforce, 11 monitor, 3 enforce (integrity)
     
   Current Session (abc123):
     Rule evaluations:  47
     Denials:           0
     Monitor events:    3
     Bypasses used:     2
   
   Monitor Mode:
     7 rules in monitor mode
     5 rules with 0 events (safe to promote)
     2 rules with events (review needed)
     Calibration: 3 days remaining
     
   Run `gdev hook audit` to review monitor events.
   Run `gdev hook enforce --clean` to promote clean rules.
   ```

5. Implement session state updates within the hook binary:
   - Atomic file writes (write to temp, rename) to prevent corruption
   - Lock file to prevent concurrent hook invocations from corrupting state
   - Lightweight: only update the session file, not read the full state

6. Write tests:
   - Session state file created on first hook evaluation
   - Rule evaluations increment counter
   - Monitor events increment counter
   - Bypass records appended with timestamp and token ID
   - Auto-cleanup removes files older than 7 days
   - `gdev status --self-protection` reads and displays state correctly
   - Concurrent hook invocations do not corrupt state (lock file test)

**Acceptance Criteria:**
- [ ] Session state file created at `~/.qsdev/self-protection/sessions/<session-id>.json`
- [ ] Tracks rule evaluations, denials, monitor events, and bypasses per session
- [ ] Auto-cleanup removes session files older than 7 days
- [ ] `gdev status --self-protection` displays current session activity and monitor mode summary
- [ ] State file writes are atomic (temp file + rename)
- [ ] Concurrent hook invocations handled safely (lock file)
- [ ] Session state is protected by SP-03/SP-07 (under ~/.qsdev/)

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/monitor-mode-research.md` Section 7.1 -- Mode storage in hook-state.yaml
- `research-spikes/gdev-agent-self-protection-design/escape-hatch-research.md` Section 3 -- Bypass audit record schema
- `research-spikes/gdev-agent-self-protection-design/verdict-model-research.md` Section 6 -- Audit logging integration

**Status:** Not Started

---

### Unit 41.5: PreToolUse Hook -- Write/Edit Tool Rules

**Description:** Implement the `gdev-hook pre-write` subcommand that evaluates all Write/Edit-targeted rules against the incoming tool input. The binary parses stdin JSON for `tool_input.file_path`, `tool_input.content` (Write), and `tool_input.new_string`/`tool_input.old_string` (Edit). File paths are canonicalized before rule evaluation. Content is inspected for Tier 2 rules that require it (CFG-01, CFG-04, CFG-07, MCP-01, MCP-02, MCP-05). The hook coordinates with Phase 32's credential scanner to avoid pattern duplication.

**Context:** The Write/Edit hook has a simpler path extraction problem than the Bash hook: the `file_path` field is a single structured value, not an arbitrary command string. Canonicalization is reliable for these tools (the canonical path research rates Write/Edit path extraction as HIGH reliability). The content inspection rules add a second dimension: not just *where* is the write, but *what* is being written.

Coordination with Phase 32's credential scanner (`credential-scan.py`) is important. Both hooks fire on the same Write/Edit tool calls. The credential scanner checks for secret patterns in content; the self-protection hook checks for protected paths and security-weakening content patterns. The two should not duplicate each other's patterns. Specifically: the credential scanner handles all secret/credential detection; the self-protection hook handles path protection and configuration-weakening detection.

**Desired Outcome:** Writing to `~/.qsdev/hooks/destructive-prevention.sh` is blocked by SP-03 (path protection). Writing `sandbox = false` to `devenv.nix` is blocked by CFG-01 (content inspection). Writing an AWS key to `config.js` is blocked by Phase 32's credential scanner (not by self-protection). Writing a normal TypeScript file to a non-protected path passes through all rules in under 10ms.

**Steps:**

1. Implement `EvaluateWriteRules` in `internal/selfprotect/write_rules.go`:
   ```go
   func EvaluateWriteRules(stdin io.Reader) (int, error) {
       input, err := ParseToolInput(stdin)
       if err != nil {
           return ExitBlock, fmt.Errorf("failed to parse tool input: %w", err)
       }

       toolName := input.ToolName
       filePath := input.ToolInput.FilePath
       if filePath == "" {
           return ExitAllow, nil
       }

       canonPath := CanonicalizePath(filePath)
       rules := LoadRulesForMatcher("Write|Edit")

       var violations []RuleViolation
       for _, rule := range rules {
           // Skip rules that don't apply to this tool
           if !MatcherApplies(rule.MatcherTool, toolName) {
               continue
           }
           // Path-based rules
           if rule.PathMatch != nil {
               if violation := EvaluatePathRule(rule, canonPath); violation != nil {
                   violations = append(violations, *violation)
                   continue
               }
           }
           // Content-based rules (require path match first if both are set)
           if rule.ContentMatch != nil {
               if violation := EvaluateContentRule(rule, input.ToolInput); violation != nil {
                   // For content rules with path constraints, verify path matches too
                   if rule.PathMatch != nil {
                       if pathViolation := EvaluatePathRule(rule, canonPath); pathViolation == nil {
                           continue // Content matches but path doesn't -- skip
                       }
                   }
                   violations = append(violations, *violation)
               }
           }
       }

       if len(violations) == 0 {
           return ExitAllow, nil
       }

       // Check bypass tokens for Tier 2 violations before blocking
       var unbypassed []RuleViolation
       for _, v := range violations {
           rule := FindRuleByID(rules, v.RuleID)
           if rule.BypassTier == 2 {
               token := ConsumeBypassToken(v.RuleID)
               if token != nil {
                   LogBypassEvent(token, v, input)
                   continue // Bypass consumed, skip this violation
               }
           }
           unbypassed = append(unbypassed, v)
       }

       if len(unbypassed) == 0 {
           return ExitAllow, nil
       }

       BlockWithViolations(unbypassed)
       return ExitBlock, nil
   }
   ```

2. Implement Read tool handling for SP-16:
   - The `pre-write` subcommand also handles Read tool (matcher is `Write|Edit|Read` for some rules)
   - SP-16 fires only on Read tool with settings.json path
   - Alternatively, register a separate `pre-read` matcher if cleaner

3. Implement filename-based rule matching for rules that target specific files by name:
   - CFG-01 targets `devenv.nix` in the project root
   - CFG-02 targets `.pre-commit-config.yaml`
   - CFG-03 targets `.gdev.yaml`
   - CFG-04 targets `CLAUDE.md`
   - CFG-06 targets `.mcp.json`
   - CFG-07 targets `.npmrc`
   - These rules need both filename matching (is this the right file?) and content inspection (does this edit weaken security?)

4. Implement the coordination boundary with Phase 32 credential scanner:
   - Document that self-protection rules do NOT check for credentials/secrets
   - Credential patterns (AKIA, PEM headers, JWT, etc.) remain in `credential-scan.py`
   - Self-protection rules check for: protected paths, security-weakening content patterns, MCP poisoning indicators
   - No overlap between the two hook scripts' pattern libraries

5. Deploy the hook in `settings.json`:
   ```json
   {
     "hooks": {
       "PreToolUse": [
         {
           "matcher": "Write|Edit",
           "hooks": [
             {"type": "command", "command": "python3 /home/user/.qsdev/hooks/credential-scan.py"},
             {"type": "command", "command": "/home/user/.qsdev/bin/gdev-hook pre-write"}
           ]
         }
       ]
     }
   }
   ```

6. Write tests:
   - SP-03: Write to `~/.qsdev/hooks/test.sh` -> blocked (path protection)
   - SP-04: Write to `~/.claude/settings.json` -> blocked (exact path)
   - SP-04: Write to `.claude/settings.json` (project-level, no `~`) -> NOT blocked (project settings are Tier 2 via CFG-05)
   - SP-14: Write to `~/.config/nix/nix.conf` -> blocked
   - CFG-01: Write to `devenv.nix` with `sandbox = false` -> blocked; Write to `devenv.nix` with normal content -> allowed
   - CFG-02: Write to `.pre-commit-config.yaml` -> blocked
   - CFG-04: Edit CLAUDE.md removing `<!-- gdev-managed -->` -> blocked; Edit CLAUDE.md adding content -> allowed
   - CFG-06: Write to `.mcp.json` -> blocked
   - MCP-01: Write to `.mcp.json` with `/tmp/server --stdio` -> blocked (Tier 1, no bypass)
   - MCP-02: Write to `.mcp.json` with `pastebin.com` URL -> blocked (Tier 1, no bypass)
   - Bypass test: CFG-01 with valid token -> allowed with audit log
   - Symlink: `ln -s ~/.claude/settings.json /tmp/x.json`, Write to `/tmp/x.json` -> blocked (canonicalized)
   - Performance: 100 invocations of Write to non-protected path -> all complete in < 10ms each

**Acceptance Criteria:**
- [ ] `gdev-hook pre-write` evaluates all Write/Edit-targeted rules (Tier 1 path rules + Tier 2 content rules)
- [ ] File paths are canonicalized via `CanonicalizePath` before matching
- [ ] Content inspection rules fire only when both path and content criteria match
- [ ] Tier 2 rules check for bypass tokens before blocking
- [ ] Write to `~/.claude/settings.json` is blocked (Tier 1, SP-04)
- [ ] Write to project `.claude/settings.json` triggers CFG-05 (Tier 2, bypassable)
- [ ] CFG-01 only fires on devenv.nix edits containing security-weakening patterns
- [ ] CFG-04 detects gdev section marker removal in CLAUDE.md Edit operations
- [ ] Credential/secret patterns are NOT checked (Phase 32 credential scanner handles those)
- [ ] Hook coordinates with Phase 32 credential scanner without pattern duplication
- [ ] Performance: all rules evaluated in < 50ms for typical Write/Edit operations

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/canonical-path-research.md` Section 3.2 -- Write/Edit path extraction rated HIGH reliability
- `research-spikes/gdev-agent-self-protection-design/prempti-patterns-research.md` Section 2.2 SP-3/SP-4 -- Write/Edit path protection translation
- `research-spikes/gdev-agent-self-protection-design/prempti-patterns-research.md` Section 3 -- gdev-specific content inspection rules (GSP-1 through GSP-6)
- `research-spikes/gdev-agent-self-protection-design/threat-model-research.md` Section 4 Rule Set A -- Protected path Write guard
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 3.1 -- Hook architecture: 3 physical scripts

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Bypass Token Lifecycle

```
Developer's terminal             Claude Code session
      |                                |
      |                                | Agent attempts protected edit
      |                                | -> gdev-hook pre-write fires
      |                                | -> CFG-01 matches (devenv.nix + sandbox=false)
      |                                | -> No bypass token found
      |                                | -> exit 2: "Run gdev hook bypass-next CFG-01"
      |                                |
      | $ gdev hook bypass-next CFG-01 |
      |   --reason "adding sandbox off |
      |   for build system compat"     |
      | Token created: tok_7f8a9b2c    |
      | Expires: 5 minutes             |
      |                                |
      |                                | Developer tells agent to retry
      |                                | -> gdev-hook pre-write fires
      |                                | -> CFG-01 matches
      |                                | -> Bypass token tok_7f8a9b2c found and consumed
      |                                | -> Bypass logged to audit trail
      |                                | -> exit 0 (allow)
      |                                | -> Edit proceeds
```

### Monitor Mode State Flow

```
[Deploy]                [Calibrate]              [Review]                [Enforce]
gdev enable hooks  ->  Rules evaluate fully  ->  gdev hook audit    ->  gdev hook enforce
  --self-protection      Verdicts logged          Shows would-deny        --clean
  New rules start        Operations allowed       per rule
  in monitor mode        Developer works           0 events = safe
                         normally                  N events = review
```

### File Locations

| File | Purpose | Protected By |
|------|---------|-------------|
| `~/.qsdev/self-protection/rules.yaml` | User rule overrides | SP-03, SP-07 |
| `~/.qsdev/self-protection/hook-state.yaml` | Per-rule mode state | SP-03, SP-07 |
| `~/.qsdev/self-protection/bypass-tokens/` | Active bypass tokens | SP-03, SP-07 |
| `~/.qsdev/self-protection/sessions/` | Per-session state files | SP-03, SP-07 |
| `~/.qsdev/audit/sessions/<date>/<id>.jsonl` | Unified audit trail | SP-03, SP-07, SP-05 |
| `~/.qsdev/audit/bypasses.jsonl` | Bypass-only convenience log | SP-03, SP-07, SP-05 |

### New Commands

| Command | Notes |
|---------|-------|
| `gdev hook bypass-next <rule-id>` | Create single-use bypass token (Tier 2 only) |
| `gdev hook monitor <rule-id>` | Set rule to monitor mode |
| `gdev hook enforce <rule-id>` | Promote rule to enforce mode |
| `gdev hook enforce --clean` | Promote all rules with 0 monitor events |
| `gdev hook off <rule-id>` | Disable a rule (warning for Tier 1) |
| `gdev hook status` | Show all rules with modes and event counts |
| `gdev hook audit` | Review monitor-mode events |
| `gdev status --self-protection` | Show self-protection activity for current session |
| `gdev-hook pre-write` | PreToolUse hook binary for Write/Edit self-protection rules |

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] Three-tier bypass system is operational: Tier 1 (no bypass), Tier 2 (CLI token), Tier 3 (magic comment)
- [ ] Chained protection verified: `gdev hook bypass-next` via Bash tool is blocked by SP-09
- [ ] Per-rule monitor mode works: rules evaluate fully, log would-deny, allow operations
- [ ] `enforce_always` rules block even in monitor mode (verified by test)
- [ ] All 11 Tier 2 rules implemented with content inspection where required
- [ ] Bypass tokens are single-use, expire after TTL, and are cleaned up
- [ ] All bypass events logged to audit trail with full context (rule ID, token ID, reason)
- [ ] `gdev hook audit` summarizes monitor events and identifies clean rules for promotion
- [ ] `gdev hook enforce --clean` promotes rules with 0 events to enforce mode
- [ ] Session state tracks evaluations, denials, monitor events, and bypasses
- [ ] `gdev-hook pre-write` evaluates all Write/Edit rules with path canonicalization and content inspection
- [ ] Calibration reminders appear at session start after 5 days in monitor mode
- [ ] Self-protection hooks coordinate with Phase 32 hooks without pattern duplication
