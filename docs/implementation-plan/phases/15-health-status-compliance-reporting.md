# Phase 15: Health, Status & Compliance Reporting

## Goal

Implement a comprehensive project health and compliance reporting system that lets developers instantly assess their project's security posture, detect configuration drift, generate compliance evidence for client audits, and produce team-level dashboards across multiple projects. The primary interface is `qsdev status` with progressive disclosure (quiet through verbose through JSON), backed by a three-layer compliance posture model, a six-category drift detection engine, machine-readable output (versioned JSON schema, SARIF 2.1.0), badge generation, and a CI-artifact-based team aggregation pipeline.

## Dependencies

Phase 1 complete (shared types in `pkg/types/`). Phase 12 complete (tool lifecycle management — the tool registry is the data source for defense coverage and tool inventory reporting). Phase 13 complete (project config — `.qsdev.yaml` supplies conformance level definitions and policy overrides). Phase 14 complete (skills — the `gdev-status` skill invokes this reporting system).

## Phase Outputs

- `qsdev status` command with progressive disclosure: `--quiet` (exit code only), default (hierarchical checkmarks), `--verbose` (per-check detail), `--json` (machine-readable)
- Three-layer compliance posture scoring: defense coverage (40%), configuration health (30%), dependency health (30%)
- 0-100 numeric score with A-F letter grades and conformance labels (baseline/enhanced PASS/FAIL)
- Six-category drift detection engine (all local, <100ms total)
- `qsdev evidence` command for compliance evidence generation (SOC2/HIPAA control mapping)
- Machine-readable output: versioned JSON schema, SARIF 2.1.0 for GitHub Code Scanning, shields.io badge JSON
- Badge generation (score, conformance, defense count variants)
- Team-level CI artifact aggregation pipeline with markdown dashboard and auto-generated GitHub issues

---

### Unit 15.1: `qsdev status` Command & Progressive Disclosure

**Description:** Implement the `qsdev status` command as the primary developer interface for project health assessment. The command uses progressive disclosure to serve both quick-glance ("is everything green?") and detailed investigation ("what exactly is wrong and how do I fix it?") use cases, with subcommands for focused views and exit codes for CI gate integration.

**Code-Grounded Implementation Note:** No existing gdev command produces `--json` output. This unit establishes the `--json` output format pattern that all subsequent commands will follow. The state tracking infrastructure already exists: `state.CheckModified()` at `internal/state/state.go:46-94` performs hash-based comparison, and `state.LoadStateFromFile()` at `internal/state/persistence.go:13-40` loads YAML state. However, state is currently split across three files: `.devinit/.qsdev-init-state.yaml`, `.devenv/.gdev-state.yaml`, `.claude/.gdev-claude-state.yaml`. This phase requires a `LoadAllStates()` aggregation function that unifies these three sources into a single posture view before scoring can proceed.

**Context:** Currently, a developer managing a gdev-bootstrapped project has no way to answer "is my security posture intact?" without manually checking each generated file, each defense layer, and each ecosystem's vulnerability status. The tool lifecycle system (Phase 12) tracks enabled/disabled tools, the state file tracks generated file hashes, and ecosystem modules know about lock files and vulnerability scanners — but nothing aggregates these signals into a single view.

The `qsdev status` command fills this gap by following established patterns from `flutter doctor` (hierarchical checks with three-state indicators), `npm audit` (severity-threshold exit codes), and OpenSSF Scorecard (weighted scoring with per-check detail). The command must complete its local checks in under 1 second; network-dependent scans (vulnerability databases) are cached by default with `--scan` for fresh results.

Phase 9 already defined `qsdev devenv doctor` for system-level diagnostics (OS, tools, package managers). `qsdev status` is project-level — it reports on the security posture of the current project, not the system. They share the `internal/doctor/` package's check infrastructure but serve different audiences and answer different questions.

**Desired Outcome:** A developer runs `qsdev status` and instantly sees: overall score with letter grade, conformance status (baseline/enhanced PASS/FAIL), defense layer checklist with per-layer status, config health summary, and dependency vulnerability counts. Running `qsdev status --verbose` expands every section with per-check detail and remediation hints. Running `qsdev status --json` produces a complete `PostureReport` for CI consumption. Running `qsdev status --audit-level high` exits non-zero if any high-or-above findings exist, gating CI builds.

**Steps:**

1. Create `internal/posture/` package as the home for all posture assessment logic. This package contains the core types, scoring engine, and report aggregation — separate from the CLI command wiring in the addon.

2. Define the top-level `PostureReport` struct that serves as the canonical data model. All output formats (text, JSON, SARIF, badge) derive from this single struct:
   ```go
   type PostureReport struct {
       SchemaVersion string              `json:"schemaVersion"`  // "1.0.0" — semver, bump on breaking changes
       GeneratedAt   time.Time           `json:"generatedAt"`
       GdevVersion   string              `json:"gdevVersion"`
       ProjectPath   string              `json:"projectPath"`
       ProjectName   string              `json:"projectName"`

       Score         AggregateScore      `json:"score"`
       Conformance   ConformanceResult   `json:"conformance"`
       Defense       DefenseCoverage     `json:"defense"`
       Config        ConfigHealth        `json:"config"`
       Dependencies  DependencyHealth    `json:"dependencies"`
       Drift         DriftReport         `json:"drift"`
       Tools         []ToolStatus        `json:"tools"`
       Ecosystems    []EcosystemStatus   `json:"ecosystems"`
   }
   ```

3. Define `AggregateScore` with dual representation (numeric + letter grade):
   ```go
   type AggregateScore struct {
       Total     float64 `json:"total"`     // 0-100 weighted aggregate
       Grade     string  `json:"grade"`     // "A+", "A", "A-", "B+", ..., "F"
       Defense   float64 `json:"defense"`   // 0-100, weighted 40%
       Config    float64 `json:"config"`    // 0-100, weighted 30%
       DepHealth float64 `json:"depHealth"` // 0-100, weighted 30%
   }
   ```
   Grade scale: A (90-100), B (80-89), C (70-79), D (60-69), F (<60). Plus/minus modifiers within each range (e.g., A+: 97-100, A: 93-96, A-: 90-92).

4. Implement the `Assess(ctx context.Context, projectDir string, opts AssessOptions) (*PostureReport, error)` function — the single entry point that orchestrates all assessment layers:
   ```go
   type AssessOptions struct {
       FreshScan    bool          // --scan: run fresh vuln scans instead of using cache
       AuditLevel   string        // --audit-level: none|info|low|moderate|high|critical
       PolicyFile   string        // path to .gdev-policy.yaml for custom conformance definitions
       CacheDir     string        // .gdev/cache/ for scan result caching
       CacheTTL     time.Duration // default 24h, auto-scan if cache older
   }
   ```
   The function reads the tool registry (Phase 12), state file (`.gdev/state.yaml`), project config (`.qsdev.yaml`), runs drift detection, collects ecosystem health, computes scores, evaluates conformance, and assembles the `PostureReport`.

5. Register `qsdev status` as a Cobra command in the devinit addon with the following flag set:
   ```
   qsdev status                        # Default: colored summary
   qsdev status --verbose              # Full detail per check with remediation
   qsdev status --quiet                # Score and grade only (one line)
   qsdev status --json                 # Complete PostureReport JSON
   qsdev status --sarif                # SARIF 2.1.0 findings
   qsdev status --format badge         # shields.io endpoint JSON
   qsdev status --fix                  # Remediation commands only
   qsdev status --scan                 # Run fresh vulnerability scans
   qsdev status --audit-level <level>  # Exit code gate: none|info|low|moderate|high|critical
   qsdev status defense                # Defense coverage section only
   qsdev status config                 # Config health section only
   qsdev status deps                   # Dependency health section only
   qsdev status tools                  # Tool inventory view
   ```

6. Implement the default text renderer in `internal/posture/render_text.go`:
   - Header: `qsdev status -- Project Security Posture`
   - Score line: `Score: 82/100 (B+)` in bold
   - Conformance line: `Conformance: baseline PASS, enhanced FAIL`
   - Defense Coverage section: hierarchical list with indicators:
     - `[✓]` (green) = fully enabled/healthy
     - `[~]` (yellow) = partially enabled or degraded
     - `[ ]` (dim) = disabled or not applicable
     - `[✗]` (red) = misconfigured, broken, or critical issue
   - Config Health section: file count summary, outdated/missing flagged
   - Dependency Health section: vulnerability counts by severity, lock file status, last scan time
   - Footer: `Run 'qsdev status --verbose' for details.`

7. Implement `--verbose` rendering: expand each defense layer with per-ecosystem detail, expand each config file with hash status and version, expand each ecosystem with individual vulnerability advisories. Include inline remediation hints (e.g., "Fix: Run `qsdev enable container-security`").

8. Implement `--quiet` rendering: single line output `82/100 B+` suitable for scripting and badge generators. Exit code reflects audit-level threshold.

