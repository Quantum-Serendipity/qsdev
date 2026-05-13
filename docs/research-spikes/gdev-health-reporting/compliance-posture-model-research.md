# Compliance Posture Model for gdev

## Problem

A consulting engineer managing multiple client projects needs to answer one question instantly: "Is this project's security posture complete?" This requires a data model that captures the state of gdev's defense layers, tools, configurations, and dependencies -- and can express that state as both a quick visual summary and a detailed machine-readable report.

## gdev's Security Surface Area

From the implementation plan, gdev manages:
- **10 defense layers**: age-gating, install script blocking, lock file enforcement, vulnerability scanning, PreToolUse hooks, Nix hardening, SAST (Semgrep), secrets scanning (Gitleaks), container security (Grype/Syft/Cosign), license compliance (ScanCode)
- **16+ toggleable tools**: semgrep, gitleaks, container-security, license-compliance, attach-guard, ripsecrets, agent-postmortem, version-sentinel, semble, context7, github-mcp, socket-dev-mcp, trail-of-bits-skills, secretspec, commitlint, changelog
- **27 language ecosystems** across 4 priority tiers
- **Generated config files**: devenv.yaml, devenv.nix, .envrc, CLAUDE.md, .claude/settings.json, .mcp.json, .pre-commit-config.yaml, CI workflows, and per-tool config files

## Posture Model Design

### Three-Layer Assessment

The posture model evaluates three independent dimensions:

#### Layer 1: Defense Coverage (What percentage of defenses are active?)

```
Defense Layer          Status    Weight    Notes
---------------------------------------------------------------
age-gating             enabled   high      npm: 72h, pip: 72h
install-script-block   enabled   high      @lavamoat/allow-scripts
lock-file-enforce      enabled   high      package-lock.json valid
vuln-scanning          enabled   high      OSV Scanner configured
pretooluse-hooks       partial   high      attach-guard installed, 2 rules disabled
nix-hardening          enabled   medium    10/10 settings applied
sast                   enabled   medium    Semgrep with auto config
secrets-scanning       enabled   medium    Gitleaks + ripsecrets
container-security     disabled  medium    No Dockerfile detected
license-compliance     disabled  low       Opt-in, not enabled
---------------------------------------------------------------
Coverage: 8/10 enabled (80%), weighted score: 87/100
```

Each defense layer has:
- **Status**: `enabled` | `partial` | `disabled` | `not-applicable`
- **Weight**: `critical` (10) | `high` (7.5) | `medium` (5) | `low` (2.5) -- following Scorecard's model
- **Details**: Per-ecosystem breakdown where applicable
- **Reason**: Why disabled/partial (auto-detected "no Dockerfile" vs user choice)

#### Layer 2: Configuration Health (Are configs correct and current?)

```
Config File                    State       Version    Notes
---------------------------------------------------------------
devenv.yaml                    current     v1.2.0     Hash matches
devenv.nix                     modified    v1.2.0     User customized (expected)
.envrc                         current     v1.2.0     Hash matches
CLAUDE.md                      current     v1.2.0     Generated sections intact
.claude/settings.json          current     v1.2.0     3 deny rules added by user
.pre-commit-config.yaml        outdated    v1.1.0     New hooks available
.semgrep.yml                   current     v1.2.0     Hash matches
.gitleaks.toml                 modified    v1.2.0     Custom allowlist (expected)
.mcp.json                      current     v1.2.0     3 servers configured
---------------------------------------------------------------
Config health: 7/9 current, 1 user-modified (OK), 1 outdated
```

Each config file has:
- **State**: `current` (hash matches latest generation) | `modified` (user edited, expected for human-owned files) | `outdated` (generated from older gdev version) | `missing` (expected but absent) | `corrupt` (fails validation)
- **Version**: gdev version that generated it
- **Hash match**: SHA256 comparison against generation-time hash
- **Category**: `machine-owned` (devenv.yaml, .envrc) | `human-edited` (devenv.nix, CLAUDE.md) | `exclusive` (per-tool configs)

