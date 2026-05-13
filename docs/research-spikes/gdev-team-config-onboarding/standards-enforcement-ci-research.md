# Standards Enforcement in CI

## Research Question

How should `gdev check` or `gdev validate` work to verify that a project meets team standards, security hardening is intact, and required tools are properly configured?

## The Problem

Generated configuration files can drift from expected state through:
- Manual edits that weaken security (removing deny rules, disabling hooks)
- Partial updates (some files updated, others not)
- New team standards not yet applied to older projects
- Engineers disabling pre-commit hooks locally and committing without them
- Tooling version skew producing different output

CI is the enforcement point -- the one place where checks run consistently regardless of individual developer behavior.

## Command Design: `gdev check`

### Scope

`gdev check` validates the project's gdev configuration against the expected state. It is a read-only command that never modifies files.

```
$ gdev check
  gdev check v0.16.0 | project: acme-widget-service

  Binary Compatibility
  ✓ gdev version 0.16.0 satisfies >= 0.15.0

  Config Integrity
  ✓ .gdev.yaml present and valid (schema version 2)
  ✓ .gdev.yaml profile "go-web-service" recognized

  Required Tools
  ✓ devenv.sh enabled
  ✓ Claude Code enabled
  ✓ pre-commit hooks enabled (ripsecrets, gitleaks, semgrep)
  ✗ age_gating: disabled in .gdev.yaml (required by org policy)

  Generated File State
  ✓ devenv.yaml matches expected state
  ✓ devenv.nix present (user-modified -- skipping content check)
  ✓ .envrc matches expected state
  ✗ .claude/settings.json: deny rule "Bash(pip install *)" missing
  ✓ .pre-commit-config.yaml matches expected state

  Security Hardening
  ✓ Package age-gating configured (.npmrc, pip.conf)
  ✓ Install script blocking configured
  ✓ Lock file enforcement configured
  ✗ Vulnerability scanning not configured (missing osv-scanner in CI)

  Summary: 2 issues found (1 critical, 1 warning)
  
  Critical:
  - age_gating disabled -- required by org security policy
  
  Warning:
  - deny rule missing in settings.json -- run `gdev init --update` to fix
  
  Exit code: 1
```

### Check Categories

1. **Binary compatibility** -- gdev version meets `.gdev.yaml` constraint
2. **Config integrity** -- `.gdev.yaml` parses correctly, profile exists, schema version supported
3. **Required tools** -- Org policy mandates certain tools are enabled (security hardening, pre-commit hooks, Claude Code)
4. **Generated file state** -- Machine-owned files match expected output; human-edited files checked for required content (deny rules in settings.json, section markers in CLAUDE.md)
5. **Security hardening** -- Per-ecosystem security configs present and correct (age-gating, install script blocking, lock file enforcement, vulnerability scanning)

### Policy Definition

The org's required standards are compiled into the binary:

```go
type OrgPolicy struct {
    RequiredTools      []string          // e.g., ["age_gating", "install_script_blocking", "pre_commit_hooks"]
    RequiredDenyRules  []string          // deny rules that must be in settings.json
    RequiredPreCommit  []string          // hooks that must be in .pre-commit-config.yaml
    RequiredCIChecks   []string          // CI workflow steps that must exist
    MinGdevVersion     string            // minimum gdev version for all projects
    SecurityLevel      string            // "baseline" | "enhanced" | "strict"
}
```

Projects can exceed the policy (add more restrictions) but never fall below it.

### Exit Codes

Following Unix convention and CI best practices:

| Exit Code | Meaning | CI Action |
|-----------|---------|-----------|
| 0 | All checks pass | Pipeline continues |
| 1 | One or more checks failed | Pipeline fails |
| 2 | gdev check itself errored (can't read config, etc.) | Pipeline fails |

### Output Formats

```
gdev check                    # Human-readable (default)
gdev check --format json      # Machine-readable JSON
gdev check --format sarif     # SARIF for GitHub Security tab
gdev check --format junit     # JUnit XML for CI test reporting
```

JSON output example:
```json
{
  "version": "0.16.0",
  "project": "acme-widget-service",
  "timestamp": "2026-05-12T15:30:00Z",
  "checks": [
    {
      "category": "security_hardening",
      "name": "age_gating",
      "status": "fail",
      "severity": "critical",
      "message": "Package age-gating is disabled in .gdev.yaml",
      "remediation": "Set security.age_gating: true in .gdev.yaml"
    }
  ],
  "summary": {
    "total": 12,
    "pass": 10,
    "fail": 2,
    "skip": 0
  }
}
```

### CI Integration

#### GitHub Actions

```yaml
# .github/workflows/gdev-check.yml
name: gdev Standards Check
on: [pull_request]

jobs:
  gdev-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install gdev
        run: curl -fsSL https://get.myxdev.dev | sh
      - name: Run gdev check
        run: gdev check --format sarif > gdev-check.sarif
      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: gdev-check.sarif
```

#### GitLab CI

```yaml
gdev-check:
  stage: validate
  script:
    - curl -fsSL https://get.myxdev.dev | sh
    - gdev check --format junit > gdev-check.xml
  artifacts:
    reports:
      junit: gdev-check.xml
```

### What Gets Checked vs What Doesn't

**Checked (deterministic, enforceable):**
- Config file presence and validity
- Required tool enablement
- Security deny rules present in settings.json
- Pre-commit hook configuration
- Package manager hardening configs (.npmrc, pip.conf, etc.)
- Lock file presence for detected ecosystems
- CLAUDE.md section markers intact

**Not checked (would cause false positives):**
- devenv.nix content (human-edited, too variable)
- Custom skill content (user-written)
- MCP server authentication state
- Whether devenv shell actually works (requires Nix evaluation)
- Individual developer's local tool versions (only project config)

### Relationship to `gdev devenv doctor`

`gdev check` and `gdev devenv doctor` serve different purposes:

| | `gdev check` | `gdev devenv doctor` |
|---|---|---|
| **Scope** | Project config compliance | Machine/system state |
| **Runs in** | CI + local | Local only |
| **Checks** | Standards, security, config state | Tool installation, versions, shell hooks |
| **Modifies files** | Never | Never (but suggests `gdev devenv setup`) |
| **Exit code** | 0 (pass) / 1 (fail) | 0 (healthy) / 1 (issues) |

They complement each other: `gdev devenv doctor` ensures the machine is ready; `gdev check` ensures the project is compliant.

## Prior Art Comparison

| Tool | Validation Command | Scope | Output |
|------|-------------------|-------|--------|
| Terraform | `terraform validate` | Config syntax + internal consistency | Human + JSON |
| ESLint | `eslint .` | Code style + quality | Human + JSON + SARIF |
| Renovate | `renovate-config-validator` | Config syntax | Human |
| Nx | `nx lint` | Workspace constraints | Human |
| OpenSSF Scorecard | `scorecard --repo` | Supply chain security posture | Human + JSON + SARIF |
| npm audit | `npm audit` | Dependency vulnerabilities | Human + JSON |
| gdev (proposed) | `gdev check` | Config compliance + security hardening | Human + JSON + SARIF + JUnit |

gdev's `gdev check` is closest to OpenSSF Scorecard in philosophy -- it evaluates security posture rather than just config syntax -- but focused on the project's gdev-managed configuration rather than the entire supply chain.

## Auto-Fix Mode

For issues that have deterministic fixes:

```
$ gdev check --fix
  ✗ .claude/settings.json: deny rule "Bash(pip install *)" missing
    → Added deny rule. Settings.json updated.
  
  ✗ Vulnerability scanning not configured
    → Cannot auto-fix: requires CI workflow changes. See remediation steps.
  
  1 issue fixed, 1 requires manual action.
```

Auto-fix only applies to:
- Missing deny rules in settings.json (additive, safe)
- Missing pre-commit hooks (additive, safe)
- Missing .gitignore entries (additive, safe)
- Outdated library-managed skills/rules (overwrite, safe)

Auto-fix does NOT apply to:
- Config structure changes (requires wizard re-run)
- Security settings that were explicitly disabled (respects user intent)
- CI workflow changes (too variable across providers)

## Depth Checklist

- [x] Underlying mechanism explained -- check categories, policy definition, exit codes
- [x] Key tradeoffs and limitations identified -- what to check vs what to skip
- [x] Compared to alternatives -- Terraform, ESLint, OpenSSF Scorecard, npm audit
- [x] Failure modes and edge cases described -- false positives, auto-fix scope
- [x] Concrete examples -- CLI output, CI workflow YAML, JSON output
- [x] Report is standalone-readable