9. Implement color support:
   - Detect terminal capability via `os.Getenv("TERM")` and `isatty` check on stdout.
   - Respect `NO_COLOR` environment variable (per no-color.org convention).
   - Respect `FORCE_COLOR` environment variable for CI environments that strip `isatty`.
   - Use `github.com/fatih/color` or equivalent — already a transitive dependency via charmbracelet libraries.
   - JSON/SARIF/badge output never includes ANSI color codes.

10. Implement exit code strategy:
    ```
    Exit 0: No findings at or above --audit-level threshold (default: none, meaning always 0)
    Exit 1: Findings at or above --audit-level threshold
    Exit 2: gdev itself failed (not initialized, corrupt state, missing config)
    ```
    The `--audit-level` flag maps severity strings to the posture model: `critical` = any critical vulns, `high` = any high+ vulns or baseline conformance FAIL, `moderate` = any moderate+ vulns, `low` = any findings, `info` = any informational findings, `none` = always exit 0, `any` = alias for `info`.

11. Implement the `tools` subcommand view (`qsdev status tools`): show all tools from the Phase 12 tool registry grouped by category (security, ai-agent, devex, infrastructure), with enabled/disabled/not-applicable status, one-line description, and config file path. Include "run `qsdev enable <tool>`" hints for available-but-disabled tools.

12. Implement caching for dependency health results:
    - Store scan results in `.gdev/cache/vuln-scan.json` (gitignored).
    - Include `lastScan` timestamp in output.
    - Auto-trigger fresh scan if cache older than `CacheTTL` (default 24h).
    - `--scan` flag bypasses cache and runs fresh ecosystem audit tools.
    - If offline (scan fails), display cached results with staleness warning.

13. Handle edge cases:
    - **First run before any tools enabled:** Display "not initialized" state with `qsdev init` suggestion, exit 2.
    - **Partially initialized project:** Show available data, mark missing sections as "unknown."
    - **Corrupt state file:** Attempt graceful degradation, warn about corruption, suggest `qsdev init --rebuild-state`.
    - **CI environment detection:** When `CI=true`, default to `--json` output behavior (no colors, structured output).

14. Write unit tests for:
    - Score calculation from known inputs (verify weighted aggregate).
    - Grade assignment at boundary values (89 = B+, 90 = A-, etc.).
    - Exit code selection for each `--audit-level` value.
    - Text rendering with mock `PostureReport` data.
    - Color stripping when `NO_COLOR` is set.
    - Subcommand routing (`defense`, `config`, `deps`, `tools`).

**Acceptance Criteria:**
- [ ] `qsdev status` displays colored hierarchical summary completing in <1s for local checks
- [ ] `qsdev status --verbose` shows per-check detail with remediation hints for every failing check
- [ ] `qsdev status --quiet` outputs single-line score suitable for scripting
- [ ] `qsdev status --json` produces valid JSON matching the `PostureReport` schema
- [ ] `qsdev status --audit-level high` exits 1 when high-severity findings exist, 0 otherwise
- [ ] `qsdev status defense` shows only the defense coverage section
- [ ] `qsdev status tools` shows tool inventory grouped by category with enable/disable hints
- [ ] `NO_COLOR=1 qsdev status` produces output with no ANSI escape codes
- [ ] `qsdev status` exits 2 when run outside a gdev-initialized project
- [ ] Cached vulnerability results displayed with staleness timestamp; `--scan` forces refresh
- [ ] Performance: defense + config checks complete in <500ms, full report (with cached deps) in <1s

**Research Citations:**
- `research-spikes/gdev-health-reporting/status-command-ux-research.md` -- progressive disclosure hierarchy, terminal mockups, flag inventory, exit code strategy, color coding, performance model
- `research-spikes/gdev-health-reporting/compliance-posture-model-research.md § Data Model (Go Types)` -- PostureReport, AggregateScore, ConformanceResult struct definitions
- `research-spikes/gdev-health-reporting/prior-art-research.md § Doctor Command Pattern` -- flutter doctor, brew doctor, rustup check UX analysis
- `research-spikes/gdev-health-reporting/prior-art-research.md § Universal Patterns` -- JSON output, severity levels, exit codes, summary-then-detail, remediation hints
- `phases/09-cross-platform-system-detection.md § Unit 9.5` -- `qsdev devenv doctor` design (system-level diagnostics, complementary to project-level `qsdev status`)

**Status:** Not Started

---

### Unit 15.2: Compliance Posture Scoring Engine

**Description:** Implement the three-layer posture scoring engine that evaluates defense coverage, configuration health, and dependency health into a weighted 0-100 aggregate score with letter grade and dual-track conformance labels (baseline/enhanced PASS/FAIL).

**Context:** The posture model must answer two distinct questions for two distinct audiences. Engineers want a nuanced score ("how well-defended is this project?") — this is the 0-100 numeric score with letter grade, analogous to OpenSSF Scorecard's per-check scoring. Compliance stakeholders want a binary answer ("does this project meet our minimum standard?") — this is the conformance track (baseline PASS/FAIL, enhanced PASS/FAIL), following Scorecard v6's evolution toward conformance evaluation.

The three layers are independently useful: defense coverage reveals which security tools are active, configuration health detects drift from generated baselines, and dependency health surfaces known vulnerabilities. The weighted aggregate (defense 40%, config 30%, deps 30%) reflects that defense coverage is gdev's core value proposition — a project with all defenses enabled but some moderate vulns is in better shape than a project with zero defenses and zero vulns (because zero vulns is transient).

A critical design principle from the research: scoring must distinguish "disabled by choice" from "disabled by oversight." Container security disabled because no Dockerfile exists is not a penalty. Container security disabled in a project with a Dockerfile is a gap. The `not-applicable` status handles this.

**Desired Outcome:** Given a project's tool registry state, file hashes, and ecosystem scan results, the scoring engine produces an `AggregateScore` and `ConformanceResult` that accurately reflect the project's security posture. The score is deterministic — the same inputs always produce the same output. Conformance definitions are configurable via `.gdev-policy.yaml` but ship with strong defaults.

**Steps:**

1. Create `internal/posture/scoring.go` with the core scoring functions.

2. Implement defense coverage scoring. Each defense layer has a weight and status:
   ```go
   type DefenseLayer struct {
       Name    string `json:"name"`      // "age-gating", "script-blocking", etc.
       Status  string `json:"status"`    // "enabled" | "partial" | "disabled" | "not-applicable"
       Weight  string `json:"weight"`    // "critical" | "high" | "medium" | "low"
       Score   int    `json:"score"`     // 0-10, like Scorecard per-check scores
       Details string `json:"details,omitempty"` // "npm: 72h, pip: 72h"
       Reason  string `json:"reason,omitempty"`  // "No Dockerfile detected"
   }
   ```
   Weight multipliers follow Scorecard's model: critical=10, high=7.5, medium=5, low=2.5.

3. Define the 10 defense layers with their default weights:
   | Layer | Weight | Source |
   |-------|--------|--------|
   | age-gating | high (7.5) | Phase 5 package manager hardening |
   | install-script-blocking | high (7.5) | Phase 5 @lavamoat/allow-scripts |
   | lock-file-enforcement | high (7.5) | Phase 5 lock file configs |
   | vulnerability-scanning | high (7.5) | Phase 5 OSV Scanner |
   | pretooluse-hooks | critical (10) | Phase 4 attach-guard |
   | nix-hardening | medium (5) | Phase 5 nix.conf settings |
   | sast | medium (5) | Phase 12 Semgrep |
   | secrets-scanning | medium (5) | Phase 12 Gitleaks + ripsecrets |
   | container-security | medium (5) | Phase 12 Grype/Syft/Cosign |
   | license-compliance | low (2.5) | Phase 12 ScanCode |

4. Implement the defense score calculation:
   ```go
   func computeDefenseScore(layers []DefenseLayer) float64 {
       var totalWeight, earnedWeight float64
       for _, layer := range layers {
           if layer.Status == "not-applicable" {
               continue // Don't penalize for inapplicable layers
           }
           w := weightMultiplier(layer.Weight)
           totalWeight += w
           switch layer.Status {
           case "enabled":
               earnedWeight += w
           case "partial":
               earnedWeight += w * float64(layer.Score) / 10.0
           case "disabled":
               // 0 earned
           }
       }
       if totalWeight == 0 {
           return 100.0 // No applicable layers = perfect (edge case)
       }
       return (earnedWeight / totalWeight) * 100.0
   }
   ```

