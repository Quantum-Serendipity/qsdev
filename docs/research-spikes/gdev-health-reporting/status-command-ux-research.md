# Status/Report Command UX Design Research

## Problem

Design the terminal UX for `qsdev status` -- the command a consulting engineer runs to instantly assess a project's security posture. Must work for both quick glances (is everything green?) and detailed investigation (what exactly is wrong and how do I fix it?).

## Prior Art UX Patterns

### Flutter Doctor Pattern
```
Doctor summary (to see all details, run flutter doctor -v):
[✓] Flutter (Channel stable, 3.27.0)
[✓] Android toolchain - develop for Android devices
[✗] Xcode - develop for iOS and macOS
    ✗ Xcode not installed
    ! Run: xcode-select --install
[✓] Chrome - develop for the web
[!] Android Studio (not installed)
[✓] Connected device (1 available)
```

**Strengths:** Instant visual scan via checkmarks. Three-state (pass/fail/warn). Remediation inline. `-v` for details.

### npm audit Pattern
```
6 vulnerabilities (1 moderate, 3 high, 2 critical)

To address all issues, run:
  npm audit fix

Run `npm audit` for details.
```

**Strengths:** Count-first summary. Actionable next step. Graduated detail levels.

### Scorecard Pattern
```
RESULTS
-------
Aggregate score: 7.5 / 10

Check scores:
  |---------|-------------------------|--------------------------------|
  |  SCORE  |          CHECK          |             REASON             |
  |---------|-------------------------|--------------------------------|
  |  10/10  | Binary-Artifacts        | no binaries found in the repo  |
  |  10/10  | Branch-Protection       | branch protection enabled      |
  |   0/10  | Signed-Releases         | no releases found              |
  |---------|-------------------------|--------------------------------|
```

**Strengths:** Table format for many checks. Score per check. Aggregate headline.

## Proposed `qsdev status` Command Design

### Default Output (Quick Scan)

```
qsdev status — Project Security Posture

  Score: 82/100 (B+)
  Conformance: baseline PASS, enhanced FAIL

  Defense Coverage (87/100)
    [✓] age-gating          npm: 72h, pip: 72h
    [✓] script-blocking     @lavamoat/allow-scripts active
    [✓] lock-file-enforce   package-lock.json + requirements.txt valid
    [✓] vuln-scanning       OSV Scanner configured
    [~] pretooluse-hooks    attach-guard: 2/48 rules disabled
    [✓] nix-hardening       10/10 settings applied
    [✓] sast                Semgrep with auto config
    [✓] secrets-scanning    Gitleaks + ripsecrets
    [ ] container-security  No Dockerfile detected (N/A)
    [ ] license-compliance  Available, not enabled

  Config Health (91/100)
    [✓] 7 files current
    [~] 1 file outdated: .pre-commit-config.yaml (v1.1.0 → v1.2.0)
    [i] 2 files user-modified (expected): devenv.nix, .gitleaks.toml

  Dependency Health (68/100)
    Vulns: 0 critical, 2 high, 6 moderate, 15 low
    Lock files: 3/3 valid
    Last scan: 2h ago

  Run 'qsdev status --verbose' for details.
  Run 'qsdev status --fix' for auto-remediation suggestions.
```

### Design Decisions

**Indicators:**
- `[✓]` (green) = fully enabled/healthy
- `[~]` (yellow) = partially enabled or outdated
- `[ ]` (dim) = disabled or not applicable
- `[✗]` (red) = misconfigured, broken, or critical issue

**Score display:** Numeric score + letter grade for instant interpretation. Both because engineers differ -- some prefer numbers, some prefer grades.

**Conformance inline:** One-line summary of conformance tracks. Details via `--verbose`.

