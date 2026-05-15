# Machine-Readable Output Formats for gdev

## Problem

gdev's posture data must be consumable by CI pipelines, compliance dashboards, GitHub Code Scanning, audit systems, and badge generators. Each consumer expects a different format. Which formats matter, what goes in each, and how should they relate?

## Format Landscape

### JSON (Primary Machine-Readable Format)

**Consumers:** CI scripts, dashboards, aggregation tools, badge generators, custom tooling.

**Design:** The PostureReport JSON is gdev's canonical machine-readable output. All other formats are derived from it.

```json
{
  "schemaVersion": "1.0.0",
  "generatedAt": "2026-05-12T14:30:00Z",
  "gdevVersion": "1.2.0",
  "projectPath": "/home/user/projects/client-app",
  "projectName": "client-app",
  "score": {
    "total": 82,
    "grade": "B+",
    "defense": 87,
    "config": 91,
    "depHealth": 68
  },
  "conformance": {
    "baseline": {
      "pass": true,
      "checks": [
        {"name": "lock-files-present", "pass": true},
        {"name": "pre-commit-hooks-installed", "pass": true},
        {"name": "no-critical-vulns", "pass": true},
        {"name": "claude-md-generated", "pass": true},
        {"name": "deny-rules-present", "pass": true},
        {"name": "tier1-defenses-enabled", "pass": false, "reason": "container-security disabled"}
      ]
    },
    "enhanced": {
      "pass": false,
      "checks": [
        {"name": "age-gating-all-ecosystems", "pass": true},
        {"name": "zero-high-vulns", "pass": false, "reason": "2 high vulns in npm"},
        {"name": "sast-enabled", "pass": true},
        {"name": "secrets-scanning-enabled", "pass": true},
        {"name": "license-compliance-enabled", "pass": false, "reason": "not enabled"},
        {"name": "ci-workflows-generated", "pass": true}
      ]
    }
  },
  "defense": {
    "score": 87,
    "enabled": 8,
    "total": 10,
    "layers": [
      {
        "name": "age-gating",
        "status": "enabled",
        "weight": "high",
        "details": "npm: 72h, pip: 72h",
        "ecosystems": [
          {"name": "npm", "configured": true, "threshold": "72h"},
          {"name": "pip", "configured": true, "threshold": "72h"},
          {"name": "go", "configured": false, "reason": "not-supported"}
        ]
      }
    ]
  },
  "config": {
    "score": 91,
    "files": [
      {
        "path": "devenv.yaml",
        "state": "current",
        "category": "machine-owned",
        "generatedVersion": "1.2.0",
        "hashMatch": true
      },
      {
        "path": ".pre-commit-config.yaml",
        "state": "outdated",
        "category": "machine-owned",
        "generatedVersion": "1.1.0",
        "latestVersion": "1.2.0",
        "hashMatch": true,
        "updateAvailable": true
      }
    ]
  },
  "dependencies": {
    "score": 68,
    "ecosystems": [
      {
        "name": "npm",
        "lockFile": "valid",
        "vulns": {"critical": 0, "high": 2, "moderate": 5, "low": 12, "info": 0},
        "ageGate": "72h",
        "lastScan": "2026-05-12T12:30:00Z"
      }
    ],
    "totals": {"critical": 0, "high": 2, "moderate": 6, "low": 15, "info": 0}
  },
  "tools": [
    {"name": "semgrep", "enabled": true, "available": true, "category": "security"},
    {"name": "container-security", "enabled": false, "available": true, "category": "security", "reason": "no-dockerfile-detected"},
    {"name": "license-compliance", "enabled": false, "available": true, "category": "security", "reason": "opt-in"}
  ]
}
```

**Schema versioning:** `schemaVersion` field at top level. Semantic versioning. Breaking changes increment major version. New fields are minor version additions. Consumers should ignore unknown fields.

### SARIF 2.1.0 (Security Findings for Code Scanning)

**Consumers:** GitHub Code Scanning, IDE integrations, SonarQube, security platforms.