5. Implement defense layer assessment functions. Each function queries the tool registry and state file:
   - `assessAgeGating()`: Check per-ecosystem age-gate configs exist (`.npmrc`, `pip.conf`, etc.). Score 10 if all detected ecosystems have age-gating, proportional if partial.
   - `assessScriptBlocking()`: Check `@lavamoat/allow-scripts` or equivalent is configured. Binary: 10 or 0.
   - `assessLockFileEnforcement()`: Check lock file enforcement configs per ecosystem. Score proportional to coverage.
   - `assessVulnScanning()`: Check OSV Scanner or equivalent is configured and running.
   - `assessPreToolUseHooks()`: Check attach-guard installed, count active vs total rules from settings.json deny list.
   - `assessNixHardening()`: Check nix.conf for the 10 hardening settings from devenv-security spike. Score = settings present / 10 * 10.
   - `assessSAST()`: Check Semgrep enabled via tool registry.
   - `assessSecretsScanning()`: Check Gitleaks and/or ripsecrets enabled.
   - `assessContainerSecurity()`: If Docker ecosystem detected, check Grype/Syft/Cosign enabled. If no Docker, status = `not-applicable`.
   - `assessLicenseCompliance()`: Check ScanCode enabled.

6. Implement configuration health scoring in `internal/posture/config_health.go`:
   ```go
   type ConfigFileStatus struct {
       Path             string `json:"path"`
       State            string `json:"state"`     // "current" | "modified" | "outdated" | "missing" | "corrupt"
       Category         string `json:"category"`  // "machine-owned" | "human-edited" | "exclusive"
       GeneratedVersion string `json:"generatedVersion,omitempty"`
       LatestVersion    string `json:"latestVersion,omitempty"`
       HashMatch        bool   `json:"hashMatch"`
       UpdateAvailable  bool   `json:"updateAvailable,omitempty"`
   }

   type ConfigHealth struct {
       Score float64            `json:"score"`
       Total int                `json:"total"`
       Current int              `json:"current"`
       Modified int             `json:"modified"`   // User-modified (expected for human-edited)
       Outdated int             `json:"outdated"`
       Missing int              `json:"missing"`
       Files []ConfigFileStatus `json:"files"`
   }
   ```
   Scoring: `current` = full credit, `modified` on human-edited file = full credit (expected), `modified` on machine-owned file = 50% credit (unexpected drift), `outdated` = 50% credit (still functional), `missing` = 0% credit, `corrupt` = 0% credit. Score = earned / total * 100.

7. Implement dependency health scoring in `internal/posture/dep_health.go`:
   ```go
   type EcosystemStatus struct {
       Name       string             `json:"name"`      // "npm", "pip", "go"
       Detected   bool               `json:"detected"`
       LockFile   string             `json:"lockFile"`   // "valid" | "missing" | "corrupt" | "stale"
       VulnCounts VulnSeverityCounts `json:"vulnCounts"`
       AgeGate    string             `json:"ageGate,omitempty"` // "72h" or ""
       LastScan   *time.Time         `json:"lastScan,omitempty"`
   }

   type VulnSeverityCounts struct {
       Critical int `json:"critical"`
       High     int `json:"high"`
       Moderate int `json:"moderate"`
       Low      int `json:"low"`
       Info     int `json:"info"`
   }

   type DependencyHealth struct {
       Score      float64            `json:"score"`
       Ecosystems []EcosystemStatus  `json:"ecosystems"`
       Totals     VulnSeverityCounts `json:"totals"`
   }
   ```
   Scoring formula: Start at 100, deduct per vulnerability. Critical = -25 each (capped), High = -10 each, Moderate = -3 each, Low = -1 each. Missing lock file = -15 per ecosystem. Floor at 0. This heavily penalizes critical/high vulns while tolerating moderate/low findings that are common in real projects.

8. Implement the weighted aggregate calculation:
   ```go
   func computeAggregateScore(defense, config, deps float64) AggregateScore {
       total := defense*0.40 + config*0.30 + deps*0.30
       return AggregateScore{
           Total:     math.Round(total*10) / 10, // One decimal place
           Grade:     scoreToGrade(total),
           Defense:   math.Round(defense*10) / 10,
           Config:    math.Round(config*10) / 10,
           DepHealth: math.Round(deps*10) / 10,
       }
   }
   ```

9. Implement conformance evaluation in `internal/posture/conformance.go`:
   ```go
   type ConformanceResult struct {
       Baseline ConformanceLevel `json:"baseline"`
       Enhanced ConformanceLevel `json:"enhanced"`
       Custom   *ConformanceLevel `json:"custom,omitempty"` // From .gdev-policy.yaml
   }

   type ConformanceLevel struct {
       Pass   bool                `json:"pass"`
       Checks []ConformanceCheck  `json:"checks"`
   }

   type ConformanceCheck struct {
       Name   string `json:"name"`
       Pass   bool   `json:"pass"`
       Reason string `json:"reason,omitempty"`
   }
   ```

10. Define baseline conformance requirements (all must pass):
    - Lock files present for all detected ecosystems
    - Pre-commit hooks installed
    - No critical vulnerabilities
    - CLAUDE.md generated sections present
    - settings.json deny rules present
    - All high-weight defense layers enabled (or not-applicable)

11. Define enhanced conformance requirements (consulting firm standard):
    - All baseline requirements pass
    - Age-gating configured for all supported ecosystems
    - Zero high-severity vulnerabilities
    - SAST enabled (Semgrep)
    - Secrets scanning enabled (Gitleaks)
    - License compliance enabled (ScanCode)
    - CI workflows generated

12. Support custom conformance via `.gdev-policy.yaml`:
    ```yaml
    conformance:
      custom:
        name: "client-acme-standard"
        requirements:
          - name: "container-security-enabled"
            check: "defense.container-security.status == enabled"
          - name: "zero-moderate-vulns"
            check: "dependencies.totals.moderate == 0"
    ```
    Custom conformance is optional — projects without a policy file get only baseline and enhanced tracks.

13. Write comprehensive unit tests:
    - Defense score: 10/10 enabled = 100, 5/10 enabled = proportional, 0/10 = 0, all not-applicable = 100.
    - Config health: all current = 100, mix of states = correct proportional score.
    - Dependency health: 0 vulns = 100, escalating deductions, floor at 0.
    - Aggregate: verify 40/30/30 weighting.
    - Conformance: verify baseline/enhanced pass/fail with known inputs.
    - Grade boundaries: 89.5 rounds to 90 = A-, 89.4 rounds to 89 = B+.
    - Edge case: project with no ecosystems detected (deps = 100 by default).

