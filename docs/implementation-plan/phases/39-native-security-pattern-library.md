# Phase 39: Native Security Pattern Library

## Goal

Implement a native Go library of security primitives within gdev, drawing on the 23 borrowable design patterns identified by the security-tooling-evaluation-gdev spike. These patterns — extracted from Prempti, npm-scan, reasoning-core, Cloudberry, and Sense — cover runtime policy enforcement, package risk assessment, security analysis pipelines, and MCP server trust scoring. All implementations are native Go with zero external tool dependencies, consistent with gdev's single-binary, zero-prerequisites architecture. Optional integration adapters allow delegation to Prempti (runtime policy) and Sense (MCP registry intelligence) when those tools are installed, controlled via `.gdev.yaml` feature flags.

## Dependencies

Phase 5 complete (security infrastructure — package manager hardening configs, pre-commit hooks, CI scanning workflows provide the enforcement points this phase's analysis feeds into). Phase 12 complete (tool lifecycle management — `gdev enable/disable` infrastructure, shared-file surgery, the tool registry that this phase registers policy engine and adapters into). Phase 32 complete (managed hook policy — JSONL audit trail, hook deployment infrastructure, and the 3-tier enforcement model that this phase's policy engine extends with declarative YAML rules).

## Phase Outputs

- `internal/security/policy/` — YAML-based policy engine with rule evaluation, severity tiers, versioning, and built-in policy sets
- `internal/security/pkgrisk/` — Multi-registry package risk assessment with publication age, maintainer analysis, and weighted risk scoring
- `internal/security/pipeline/` — Security finding data model, scan-categorize-score-report pipeline, and SARIF 2.1.0 output
- `internal/security/mcptrust/` — MCP server trust scoring algorithm with capability and permission analysis
- `internal/security/adapters/` — Optional Prempti and Sense integration adapters with feature-flag switching
- Built-in policy sets embedded via `embed.FS`: `default.yaml`, `strict.yaml`, `permissive.yaml`
- `gdev policy lint/check/list` commands for policy management
- `gdev status --security` integration (feeds Phase 15 posture reporting)

---

### Unit 39.1: Policy Engine Core

**Description:** Implement a YAML-based declarative policy engine in Go that evaluates security rules against tool operations, project state, and configuration. The engine borrows Prempti's rule-based enforcement model (YAML definitions, severity-tiered evaluation, tag-based verdict resolution) but runs entirely within the gdev binary — no Falco daemon, no IPC, no external process.

**Context:** Prempti demonstrates that a rule engine with YAML-based policy definitions, severity tiers, and three-outcome verdicts (allow/deny/ask) is a powerful enforcement primitive. However, Prempti requires Falco as a runtime dependency (heavy infrastructure, no Nix package, fail-closed daemon risk). gdev needs the same declarative rule capability without the infrastructure weight. The policy engine also incorporates npm-scan's policy-as-code pattern: context-aware suppressions with unsuppressible safety guards for lifecycle hooks, and reasoning-core's shadow-mode calibration for non-blocking rollout of new rules.

Phase 32 already deploys 6 shell-script hooks with hardcoded patterns. This unit provides the underlying engine that allows those patterns (and new ones) to be expressed as data (YAML rules) rather than code (shell scripts), enabling user customization, policy versioning, and audit trail integration without modifying hook scripts.

**Desired Outcome:** A developer can define security policies in `.gdev.yaml` or standalone YAML files, gdev evaluates operations against those policies with severity-aware verdicts, and the evaluation results integrate with the Phase 32 JSONL audit trail. Built-in policy sets cover common postures without requiring any user-authored YAML.

**Steps:**

1. Create `internal/security/policy/` package with core types:
   ```go
   // Rule represents a single policy rule evaluated against an operation context.
   type Rule struct {
       ID          string            `yaml:"id"`          // e.g., "SPR-001" (Security Policy Rule)
       Name        string            `yaml:"name"`
       Description string            `yaml:"description"`
       Category    Category          `yaml:"category"`    // boundary, sensitive_paths, sandbox, threats, mcp, persistence, self_protection
       Severity    Severity          `yaml:"severity"`    // critical, high, medium, low, info
       Verdict     Verdict           `yaml:"verdict"`     // deny, ask, allow (default: deny)
       Match       MatchSpec         `yaml:"match"`       // conditions that trigger this rule
       Tags        []string          `yaml:"tags"`
       Enabled     bool              `yaml:"enabled"`
   }

   type MatchSpec struct {
       Tools       []string          `yaml:"tools"`       // tool names: Bash, Write, Edit, etc.
       Patterns    []string          `yaml:"patterns"`    // regex patterns against tool input
       Paths       []string          `yaml:"paths"`       // glob patterns for file paths
       Commands    []string          `yaml:"commands"`    // regex patterns for command strings
       Negate      bool              `yaml:"negate"`      // invert match (allowlist pattern)
   }

   type Severity int
   const (
       SeverityInfo Severity = iota
       SeverityLow
       SeverityMedium
       SeverityHigh
       SeverityCritical
   )

   type Verdict int
   const (
       VerdictAllow Verdict = iota
       VerdictAsk
       VerdictDeny
   )
   ```