#### Layer 3: Dependency Health (Are dependencies safe?)

```
Ecosystem     Lock File   Vulns (C/H/M/L)   Age-Gate   Last Scan
---------------------------------------------------------------
npm           valid       0/2/5/12           72h        2h ago
pip           valid       0/0/1/3            72h        2h ago
go.mod        valid       0/0/0/0            N/A        2h ago
---------------------------------------------------------------
Total vulnerabilities: 0 critical, 2 high, 6 moderate, 15 low
```

Each ecosystem tracks:
- **Lock file**: `valid` | `missing` | `corrupt` | `stale` (older than source manifest)
- **Vulnerability counts**: by severity tier
- **Age-gate status**: configured threshold
- **Last scan time**: freshness of vulnerability data

### Aggregate Score

Following Scorecard's weighted model:

```
Overall Security Posture: 82/100 (B+)

  Defense Coverage:      87/100 (A-)    -- 8/10 layers active, weighted
  Configuration Health:  91/100 (A)     -- 1 outdated file
  Dependency Health:     68/100 (C+)    -- 2 high vulns outstanding

Grade Scale: A (90-100), B (75-89), C (60-74), D (45-59), F (<45)
```

The three layers feed into a weighted aggregate:
- Defense Coverage: 40% weight (the core value proposition)
- Configuration Health: 30% weight (drift = erosion of defenses)
- Dependency Health: 30% weight (runtime risk)

### Conformance Track (PASS/FAIL)

In addition to the numeric score, a binary conformance check:

```
Conformance: PASS (baseline) / FAIL (enhanced)

Baseline Requirements (all must pass):
  [PASS] Lock files present for all detected ecosystems
  [PASS] Pre-commit hooks installed
  [PASS] No critical vulnerabilities
  [PASS] CLAUDE.md generated sections present
  [PASS] settings.json deny rules present
  [FAIL] All Tier 1 defense layers enabled  <-- container-security disabled

Enhanced Requirements (consulting firm standard):
  [PASS] Age-gating configured for all supported ecosystems
  [FAIL] Zero high-severity vulnerabilities
  [PASS] SAST enabled (Semgrep)
  [PASS] Secrets scanning enabled (Gitleaks)
  [FAIL] License compliance enabled
  [PASS] CI workflows generated
```

Conformance levels:
- **Baseline**: Minimum acceptable posture. All Tier 1 defenses, no critical vulns, lock files present.
- **Enhanced**: Consulting firm standard. All defenses including optional ones, zero high vulns, full CI pipeline.
- **Custom**: Per-project overrides via `.gdev-policy.yaml`.

## Data Model (Go Types)