**Scope:** SARIF only carries security findings -- it is not a general posture format. Map gdev findings to SARIF results:

```json
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [{
    "tool": {
      "driver": {
        "name": "gdev",
        "version": "1.2.0",
        "informationUri": "https://github.com/example/gdev",
        "rules": [
          {
            "id": "gdev/defense-disabled",
            "shortDescription": {"text": "Security defense layer is disabled"},
            "helpUri": "https://gdev.example.com/docs/defenses",
            "defaultConfiguration": {"level": "warning"}
          },
          {
            "id": "gdev/config-outdated",
            "shortDescription": {"text": "Generated configuration is outdated"},
            "defaultConfiguration": {"level": "note"}
          },
          {
            "id": "gdev/config-missing",
            "shortDescription": {"text": "Expected configuration file is missing"},
            "defaultConfiguration": {"level": "error"}
          },
          {
            "id": "gdev/vuln-high",
            "shortDescription": {"text": "High-severity vulnerability in dependency"},
            "defaultConfiguration": {"level": "error"}
          },
          {
            "id": "gdev/lockfile-missing",
            "shortDescription": {"text": "Lock file missing for detected ecosystem"},
            "defaultConfiguration": {"level": "error"}
          },
          {
            "id": "gdev/hooks-not-installed",
            "shortDescription": {"text": "Pre-commit hooks not installed"},
            "defaultConfiguration": {"level": "warning"}
          }
        ]
      }
    },
    "results": [
      {
        "ruleId": "gdev/defense-disabled",
        "level": "warning",
        "message": {
          "text": "Defense layer 'container-security' is disabled. No Dockerfile detected. Enable with: qsdev enable container-security"
        },
        "locations": [{
          "physicalLocation": {
            "artifactLocation": {"uri": ".gdev/state.yaml"}
          }
        }]
      },
      {
        "ruleId": "gdev/vuln-high",
        "level": "error",
        "message": {
          "text": "2 high-severity vulnerabilities found in npm dependencies. Run 'npm audit' for details."
        },
        "locations": [{
          "physicalLocation": {
            "artifactLocation": {"uri": "package-lock.json"}
          }
        }]
      }
    ]
  }]
}
```

**What maps to SARIF:**
- Disabled defense layers -> `gdev/defense-disabled` (warning)
- Missing config files -> `gdev/config-missing` (error)
- Outdated configs -> `gdev/config-outdated` (note)
- High+ vulnerabilities -> `gdev/vuln-high` or `gdev/vuln-critical` (error)
- Missing lock files -> `gdev/lockfile-missing` (error)
- Missing pre-commit hooks -> `gdev/hooks-not-installed` (warning)

**What does NOT map to SARIF:** Scores, grades, tool inventory, conformance results. SARIF is for discrete findings, not aggregate posture.

### OWASP ASVS Alignment

**Consumers:** Audit teams, compliance officers, enterprise security reviews.

**Approach:** ASVS is a verification standard (checklist of requirements), not an output format. gdev doesn't produce ASVS-formatted output but can map its defense layers to ASVS requirements for audit evidence.

Relevant ASVS v5.0 chapters:
- Chapter 10: Malicious Code (maps to: age-gating, install script blocking, vuln scanning)
- Chapter 14: Configuration (maps to: nix-hardening, config health, lock file enforcement)
- Chapter 1: Architecture (maps to: defense-in-depth design documentation)

**Implementation:** A `qsdev status --compliance asvs` mode could generate a mapping document showing which ASVS requirements are addressed by which gdev defense layers. This is a report for auditors, not a machine-to-machine format.

```
OWASP ASVS v5.0 Coverage Report

  10.3 Dependency Integrity
    10.3.1 Application dependencies from trusted sources    [PASS] age-gating configured
    10.3.2 SCA tool identifies components and versions      [PASS] OSV Scanner, npm audit
    10.3.3 Unused dependencies removed                      [N/A]  manual process
    
  14.2 Dependency Security  
    14.2.1 All components are up to date                    [PARTIAL] 2 high vulns
    14.2.2 Unnecessary features disabled                    [PASS] nix-hardening
```