2. Implement the rule evaluation engine in `internal/security/policy/engine.go`:
   - Load rules from YAML files (embedded built-in sets + user-defined files)
   - Evaluate an `OperationContext` (tool name, input, file paths, working directory) against all enabled rules
   - Apply Prempti's escalation model: when multiple rules match, deny > ask > allow
   - Resolve canonical paths before matching (symlink resolution per Prempti's raw/real path pair pattern) to prevent traversal bypass
   - Return `EvaluationResult` with matched rules, final verdict, and LLM-friendly explanation string
   - Support shadow mode: evaluate and log but return allow regardless (per reasoning-core's calibration pattern)

3. Implement policy versioning in `internal/security/policy/version.go`:
   - Each policy set carries a semver version string
   - `PolicyVersion` constraint in `.gdev.yaml` allows pinning: `security.policy_version: ">=1.2.0 <2.0.0"`
   - Version mismatch produces a warning in `gdev status`, not a hard failure

4. Create three embedded built-in policy sets in `internal/security/policy/builtin/`:
   - `default.yaml` — Production-reasonable defaults: block credential access, destructive commands, sensitive path writes; ask for operations outside working directory. Maps to Phase 32's existing hook patterns expressed as data.
   - `strict.yaml` — All `default` rules plus: deny all network operations, deny writes to any dotfile directory, deny MCP config modifications, deny agent settings changes (self-protection per Prempti pattern)
   - `permissive.yaml` — Reduced set: only block credential exfiltration and destructive infrastructure commands. For trusted environments or experienced developers who want minimal friction.

5. Implement context-aware suppression system (borrowed from npm-scan's policy engine):
   - Suppression rules in `.gdev.yaml` under `security.suppressions[]` with required `reason` field
   - Context qualifiers: file path glob, tool name, environment (CI vs local)
   - Unsuppressible safety guards: rules tagged `safety_guard: true` cannot be suppressed regardless of user configuration (e.g., credential exfiltration blocking)
   - Suppression audit: every suppression is logged to the Phase 32 JSONL audit trail with the reason

6. Integrate with gdev config resolution:
   - Policy selection in `.gdev.yaml`: `security.policy: default | strict | permissive | <path-to-custom.yaml>`
   - Local override support: project `.gdev.yaml` can extend (not replace) the selected policy set
   - Binary defaults apply when no `.gdev.yaml` exists
   - Client profile integration: Phase 30's `GDEV_CLIENT_PROFILE` can specify a policy set per client

7. Implement `gdev policy` subcommands:
   - `gdev policy list` — show all active rules with severity and verdict
   - `gdev policy check <operation>` — dry-run evaluation of a hypothetical operation
   - `gdev policy lint <file>` — validate a custom policy YAML file

**Acceptance Criteria:**
- [ ] Policy engine loads and evaluates YAML rules with <5ms latency per evaluation (no external process, no IPC)
- [ ] Three built-in policy sets (`default`, `strict`, `permissive`) are embedded and selectable via `.gdev.yaml`
- [ ] Rule evaluation implements deny > ask > allow escalation when multiple rules match
- [ ] Canonical path resolution prevents symlink traversal bypass (test with symlinked paths)
- [ ] Shadow mode evaluates and logs without blocking (returns allow for all verdicts)
- [ ] Context-aware suppressions work with unsuppressible safety guards enforced
- [ ] `gdev policy list/check/lint` commands produce correct output
- [ ] Policy versioning with semver constraints warns on mismatch
- [ ] All evaluations emit JSONL audit entries compatible with Phase 32 audit trail schema
- [ ] LLM-friendly explanation strings are included in deny/ask verdicts (agent can adapt behavior)

**Research Citations:**
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` Section 2.3 — tag-based verdict resolution, escalation model, fail-closed design
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` Section 3 — 7-domain rule taxonomy (boundary, sensitive_paths, sandbox, threats, MCP, persistence, self_protection)
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` Section 5.3 — canonical path resolution, ask verdict, monitor mode, LLM-friendly output
- `research-spikes/security-tooling-evaluation-gdev/npm-scan-research.md` Section 1.5 — policy-as-code engine with context-aware suppressions and unsuppressible safety guards
- `research-spikes/security-tooling-evaluation-gdev/reasoning-core-research.md` Section 4C — shadow-mode calibration, escape hatch design
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 4, patterns 1-4, 7-8, 10-11, 19

**Status:** Not Started

---

### Unit 39.2: Package Risk Assessment Primitives

**Description:** Implement a multi-registry package metadata fetcher and risk scoring engine that evaluates individual packages against supply chain attack signals. The scoring model uses publication age as the primary signal (92% of malicious packages are caught within 24 hours of publication), augmented by maintainer count, download volume, and version history analysis.

**Context:** npm-scan's detection architecture is fundamentally shallow (9 of 11 detectors are single-regex matchers on concatenated source), but the signals it targets are real and validated by industry data. Socket CLI is the recommended external tool for deep behavioral analysis (70+ risk types, funded team). This unit builds the lightweight, zero-dependency complement: fast metadata checks that can run locally without API keys or cloud services. These primitives feed into Phase 5's package manager hardening (age-gating configs) and Phase 15's posture reporting (dependency health scoring).

The key insight from the npm-scan evaluation: package metadata signals (age, maintainer count, download count) are cheap to compute and high-signal. The regex-based source code analysis is neither — gdev delegates that to Socket CLI via Phase 12's tool lifecycle. This unit focuses on the metadata layer only.

**Desired Outcome:** A developer can run `gdev pkg check <package>` to get a risk assessment of any package across supported registries. The risk score feeds into `gdev status --security` for project-level dependency health. CI integration via `gdev pkg check --lockfile` scans all dependencies in the project's lockfile.

**Steps:**

1. Create `internal/security/pkgrisk/` package with registry client interfaces:
   ```go
   // RegistryClient fetches package metadata from a specific ecosystem registry.
   type RegistryClient interface {
       FetchMetadata(ctx context.Context, name, version string) (*PackageMetadata, error)
       Name() string // "npm", "pypi", "crates.io", etc.
   }

   type PackageMetadata struct {
       Name              string
       Version           string
       Registry          string
       FirstPublished    time.Time
       VersionPublished  time.Time
       MaintainerCount   int
       Maintainers       []string
       DownloadCount     int64      // last 30 days where available
       VersionCount      int
       LatestVersion     string
       License           string
       Repository        string     // source repo URL
       HasInstallScripts bool       // preinstall/postinstall/etc.
       DepCount          int        // direct dependency count
   }
   ```

2. Implement registry clients for Tier 1 ecosystems:
   - **npm**: `https://registry.npmjs.org/<package>` — full metadata endpoint, no auth required
   - **PyPI**: `https://pypi.org/pypi/<package>/json` — JSON API, no auth required
   - **crates.io**: `https://crates.io/api/v1/crates/<package>` — JSON API, requires User-Agent
   - **Go modules**: `https://proxy.golang.org/<module>/@v/list` + `<version>.info` — no auth required
   - All clients: configurable timeout (default 5s), respect rate limits, cache responses in `~/.cache/gdev/pkg/` with 1-hour TTL

3. Implement the publication age calculator:
   - Compute days since the specific version was first published (not the package — the version)
   - Flag packages published within configurable threshold (default: 7 days, matching gdev's age-gating configs)
   - Extreme risk flag for packages published within 24 hours (92% malware detection window per industry data)

4. Implement maintainer analysis:
   - Single-maintainer flag (bus factor = 1)
   - Maintainer change detection: compare current maintainer list against previous version's maintainers (npm's `maintainers` field provides this)
   - New maintainer on existing package: elevated risk signal (account takeover vector)

5. Implement risk score computation:
   ```go
   type RiskScore struct {
       Overall     float64        // 0.0 (safe) to 1.0 (critical risk)
       Grade       string         // A through F
       Signals     []RiskSignal   // individual contributing signals
       Explanation string         // human-readable summary
   }

   type RiskSignal struct {
       Name        string         // "publication_age", "maintainer_count", etc.
       Weight      float64        // contribution weight (0.0-1.0, sum to 1.0)
       RawValue    interface{}    // the measured value
       Score       float64        // normalized 0.0-1.0 score for this signal
       Detail      string         // "Published 2 days ago (threshold: 7 days)"
   }
   ```
   - Weighted multi-signal scoring: publication age (40%), maintainer count (15%), download count (15%), install scripts (15%), dependency count (10%), version history (5%)
   - Weight distribution rationale: publication age is the strongest single predictor; install scripts are the primary execution vector; maintainer analysis catches takeovers

6. Implement `gdev pkg check` command:
   - `gdev pkg check <registry>:<package>@<version>` — single package assessment
   - `gdev pkg check --lockfile` — scan all packages in the project's lockfile (detect lockfile format automatically)
   - Output formats: human-readable (default), JSON (`--json`), integration with Phase 15 posture report
   - Exit code: 0 = all clear, 1 = high risk packages found (CI gate)

7. Integrate with Phase 5 package manager hardening:
   - When `gdev pkg check --lockfile` finds high-risk packages, recommend specific package manager config changes (e.g., increase `min-release-age` in `.npmrc`)
   - Cross-reference age-gating configs: if a flagged package's age is below the configured gate, note that the gate already blocks it

**Acceptance Criteria:**
- [ ] Registry clients fetch metadata from npm, PyPI, crates.io, and Go module proxy without API keys
- [ ] Publication age calculation correctly computes version-specific age (not package creation date)
- [ ] Packages published within 24 hours are flagged as extreme risk
- [ ] Maintainer change detection identifies new maintainers on existing packages
- [ ] Risk score computation produces weighted multi-signal scores with per-signal breakdowns
- [ ] `gdev pkg check` works for single packages and lockfile scanning
- [ ] Response caching prevents redundant API calls within TTL window
- [ ] Registry clients respect rate limits and handle timeouts gracefully
- [ ] Lockfile parser handles npm, yarn, pnpm, and Go module lockfile formats
- [ ] JSON output format is compatible with Phase 15 posture report schema

**Research Citations:**
- `research-spikes/security-tooling-evaluation-gdev/npm-scan-research.md` Section 1.2-1.4 — scan pipeline, detector architecture, lockfile analysis
- `research-spikes/security-tooling-evaluation-gdev/npm-scan-research.md` Section 5.3 — ATK taxonomy, policy-as-code, lockfile-triggered scanning as borrowable patterns
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 4, patterns 15, 20 — ATK taxonomy with NIST mappings, lockfile-triggered scanning
- `research-spikes/package-supply-chain-security/quarantine-gates-research.md` — age-gating configs and malware detection window data
- `research-spikes/package-supply-chain-security/research.md` — 6-layer defense-in-depth model, age-gating as highest-impact defense

**Status:** Not Started

---

### Unit 39.3: Security Analysis Pipeline

**Description:** Implement a structured security finding data model and a multi-stage analysis pipeline (scan, categorize, score, report) with output formatters for human-readable, JSON, and SARIF 2.1.0 formats. This unit provides the unified reporting layer that all other security primitives (policy engine, package risk, MCP trust) feed their findings into.

**Context:** Cloudberry's six-phase pipeline (prep, map, hunt, dedup, validate, aggregate) demonstrates that the value of a security analysis pipeline is in noise reduction, not finding generation — 60% of raw findings are discarded through dedup and validation. npm-scan's SARIF v2.1 output format enables native rendering in GitHub's Security tab, a critical CI integration point. reasoning-core's JSONL audit log provides structured decision metadata for forensic review.

This unit does not perform the scanning itself (that is done by the policy engine, package risk module, and external tools like Socket CLI). Instead, it provides the common data model and pipeline stages that normalize, deduplicate, and render findings from all sources into consistent output formats.

**Desired Outcome:** Security findings from any gdev subsystem (policy violations, package risks, MCP trust warnings, drift detection) flow through a single pipeline that categorizes, deduplicates, scores, and renders them in multiple formats. `gdev status --security` uses this pipeline for its security section. CI systems consume SARIF output for GitHub Security tab integration.

**Steps:**

1. Create `internal/security/pipeline/` package with the finding data model:
   ```go
   // Finding represents a single security observation from any gdev subsystem.
   type Finding struct {
       ID            string          `json:"id"`            // unique finding ID (e.g., "POL-SPR-001-20260515")
       Source        string          `json:"source"`        // "policy_engine", "pkg_risk", "mcp_trust", "drift"
       RuleID        string          `json:"ruleId"`        // originating rule or check ID
       Category      string          `json:"category"`      // NIST 800-161 category mapping
       Severity      Severity        `json:"severity"`      // critical/high/medium/low/info
       Title         string          `json:"title"`         // one-line summary
       Description   string          `json:"description"`   // detailed explanation
       Evidence      []Evidence      `json:"evidence"`      // supporting data
       Remediation   string          `json:"remediation"`   // actionable fix recommendation
       Suppressible  bool            `json:"suppressible"`  // whether user can suppress this finding
       Deduplicated  bool            `json:"deduplicated"`  // true if merged with another finding
       MergedFrom    []string        `json:"mergedFrom"`    // IDs of findings merged into this one
   }

   type Evidence struct {
       Type   string `json:"type"`   // "file", "command", "config", "metadata"
       Path   string `json:"path"`   // file path or resource identifier
       Line   int    `json:"line"`   // line number (0 if N/A)
       Detail string `json:"detail"` // evidence content or description
   }
   ```

2. Implement the four pipeline stages:
   - **Scan**: Collect `[]Finding` from registered sources (policy engine, package risk, MCP trust, Phase 15 drift detection). Each source implements a `Scanner` interface: `Scan(ctx context.Context, project *ProjectContext) ([]Finding, error)`
   - **Categorize**: Map each finding to NIST 800-161 categories using the ATK taxonomy structure borrowed from npm-scan. Categories: supply_chain, credential_exposure, destructive_operation, configuration_drift, dependency_risk, mcp_trust, policy_violation
   - **Score**: Apply severity-based scoring with configurable weights. Produce an aggregate security score (0-100) that feeds into Phase 15's posture score
   - **Report**: Format findings into the selected output format

3. Implement deduplication (Cloudberry's dedup-before-validate principle):
   - Group findings by root cause: same file + same category + overlapping evidence
   - Merge grouped findings: keep the highest severity, combine evidence, track merged IDs
   - Log dedup statistics: "12 raw findings, 3 duplicates merged, 9 unique findings"

4. Implement anti-false-positive support (Cloudberry's known-safe-patterns concept):
   - `.gdev.yaml` section `security.known_safe_patterns[]` with glob + description
   - Findings matching known-safe patterns are automatically downgraded to `info` severity (not removed — visible in verbose output)
   - Example: `{ pattern: "internal/test/**", description: "Test fixtures use hardcoded credentials intentionally" }`

5. Implement output formatters:
   - **Human-readable**: Hierarchical display grouped by severity, with color coding (red = critical/high, yellow = medium, blue = low/info). Remediation actions as actionable one-liners.
   - **JSON**: Full `PipelineReport` struct with schema version, compatible with Phase 15's `PostureReport.SecurityFindings` field
   - **SARIF 2.1.0**: Standard Static Analysis Results Interchange Format for GitHub Security tab integration. Map gdev finding fields to SARIF `result`, `ruleDescriptor`, `physicalLocation`, and `level` fields. Include `tool.driver` metadata identifying gdev version and active policy set.

6. Integrate with Phase 15 health/status reporting:
   - `gdev status --security` invokes the pipeline with all registered scanners
   - Security score contributes to the overall posture score (feeds the dependency health 30% bucket)
   - Findings are rendered in the `--verbose` output with per-finding detail
   - SARIF output is available via `gdev status --security --format sarif`

7. Implement `gdev security report` command:
   - Full pipeline execution with all scanners
   - `--format` flag: `text` (default), `json`, `sarif`
   - `--severity` flag: minimum severity to include (default: `low`)
   - `--output` flag: write to file (default: stdout)
   - Exit code: configurable via `--fail-on` (default: `high`) — exits non-zero if findings at or above the threshold exist

**Acceptance Criteria:**
- [ ] Finding data model supports all gdev security subsystems (policy, package risk, MCP trust, drift)
- [ ] Pipeline stages execute in order: scan, categorize, score, report
- [ ] Deduplication merges findings with same root cause and tracks merge provenance
- [ ] Anti-false-positive patterns downgrade matching findings without removing them
- [ ] SARIF 2.1.0 output validates against the SARIF JSON schema
- [ ] SARIF output renders correctly in GitHub Security tab (manual verification with a test repo)
- [ ] Human-readable output groups by severity with remediation actions
- [ ] JSON output is compatible with Phase 15's `PostureReport` schema
- [ ] `gdev security report` command supports all format and severity options
- [ ] `--fail-on` exit code correctly gates CI pipelines
- [ ] NIST 800-161 category mappings are documented in the SARIF `ruleDescriptor.helpUri` field

**Research Citations:**
- `research-spikes/security-tooling-evaluation-gdev/cloudberry-security-reviews-research.md` Section 1 — six-phase pipeline, dedup-before-validate principle, 60% finding discard rate
- `research-spikes/security-tooling-evaluation-gdev/cloudberry-security-reviews-research.md` Section 3 — anti-false-positive patterns, separated security context
- `research-spikes/security-tooling-evaluation-gdev/cloudberry-security-reviews-research.md` Section 6 — replicable principles (encode deterministic steps deterministically, build dedup before tuning generation)
- `research-spikes/security-tooling-evaluation-gdev/npm-scan-research.md` Section 1.6 — SARIF v2.1 output format, ATK taxonomy with NIST 800-161 mappings
- `research-spikes/security-tooling-evaluation-gdev/reasoning-core-research.md` Section 4C — JSONL audit log schema with decision metadata
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 4, patterns 14-18 — JSONL audit, ATK taxonomy, SARIF output, anti-false-positive documentation, review benchmarking

**Status:** Not Started

---

### Unit 39.4: MCP Server Trust Assessment

**Description:** Implement an MCP server trust scoring algorithm that evaluates MCP servers based on their metadata, capabilities, permissions, and source reputation. Trust scores are displayed in `gdev mcp list --trust-scores` and feed into Phase 28's MCP registry decisions.

**Context:** Sense demonstrates that MCP servers are the primary extension mechanism for AI coding agents, but the current MCP ecosystem has no standardized trust evaluation. Phase 28's MCP registry classifies servers into security tiers (low/medium/high), but this classification is static and manually maintained. This unit adds dynamic trust scoring based on inspectable server properties: what tools does the server expose? What permissions does it require? Does it need network access? Does it read/write files? Is the source repo maintained?

The trust scoring algorithm borrows Sense's capability analysis approach (enumerate tools, permissions, and data access patterns) and extends it with source reputation signals (GitHub stars, contributor count, maintenance cadence) similar to the package risk assessment in Unit 39.2.

**Desired Outcome:** Every MCP server in the Phase 28 registry has a computed trust score visible to developers. The score informs detect-and-offer decisions (low-trust servers require explicit confirmation), generates security notices in CLAUDE.md, and is queryable via `gdev mcp list --trust-scores`.

**Steps:**

1. Create `internal/security/mcptrust/` package with the trust model:
   ```go
   // McpTrustProfile captures the security-relevant metadata of an MCP server.
   type McpTrustProfile struct {
       ServerName       string
       ToolCount        int
       Tools            []ToolCapability
       Permissions      PermissionSet
       DataAccess       DataAccessPattern
       Source           SourceReputation
       TrustScore       TrustScore
   }

   type ToolCapability struct {
       Name        string
       ReadOnly    bool       // true if tool only reads data
       NetworkReq  bool       // true if tool makes network requests
       FileWrite   bool       // true if tool writes to filesystem
       Scope       string     // "project", "user", "system"
   }

   type PermissionSet struct {
       RequiresCredentials bool
       RequiresNetwork     bool
       RequiresFileSystem  bool
       RequiresShell       bool
       ScopeLevel          string   // "project" (safest), "user", "system" (riskiest)
   }

   type DataAccessPattern struct {
       ReadsSourceCode    bool
       ReadsSecrets       bool
       ReadsSystemFiles   bool
       WritesFiles        bool
       SendsDataExternal  bool
   }

   type SourceReputation struct {
       RepoURL           string
       Stars             int
       Contributors      int
       LastCommitAge     time.Duration
       License           string
       OrgBacked         bool     // true if backed by a known org (CNCF, company, etc.)
       KnownVulns        int
   }

   type TrustScore struct {
       Overall     float64   // 0.0 (untrusted) to 1.0 (fully trusted)
       Grade       string    // A through F
       Tier        string    // maps to Phase 28 security tier: low, medium, high
       Signals     []TrustSignal
       Explanation string
   }
   ```

2. Implement the trust scoring algorithm:
   - **Capability scope** (30% weight): Read-only, project-scoped tools score highest. System-scoped tools with file writes and network access score lowest.
   - **Permission requirements** (25% weight): Servers requiring credentials, shell access, or system-level permissions score lower. No-credential, read-only servers score highest.
   - **Data access** (20% weight): Servers that read source code only score well. Servers that send data externally or read secrets score poorly.
   - **Source reputation** (25% weight): Stars, contributor count, maintenance cadence, organizational backing, known vulnerabilities. Thresholds informed by the security tooling evaluation findings (4-star tools are risky, CNCF-backed tools are more trustworthy).

3. Implement static trust profiles for Phase 28 registry servers:
   - Pre-computed trust profiles for all servers in the existing registry (Socket.dev, Context7, Terraform, etc.)
   - Profiles are embedded data, updated when the registry is updated
   - Developers can override scores in `.gdev.yaml` under `mcp.trust_overrides`

4. Implement dynamic trust analysis for user-added MCP servers:
   - When a developer adds an MCP server not in the registry (`gdev mcp add <config>`), run trust analysis
   - Parse the MCP server's tool list from its manifest/handshake
   - Classify each tool's capabilities based on name patterns and description parsing
   - Fetch source reputation from GitHub API (if repo URL is available) with caching
   - Display trust score and ask for confirmation if score is below threshold

5. Integrate with Phase 28 MCP registry:
   - `gdev mcp list` default output includes trust grade (A-F) per server
   - `gdev mcp list --trust-scores` shows full trust breakdown per server
   - Detect-and-offer policy uses trust score: servers with grade C or below require explicit user confirmation
   - Security notices in CLAUDE.md include trust-derived warnings for medium/high-tier servers

6. Integrate with Phase 15 posture reporting:
   - MCP trust scores contribute to the security posture assessment
   - Low-trust enabled servers produce findings in the security pipeline (Unit 39.3)
   - `gdev status --security` includes MCP trust summary

**Acceptance Criteria:**
- [ ] Trust scoring algorithm produces 0.0-1.0 scores with A-F grades for all MCP servers
- [ ] Static trust profiles exist for all Phase 28 registry servers
- [ ] Dynamic trust analysis runs for user-added MCP servers not in the registry
- [ ] Capability analysis correctly classifies read-only vs. read-write vs. network-accessing tools
- [ ] Source reputation fetcher handles GitHub API responses with caching and rate limiting
- [ ] `gdev mcp list --trust-scores` displays per-server trust breakdowns
- [ ] Low-trust servers (grade C or below) trigger confirmation prompts in detect-and-offer flow
- [ ] Trust overrides in `.gdev.yaml` are respected
- [ ] MCP trust findings integrate with the security analysis pipeline (Unit 39.3)
- [ ] Trust scores are deterministic given the same inputs (no randomness)

**Research Citations:**
- `research-spikes/security-tooling-evaluation-gdev/sense-research.md` Section 2 — MCP server architecture, tool capability model, hook system
- `research-spikes/security-tooling-evaluation-gdev/sense-research.md` Section 4 — integration fit, McpServerEntry struct, security tier classification
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 4, patterns 21-23 — post-tool-use re-indexing, pre-compact context injection, idempotent setup
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 2 — Sense as detect-and-offer entry in Phase 28

**Status:** Not Started

---

### Unit 39.5: Optional Tool Integration Points

**Description:** Implement integration adapters that allow the native policy engine (Unit 39.1) and MCP trust assessment (Unit 39.4) to optionally delegate to external tools (Prempti for runtime policy, Sense for MCP registry intelligence) when those tools are installed. This provides an upgrade path for teams that want deeper capabilities than gdev's native implementations while keeping the native implementations as the zero-dependency default.

**Context:** The security-tooling-evaluation-gdev spike recommended Prempti and Sense as the only two tools warranting optional configuration options. Prempti adds capabilities gdev's native policy engine cannot replicate without a daemon: real-time Falco rule evaluation across all tool calls (not just gdev-managed hooks), audit trail with correlation IDs across sessions, and monitor mode with Falco's web UI. Sense adds capabilities gdev's native MCP trust scoring cannot replicate without an indexer: live capability analysis from the MCP handshake, structural codebase context for trust decisions, and convention detection.

The adapter pattern ensures gdev functions identically whether these tools are installed or not. The feature flag system makes the choice explicit and auditable.

**Desired Outcome:** A developer can run `gdev enable prempti` or configure `security.policy_engine: prempti` in `.gdev.yaml` to delegate policy evaluation to Prempti when it is installed. Similarly, `security.mcp_trust: sense` delegates MCP trust scoring to Sense. When the external tool is not installed or not configured, gdev falls back to native implementations transparently. The feature flag state is visible in `gdev status` and logged in the audit trail.

**Steps:**

1. Define the adapter interface in `internal/security/adapters/`:
   ```go
   // PolicyAdapter allows pluggable policy evaluation backends.
   type PolicyAdapter interface {
       Name() string
       Available() bool                                        // is the external tool installed?
       Evaluate(ctx context.Context, op OperationContext) (*EvaluationResult, error)
       ListRules() ([]Rule, error)
       Mode() string                                           // "native", "prempti"
   }

   // TrustAdapter allows pluggable MCP trust evaluation backends.
   type TrustAdapter interface {
       Name() string
       Available() bool
       Score(ctx context.Context, server McpServerConfig) (*TrustScore, error)
       Mode() string                                           // "native", "sense"
   }
   ```

2. Implement the Prempti adapter in `internal/security/adapters/prempti.go`:
   - Detect Prempti installation: check for `~/.prempti/bin/premptictl` or `premptictl` in PATH
   - Detect Prempti service status: run `premptictl status` and parse output
   - Evaluate operations by invoking Prempti's interceptor binary with the operation context as JSON on stdin
   - Map Prempti verdicts (allow/deny/ask) to gdev's `EvaluationResult`
   - Handle Prempti unavailability: if the Prempti service is down, log a warning and fall back to native policy engine (fail-open to native, not fail-closed per Prempti's default — gdev wraps the failure mode)
   - Generate gdev-specific Falco rules in `~/.prempti/rules/user/gdev-rules.yaml` that extend Prempti's defaults with gdev's active policy set

3. Implement the Sense adapter in `internal/security/adapters/sense.go`:
   - Detect Sense installation: check for `sense` binary in PATH
   - Detect Sense index: check for `.sense/index.db` in project directory
   - Query Sense's MCP server for tool capabilities via stdio transport
   - Map Sense's tool descriptions and capability analysis to the `TrustScore` model
   - Handle Sense unavailability: log and fall back to native trust scoring

4. Implement the feature flag system in `.gdev.yaml`:
   ```yaml
   security:
     # Policy engine backend: "native" (default) or "prempti"
     policy_engine: native

     # MCP trust backend: "native" (default) or "sense"
     mcp_trust: native

     # Policy set selection (used by native engine; ignored when delegating to prempti)
     policy: default

     # Shadow mode: evaluate and log without enforcing (applies to both backends)
     shadow_mode: false
   ```

5. Implement `gdev enable prempti` and `gdev enable sense` commands:
   - `gdev enable prempti`:
     1. Check if Prempti is installed; if not, display installation instructions (not auto-install — respect gdev's "tell don't install" principle for heavyweight dependencies)
     2. Check if Prempti service is running; if not, offer to start it
     3. Generate gdev-specific Falco rules
     4. Set `security.policy_engine: prempti` in project `.gdev.yaml`
     5. Document the interaction with existing gdev hooks in CLAUDE.md (dual-hook overhead warning)
   - `gdev enable sense`:
     1. Check if Sense is installed; if not, display installation instructions
     2. Check if `.sense/index.db` exists; if not, offer to run `sense scan`
     3. Set `security.mcp_trust: sense` in project `.gdev.yaml`
     4. Add Sense to Phase 28 MCP registry as an active server

6. Write documentation for when to use native vs. external tool:
   - Generate `docs/security-policy-engines.md` when either adapter is enabled
   - Content: comparison table (native vs. Prempti: latency, capabilities, dependencies, failure modes), decision criteria (native for most teams; Prempti for high-security environments with Falco expertise), configuration examples
   - Similar comparison for native MCP trust vs. Sense

**Acceptance Criteria:**
- [ ] Prempti adapter detects installation and service status correctly
- [ ] Prempti adapter falls back to native engine when service is unavailable (fail-open to native)
- [ ] Prempti adapter maps Prempti verdicts to gdev's EvaluationResult
- [ ] Sense adapter detects installation and index presence correctly
- [ ] Sense adapter falls back to native trust scoring when unavailable
- [ ] Feature flags in `.gdev.yaml` switch between native and external backends
- [ ] `gdev enable prempti` generates Falco rules and updates config
- [ ] `gdev enable sense` adds Sense to MCP registry and updates config
- [ ] `gdev status` shows active backend for policy and MCP trust
- [ ] Audit trail logs which backend processed each evaluation
- [ ] Documentation generated when adapters are enabled covers decision criteria and failure modes
- [ ] System functions identically with neither external tool installed (native-only path)

**Research Citations:**
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` Section 5.1 — integration as configuration option, implementation path, value-add over bare install
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` Section 5.2 — why not default (infrastructure weight, overlap, maturity, fail-closed risk)
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` Section 6 — tradeoffs, fail-closed failure mode, NixOS concerns
- `research-spikes/security-tooling-evaluation-gdev/sense-research.md` Section 4 — Option A (configuration option), MCP registry entry, interaction with existing tools
- `research-spikes/security-tooling-evaluation-gdev/sense-research.md` Section 5 — limitations, single-author risk, O'Saasy license
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 2 — Tier 1 recommendations (Prempti as optional + concept source, Sense as optional + concept source)
- `research-spikes/security-tooling-evaluation-gdev/cross-tool-comparison-research.md` Section 5, Theme 3 — concept borrowing beats direct integration

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### New Packages

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `internal/security/policy/` | YAML-based policy engine | `Rule`, `MatchSpec`, `Engine`, `EvaluationResult`, `PolicySet` |
| `internal/security/policy/builtin/` | Embedded policy sets | `default.yaml`, `strict.yaml`, `permissive.yaml` (via `embed.FS`) |
| `internal/security/pkgrisk/` | Package risk assessment | `RegistryClient`, `PackageMetadata`, `RiskScore`, `RiskSignal` |
| `internal/security/pkgrisk/registry/` | Per-ecosystem registry clients | `NpmClient`, `PyPIClient`, `CratesClient`, `GoProxyClient` |
| `internal/security/pipeline/` | Security analysis pipeline | `Finding`, `Evidence`, `PipelineReport`, `Scanner` interface |
| `internal/security/pipeline/sarif/` | SARIF 2.1.0 formatter | `SarifReport`, `Run`, `Result`, `Location` |
| `internal/security/mcptrust/` | MCP server trust scoring | `McpTrustProfile`, `TrustScore`, `TrustSignal`, `ToolCapability` |
| `internal/security/adapters/` | External tool adapters | `PolicyAdapter`, `TrustAdapter`, `PremptiAdapter`, `SenseAdapter` |

### Integration Points with Existing Code

- **Phase 32 audit trail** (`~/.qsdev/audit/sessions/<date>/<session-id>.jsonl`): Policy engine evaluations append entries using the same JSONL schema. New fields: `source: "policy_engine"`, `rule_id`, `verdict`, `shadow_mode`.
- **Phase 15 posture reporting** (`internal/posture/`): Security pipeline findings feed into `PostureReport.SecurityFindings`. Package risk scores contribute to the dependency health bucket (30% of posture score). MCP trust scores produce findings for enabled low-trust servers.
- **Phase 28 MCP registry** (`McpServerRegistry`): Trust scores are stored alongside existing `SecurityTier` field. `gdev mcp list` output extended with trust grade column. Detect-and-offer logic consults trust score for confirmation threshold.
- **Phase 12 tool lifecycle** (`internal/tools/`): Prempti and Sense adapters register as lifecycle-managed tools. `gdev enable/disable` commands manage their activation state.
- **Phase 5 package manager configs**: Package risk findings cross-reference age-gating configs to identify whether existing protections already cover the flagged risk.
- **`.gdev.yaml` config resolution**: New `security` section with `policy_engine`, `mcp_trust`, `policy`, `shadow_mode`, `suppressions`, `known_safe_patterns`, and `trust_overrides` fields. Follows existing config resolution order: `.gdev.yaml` (project) > local config > binary defaults.

### Embed Strategy

Built-in policy sets (`default.yaml`, `strict.yaml`, `permissive.yaml`) are embedded via `//go:embed builtin/*.yaml` in the policy package. This keeps policy definitions as human-readable YAML files in the source tree while compiling them into the single binary. The same pattern is used by Phase 32's hook scripts (`//go:embed hooks/*.sh`).

### Testing Strategy

- **Policy engine**: Table-driven tests with operation contexts and expected verdicts. Fuzz testing on YAML rule parsing. Benchmark tests confirming <5ms evaluation latency.
- **Package risk**: Mock registry clients returning fixture responses. Integration tests against real registries (gated behind `GDEV_INTEGRATION_TESTS=1`). Test coverage for each risk signal in isolation and in combination.
- **Pipeline**: Test deduplication with known-duplicate finding sets. SARIF output validated against the official SARIF JSON schema. Test anti-false-positive pattern matching.
- **MCP trust**: Test scoring determinism (same inputs produce same scores). Test fallback behavior when external tools are unavailable.
- **Adapters**: Mock external tool binaries. Test fail-open behavior when Prempti service is down. Test feature flag switching between native and external backends.

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] Policy engine evaluates rules in <5ms with zero external dependencies
- [ ] Package risk assessment fetches metadata from 4 registries (npm, PyPI, crates.io, Go proxy)
- [ ] Security analysis pipeline produces valid SARIF 2.1.0 output
- [ ] MCP trust scores are visible in `gdev mcp list --trust-scores`
- [ ] Feature flag system cleanly switches between native and external backends
- [ ] Native implementations function identically whether external tools are installed or not
- [ ] All 23 borrowable patterns from the security tooling evaluation are addressed (implemented natively, delegated to adapter, or documented as N/A with rationale)
- [ ] `gdev status --security` displays unified security posture including policy, package risk, MCP trust, and drift findings
- [ ] Built-in policy sets cover the 7 security domains identified in Prempti's rule taxonomy
- [ ] Shadow mode works end-to-end: evaluate, log, but do not enforce
- [ ] All new commands documented in `gdev help` output