**Section ordering:** Defense first (it's what gdev is about), config second (drift detection), dependencies third (slowest to assess, most volatile).

**Remediation hints:** Inline for simple cases ("Available, not enabled"). `--fix` mode for detailed suggestions. Never bury the fix.

### Verbose Output (`qsdev status --verbose`)

Expands each section with full detail:

```
  Defense Coverage (87/100)

    [✓] age-gating (weight: high)
        npm:  72h quarantine via .npmrc registry proxy
        pip:  72h quarantine via pip.conf index-url
        go:   N/A (go.sum provides integrity, not age-gating)
        rust: N/A (crates.io has no age-gating support)

    [~] pretooluse-hooks (weight: high)
        attach-guard installed: .claude/hooks/package-guard.py
        Active rules: 46/48
        Disabled rules:
          - npm-install-global (disabled by user in settings.json)
          - pip-install-editable (disabled by user in settings.json)
        Fix: Review disabled rules with 'qsdev claudecode rules'
```

### Subcommands and Flags

```
qsdev status                    # Quick posture summary (default)
qsdev status --verbose          # Full detail on every check
qsdev status --json             # Machine-readable JSON (PostureReport)
qsdev status --sarif            # SARIF 2.1.0 output
qsdev status --format badge     # Shields.io endpoint JSON
qsdev status --fix              # Show remediation commands
qsdev status --audit-level high # Exit non-zero if high+ findings
qsdev status --scan             # Run fresh vulnerability scans (slow)
qsdev status --quiet            # Score and grade only, no detail
qsdev status defense            # Defense coverage section only
qsdev status config             # Config health section only
qsdev status deps               # Dependency health section only
qsdev status tools              # Tool inventory (enabled/available/disabled)
```

### `qsdev status tools` — Tool Inventory

Specific view for "what's available and what's enabled":

```
qsdev status tools

  Enabled Tools (12/16)
    Security:
      [✓] semgrep          SAST, .semgrep.yml
      [✓] gitleaks         Secrets scanning, .gitleaks.toml
      [✓] ripsecrets       Pre-commit secrets, hook only
      [✓] attach-guard     Package guardrails, .claude/hooks/package-guard.py
      [ ] container-sec    Grype/Syft/Cosign — not detected (no Dockerfile)
      [ ] license-scan     ScanCode — opt-in, run 'qsdev enable license-compliance'

    AI Agent:
      [✓] agent-postmortem Verification skill
      [✓] version-sentinel Dependency change guardrails
      [✓] semble           Semantic code search MCP
      [✓] context7         Library docs MCP
      [✓] trail-of-bits    Security audit skills (3 active)

    Workflow:
      [✓] commitlint       Commit message linting
      [✓] changelog        git-cliff changelog generation
      [ ] secretspec        Dev secrets management — run 'qsdev enable secretspec'
```

### Color Coding Strategy

Following terminal conventions and accessibility:
- **Green**: Healthy, enabled, current, no issues
- **Yellow/Amber**: Partial, outdated, warnings, available-but-disabled
- **Red**: Broken, critical issues, misconfigured, missing required
- **Dim/Gray**: Not applicable, informational
- **Bold white**: Section headers, scores, grades

Support `NO_COLOR` and `FORCE_COLOR` environment variables per the [no-color.org](https://no-color.org) convention. JSON/SARIF output never includes color codes.

### Exit Code Strategy

```
Exit 0: No findings at or above --audit-level threshold (default: none)
Exit 1: Findings at or above --audit-level threshold
Exit 2: gdev itself failed (not initialized, corrupt state, etc.)
```

The `--audit-level` flag accepts: `none` (always 0), `critical`, `high`, `moderate`, `low`, `info`, `any`.

For CI:
```yaml
# GitHub Actions example
- run: qsdev status --json > posture.json
- run: qsdev status --audit-level high  # Fails build if high+ vulns
- run: qsdev status --sarif > results.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

### Progressive Detail Levels

Following the npm pattern of graduated disclosure:

| Mode | Content | Use Case |
|------|---------|----------|
| `--quiet` | Score + grade only | Badge generators, scripts |
| (default) | Summary with section scores | Daily developer use |
| `--verbose` | Full detail per check | Investigation, debugging |
| `--json` | Complete PostureReport | CI, dashboards, audit |
| `--sarif` | Security findings only | GitHub Code Scanning |
| `--fix` | Remediation commands | Fixing issues |

### Performance Considerations

**Fast path (< 1 second):** Defense coverage and config health are purely local operations -- read gdev state, hash files, compare. These should always be instant.

**Slow path (5-30 seconds):** Dependency health requires running ecosystem audit tools (`npm audit`, `govulncheck`, etc.). Default to cached results with timestamp. `--scan` flag for fresh results.

**Caching strategy:**
- Store last scan results in `.gdev/cache/vuln-scan.json`
- Show `Last scan: 2h ago` in output
- Auto-scan if cache older than configurable threshold (default: 24h)
- CI always uses `--scan` for fresh results

## Tradeoffs

**Score granularity vs simplicity:** A single 0-100 score is easy to understand but hides nuance. The three sub-scores (defense/config/deps) provide context without overwhelming. Individual per-check scores are available in `--verbose` and JSON but not in default output.

**Opinionated defaults vs configurability:** The grade scale (A/B/C/D/F) and weight assignments are opinionated. Making everything configurable creates analysis paralysis. Recommendation: ship strong defaults, allow override via `.gdev-policy.yaml`, but don't make it a first-run question.

**Remediation detail:** `qsdev status --fix` vs embedding fixes in default output. Default output should hint ("run 'qsdev enable X'") but not overwhelm. `--fix` mode generates a script-like sequence of commands.

## Comparison to Alternatives

**vs `flutter doctor`:** Similar check-based approach, but gdev adds scoring (flutter doctor is pure pass/fail). gdev's defense layers are more granular than flutter's 5-6 categories.

**vs `npm audit`:** npm audit is vuln-only. qsdev status covers a broader surface (defenses + config + vulns). npm audit's severity threshold pattern directly adopted.

**vs Scorecard:** Scorecard is repo-external (analyzes from outside). qsdev status is project-internal (knows its own config state). This gives gdev much richer config-health insight but no supply-chain perspective.

## Depth Checklist

- [x] Underlying mechanism explained: Progressive disclosure hierarchy, exit code strategy, caching model
- [x] Key tradeoffs and limitations identified: Score granularity, opinionated defaults, scan performance
- [x] Compared to at least one alternative: flutter doctor, npm audit, Scorecard
- [x] Failure modes and edge cases: Offline mode, stale cache, corrupt state, NO_COLOR support
- [x] Concrete examples or reference implementations: Full terminal mockups, CLI flag inventory, CI example
- [x] Report is standalone-readable: Complete command design spec with rationale