### Shields.io Badge JSON

**Consumers:** README badges, dashboards, quick visual indicators.

**Output:** Minimal JSON matching shields.io endpoint badge schema:

```json
{
  "schemaVersion": 1,
  "label": "gdev security",
  "message": "82/100 B+",
  "color": "green"
}
```

Color mapping:
- `brightgreen`: 90-100 (A range)
- `green`: 75-89 (B range)
- `yellow`: 60-74 (C range)
- `orange`: 45-59 (D range)
- `red`: 0-44 (F range)

**Delivery options:**
1. `qsdev status --format badge > badge.json` -- Static file committed to repo, served via raw GitHub URL
2. CI job generates badge JSON as artifact, served via GitHub Pages or similar
3. Self-hosted endpoint that runs `qsdev status --format badge` on demand (for dynamic badges)

### JUnit XML (Optional)

**Consumers:** CI systems with native JUnit support (Jenkins, GitLab, most CI).

**Mapping:** Each defense check, config check, and conformance requirement becomes a test case. Failed checks are test failures.

```xml
<testsuites name="gdev-posture" tests="22" failures="3">
  <testsuite name="defense-coverage" tests="10" failures="1">
    <testcase name="age-gating" classname="defense"/>
    <testcase name="container-security" classname="defense">
      <failure message="No Dockerfile detected"/>
    </testcase>
  </testsuite>
  <testsuite name="config-health" tests="9" failures="1">
    <testcase name=".pre-commit-config.yaml" classname="config">
      <failure message="Outdated: v1.1.0, latest v1.2.0"/>
    </testcase>
  </testsuite>
</testsuites>
```

**Value:** Leverages existing CI test reporting infrastructure without requiring SARIF support.

## Format Selection Matrix

| Consumer | Primary Format | Fallback |
|----------|---------------|----------|
| Developer terminal | Default text | -- |
| CI pass/fail gate | Exit code + `--audit-level` | JUnit |
| GitHub Code Scanning | SARIF | -- |
| Custom dashboard | JSON | -- |
| Badge generator | Badge JSON | JSON (parse score) |
| Audit evidence | JSON + ASVS mapping | -- |
| Multi-repo aggregation | JSON | -- |
| GitLab CI | JUnit | SARIF |
| IDE integration | SARIF | -- |

## Implementation Priority

1. **JSON** (must-have): Canonical format, all others derive from it. Version the schema from day one.
2. **Default text** (must-have): Terminal UX, the primary developer interface.
3. **SARIF** (high priority): GitHub Code Scanning is table stakes for any security tool.
4. **Exit codes** (must-have): CI gate integration.
5. **Badge JSON** (medium priority): Low effort, high visibility.
6. **JUnit** (low priority): Nice for Jenkins/GitLab shops, but SARIF covers most cases.
7. **ASVS mapping** (low priority): Audit evidence, niche audience.

## Tradeoffs

**SARIF scope limitation:** SARIF is designed for discrete findings, not aggregate posture scores. gdev's "score is 82" doesn't map to SARIF. Use SARIF for individual findings and JSON for the full posture.

**Format proliferation:** Every new format is maintenance burden. Starting with JSON + text + SARIF covers 90% of use cases. Add JUnit and badge only if requested.

**Schema stability:** Versioning the JSON schema from day one (lesson from cargo-audit's instability) prevents breaking downstream tools. Use `schemaVersion` field. Document the schema in the repo.

## Depth Checklist

- [x] Underlying mechanism explained: Format structures, mapping between posture model and output formats
- [x] Key tradeoffs and limitations identified: SARIF scope, format proliferation, schema stability
- [x] Compared to at least one alternative: SARIF vs JUnit for CI, JSON vs OSCAL for compliance
- [x] Failure modes and edge cases: SARIF doesn't carry scores, badge color mapping edge cases, schema versioning
- [x] Concrete examples or reference implementations: Full JSON, SARIF, badge, JUnit examples
- [x] Report is standalone-readable: Complete format spec with implementation priority and consumer matrix
