# Phase 40: Agent Self-Protection -- Tier 1 Rules & Fail-Closed Harness

## Goal

Deploy the foundational self-protection enforcement layer: a YAML-based rule definition schema, two-tier path canonicalization infrastructure, a fail-closed hook harness, and the 18 Tier 1 (absolute deny, never bypassable) rules that prevent the AI agent from dismantling gdev's own security infrastructure. This phase closes the three P0 gaps identified in the threat model: no protection for settings.json mutations, no protection against indirect Bash writes to protected paths, and no protection against hook registration manipulation. Every rule uses exit code 2 for denials -- never Claude Code's `permissionDecision` JSON, which has two open bugs (#39344, #52822) that make it unreliable.

## Dependencies

Phase 4 complete (Claude Code addon -- settings.json generation, hook deployment infrastructure, section markers). Phase 32 complete (hook deployment, `gdev enable hooks` / `gdev disable hooks`, 3-tier deployment model, `~/.qsdev/hooks/` runtime directory). Phase 32 hooks continue to operate alongside self-protection hooks; this phase does not replace them.

## Phase Outputs

- Rule definition schema (Go structs with `embed.FS` defaults + YAML user overrides)
- `CanonicalizePath()` Go function with two-tier resolution (filesystem first, lexical fallback)
- Fail-closed hook harness that converts any internal error to exit code 2
- 18 Tier 1 rules across 4 categories: self-protection (SP-01 through SP-14), MCP poisoning (MCP-01 through MCP-04)
- PreToolUse hook binary for Bash tool (`gdev-hook pre-bash`)
- Unit and integration tests validating all rules against documented evasion vectors

---

### Unit 40.1: Rule Definition Schema & Configuration

**Description:** Define the YAML-based rule definition format and the Go struct representation. Built-in rules are embedded in the gdev binary via `embed.FS` as a default `rules.yaml`. User overrides in `~/.qsdev/self-protection/rules.yaml` can disable Tier 2 rules or add custom protected paths but cannot weaken Tier 1 rules. The schema supports per-rule fields for mode control (`enforce`, `monitor`, `off`), bypass tier assignment, and the `enforce_always` flag that prevents monitor mode from overriding critical rules.

**Context:** The synthesis research concluded that gdev needs 32 rules across 4 categories (self-protection, configuration guard, MCP poisoning, integrity checks). These rules must be defined declaratively so they can be inspected, audited, and extended without recompilation. The Prempti patterns research designed a Go struct schema with path matching, command matching, and content matching rule types. The monitor mode research established that per-rule mode control is essential for incremental enforcement -- the same format used here drives the mode logic in Phase 41.

The rule format must encode: rule identity (ID, name, category), matching criteria (tool matcher, path patterns, command patterns), enforcement behavior (verdict, bypass tier, enforce_always flag, fail policy), and metadata (description, message template for LLM-friendly denial output). Rules are grouped by `(HookEvent, Matcher)` at load time to determine which rules apply to each hook invocation.

**Desired Outcome:** `gdev hook list` prints all 32 rules with their IDs, categories, verdicts, bypass tiers, and current modes. A developer can create `~/.qsdev/self-protection/rules.yaml` to add a custom protected path and see it enforced on the next Claude Code tool call.

**Steps:**

1. Define the core rule struct in `internal/selfprotect/rule.go`:
   ```go
   package selfprotect

   // Rule defines a single self-protection rule.
   type Rule struct {
       ID            string        `yaml:"id"`
       Name          string        `yaml:"name"`
       Category      string        `yaml:"category"` // self-protection, config-guard, mcp-poisoning, integrity
       Severity      string        `yaml:"severity"` // critical, high, medium, info
       HookEvent     string        `yaml:"hook_event"` // PreToolUse, SessionStart
       MatcherTool   string        `yaml:"matcher_tool"` // Bash, Write|Edit, Read, Task, (all)
       Verdict       string        `yaml:"verdict"` // deny, warn
       BypassTier    int           `yaml:"bypass_tier"` // 1=absolute, 2=interactive, 3=magic-comment
       EnforceAlways bool          `yaml:"enforce_always"` // true = enforces even in monitor mode
       FailPolicy    string        `yaml:"fail_policy"` // fail-closed, fail-open
       Description   string        `yaml:"description"`
       Message       string        `yaml:"message"` // LLM-friendly denial message template

       // Matching criteria (at least one must be set)
       PathMatch    *PathMatchRule    `yaml:"path_match,omitempty"`
       CommandMatch *CommandMatchRule `yaml:"command_match,omitempty"`
       ContentMatch *ContentMatchRule `yaml:"content_match,omitempty"`
   }

   type PathMatchRule struct {
       ExactPaths   []string `yaml:"exact_paths,omitempty"`   // Match exact canonicalized path
       Prefixes     []string `yaml:"prefixes,omitempty"`      // Match path starting with prefix
       Canonicalize bool     `yaml:"canonicalize"`            // Default true
   }

   type CommandMatchRule struct {
       Contains     []string `yaml:"contains,omitempty"`      // Substring match
       Regex        []string `yaml:"regex,omitempty"`         // Regex match
       WordBoundary bool     `yaml:"word_boundary"`           // Wrap Contains in \b
   }

   type ContentMatchRule struct {
       Contains []string `yaml:"contains,omitempty"`
       Regex    []string `yaml:"regex,omitempty"`
   }
   ```

2. Define the embedded defaults in `internal/selfprotect/defaults.go` using `embed.FS`:
   ```go
   //go:embed rules/defaults.yaml
   var defaultRulesFS embed.FS
   ```

3. Implement the rule loader in `internal/selfprotect/loader.go`:
   - Load embedded defaults from `rules/defaults.yaml`
   - If `~/.qsdev/self-protection/rules.yaml` exists, load user overrides
   - Merge: user rules with matching IDs override default fields; user rules with new IDs are appended
   - Validation: reject user overrides that attempt to set `enforce_always: false` on Tier 1 rules or change a Tier 1 rule's verdict to `allow`
   - Group rules by `(HookEvent, MatcherTool)` for efficient lookup during evaluation

4. Write the defaults YAML at `internal/selfprotect/rules/defaults.yaml` containing all 32 rules from the synthesis research rule catalog (Section 2.2). Each rule fully specified with all fields.

5. Implement `gdev hook list` command in `cmd/hook.go`:
   ```
   $ gdev hook list
   ID          Category          Verdict  Tier  Mode     Description
   SP-01       self-protection   deny     1     enforce  Deny gdev CLI invocation via Bash
   SP-02       self-protection   deny     1     enforce  Deny process-kill targeting security tools
   ...
   INT-03      integrity         warn     N/A   enforce  Verify hook script checksums
   
   32 rules total: 18 Tier 1, 11 Tier 2, 3 integrity checks
   ```

**Acceptance Criteria:**
- [ ] Rule struct supports all fields: ID, name, category, severity, hook_event, matcher_tool, verdict, bypass_tier, enforce_always, fail_policy, description, message, path_match, command_match, content_match
- [ ] Embedded defaults contain all 32 rules from the synthesis research catalog
- [ ] User override file at `~/.qsdev/self-protection/rules.yaml` merges with defaults
- [ ] User overrides cannot weaken Tier 1 rules (enforce_always, verdict, bypass_tier are immutable for Tier 1)
- [ ] `gdev hook list` displays all rules with current configuration
- [ ] Rule loader validates YAML syntax and reports errors with line numbers

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 2 -- Complete rule catalog (32 rules)
- `research-spikes/gdev-agent-self-protection-design/prempti-patterns-research.md` Section 4 -- Rule format design, Go struct schema
- `research-spikes/gdev-agent-self-protection-design/verdict-model-research.md` Section 3.4 -- EvaluatedRule struct design

**Status:** Not Started

---

### Unit 40.2: Path Canonicalization Infrastructure

**Description:** Implement the `CanonicalizePath(rawPath string) string` function that resolves symlinks, normalizes `..` and `.`, expands `~`, and handles the `/proc/self/fd/` indirection trick. The function uses a two-tier strategy adapted from Prempti: Tier 1 is filesystem resolution via `filepath.EvalSymlinks` (resolves all symlinks, requires path to exist); Tier 2 is a parent-walk fallback that resolves the existing prefix and lexically normalizes the non-existent remainder. All path-based rules in the self-protection system depend on this function.

**Context:** The canonical path research cataloged 9 bypass technique categories. The three most critical for gdev are symlink traversal (documented real-world bypass in Gemini CLI issue #1121 and the Ona `/proc/self/root` bypass), relative path manipulation, and `/proc/self/fd/` file descriptor tricks. The research concluded that `realpath` in Bash (with `-m` fallback) and `filepath.EvalSymlinks` in Go (with parent-walk fallback) are the correct implementations. The research also established that path canonicalization adds approximately 2-20 microseconds per call -- negligible compared to process spawning overhead.

A critical finding from the canonical path research: the Write tool creates files that do not exist yet. `filepath.EvalSymlinks` fails on non-existent paths. The parent-walk fallback resolves symlinks in existing parent directories while lexically normalizing the non-existent leaf components. Without this, the symlink `ln -s /home/user/.claude /home/user/project/safe-dir` followed by `Write: file_path="/home/user/project/safe-dir/settings.json"` would bypass protection because the leaf `settings.json` does not yet exist.

**Desired Outcome:** Given `ln -s ~/.claude/settings.json /tmp/innocent.json`, calling `CanonicalizePath("/tmp/innocent.json")` returns `/home/user/.claude/settings.json`. Given a non-existent path `/home/user/project/newdir/newfile.txt` where `/home/user/project/` exists, the function resolves any symlinks in the existing prefix and returns the normalized full path.

**Steps:**

1. Implement `CanonicalizePath` in `internal/selfprotect/canonicalize.go`:
   ```go
   package selfprotect

   import (
       "os"
       "path/filepath"
       "strings"
   )

   // CanonicalizePath resolves a file path to its canonical form.
   // Tier 1: filepath.EvalSymlinks resolves all symlinks (requires path to exist).
   // Tier 2: Resolve parent directory symlinks, lexically normalize leaf.
   // Tier 3: Walk up path tree resolving what exists, normalize remainder.
   func CanonicalizePath(rawPath string) string {
       if rawPath == "" {
           return ""
       }

       // Expand ~ to home directory
       if strings.HasPrefix(rawPath, "~/") {
           home, err := os.UserHomeDir()
           if err == nil {
               rawPath = filepath.Join(home, rawPath[2:])
           }
       } else if rawPath == "~" {
           home, _ := os.UserHomeDir()
           return home
       }

       // Make absolute
       absPath, err := filepath.Abs(rawPath)
       if err != nil {
           return filepath.Clean(rawPath)
       }

       // Tier 1: Full filesystem canonicalization
       if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
           return filepath.Clean(resolved)
       }

       // Tier 2: Resolve parent, append leaf
       dir := filepath.Dir(absPath)
       base := filepath.Base(absPath)
       if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
           return filepath.Join(filepath.Clean(resolvedDir), base)
       }

       // Tier 3: Walk up resolving what exists
       parts := strings.Split(absPath, string(filepath.Separator))
       resolved := string(filepath.Separator)
       for i, part := range parts {
           if part == "" {
               continue
           }
           candidate := filepath.Join(resolved, part)
           if evalCandidate, err := filepath.EvalSymlinks(candidate); err != nil {
               remaining := strings.Join(parts[i:], string(filepath.Separator))
               return filepath.Join(resolved, remaining)
           } else {
               resolved = evalCandidate
           }
       }
       return resolved
   }
   ```

2. Implement `IsProtectedPath` in `internal/selfprotect/path_matcher.go`:
   ```go
   // IsProtectedPath checks if a canonicalized path matches any protection rule.
   // Returns the matching rule (if any) and the match reason.
   func IsProtectedPath(canonPath string, rules []Rule) (*Rule, string) {
       for i := range rules {
           rule := &rules[i]
           if rule.PathMatch == nil {
               continue
           }
           for _, exact := range rule.PathMatch.ExactPaths {
               if canonPath == exact {
                   return rule, fmt.Sprintf("exact match: %s", exact)
               }
           }
           for _, prefix := range rule.PathMatch.Prefixes {
               if strings.HasPrefix(canonPath, prefix) {
                   return rule, fmt.Sprintf("prefix match: %s", prefix)
               }
           }
       }
       return nil, ""
   }
   ```

3. Handle special path patterns:
   - `/proc/self/root/` prefix: resolved automatically by `EvalSymlinks` (follows symlink to `/`)
   - `/proc/self/fd/N` paths: detect by prefix before canonicalization, resolve via `os.Readlink`
   - `/dev/fd/N` paths: same treatment as `/proc/self/fd/`
   - Recursive symlinks (ELOOP): treat as suspicious, return error that fail-closed harness converts to deny

4. Write comprehensive tests in `internal/selfprotect/canonicalize_test.go`:
   - Basic: absolute path unchanged
   - Tilde expansion: `~/foo` resolves to `$HOME/foo`
   - Relative path: `../foo` resolves to absolute
   - Symlink resolution: create temp symlink, verify it resolves to target
   - Non-existent path with existing parent: resolves parent symlinks, normalizes leaf
   - `/proc/self/root/` prefix: resolves to real path (test on Linux only)
   - Double-dot traversal: `~/.qsdev/../.qsdev/hooks/` normalizes correctly
   - Empty string: returns empty
   - ELOOP handling: recursive symlink returns error path

**Acceptance Criteria:**
- [ ] `CanonicalizePath` resolves symlinks via `filepath.EvalSymlinks` (Tier 1)
- [ ] Non-existent paths fall back to parent-walk resolution (Tier 2/3)
- [ ] `~` expands to `os.UserHomeDir()`
- [ ] `/proc/self/root/` prefix resolves to real path
- [ ] `/proc/self/fd/N` and `/dev/fd/N` paths are detected and resolved via `os.Readlink`
- [ ] Relative paths are converted to absolute via `filepath.Abs`
- [ ] Path resolution adds < 1ms overhead per call (verified by benchmark)
- [ ] All 9 bypass techniques from the canonical path research are tested

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/canonical-path-research.md` -- Complete bypass catalog (9 techniques), two-tier canonicalization design, Go implementation pseudocode
- `research-spikes/gdev-agent-self-protection-design/canonical-path-research.md` Section 2.3 -- Go CanonicalizePath implementation
- `research-spikes/gdev-agent-self-protection-design/threat-model-research.md` Section 1 Vector 4 -- Path manipulation threat analysis

**Status:** Not Started

---

### Unit 40.3: Fail-Closed Hook Harness

**Description:** Implement the hook execution wrapper that ensures security-critical hooks block operations on ANY internal error. Claude Code's hook system is inherently fail-open: any non-exit-2 error allows the operation to proceed. The harness wraps all rule evaluation in a `recover()` block that catches panics and converts them to exit code 2 with a diagnostic stderr message. The harness also enforces a 100ms timeout for rule evaluation and implements the severity-tiered fail policy (fail-closed for security hooks, fail-open for advisory hooks).

**Context:** The fail-policy research surveyed 8 security systems (Prempti, reasoning-core, Claude Code, SELinux, AppArmor, AWS WAF, seccomp, gVisor) and concluded that gdev needs severity-tiered failure: fail-closed for self-protection, destructive prevention, and credential scanning; fail-open for cost alerting, audit logging, test enforcement, and client isolation. The research identified 7 specific failure scenarios (hook crash, timeout, malformed JSON, unreadable config, binary crash, version regression, engineered failure) and designed the Go harness as the cleanest fail-closed implementation -- it eliminates most failure modes (missing interpreter, dependency issues, shell profile interference, regex backtracking).

The key design principle from the escape hatch research: exit code 2 is used for ALL security denials, not JSON `permissionDecision: "deny"`. This is because exit code 2 blocks the tool call *before* Claude Code's permission rules are evaluated, making it immune to bug #39344 (where ask overrides deny).

**Desired Outcome:** If the self-protection hook binary panics, runs out of memory, or encounters any unexpected error during rule evaluation, the operation is blocked with a clear error message: "gdev self-protection hook error -- blocking for safety. Run `gdev doctor` to diagnose." The developer sees the error and can use `gdev doctor --fix` to repair.

**Steps:**

1. Implement the harness in `internal/selfprotect/harness.go`:
   ```go
   package selfprotect

   import (
       "context"
       "fmt"
       "os"
       "time"
   )

   const (
       ExitAllow      = 0
       ExitBlock      = 2
       DefaultTimeout = 100 * time.Millisecond
   )

   // RunWithHarness executes a hook evaluation function with fail-closed protection.
   // If the function panics, times out, or returns an error, the harness
   // writes a diagnostic message to stderr and exits with code 2 (block).
   func RunWithHarness(failPolicy string, timeout time.Duration, fn func() (int, error)) {
       if timeout == 0 {
           timeout = DefaultTimeout
       }

       ctx, cancel := context.WithTimeout(context.Background(), timeout)
       defer cancel()

       resultCh := make(chan harnessResult, 1)

       go func() {
           defer func() {
               if r := recover(); r != nil {
                   resultCh <- harnessResult{
                       exitCode: ExitBlock,
                       err:      fmt.Errorf("panic: %v", r),
                   }
               }
           }()
           exitCode, err := fn()
           resultCh <- harnessResult{exitCode: exitCode, err: err}
       }()

       select {
       case result := <-resultCh:
           if result.err != nil {
               if failPolicy == "fail-closed" {
                   fmt.Fprintf(os.Stderr,
                       "gdev self-protection hook error -- blocking for safety.\n"+
                       "  Error: %v\n"+
                       "  Run `gdev doctor` to diagnose.\n", result.err)
                   os.Exit(ExitBlock)
               }
               // fail-open: log and allow
               fmt.Fprintf(os.Stderr,
                   "gdev hook warning: %v (advisory hook, operation continues)\n", result.err)
               os.Exit(ExitAllow)
           }
           os.Exit(result.exitCode)

       case <-ctx.Done():
           if failPolicy == "fail-closed" {
               fmt.Fprintf(os.Stderr,
                   "gdev self-protection hook timeout (%v) -- blocking for safety.\n"+
                   "  Run `gdev doctor` to diagnose.\n", timeout)
               os.Exit(ExitBlock)
           }
           fmt.Fprintf(os.Stderr,
               "gdev hook timeout (%v) -- advisory hook, operation continues.\n", timeout)
           os.Exit(ExitAllow)
       }
   }

   type harnessResult struct {
       exitCode int
       err      error
   }
   ```

2. Implement structured error output for blocked operations:
   ```go
   // BlockWithMessage writes a structured denial message to stderr and exits 2.
   func BlockWithMessage(ruleID, reason, remediation string) {
       fmt.Fprintf(os.Stderr,
           "BLOCKED by gdev self-protection [%s]:\n"+
           "  %s\n"+
           "  %s\n", ruleID, reason, remediation)
       os.Exit(ExitBlock)
   }
   ```

3. Implement the fail policy resolver:
   - Read the rule's `fail_policy` field
   - Critical/high severity rules: always fail-closed
   - Info severity rules: always fail-open
   - Medium severity rules: configurable via `~/.qsdev/self-protection/rules.yaml`

4. Write tests:
   - Panic recovery: function that panics -> exit 2
   - Timeout: function that sleeps 200ms with 100ms timeout -> exit 2
   - Clean deny: function returns ExitBlock -> exit 2
   - Clean allow: function returns ExitAllow -> exit 0
   - Error in fail-open mode: function returns error -> exit 0 (logged, not blocked)
   - Error in fail-closed mode: function returns error -> exit 2

**Acceptance Criteria:**
- [ ] Panics in rule evaluation are caught and converted to exit code 2
- [ ] Rule evaluation that exceeds 100ms timeout is terminated and blocked (exit 2)
- [ ] Error messages to stderr identify the failing hook and suggest `gdev doctor`
- [ ] Fail-closed policy applies to all security-critical rules (self-protection, destructive prevention, credential scanning)
- [ ] Fail-open policy applies to advisory rules (cost alerting, audit logging, test enforcement)
- [ ] Exit code 2 is used for ALL security blocks -- never JSON `permissionDecision: "deny"`
- [ ] `gdev doctor` can diagnose harness-level failures (hook health check)

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/fail-policy-research.md` -- Severity-tiered fail policy, Go harness design, 7 failure scenarios
- `research-spikes/gdev-agent-self-protection-design/fail-policy-research.md` Section 4.2 -- The fail-closed harness (Go implementation)
- `research-spikes/gdev-agent-self-protection-design/escape-hatch-research.md` Section 6.2 -- Why exit code 2 for all denials
- `research-spikes/gdev-agent-self-protection-design/verdict-model-research.md` Section 1.2 -- Verdict vs exit code mapping

**Status:** Not Started

---

### Unit 40.4: Tier 1 Critical Security Rules (enforce_always)

**Description:** Implement the 18 Tier 1 rules that can NEVER be bypassed -- not by magic comments, CLI commands, or environment variables. These rules protect the root of trust: settings.json hook registrations, hook scripts, audit trail, bypass mechanisms, and the gdev CLI itself. All 18 rules have `enforce_always: true`, meaning they enforce even when their category is in monitor mode. All use exit code 2 for denials with descriptive stderr messages that guide the agent toward compliant alternatives.

**Context:** The synthesis research consolidated findings from the threat model (12 attack vectors), Prempti patterns (6 self-protection rules), and escape hatch research (3-tier bypass policy) into a definitive 32-rule catalog. The 18 Tier 1 rules are the subset that protects gdev's own security infrastructure -- the "guard the guards" layer that no other defense provides. The threat model rates the gaps these rules close as P0 (Critical): settings.json mutations, indirect Bash writes, and hook registration manipulation.

The rules are organized into 3 logical groups deployed across 2 physical matchers:
- **Bash matcher** (SP-01, SP-02, SP-06 through SP-13, MCP-03, MCP-04): 13 rules matching Bash tool commands
- **Write|Edit matcher** (SP-03, SP-04, SP-05, SP-14, MCP-01, MCP-02): 6 rules matching Write/Edit tool file paths (note: SP-04 covers Write|Edit; SP-06 covers the Bash vector for the same files -- both are needed for defense in depth)

All path matching uses the `CanonicalizePath` function from Unit 40.2. All command matching uses pre-compiled regex patterns from Unit 40.5.

**Desired Outcome:** An agent that attempts `sed -i 's/hooks//' ~/.claude/settings.json` via the Bash tool is immediately blocked with: "BLOCKED by gdev self-protection [SP-06]: Cannot modify Claude Code settings via Bash. Use `gdev enable hooks` to manage hook configuration." An agent that attempts `Write: file_path="~/.qsdev/hooks/destructive-prevention.sh"` is blocked with: "BLOCKED by gdev self-protection [SP-03]: Cannot write to gdev installation directory (~/.qsdev/). Hook scripts are managed by `gdev enable hooks`."

**Steps:**

1. Define all 18 Tier 1 rules in `internal/selfprotect/rules/defaults.yaml`:
   ```yaml
   rules:
     # --- Self-Protection: Write/Edit rules ---
     - id: SP-03
       name: "Deny writes to gdev install prefix"
       category: self-protection
       severity: critical
       hook_event: PreToolUse
       matcher_tool: "Write|Edit"
       verdict: deny
       bypass_tier: 1
       enforce_always: true
       fail_policy: fail-closed
       description: "Block Write/Edit to ~/.qsdev/ (hooks, config, audit logs)"
       message: "Cannot write to gdev installation directory (~/.qsdev/). Hook scripts and configuration are managed by `gdev enable hooks`."
       path_match:
         canonicalize: true
         prefixes:
           - "{{HOME}}/.qsdev/"

     - id: SP-04
       name: "Deny writes to Claude Code settings"
       category: self-protection
       severity: critical
       hook_event: PreToolUse
       matcher_tool: "Write|Edit"
       verdict: deny
       bypass_tier: 1
       enforce_always: true
       fail_policy: fail-closed
       description: "Block Write/Edit to ~/.claude/settings.json and settings.local.json"
       message: "Cannot modify Claude Code settings. Hook registrations and security configuration must not be modified by the agent. Use `gdev enable hooks` to reconfigure."
       path_match:
         canonicalize: true
         exact_paths:
           - "{{HOME}}/.claude/settings.json"
           - "{{HOME}}/.claude/settings.local.json"

     # ... (remaining 16 rules following same pattern)
   ```

2. Implement the `{{HOME}}` template expansion in the rule loader:
   - At rule load time, replace `{{HOME}}` with `CanonicalizePath(os.UserHomeDir())`
   - This ensures the protected path list uses the canonicalized home directory, catching symlinked home directories

3. For each rule, write the specific matching logic:
   - **SP-01** (deny gdev CLI): `\bgdev\b` word boundary in command, excluding `gdev-allow-` and `gdev-hook-` substrings
   - **SP-02** (deny process-kill): match `pkill|killall` followed by `pre-commit|gitleaks|ripsecrets|gdev`
   - **SP-03** (deny writes ~/.qsdev/): prefix match on canonicalized path
   - **SP-04** (deny writes settings.json): exact match on canonicalized path
   - **SP-05** (deny writes policy-limits.json): exact match
   - **SP-06** (deny Bash writes to settings): command contains protected path + write operator
   - **SP-07** (deny Bash writes to ~/.qsdev/): command contains `/.qsdev/` + write/delete/permission operator
   - **SP-08** (deny bypass export): `export GDEV_HOOK_BYPASS`, `export GDEV_BYPASS_`, `export GDEV_SELF_PROTECTION`
   - **SP-09** (deny bypass-next via Bash): `gdev hook bypass-next` in command
   - **SP-10** (deny history destruction): `export HISTSIZE=0`, `history -c`
   - **SP-11** (deny git hooks disabling): `git config core.hooksPath`, `git config core.fsmonitor`
   - **SP-12** (deny obfuscation): `base64 -d | bash`, `base64 --decode | sh`, pipe-to-interpreter patterns
   - **SP-13** (deny Nix removal): `nix profile remove.*gdev`
   - **SP-14** (deny writes nix.conf): exact match on `~/.config/nix/nix.conf`
   - **MCP-01** (deny MCP from temp dir): content inspection for `/tmp/`, `/dev/shm/`, `/var/tmp/` in `.mcp.json` writes
   - **MCP-02** (deny MCP with IOC domain): content inspection for pastebin.com, transfer.sh, etc.
   - **MCP-03** (deny MCP execution from temp): command contains `--stdio|--sse` + `/tmp/`
   - **MCP-04** (deny install from IOC domain): `npm install|pip install` + IOC domain patterns

4. For Bash matcher rules, implement the path-in-command detection:
   ```go
   // ExtractProtectedPathsFromCommand checks if a Bash command references
   // any protected path combined with a write operator.
   func ExtractProtectedPathsFromCommand(command string, protectedPaths []string) []string {
       // Expand ~ and $HOME in the command
       expanded := expandHomeVars(command)
       var matches []string
       for _, path := range protectedPaths {
           if strings.Contains(expanded, path) {
               matches = append(matches, path)
           }
       }
       return matches
   }
   ```

5. Write tests for every rule against both positive (should block) and negative (should allow) cases. Use the test cases from Phase 32 as a pattern, extended with self-protection scenarios.

**Acceptance Criteria:**
- [ ] All 18 Tier 1 rules are defined in `defaults.yaml` and loaded by the rule loader
- [ ] All rules have `enforce_always: true` and `bypass_tier: 1`
- [ ] SP-03: blocks Write/Edit to any path under `~/.qsdev/` (including via symlinks)
- [ ] SP-04: blocks Write/Edit to `~/.claude/settings.json` and `settings.local.json` (not project `.claude/settings.json`)
- [ ] SP-06: blocks Bash commands writing to settings.json (sed -i, cat >, tee, cp, mv, echo >, perl -pi, python -c, awk >, dd of=)
- [ ] SP-07: blocks Bash commands writing to, deleting from, or changing permissions on `~/.qsdev/` paths
- [ ] SP-08: blocks `export GDEV_HOOK_BYPASS`, `export GDEV_BYPASS_*`, `export GDEV_SELF_PROTECTION`
- [ ] SP-09: blocks `gdev hook bypass-next` when invoked via Bash tool
- [ ] SP-12: blocks `base64 -d | bash`, `base64 --decode | sh` obfuscation patterns
- [ ] MCP-01/02: blocks MCP config writes with temp directory paths or IOC domains
- [ ] All rules use exit code 2 for denials (never JSON permissionDecision)
- [ ] Denial messages name the violated rule and suggest a compliant alternative
- [ ] All path matching uses `CanonicalizePath` -- no raw path comparisons

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 2.2 -- Complete rule catalog table
- `research-spikes/gdev-agent-self-protection-design/threat-model-research.md` Section 4 -- Minimum self-protection rule sets A-E
- `research-spikes/gdev-agent-self-protection-design/prempti-patterns-research.md` Section 2.2 -- Rule-by-rule translation from Prempti
- `research-spikes/gdev-agent-self-protection-design/escape-hatch-research.md` Section 2.1 -- Tier 1 absolute deny rules
- `research-spikes/gdev-agent-self-protection-design/monitor-mode-research.md` Section 3.4 -- enforce_always pattern from AppArmor

**Status:** Not Started

---

### Unit 40.5: PreToolUse Hook -- Bash Tool Rules

**Description:** Implement the `gdev-hook pre-bash` subcommand as a compiled Go binary that evaluates all Bash-targeted rules against the incoming tool input. The binary parses stdin JSON for `tool_input.command`, expands `~` and `$HOME` references in the command string, extracts redirect targets where possible, canonicalizes extracted paths, evaluates all applicable rules using all-match-then-escalate with deny-overrides, and returns exit code 0 (allow) or 2 (block) with a structured stderr message.

**Context:** The Bash tool is the most powerful tool available to the agent and the hardest to constrain. The canonical path research concluded that extracting write targets from arbitrary Bash commands is equivalent to static analysis of shell scripts -- undecidable in the general case. The recommended strategy is layered: regex pattern matching for common patterns (catches 80-90% of naive attacks), an evasion-mechanism blocklist for known obfuscation techniques, and runtime integrity verification (Phase 41) as a backstop.

The Go binary approach (rather than bash scripts) eliminates the failure modes documented in the fail-policy research: missing interpreter, dependency issues, shell profile interference printing text before JSON, and regex backtracking DoS. Go's RE2 regex engine has no backtracking, making timeout impossible from adversarial input.

**Desired Outcome:** The `gdev-hook pre-bash` binary starts in under 50ms, evaluates all 13 Bash-targeted rules against the command, and blocks with a descriptive error if any rule fires. Legitimate commands like `ls ~/.qsdev/` (read-only) pass through. Commands like `sed -i 's/deny/allow/' ~/.claude/settings.json` are blocked immediately.

**Steps:**

1. Create the hook binary entry point at `cmd/gdev-hook/main.go`:
   ```go
   package main

   import (
       "fmt"
       "os"

       "gdev/internal/selfprotect"
   )

   func main() {
       if len(os.Args) < 2 {
           fmt.Fprintln(os.Stderr, "Usage: gdev-hook <pre-bash|pre-write|pre-agent>")
           os.Exit(1)
       }

       switch os.Args[1] {
       case "pre-bash":
           selfprotect.RunWithHarness("fail-closed", selfprotect.DefaultTimeout, func() (int, error) {
               return selfprotect.EvaluateBashRules(os.Stdin)
           })
       case "pre-write":
           selfprotect.RunWithHarness("fail-closed", selfprotect.DefaultTimeout, func() (int, error) {
               return selfprotect.EvaluateWriteRules(os.Stdin)
           })
       default:
           fmt.Fprintf(os.Stderr, "Unknown hook type: %s\n", os.Args[1])
           os.Exit(1)
       }
   }
   ```

2. Implement `EvaluateBashRules` in `internal/selfprotect/bash_rules.go`:
   ```go
   func EvaluateBashRules(stdin io.Reader) (int, error) {
       input, err := ParseToolInput(stdin)
       if err != nil {
           return ExitBlock, fmt.Errorf("failed to parse tool input: %w", err)
       }
       command := input.ToolInput.Command
       if command == "" {
           return ExitAllow, nil
       }

       rules := LoadRulesForMatcher("Bash")
       expandedCmd := ExpandHomeVars(command)

       var violations []RuleViolation
       for _, rule := range rules {
           if violation := EvaluateRule(rule, command, expandedCmd); violation != nil {
               violations = append(violations, *violation)
           }
       }

       if len(violations) == 0 {
           return ExitAllow, nil
       }

       // Deny-overrides: report all matching rules
       BlockWithViolations(violations)
       return ExitBlock, nil // unreachable (BlockWithViolations exits)
   }
   ```

3. Implement pre-compiled regex pool:
   ```go
   var bashRegexPool = sync.OnceValues(func() (map[string]*regexp.Regexp, error) {
       patterns := map[string]string{
           "write_ops":      `(>|>>|tee\s|dd\s.*of=|sed\s+-i|cp\s|mv\s|chmod\s|chown\s|truncate\s|cat\s.*>|echo\s.*>|perl\s+-pi)`,
           "delete_ops":     `(rm\s|unlink\s|truncate\s+-s\s+0|shred\s)`,
           "gdev_word":      `\bgdev\b`,
           "process_kill":   `(pkill|killall)\s+.*(pre-commit|gitleaks|ripsecrets|gdev|falco)`,
           "bypass_export":  `export\s+(GDEV_HOOK_BYPASS|GDEV_BYPASS_|GDEV_SELF_PROTECTION|GDEV_SKIP_ISOLATION)`,
           "history_destroy": `(export\s+HISTSIZE=0|history\s+-c)`,
           "git_hooks_path": `git\s+config\s+.*core\.(hooksPath|fsmonitor)`,
           "base64_pipe":    `base64\s+(-d|--decode)\s*\|\s*(ba)?sh`,
           "nix_remove":     `nix\s+profile\s+remove.*gdev`,
           "bypass_next":    `gdev\s+hook\s+bypass-next`,
           "mcp_temp_exec":  `(--stdio|--sse).*(/tmp/|/dev/shm/|/var/tmp/)`,
           "ioc_install":    `(npm|pip|pip3)\s+install.*` + iocDomainPattern,
       }
       compiled := make(map[string]*regexp.Regexp, len(patterns))
       for name, pat := range patterns {
           re, err := regexp.Compile(pat)
           if err != nil {
               return nil, fmt.Errorf("compile %s: %w", name, err)
           }
           compiled[name] = re
       }
       return compiled, nil
   })
   ```

4. Implement command path expansion:
   - Replace `~/` with canonicalized `$HOME/`
   - Replace `$HOME/` and `${HOME}/` with canonicalized home path
   - Extract redirect targets using regex: `[0-9]*>{1,2}\s*[^\s;|&]+`
   - Canonicalize each extracted redirect target before matching

5. Implement settings.json deployment for the hook:
   ```json
   {
     "hooks": {
       "PreToolUse": [
         {
           "matcher": "Bash",
           "hooks": [
             {"type": "command", "command": "/home/user/.qsdev/bin/gdev-hook pre-bash"}
           ]
         }
       ]
     }
   }
   ```

6. Write tests covering all 13 Bash-targeted rules:
   - SP-01: `gdev disable hooks` -> blocked; `echo gdev_config_path` -> allowed (word boundary)
   - SP-02: `pkill pre-commit` -> blocked; `pkill my-app` -> allowed
   - SP-06: `sed -i 's/hooks//' ~/.claude/settings.json` -> blocked; `cat ~/.claude/settings.json` -> allowed (read, not write)
   - SP-07: `echo > ~/.qsdev/hooks/foo.sh` -> blocked; `ls ~/.qsdev/` -> allowed
   - SP-08: `export GDEV_HOOK_BYPASS=1` -> blocked; `echo $GDEV_HOOK_BYPASS` -> allowed
   - SP-09: `gdev hook bypass-next sp-04` -> blocked
   - SP-10: `export HISTSIZE=0` -> blocked; `export HISTSIZE=1000` -> allowed
   - SP-11: `git config core.hooksPath /dev/null` -> blocked; `git config user.email foo@bar.com` -> allowed
   - SP-12: `echo payload | base64 -d | bash` -> blocked; `base64 -d file.b64 > output.bin` -> allowed (no pipe to shell)
   - SP-13: `nix profile remove gdev` -> blocked; `nix profile remove hello` -> allowed
   - MCP-03: `./server --stdio` from `/tmp/` -> blocked; `./server --stdio` from project dir -> allowed
   - MCP-04: `npm install https://pastebin.com/raw/xyz` -> blocked; `npm install express` -> allowed
   - Symlink evasion: `sed -i 's/foo/bar/' /tmp/symlink-to-settings` -> blocked (after canonicalization)

**Acceptance Criteria:**
- [ ] Hook binary starts in under 50ms (Go binary cold start, verified by benchmark)
- [ ] All 13 Bash-targeted Tier 1 rules are evaluated on every Bash tool invocation
- [ ] Pre-compiled regex pool eliminates per-invocation compilation overhead
- [ ] `~` and `$HOME` in command strings are expanded to canonicalized home path before matching
- [ ] Redirect targets (`> /path`, `>> /path`) are extracted and canonicalized
- [ ] All-match-then-escalate evaluation: all rules fire, deny-overrides combining
- [ ] Block messages name the specific rule ID and provide a compliant alternative
- [ ] Legitimate Bash commands (ls, cat, echo to non-protected paths) are not blocked
- [ ] Hook is deployed via `gdev enable hooks` to `~/.claude/settings.json` under `[gdev-managed-policy]`

**Research Citations:**
- `research-spikes/gdev-agent-self-protection-design/canonical-path-research.md` Section 3 -- Bash write target extraction strategies A/B/C
- `research-spikes/gdev-agent-self-protection-design/threat-model-research.md` Section 4 Rule Set B -- Protected path Bash guard patterns
- `research-spikes/gdev-agent-self-protection-design/verdict-model-research.md` Section 3.2 -- Why all-match-then-escalate (not first-match)
- `research-spikes/gdev-agent-self-protection-design/fail-policy-research.md` Section 6 -- Go binary advantage (no backtracking, no missing interpreter)
- `research-spikes/gdev-agent-self-protection-design/synthesis-research.md` Section 3.1 -- 3 physical scripts architecture

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Go Binary Architecture

The `gdev-hook` binary replaces the shell/Python hook scripts from Phase 32 for self-protection rules. Phase 32's consulting hooks (destructive prevention, credential scanning, etc.) continue to run as their existing scripts until Phase 42 consolidates everything.

```
~/.qsdev/bin/gdev-hook          # Compiled Go binary
~/.qsdev/hooks/                  # Phase 32 shell/Python scripts (unchanged)
~/.qsdev/self-protection/        # Rule configs and state
  rules.yaml                     # User overrides (optional)
```

### settings.json Hook Registration

Self-protection hooks are registered alongside Phase 32 hooks in `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "/home/user/.qsdev/hooks/destructive-prevention.sh"},
          {"type": "command", "command": "/home/user/.qsdev/bin/gdev-hook pre-bash"}
        ]
      },
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

Claude Code runs all registered hooks for a matcher in parallel. If ANY hook returns exit 2, the operation is blocked. This means self-protection hooks and Phase 32 hooks operate independently -- both must allow for the operation to proceed.

### Exit Code Semantics

| Scenario | Exit Code | Stderr | Claude Code Behavior |
|----------|-----------|--------|---------------------|
| All rules pass | 0 | (empty) | Proceeds (allow) |
| Any Tier 1 rule fires | 2 | Rule ID + reason + remediation | Blocks tool call |
| Hook internal error (fail-closed) | 2 | Error details + `gdev doctor` suggestion | Blocks tool call |
| Hook internal error (fail-open) | 0 | Warning message | Proceeds |

### Path Template Expansion

Protected paths in `defaults.yaml` use `{{HOME}}` as a placeholder:
```yaml
exact_paths:
  - "{{HOME}}/.claude/settings.json"
prefixes:
  - "{{HOME}}/.qsdev/"
```

At load time, `{{HOME}}` is replaced with `CanonicalizePath(os.UserHomeDir())`. This ensures:
- The protected path list uses the real home directory (resolving any symlinks)
- Matching works correctly even if `$HOME` is a symlink (e.g., NixOS home-manager)
- The comparison is canonical-to-canonical (no bypasses via non-canonical home path)

### New Packages

| Package | Path | Purpose |
|---------|------|---------|
| `selfprotect` | `internal/selfprotect/` | Rule definitions, loader, evaluation, path canonicalization, harness |

### New Commands

| Command | Notes |
|---------|-------|
| `gdev hook list` | Lists all rules with IDs, categories, verdicts, tiers, modes |
| `gdev-hook pre-bash` | PreToolUse hook binary for Bash tool self-protection rules |
| `gdev-hook pre-write` | PreToolUse hook binary for Write/Edit tool self-protection rules |

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] Rule definition schema supports all 32 rules from synthesis research catalog
- [ ] Path canonicalization handles all 9 bypass techniques from canonical path research
- [ ] Fail-closed harness catches panics, timeouts, and errors; converts to exit 2
- [ ] All 18 Tier 1 rules are implemented and tested with both positive and negative test cases
- [ ] `gdev-hook pre-bash` binary starts in < 50ms and evaluates all rules in < 50ms
- [ ] Symlink evasion attempts are caught by path canonicalization (verified by tests)
- [ ] `/proc/self/root/` path bypass is caught by canonicalization (verified by tests on Linux)
- [ ] Legitimate developer operations (reading files, listing directories, non-protected writes) are not blocked
- [ ] Self-protection hooks coexist with Phase 32 hooks without interference
- [ ] `gdev enable hooks` deploys self-protection hooks alongside existing Phase 32 hooks
- [ ] All denials use exit code 2 -- no JSON `permissionDecision` used for security decisions