```go
type PostureReport struct {
    SchemaVersion string              `json:"schemaVersion"`  // "1.0.0"
    GeneratedAt   time.Time           `json:"generatedAt"`
    GdevVersion   string              `json:"gdevVersion"`
    ProjectPath   string              `json:"projectPath"`
    
    Score         AggregateScore      `json:"score"`
    Conformance   ConformanceResult   `json:"conformance"`
    Defense       DefenseCoverage     `json:"defense"`
    Config        ConfigHealth        `json:"config"`
    Dependencies  DependencyHealth    `json:"dependencies"`
    Tools         []ToolStatus        `json:"tools"`
    Ecosystems    []EcosystemStatus   `json:"ecosystems"`
}

type AggregateScore struct {
    Total    float64 `json:"total"`     // 0-100
    Grade    string  `json:"grade"`     // A+, A, A-, B+, ...
    Defense  float64 `json:"defense"`   // 0-100
    Config   float64 `json:"config"`    // 0-100
    DepHealth float64 `json:"depHealth"` // 0-100
}

type ConformanceResult struct {
    Baseline ConformanceLevel `json:"baseline"`
    Enhanced ConformanceLevel `json:"enhanced"`
    Custom   ConformanceLevel `json:"custom,omitempty"`
}

type ConformanceLevel struct {
    Pass   bool                `json:"pass"`
    Checks []ConformanceCheck  `json:"checks"`
}

type ConformanceCheck struct {
    Name    string `json:"name"`
    Pass    bool   `json:"pass"`
    Reason  string `json:"reason,omitempty"`
}

type DefenseCoverage struct {
    Score      float64        `json:"score"`    // 0-100
    Enabled    int            `json:"enabled"`
    Total      int            `json:"total"`
    Layers     []DefenseLayer `json:"layers"`
}

type DefenseLayer struct {
    Name    string `json:"name"`
    Status  string `json:"status"` // enabled|partial|disabled|not-applicable
    Weight  string `json:"weight"` // critical|high|medium|low
    Details string `json:"details,omitempty"`
    Reason  string `json:"reason,omitempty"`
}

type ToolStatus struct {
    Name       string `json:"name"`
    Enabled    bool   `json:"enabled"`
    Available  bool   `json:"available"`  // tool exists in gdev but not enabled
    ConfigFile string `json:"configFile,omitempty"`
    ConfigHash string `json:"configHash,omitempty"`
    Version    string `json:"version,omitempty"`
}

type EcosystemStatus struct {
    Name          string `json:"name"`     // "npm", "pip", "go"
    Detected      bool   `json:"detected"`
    LockFile      string `json:"lockFile"` // valid|missing|corrupt|stale
    VulnCounts    VulnSeverityCounts `json:"vulnCounts"`
    AgeGate       string `json:"ageGate,omitempty"` // "72h" or ""
    LastScan      *time.Time `json:"lastScan,omitempty"`
}

type VulnSeverityCounts struct {
    Critical int `json:"critical"`
    High     int `json:"high"`
    Moderate int `json:"moderate"`
    Low      int `json:"low"`
    Info     int `json:"info"`
}
```

## Consulting Engineer User Stories

1. **"Show me the quick picture"** -> `gdev status` prints colored summary with score and grade
2. **"What's wrong?"** -> `gdev status --verbose` shows per-layer detail with remediation hints
3. **"Is CI going to pass?"** -> `gdev status --audit-level high` exits non-zero if high+ findings
4. **"Generate audit evidence"** -> `gdev status --format json > posture.json` for compliance records
5. **"What can I enable?"** -> `gdev status` shows available-but-disabled tools
6. **"Did someone break the config?"** -> `gdev status` flags modified machine-owned files
7. **"Update the badge"** -> `gdev status --format badge` generates shields.io-compatible JSON

## Tradeoffs and Limitations

**Scoring subjectivity:** Any weighted score is opinionated. The weights (defense 40%, config 30%, deps 30%) and per-layer weights need to be tunable. Hardcoded weights will be wrong for some contexts.

**False sense of security:** A score of 90/100 doesn't mean "secure" -- it means "gdev's managed defenses are mostly enabled." Supply chain attacks, zero-days, and logic bugs are outside gdev's scope. The report should include a disclaimer.

**Scan freshness:** Dependency health depends on vulnerability databases being queried. `gdev status` can't run npm audit, cargo audit, and govulncheck on every invocation (too slow). Options: cache results, show last-scan time, offer `gdev status --scan` for fresh results.

**Offline capability:** Some checks (vulnerability scanning) require network access. The posture model should degrade gracefully -- show defense and config layers even when dependency health can't be assessed.

**Conformance definition:** Who defines "baseline" vs "enhanced"? For a consulting firm, this should come from the profile system. For open source, sensible defaults. The conformance definition itself is policy, not code.

## Depth Checklist

- [x] Underlying mechanism explained: Three-layer assessment model with weighted scoring
- [x] Key tradeoffs and limitations identified: Scoring subjectivity, false sense of security, scan freshness, offline capability
- [x] Compared to at least one alternative: Scorecard's scoring model used as foundation, dual-track (numeric + conformance) from Scorecard v6
- [x] Failure modes and edge cases: Offline mode, stale scan data, user-modified machine-owned files, missing lock files
- [x] Concrete examples or reference implementations: Full Go type definitions, terminal output mockups, conformance check lists
- [x] Report is standalone-readable: Complete data model with rationale for every design decision