**Acceptance Criteria:**
- [ ] Defense coverage score correctly weights 10 layers with critical/high/medium/low multipliers
- [ ] `not-applicable` layers excluded from scoring (don't penalize for absent Docker when no Dockerfile)
- [ ] `partial` status scores proportionally (e.g., 46/48 attach-guard rules = partial credit)
- [ ] Config health correctly distinguishes machine-owned modification (penalty) from human-edited modification (expected)
- [ ] Dependency health deductions are severity-proportional: critical >> high >> moderate >> low
- [ ] Weighted aggregate uses 40/30/30 split producing deterministic results
- [ ] Baseline conformance fails if any high-weight defense is disabled (not just not-applicable)
- [ ] Enhanced conformance requires SAST, secrets scanning, and license compliance
- [ ] Grade assignment is correct at all boundary values
- [ ] Custom conformance from `.gdev-policy.yaml` is evaluated when present
- [ ] Score is deterministic: same inputs always produce same output

**Research Citations:**
- `research-spikes/gdev-health-reporting/compliance-posture-model-research.md` -- three-layer model, weighted scoring, conformance tracks, full Go type definitions, grade scale
- `research-spikes/gdev-health-reporting/prior-art-research.md § OpenSSF Scorecard` -- weighted scoring model (critical 10x, high 7.5x, medium 5x, low 2.5x), dual-track evaluation
- `research-spikes/gdev-health-reporting/compliance-posture-model-research.md § Tradeoffs and Limitations` -- scoring subjectivity, false sense of security, conformance definition as policy
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` -- tool registry as data source for defense layer and tool inventory assessment

**Status:** Not Started

---

### Unit 15.3: Drift Detection Engine

**Description:** Implement the six-category drift detection engine that identifies configuration drift across all gdev-managed files, tools, and hooks. All detection is local-only and completes in under 100ms for a typical project.

**Code-Grounded Implementation Note:** SHA-256 hash-based drift detection ALREADY exists via `state.CheckModified()` at `internal/state/state.go:46-94`, which returns `map[string]FileStatus`. The `FileStatus` struct at `internal/state/state.go:14-20` contains Path, Status (a `ModificationStatus` enum), Error, StoredHash, and CurrentHash fields. This fully covers file modification drift (Category 1 below). The remaining 5 categories (version drift, tool availability, section markers, lock files, pre-commit hooks) are new implementation. For devenv.nix specifically: markers are informal comments (not formal delimiters) — use hash-only detection for devenv.nix, NOT section marker checking. The `toolcheck.Detect()` function is already available for tool availability checking (Category 3).

**Context:** After `qsdev init` generates config files, multiple forms of drift can occur: someone manually edits a machine-owned file, gdev releases a new version with updated defaults, new tools become available, pre-commit hooks get uninstalled, lock files become stale, or section markers get accidentally deleted during editing. The migration strategy design (from the gdev-extension-design spike) established SHA256 hash tracking in `.gdev/state.yaml` as the foundation — this unit builds the full detection engine on top of that foundation.

Drift detection is the mechanism that makes the compliance posture score meaningful over time. Without it, a project could score 90/100 at `qsdev init` and silently degrade to 50/100 as configs are modified, tools are uninstalled, and dependencies accumulate vulnerabilities. The drift engine transforms a point-in-time snapshot into ongoing posture monitoring.

All six categories are deliberately local-only operations — no network calls, no database queries, no external services. This means drift detection works offline, in air-gapped environments, and completes fast enough to run on every `qsdev status` invocation without caching.

**Desired Outcome:** `qsdev status` includes a drift report identifying every deviation from the last-known-good state. Each drift finding has a category, severity, description, and remediation suggestion. The drift report feeds into the config health score (Unit 15.2) and the SARIF output (Unit 15.5).

**Steps:**

1. Create `internal/posture/drift.go` with the drift detection engine types:
   ```go
   type DriftReport struct {
       Categories []DriftCategory  `json:"categories"`
       TotalFindings int           `json:"totalFindings"`
       BySeverity  map[string]int  `json:"bySeverity"` // "critical": 0, "error": 1, "warning": 2, "info": 5
   }

   type DriftCategory struct {
       Name     string          `json:"name"`     // "file-modification", "version", "tool-availability", etc.
       Findings []DriftFinding  `json:"findings"`
   }

   type DriftFinding struct {
       Category    string `json:"category"`
       Severity    string `json:"severity"`    // "critical" | "error" | "warning" | "info"
       Subject     string `json:"subject"`     // file path, tool name, marker ID, etc.
       Description string `json:"description"` // human-readable description
       Expected    string `json:"expected,omitempty"` // expected value/hash
       Actual      string `json:"actual,omitempty"`   // actual value/hash
       Remediation string `json:"remediation,omitempty"` // "Run: qsdev update"
       AutoFixable bool   `json:"autoFixable"` // can gdev fix this automatically?
   }
   ```

2. Implement **Category 1: File Modification Drift** in `internal/posture/drift_files.go`:
   - Read `.gdev/state.yaml` to get stored SHA256 hashes for all tracked files.
   - Compute current SHA256 hash for each tracked file.
   - Compare current hash against stored hash.
   - Severity assignment by file category:
     - Machine-owned file modified: `warning` (unexpected modification, auto-fixable via `qsdev update`).
     - Human-edited file with intact section markers: `info` (expected modification).
     - Human-edited file with missing/broken section markers: `warning` (generated sections may be lost).
     - Tracked file missing entirely: `error` (defense may be inactive).
     - File exists but no stored hash (pre-hash-tracking): `info` with suggestion to run `qsdev init --rebuild-state`.
   - Performance: one `os.ReadFile` + `sha256.Sum256` per file. For a typical project with 10-15 tracked files, <10ms total.

3. Implement **Category 2: Version Drift** in `internal/posture/drift_version.go`:
   - Read `gdev_version` from `.gdev/state.yaml` (the version that generated the project config).
   - Compare against the running gdev binary's embedded version.
   - If versions differ, report `info` severity with the version delta.
   - Embed a per-version config-change manifest in the gdev binary so the report can say "Changes in v1.2.0: new defense layer (license-compliance), updated Semgrep rules, new pre-commit hook (ripsecrets)."
   - Suggest `qsdev update` with preview.
   - Performance: string comparison, <1ms.

4. Implement **Category 3: Tool Availability Drift** in `internal/posture/drift_tools.go`:
   - Read enabled tools from `.gdev/state.yaml`.
   - For each enabled tool, check that its binary is available on `$PATH` via `exec.LookPath`.
   - Report `warning` for tools that are enabled in qsdev config but whose binaries are missing (e.g., user removed Semgrep from their Nix profile).
   - Also check for newly available tools: re-run detection heuristics and compare against enabled set. Report `info` for tools that are now applicable but weren't when `qsdev init` ran (e.g., Dockerfile added since initialization → container-security now applicable).
   - Performance: one `exec.LookPath` per enabled tool. For 16 tools, <50ms total.

5. Implement **Category 4: Section Marker Integrity** in `internal/posture/drift_markers.go`:
   - For human-edited files with section markers (CLAUDE.md, devenv.nix), parse for expected marker pairs.
   - CLAUDE.md markers: `<!-- gdev:<tool> -->` ... `<!-- /gdev:<tool> -->`
   - devenv.nix markers: `# --- <tool> ---` ... `# --- end <tool> ---`
   - Report `warning` for:
     - Opening marker present but closing marker missing.
     - Closing marker present but opening marker missing.
     - Expected marker pair absent entirely (tool enabled but markers removed).
   - Report `info` for markers present and well-formed.
   - Suggest `qsdev update --repair-markers` for auto-fix.
   - Performance: one file read + regex scan per tracked human-edited file. <10ms.

6. Implement **Category 5: Lock File Drift** in `internal/posture/drift_lockfiles.go`:
   - For each detected ecosystem, check manifest-to-lock-file relationship:
     - `valid`: lock file exists and has modification time >= manifest modification time.
     - `stale`: manifest modified after lock file (dependencies may have changed without lock update).
     - `missing`: manifest exists but no lock file.
     - `corrupt`: lock file exists but fails basic integrity validation (e.g., not valid JSON for package-lock.json, not valid TOML for Cargo.lock).
   - Manifest-to-lockfile pairs: `package.json`→`package-lock.json`, `pyproject.toml`→`uv.lock`/`poetry.lock`/`requirements.txt`, `go.mod`→`go.sum`, `Cargo.toml`→`Cargo.lock`, `pom.xml`→none (Maven has no lock), `*.csproj`→`packages.lock.json`.
   - Severity: `missing` = `error`, `stale` = `warning`, `corrupt` = `error`, `valid` = no finding.
   - Performance: `os.Stat` calls only (no file content reads needed for freshness check). <5ms.

7. Implement **Category 6: Pre-Commit Hook Drift** in `internal/posture/drift_hooks.go`:
   - Check `.git/hooks/pre-commit` exists and is executable.
   - Verify the hook file references the expected runner (`prek` for devenv 1.11+, `pre-commit` for older).
   - Compare configured hooks (from `.pre-commit-config.yaml` or devenv hook config) against what's expected for enabled tools.
   - Check `.git/hooks/commit-msg` exists if commitlint is enabled.
   - Severity: hooks not installed = `warning` (auto-fixable via `qsdev hooks install`), hook runner mismatch = `info`, configured hooks missing = `warning`.
   - Performance: 2-3 file existence checks + one file read. <20ms.

8. Implement the drift engine orchestrator:
   ```go
   func DetectDrift(ctx context.Context, projectDir string, state *GeneratedState, registry *ToolRegistry) (*DriftReport, error) {
       report := &DriftReport{}
       // Run all 6 categories — they're independent and could be parallel,
       // but sequential is fine at <100ms total
       report.Categories = append(report.Categories, detectFileModificationDrift(projectDir, state))
       report.Categories = append(report.Categories, detectVersionDrift(state))
       report.Categories = append(report.Categories, detectToolAvailabilityDrift(state, registry))
       report.Categories = append(report.Categories, detectMarkerIntegrity(projectDir, state))
       report.Categories = append(report.Categories, detectLockFileDrift(projectDir, state))
       report.Categories = append(report.Categories, detectHookDrift(projectDir, state))
       // Aggregate severity counts
       for _, cat := range report.Categories {
           for _, f := range cat.Findings {
               report.TotalFindings++
               report.BySeverity[f.Severity]++
           }
       }
       return report, nil
   }
   ```

9. Define the state file schema that drift detection reads from. This extends the existing `GeneratedState` from Phase 1/8:
   ```yaml
   # .gdev/state.yaml
   gdev_version: "1.2.0"
   initialized_at: "2026-05-12T14:30:00Z"
   last_update: "2026-05-12T14:30:00Z"
   profile: "consulting-default"

   tools_enabled:
     - semgrep
     - gitleaks
     - attach-guard

   files:
     devenv.yaml:
       hash: "sha256:abc123..."
       generated_at: "2026-05-12T14:30:00Z"
       category: "machine-owned"
     devenv.nix:
       hash: "sha256:def456..."
       generated_at: "2026-05-12T14:30:00Z"
       category: "human-edited"
       markers:
         - "# --- semgrep ---"
         - "# --- gitleaks ---"
     CLAUDE.md:
       hash: "sha256:789abc..."
       generated_at: "2026-05-12T14:30:00Z"
       category: "human-edited"
       markers:
         - "<!-- gdev:semgrep -->"
         - "<!-- gdev:gitleaks -->"

   ecosystems_detected:
     - name: "npm"
       manifest: "package.json"
       lockfile: "package-lock.json"
     - name: "go"
       manifest: "go.mod"
       lockfile: "go.sum"
   ```

10. Write unit tests with fixture data:
    - File modification: create temp files, modify one, verify detection.
    - Version drift: mock state with older version, verify finding.
    - Tool availability: mock `exec.LookPath` to return error for one tool.
    - Marker integrity: create CLAUDE.md with one missing closing marker.
    - Lock file drift: set manifest mtime after lock file mtime.
    - Hook drift: create `.git/hooks/` with missing `pre-commit`.
    - Performance: benchmark all 6 categories combined, assert <100ms.

**Acceptance Criteria:**
- [ ] File modification drift detects machine-owned file changes and reports as `warning`
- [ ] File modification drift reports human-edited changes as `info` (not penalizing expected edits)
- [ ] Missing tracked files detected as `error` severity
- [ ] Version drift reports gdev version mismatch with config-change summary
- [ ] Tool availability drift detects enabled tools whose binaries are missing from PATH
- [ ] Tool availability drift reports newly applicable tools (e.g., Dockerfile added) as `info`
- [ ] Section marker integrity detects orphaned opening/closing markers
- [ ] Lock file drift detects stale lock files (manifest newer than lock)
- [ ] Lock file drift detects missing lock files as `error`
- [ ] Pre-commit hook drift detects uninstalled hooks
- [ ] All 6 categories combined complete in <100ms on a typical project
- [ ] Every finding includes a remediation suggestion and auto-fixable flag
- [ ] Drift findings feed into the config health score from Unit 15.2

**Research Citations:**
- `research-spikes/gdev-health-reporting/drift-detection-research.md` -- six drift categories, detection performance table, state file schema, remediation strategies, comparison to Terraform/Kubernetes drift detection
- `research-spikes/gdev-extension-design/migration-strategy-design.md` -- SHA256 hash tracking foundation, file categories (machine-owned/human-edited/exclusive), section markers
- `research-spikes/gdev-health-reporting/drift-detection-research.md § State Storage` -- .gdev/state.yaml schema with files, markers, and ecosystems_detected

**Status:** Not Started

---

### Unit 15.4: `qsdev evidence` Command & Compliance Evidence Generation

**Description:** Implement the `qsdev evidence` command that generates compliance evidence reports mapping gdev's defense layers and tool configurations to specific regulatory control frameworks (SOC2 Trust Service Criteria, HIPAA Security Rule), producing machine-readable evidence artifacts suitable for client audit submissions.

**Context:** A consulting firm managing client projects needs to demonstrate security controls during audits. Currently, this requires manually documenting which security tools are in place, what they do, and how they map to compliance requirements. The `qsdev evidence` command automates this by introspecting the project's posture report and generating control mapping documents.

This is distinct from `qsdev status` (which answers "how healthy is my project?") — `qsdev evidence` answers "what controls can I demonstrate for an auditor?" The evidence command consumes the same `PostureReport` data but reshapes it through the lens of specific compliance frameworks.

The ASVS mapping from the machine-readable output research provides the pattern: gdev defense layers map to specific compliance controls, and the posture data provides evidence that those controls are active. The evidence report includes: control identifier, control description, gdev defense layer(s) that address it, current status, and supporting evidence (tool configs, scan results, timestamps).

**Desired Outcome:** `qsdev evidence --framework soc2` generates a JSON report mapping gdev's defenses to SOC2 Trust Service Criteria, with each mapping including the current posture status as evidence. `qsdev evidence --framework soc2 --format markdown` produces a human-readable report suitable for inclusion in audit documentation.

**Steps:**

1. Create `internal/evidence/` package with framework mapping types:
   ```go
   type EvidenceReport struct {
       SchemaVersion string             `json:"schemaVersion"` // "1.0.0"
       GeneratedAt   time.Time          `json:"generatedAt"`
       GdevVersion   string             `json:"gdevVersion"`
       ProjectName   string             `json:"projectName"`
       Framework     string             `json:"framework"`     // "soc2" | "hipaa" | "asvs"
       FrameworkVer  string             `json:"frameworkVersion"` // "2017" for SOC2, "5.0" for ASVS
       Summary       EvidenceSummary    `json:"summary"`
       Controls      []ControlMapping   `json:"controls"`
       Posture       *PostureReport     `json:"posture"`       // Full posture report as evidence
   }

   type EvidenceSummary struct {
       TotalControls    int     `json:"totalControls"`
       AddressedFully   int     `json:"addressedFully"`
       AddressedPartial int     `json:"addressedPartially"`
       NotAddressed     int     `json:"notAddressed"`
       NotApplicable    int     `json:"notApplicable"`
       CoveragePercent  float64 `json:"coveragePercent"`
   }

   type ControlMapping struct {
       ControlID    string            `json:"controlId"`     // "CC6.1", "164.312(a)(1)"
       ControlName  string            `json:"controlName"`   // "Logical and Physical Access Controls"
       ControlDesc  string            `json:"controlDesc"`   // Full control description
       Category     string            `json:"category"`      // "Access Control", "Change Management"
       Status       string            `json:"status"`        // "addressed" | "partial" | "not-addressed" | "not-applicable"
       GdevLayers   []LayerEvidence   `json:"gdevLayers"`    // Which gdev defenses address this
       Artifacts    []EvidenceArtifact `json:"artifacts"`     // Supporting evidence files
       Notes        string            `json:"notes,omitempty"`
   }

   type LayerEvidence struct {
       LayerName   string `json:"layerName"`    // "pretooluse-hooks"
       Status      string `json:"status"`       // from defense layer assessment
       Relevance   string `json:"relevance"`    // "primary" | "supporting"
       Description string `json:"description"`  // How this layer addresses the control
   }

   type EvidenceArtifact struct {
       Type        string `json:"type"`         // "config-file" | "scan-result" | "tool-version"
       Path        string `json:"path"`         // Relative file path
       Description string `json:"description"`
       Hash        string `json:"hash,omitempty"` // SHA256 for integrity
       Timestamp   string `json:"timestamp,omitempty"`
   }
   ```

2. Implement SOC2 Trust Service Criteria mapping in `internal/evidence/soc2.go`. Map gdev defense layers to relevant SOC2 criteria:
   - **CC6.1 (Logical and Physical Access Controls):** pretooluse-hooks (primary — controls what AI agents can install), nix-hardening (supporting — restricts eval-based code execution).
   - **CC6.6 (System Boundary Protection):** install-script-blocking (primary), age-gating (supporting — quarantine reduces exposure window).
   - **CC6.8 (Malicious Code Prevention):** secrets-scanning (primary), sast (primary), vulnerability-scanning (primary), age-gating (supporting).
   - **CC7.1 (Detection of Anomalies and Events):** sast (primary — detects code-level security issues), secrets-scanning (primary — detects credential exposure).
   - **CC7.2 (Security Event Monitoring):** CI workflow generation (supporting — automated security checks on every commit).
   - **CC8.1 (Change Management):** lock-file-enforcement (primary — ensures reproducible builds), pretooluse-hooks (supporting — controls dependency changes via AI).
   - **CC8.2 (Configuration Management):** nix-hardening (primary), drift detection results (supporting — detects configuration deviation).
   - **CC8.3 (Testing of Changes):** sast (primary), vulnerability-scanning (supporting — catches known-vuln introductions), pre-commit hooks (supporting).

3. Implement HIPAA Security Rule mapping in `internal/evidence/hipaa.go` for projects handling PHI:
   - **164.312(a)(1) (Access Control):** pretooluse-hooks, nix-hardening.
   - **164.312(b) (Audit Controls):** drift detection, CI workflows, scan result logging.
   - **164.312(c)(1) (Integrity):** lock-file-enforcement, hash tracking, section marker integrity.
   - **164.312(d) (Person or Entity Authentication):** Not directly addressed by gdev (note as N/A with recommendation for upstream SSO/MFA).
   - **164.312(e)(1) (Transmission Security):** Not directly addressed by gdev (note as N/A).

4. Implement OWASP ASVS v5.0 mapping in `internal/evidence/asvs.go`:
   - **10.3 (Dependency Integrity):** age-gating, lock-file-enforcement, vulnerability-scanning.
   - **14.2 (Dependency Security):** vulnerability-scanning, sast, container-security.
   - **1.14 (Configuration):** nix-hardening, drift detection, config health.

5. Register `qsdev evidence` as a Cobra command:
   ```
   gdev evidence --framework soc2               # SOC2 evidence, JSON output
   gdev evidence --framework hipaa              # HIPAA evidence, JSON output
   gdev evidence --framework asvs               # OWASP ASVS evidence
   gdev evidence --framework soc2 --format md   # Markdown report for humans
   gdev evidence --framework soc2 --format json # JSON report (default)
   gdev evidence --list-frameworks              # Show available frameworks
   ```

6. Implement the markdown renderer for evidence reports:
   ```markdown
   # SOC2 Compliance Evidence Report
   ## Project: client-app
   ## Generated: 2026-05-12T14:30:00Z
   ## gdev Version: 1.2.0

   ### Summary
   | Metric | Value |
   |--------|-------|
   | Total Controls Mapped | 8 |
   | Fully Addressed | 6 |
   | Partially Addressed | 1 |
   | Not Addressed | 0 |
   | Not Applicable | 1 |
   | Coverage | 87.5% |

   ### CC6.1 — Logical and Physical Access Controls
   **Status:** Addressed

   **gdev Defenses:**
   - pretooluse-hooks (primary): 46/48 PreToolUse rules active via attach-guard
   - nix-hardening (supporting): 10/10 Nix evaluation restrictions applied

   **Evidence Artifacts:**
   - `.claude/settings.json` — deny rules configuration (SHA256: abc123...)
   - `.claude/hooks/package-guard.py` — PreToolUse hook script
   ```

7. Include posture artifacts as evidence: list all config files with their SHA256 hashes and timestamps, tool versions, scan result timestamps. These serve as point-in-time evidence that controls were active.

8. Add a disclaimer to all evidence output: "This report documents gdev-managed security controls. It does not constitute a complete SOC2/HIPAA assessment. Organizational, physical, and procedural controls are outside gdev's scope."

9. Write tests:
   - SOC2 mapping completeness: all defined control IDs have valid mappings.
   - Evidence status derivation: fully enabled defenses = "addressed," partially enabled = "partial."
   - Markdown rendering: verify output contains expected sections and formatting.
   - Framework selection: `--framework invalid` produces clear error.

**Acceptance Criteria:**
- [ ] `qsdev evidence --framework soc2` produces valid JSON with control mappings
- [ ] Each SOC2 control mapping includes specific gdev defense layers with current status
- [ ] Evidence artifacts include file hashes and timestamps for audit trail
- [ ] `qsdev evidence --framework soc2 --format md` produces readable markdown
- [ ] HIPAA mapping correctly marks network/authentication controls as N/A
- [ ] ASVS mapping covers Chapter 10 (Malicious Code) and Chapter 14 (Configuration)
- [ ] Evidence summary shows coverage percentage (addressed / total applicable)
- [ ] `qsdev evidence --list-frameworks` shows available frameworks
- [ ] Disclaimer included in all output formats
- [ ] Evidence report includes the full PostureReport as supporting data

**Research Citations:**
- `research-spikes/gdev-health-reporting/machine-readable-output-research.md § OWASP ASVS Alignment` -- ASVS chapter mappings, compliance report format, audit evidence approach
- `research-spikes/gdev-health-reporting/compliance-posture-model-research.md § Consulting Engineer User Stories` -- "Generate audit evidence" user story
- `research-spikes/gdev-health-reporting/compliance-posture-model-research.md § Conformance Track (PASS/FAIL)` -- baseline/enhanced requirements as compliance checks

**Status:** Not Started

---

### Unit 15.5: Machine-Readable Output & Badge Generation

**Description:** Implement the machine-readable output pipeline: versioned JSON schema, SARIF 2.1.0 for GitHub Code Scanning integration, and shields.io-compatible badge JSON generation with multiple badge variants.

**Code-Grounded Implementation Note:** The three-state-file problem affects this unit. Badge generation and JSON output need to aggregate state from `.devinit/.qsdev-init-state.yaml`, `.devenv/.gdev-state.yaml`, and `.claude/.gdev-claude-state.yaml`. The `LoadAllStates()` aggregation function introduced in Unit 15.1 is a prerequisite here — all renderers consume the unified `PostureReport` which draws from all three state sources.

**Context:** The PostureReport struct (Unit 15.1) is the canonical data model. This unit implements the serialization layer that transforms it into formats consumed by CI pipelines (JSON), security platforms (SARIF), and visual indicators (badges). Three lessons from prior art drive the design: (1) version the JSON schema from day one — cargo-audit's unstable JSON broke downstream tools; (2) never paywall machine-readable output — Safety CLI's JSON-behind-API-key blocks automation; (3) SARIF maps discrete findings, not aggregate scores — use SARIF for individual drift/vulnerability findings, JSON for the full posture.

**Desired Outcome:** `qsdev status --json` produces schema-versioned JSON that downstream tools can parse reliably across gdev versions. `qsdev status --sarif` produces valid SARIF 2.1.0 that GitHub Code Scanning accepts. `qsdev status --format badge` produces shields.io-compatible JSON. CI workflows can generate, upload, and consume all three formats.

**Steps:**

1. Create `internal/posture/render_json.go` with the JSON serialization:
   ```go
   func RenderJSON(report *PostureReport) ([]byte, error) {
       report.SchemaVersion = "1.0.0" // Always set, never let callers override
       return json.MarshalIndent(report, "", "  ")
   }
   ```
   Schema versioning rules:
   - **Major bump** (2.0.0): removing or renaming fields, changing field types, changing scoring algorithm.
   - **Minor bump** (1.1.0): adding new fields (consumers must ignore unknown fields).
   - **Patch bump** (1.0.1): documentation changes, bug fixes that don't change output shape.
   - Embed `schemaVersion` at the top level of every JSON output.

2. Document the JSON schema in a Go source file (`internal/posture/schema.go`) with comments explaining every field, valid value ranges, and stability guarantees. Include a `SchemaChangeLog` string constant tracking schema evolution.

3. Create `internal/posture/render_sarif.go` implementing SARIF 2.1.0 output. SARIF carries discrete findings, not aggregate posture:
   ```go
   type SARIFLog struct {
       Schema  string     `json:"$schema"`
       Version string     `json:"version"`     // "2.1.0"
       Runs    []SARIFRun `json:"runs"`
   }

   type SARIFRun struct {
       Tool    SARIFTool     `json:"tool"`
       Results []SARIFResult `json:"results"`
   }

   type SARIFTool struct {
       Driver SARIFDriver `json:"driver"`
   }

   type SARIFDriver struct {
       Name           string      `json:"name"`            // "gdev"
       Version        string      `json:"version"`
       InformationURI string      `json:"informationUri"`
       Rules          []SARIFRule `json:"rules"`
   }
   ```

4. Define SARIF rule IDs and map posture findings to SARIF results:
   | Posture Finding | SARIF Rule ID | SARIF Level | Location |
   |----------------|---------------|-------------|----------|
   | Defense layer disabled (applicable) | `gdev/defense-disabled` | warning | `.gdev/state.yaml` |
   | Defense layer partial | `gdev/defense-partial` | warning | `.gdev/state.yaml` |
   | Config file missing | `gdev/config-missing` | error | expected file path |
   | Config file outdated | `gdev/config-outdated` | note | file path |
   | Config file unexpectedly modified | `gdev/config-modified` | warning | file path |
   | Critical vulnerability | `gdev/vuln-critical` | error | lock file path |
   | High vulnerability | `gdev/vuln-high` | error | lock file path |
   | Lock file missing | `gdev/lockfile-missing` | error | manifest file path |
   | Lock file stale | `gdev/lockfile-stale` | warning | lock file path |
   | Pre-commit hooks not installed | `gdev/hooks-not-installed` | warning | `.git/hooks/` |
   | Section markers broken | `gdev/markers-broken` | warning | file path |
   | Tool binary missing | `gdev/tool-unavailable` | warning | `.gdev/state.yaml` |

5. Implement the SARIF result mapper:
   ```go
   func RenderSARIF(report *PostureReport) ([]byte, error) {
       log := SARIFLog{
           Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
           Version: "2.1.0",
       }
       run := SARIFRun{
           Tool: SARIFTool{Driver: SARIFDriver{
               Name: "gdev", Version: report.GdevVersion,
               Rules: allSARIFRules(),
           }},
       }
       // Map defense findings
       for _, layer := range report.Defense.Layers {
           if layer.Status == "disabled" && layer.Status != "not-applicable" {
               run.Results = append(run.Results, defenseToSARIF(layer))
           }
       }
       // Map drift findings
       for _, cat := range report.Drift.Categories {
           for _, finding := range cat.Findings {
               run.Results = append(run.Results, driftToSARIF(finding))
           }
       }
       // Map vulnerability findings
       for _, eco := range report.Ecosystems {
           if eco.VulnCounts.Critical > 0 || eco.VulnCounts.High > 0 {
               run.Results = append(run.Results, vulnsToSARIF(eco))
           }
       }
       log.Runs = []SARIFRun{run}
       return json.MarshalIndent(log, "", "  ")
   }
   ```

6. Validate SARIF output against the SARIF 2.1.0 JSON Schema. Include the `$schema` field so tools can validate independently. Test that GitHub Code Scanning accepts the output by validating against the schema in unit tests.

7. Create `internal/posture/render_badge.go` implementing shields.io endpoint badge format:
   ```go
   type BadgeJSON struct {
       SchemaVersion int    `json:"schemaVersion"` // Always 1
       Label         string `json:"label"`
       Message       string `json:"message"`
       Color         string `json:"color"`
   }

   func RenderBadge(report *PostureReport, variant string) ([]byte, error) {
       switch variant {
       case "score", "":
           return renderScoreBadge(report)
       case "conformance":
           return renderConformanceBadge(report)
       case "defense":
           return renderDefenseBadge(report)
       default:
           return nil, fmt.Errorf("unknown badge variant: %s", variant)
       }
   }
   ```

8. Implement three badge variants:
   - **Score badge:** `{"schemaVersion": 1, "label": "gdev security", "message": "82/100 B+", "color": "green"}`
   - **Conformance badge:** `{"schemaVersion": 1, "label": "gdev baseline", "message": "PASS", "color": "brightgreen"}` (or "FAIL" with "red")
   - **Defense badge:** `{"schemaVersion": 1, "label": "defenses", "message": "8/10 enabled", "color": "green"}`

9. Implement color mapping from score to shields.io color:
   ```go
   func scoreToColor(score float64) string {
       switch {
       case score >= 90: return "brightgreen"
       case score >= 80: return "green"      // Note: research said 75, but 80 aligns with B grade
       case score >= 70: return "yellow"
       case score >= 60: return "orange"
       default:          return "red"
       }
   }
   ```

10. Support `--format badge --badge-type <variant>` flag and `--all-badges --output-dir <dir>` for generating all variants at once:
    ```
    qsdev status --format badge                               # Score badge (default)
    qsdev status --format badge --badge-type conformance      # Baseline conformance
    qsdev status --format badge --badge-type defense          # Defense coverage
    qsdev status --format badge --all-badges --output-dir .gdev/badges/  # All at once
    ```

11. Wire the output format selection into the `qsdev status` command from Unit 15.1. The `--json`, `--sarif`, and `--format badge` flags are mutually exclusive — only one output format per invocation (except `--all-badges`).

12. Write tests:
    - JSON: round-trip marshal/unmarshal, verify `schemaVersion` always present.
    - SARIF: validate output against SARIF 2.1.0 JSON Schema, verify rule IDs, verify result levels.
    - Badge: verify color mapping at boundary values (89 = green, 90 = brightgreen), verify all three variants produce valid JSON.
    - Schema versioning: verify the `SchemaVersion` constant matches expected value.

**Acceptance Criteria:**
- [ ] `qsdev status --json` produces JSON with `schemaVersion` field at top level
- [ ] JSON output is deterministic (same report always produces identical JSON, ignoring timestamps)
- [ ] `qsdev status --sarif` produces valid SARIF 2.1.0 that passes schema validation
- [ ] SARIF maps disabled defenses, config drift, and vulnerabilities as discrete findings
- [ ] SARIF does NOT include aggregate scores (scores don't belong in SARIF)
- [ ] SARIF includes `informationUri` and `helpUri` for each rule
- [ ] `qsdev status --format badge` produces shields.io-compatible JSON
- [ ] Badge color mapping: score 90+ = brightgreen, 80-89 = green, 70-79 = yellow, 60-69 = orange, <60 = red
- [ ] Conformance badge shows PASS (brightgreen) or FAIL (red)
- [ ] Defense badge shows count (e.g., "8/10 enabled")
- [ ] `--all-badges` generates all three variants to output directory
- [ ] All output formats handle empty/minimal PostureReport gracefully (no panics on zero-value fields)

**Research Citations:**
- `research-spikes/gdev-health-reporting/machine-readable-output-research.md` -- JSON schema design, SARIF rule mapping, badge JSON format, consumer matrix, format priority
- `research-spikes/gdev-health-reporting/prior-art-research.md § Anti-Patterns` -- unstable JSON schema (cargo-audit), paywalled JSON (Safety CLI), exit code inconsistency (govulncheck)
- `research-spikes/gdev-health-reporting/badge-generation-research.md` -- shields.io endpoint protocol, color mapping table, generation methods (static file recommended), multiple badge variants
- `research-spikes/gdev-health-reporting/machine-readable-output-research.md § SARIF 2.1.0` -- SARIF structure, what maps/doesn't map to SARIF, rule definitions

**Status:** Not Started

---

### Unit 15.6: Team Aggregation Pipeline & CI Integration

**Description:** Implement the team-level CI artifact aggregation pipeline that collects per-project posture reports across multiple repositories and generates a markdown dashboard with score tables, trend tracking, and auto-generated GitHub issues for posture degradation.

**Context:** A consulting firm managing 10-50 client projects needs organizational visibility into security posture. The research evaluated three architecture options: CI artifact aggregation (recommended), git-based collection (scorecard-monitor pattern), and push-based webhook (DefectDojo pattern). CI artifact aggregation was selected because it requires no new infrastructure, uses existing GitHub Actions, and scales to the firm's project count.

The pipeline works in two stages. Stage 1: each project's CI generates `qsdev status --json > posture.json` and uploads it as a build artifact (per-project, already handled by Unit 15.1/15.5 JSON output). Stage 2: a central aggregation repository runs a scheduled workflow that collects posture artifacts from all tracked repos via `gh run download`, aggregates them, generates a markdown dashboard, and optionally creates GitHub issues for degraded projects.

This unit implements Stage 2 — the aggregation logic, dashboard generation, issue creation, and the CI workflow template that ties it together. The `qsdev team-report` command handles the aggregation locally; the generated GitHub Actions workflow automates it.

**Desired Outcome:** An engineering lead runs `qsdev team-report --input-dir reports/` against a directory of per-project posture JSONs and gets a markdown dashboard showing all projects' scores, conformance status, vulnerability counts, and trend data. A generated GitHub Actions workflow automates this on a weekly schedule, creating issues when projects degrade.

**Steps:**

1. Create `internal/teamreport/` package with aggregation types:
   ```go
   type TeamReport struct {
       SchemaVersion   string            `json:"schemaVersion"` // "1.0.0"
       GeneratedAt     time.Time         `json:"generatedAt"`
       Summary         TeamSummary       `json:"summary"`
       Projects        []ProjectSummary  `json:"projects"`
       Trends          []ProjectTrend    `json:"trends,omitempty"`
       Alerts          []PostureAlert    `json:"alerts"`
   }

   type TeamSummary struct {
       ProjectCount      int     `json:"projectCount"`
       AverageScore      float64 `json:"averageScore"`
       MedianScore       float64 `json:"medianScore"`
       BaselinePassRate  float64 `json:"baselinePassRate"`  // 0.0-1.0
       EnhancedPassRate  float64 `json:"enhancedPassRate"`
       TotalCriticalVulns int    `json:"totalCriticalVulns"`
       TotalHighVulns    int     `json:"totalHighVulns"`
       ProjectsNeedUpdate int    `json:"projectsNeedingUpdate"`
   }

   type ProjectSummary struct {
       Name         string         `json:"name"`
       Repo         string         `json:"repo,omitempty"`
       Score        AggregateScore `json:"score"`
       Conformance  ConformanceResult `json:"conformance"`
       VulnTotals   VulnSeverityCounts `json:"vulnTotals"`
       GdevVersion  string         `json:"gdevVersion"`
       LastScan     time.Time      `json:"lastScan"`
   }

   type ProjectTrend struct {
       Project    string       `json:"project"`
       DataPoints []TrendPoint `json:"dataPoints"`
   }

   type TrendPoint struct {
       Date  string  `json:"date"`  // "2026-05-12"
       Score float64 `json:"score"`
   }

   type PostureAlert struct {
       Project  string `json:"project"`
       Severity string `json:"severity"` // "critical" | "high" | "medium"
       Message  string `json:"message"`
       Action   string `json:"action,omitempty"`
   }
   ```

2. Implement the aggregation engine in `internal/teamreport/aggregate.go`:
   ```go
   func Aggregate(reports []*PostureReport) (*TeamReport, error) {
       team := &TeamReport{
           SchemaVersion: "1.0.0",
           GeneratedAt:   time.Now(),
       }
       // Sort projects by score ascending (worst first for attention)
       // Compute summary statistics
       // Generate alerts for degraded projects
       // Load historical data for trends if available
       return team, nil
   }
   ```

3. Implement alert generation rules:
   - **Critical alert:** Any project with critical vulnerabilities.
   - **High alert:** Baseline conformance FAIL, or score dropped >10 points from previous report.
   - **Medium alert:** Score dropped >5 points, or project running outdated gdev version (>2 minor versions behind).
   - Alerts include remediation actions: "Run `qsdev update` in project X", "Run `npm audit fix` in project Y."

4. Implement trend tracking in `internal/teamreport/trends.go`:
   - Store historical data in a JSON file (`team-posture-history.json`) committed to the aggregation repo.
   - On each aggregation run, append current scores to history.
   - Retain 90 days of weekly snapshots (configurable).
   - Trend data enables "Score Changes This Week" table in the dashboard.
   ```go
   type HistoryStore struct {
       SchemaVersion string                    `json:"schemaVersion"`
       Entries       map[string][]TrendPoint   `json:"entries"` // project name -> data points
   }

   func (h *HistoryStore) Append(projects []ProjectSummary) {
       today := time.Now().Format("2006-01-02")
       for _, p := range projects {
           h.Entries[p.Name] = append(h.Entries[p.Name], TrendPoint{Date: today, Score: p.Score.Total})
       }
       h.prune(90 * 24 * time.Hour) // Keep 90 days
   }
   ```

5. Implement the markdown dashboard renderer in `internal/teamreport/render_md.go`:
   ```markdown
   # Team Security Posture -- 2026-05-12

   ## Overview
   | Metric | Value |
   |--------|-------|
   | Projects tracked | 12 |
   | Average score | 79/100 (B) |
   | Median score | 82/100 (B+) |
   | Baseline pass rate | 83% (10/12) |
   | Enhanced pass rate | 50% (6/12) |
   | Total critical vulns | 0 |
   | Total high vulns | 7 |

   ## Project Scores
   | Project | Score | Grade | Baseline | Enhanced | Vulns (C/H) | gdev Version | Last Scan |
   |---------|-------|-------|----------|----------|-------------|-------------|-----------|
   | client-a-api | 92 | A | PASS | PASS | 0/0 | 1.2.0 | 1h ago |
   | internal-tools | 65 | C | FAIL | FAIL | 0/1 | 1.1.0 | 12h ago |

   ## Attention Required
   ### High Priority
   - **internal-tools**: Baseline FAIL -- pre-commit hooks not installed
   - **client-c-monorepo**: Baseline FAIL -- lock file missing for Python

   ## Score Changes (Last 7 Days)
   | Project | Previous | Current | Change |
   |---------|----------|---------|--------|
   | client-a-api | 88 | 92 | +4 |
   | internal-tools | 72 | 65 | -7 |
   ```
   Sort projects by score ascending in the "Attention Required" section (worst first), by score descending in the main table (best first).

6. Implement GitHub issue generation in `internal/teamreport/issues.go`:
   ```go
   type IssueSpec struct {
       Title     string
       Body      string
       Labels    []string
       Assignee  string
       Repo      string
   }

   func GenerateIssues(report *TeamReport, threshold float64) []IssueSpec {
       var issues []IssueSpec
       for _, alert := range report.Alerts {
           if alert.Severity == "critical" || alert.Severity == "high" {
               issues = append(issues, alertToIssue(alert, report))
           }
       }
       return issues
   }
   ```
   Issue title format: `[gdev] Security posture degraded: <project> (<score>/100, <delta>)`.
   Issue body includes: score comparison table, specific findings, recommended actions, labels `security` and `gdev-posture`.

7. Register `qsdev team-report` as a Cobra command:
   ```
   qsdev team-report --input-dir reports/          # Aggregate from directory of posture JSONs
   qsdev team-report --scope scope.json            # Use scope file listing repos to collect from
   qsdev team-report --format md                   # Markdown output (default)
   qsdev team-report --format json                 # JSON output
   qsdev team-report --threshold 75                # Alert on projects below this score
   qsdev team-report --trend                       # Include trend data from history file
   qsdev team-report --create-issues               # Create GitHub issues for alerts (requires gh CLI)
   qsdev team-report --history-file history.json   # Path to trend history file
   ```

8. Implement scope file support for CI-based collection:
   ```json
   {
     "projects": [
       {"repo": "org/client-a-api", "branch": "main"},
       {"repo": "org/client-b-app", "branch": "develop"},
       {"repo": "org/internal-tools", "branch": "main"}
     ]
   }
   ```
   When `--scope` is provided, the command uses `gh run download` to collect the latest `gdev-posture` artifact from each listed repo, then aggregates.

9. Generate a GitHub Actions workflow template via `qsdev team-report --generate-workflow`:
   ```yaml
   # .github/workflows/team-posture.yml
   name: Team Security Posture
   on:
     schedule:
       - cron: '0 6 * * 1'  # Weekly Monday 6am UTC
     workflow_dispatch:

   permissions:
     contents: write
     issues: write
     actions: read

   jobs:
     aggregate:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@<sha-pinned>
         - name: Install gdev
           run: curl -sSfL https://get.gdev.dev | sh
         - name: Collect posture reports
           env:
             GH_TOKEN: ${{ secrets.POSTURE_PAT }}
           run: |
             mkdir -p reports
             while IFS= read -r repo; do
               gh run download --repo "$repo" --name gdev-posture -D "reports/$(basename $repo)" || echo "No artifact for $repo"
             done < scope.txt
         - name: Aggregate
           run: qsdev team-report --input-dir reports/ --trend --history-file team-posture-history.json > team-posture.md
         - name: Create issues for degraded projects
           env:
             GH_TOKEN: ${{ secrets.POSTURE_PAT }}
           run: qsdev team-report --input-dir reports/ --create-issues --threshold 70
         - name: Commit dashboard
           run: |
             git add team-posture.md team-posture-history.json
             git diff --cached --quiet || git commit -m "Update team security posture dashboard"
             git push
   ```
   Action references are SHA-pinned. The workflow uses a PAT (`POSTURE_PAT`) with read access to all tracked repos' actions artifacts.

10. Also generate the per-project CI step template for uploading posture artifacts:
    ```yaml
    # Add to each project's CI workflow
    - name: Generate security posture
      run: qsdev status --json > posture.json
    - name: Upload posture artifact
      uses: actions/upload-artifact@<sha-pinned>
      with:
        name: gdev-posture
        path: posture.json
        retention-days: 90
    - name: Update badge
      run: qsdev status --format badge > .gdev/badge.json
    - name: CI gate
      run: qsdev status --audit-level high
    ```

11. Handle scaling considerations:
    - 10 projects: direct `gh run download` loop works fine (<30s).
    - 50 projects: parallelize downloads with `xargs -P4` or Go goroutines.
    - Rate limiting: GitHub REST API allows 5,000 requests/hour with PAT. 50 projects = 50 API calls per aggregation run, well within limits.
    - Stale artifacts: flag projects whose latest artifact is >7 days old as "stale scan" in the dashboard.

12. Write tests:
    - Aggregation: verify summary stats from known inputs (average, median, pass rates).
    - Alert generation: verify critical/high/medium thresholds.
    - Trend tracking: verify history append and 90-day pruning.
    - Markdown rendering: verify table formatting, sort order.
    - Issue generation: verify title/body format.
    - Scope file parsing: verify repo list extraction.
    - Empty input: verify graceful handling when no reports are found.

**Acceptance Criteria:**
- [ ] `qsdev team-report --input-dir reports/` aggregates multiple posture JSONs into a summary
- [ ] Markdown dashboard includes overview table, project scores, attention-required section, and trend data
- [ ] Projects sorted by score ascending in attention section (worst first)
- [ ] Alert generation fires for baseline FAIL and score drops >10 points
- [ ] `--create-issues` generates GitHub issues via `gh` CLI for high-severity alerts
- [ ] Issue title follows format: `[gdev] Security posture degraded: <project> (<score>/100, <delta>)`
- [ ] Trend history stores 90 days of weekly snapshots
- [ ] `--generate-workflow` produces valid GitHub Actions YAML with SHA-pinned actions
- [ ] Per-project CI step template includes posture generation, artifact upload, badge update, and CI gate
- [ ] Scope file collection via `gh run download` handles missing artifacts gracefully
- [ ] Aggregation handles 50+ project reports without performance degradation
- [ ] Stale artifacts (>7 days) flagged in dashboard

**Research Citations:**
- `research-spikes/gdev-health-reporting/team-reporting-research.md` -- CI artifact aggregation architecture, markdown dashboard design, JSON aggregation format, GitHub issue generation pattern, scaling considerations
- `research-spikes/gdev-health-reporting/team-reporting-research.md § Prior Art: Multi-Repo Aggregation Tools` -- scorecard-monitor pattern (JSON database + markdown reports + issue generation), DefectDojo (normalize/deduplicate), GitLab Security Dashboard (zero-config via CI)
- `research-spikes/gdev-health-reporting/badge-generation-research.md § Method 1: Static File in Repo` -- CI workflow for badge generation and commit
- `research-spikes/gdev-health-reporting/machine-readable-output-research.md § Format Selection Matrix` -- JSON as primary format for dashboards and aggregation

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] `qsdev status` displays correct posture for a project with 8/10 defenses enabled, 2 high vulns, 1 outdated config
- [ ] `qsdev status --json | jq .score.total` returns a numeric value matching the terminal display
- [ ] `qsdev status --sarif` accepted by GitHub Code Scanning (validated against SARIF 2.1.0 schema)
- [ ] `qsdev status --audit-level high` correctly gates CI builds (exit 1 when high vulns present, exit 0 otherwise)
- [ ] `qsdev status --format badge` generates shields.io-compatible JSON with correct color for the score range
- [ ] Drift detection identifies all 6 categories and completes in <100ms
- [ ] `qsdev evidence --framework soc2` maps defense layers to SOC2 controls with current status
- [ ] `qsdev team-report` aggregates 10+ project posture JSONs into readable markdown dashboard
- [ ] Score is deterministic: running `qsdev status` twice with no changes produces identical scores
- [ ] All output formats handle edge cases: uninitialized project (exit 2), zero defenses, no ecosystems
- [ ] Performance: `qsdev status` completes in <1s for local checks, <5s with `--scan`
- [ ] JSON schema version is "1.0.0" and documented in source
