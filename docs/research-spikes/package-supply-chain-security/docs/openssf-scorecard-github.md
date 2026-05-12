<!-- Source: https://github.com/ossf/scorecard -->
<!-- Retrieved: 2026-05-12 -->

# OpenSSF Scorecard: Comprehensive Overview

## What Is Scorecard?

OpenSSF Scorecard is an automated security assessment tool for open source projects. It evaluates repositories against multiple security best practices and assigns scores of 0-10 for each check, with an aggregate weighted score.

**Primary Goals:**
- Automate security posture analysis and trust decisions
- Improve security practices of critical open source projects
- Measure compliance with defined security policies

## Scoring System

Each check receives a 0-10 score. The aggregate score uses risk-weighted averaging:
- Critical risk checks: weight of 10
- High risk checks: weight of 7.5
- Medium risk checks: weight of 5
- Low risk checks: weight of 2.5

## Complete List of Scorecard Checks

| Check | Risk Level | Purpose |
|-------|-----------|---------|
| Binary-Artifacts | High | Ensures no compiled binaries are committed |
| Branch-Protection | High | Validates GitHub branch protection settings |
| CI-Tests | Low | Confirms automated testing in CI/CD pipelines |
| CII-Best-Practices | Low | Checks for OpenSSF Best Practices Badge |
| Code-Review | High | Verifies code review requirements before merging |
| Contributors | Low | Confirms contributors from >= 2 organizations |
| Dangerous-Workflow | Critical | Identifies risky GitHub Actions patterns |
| Dependency-Update-Tool | High | Detects automated dependency management tools |
| Fuzzing | Medium | Checks for fuzzing tools like OSS-Fuzz |
| License | Low | Confirms declared project license |
| Maintained | High | Verifies active maintenance (90+ days old) |
| Pinned-Dependencies | Medium | Ensures declared, pinned dependencies |
| Packaging | Medium | Confirms official package publishing from CI/CD |
| SAST | Medium | Detects static analysis tools (CodeQL, SonarCloud) |
| Security-Policy | Medium | Validates security policy file presence |
| Signed-Releases | High | Checks cryptographic release signing |
| Token-Permissions | High | Confirms GitHub workflow tokens are read-only |
| Vulnerabilities | High | Detects unfixed vulnerabilities via OSV service |
| Webhooks | Critical | Verifies webhook authentication tokens |

## How to Use Scorecard

### GitHub Action
The easiest approach for repository owners is the Scorecard GitHub Action, which automatically runs on code changes and displays alerts in the Security tab.

### Command-Line Interface
Installation methods: Docker, standalone binaries (Linux/OSX), package managers (Homebrew, Nix, AUR).

Basic command: `scorecard --repo=github.com/owner/repo`
Additional options: `--show-details`, `--checks=CheckName`, `--format=json`

### REST API
Pre-calculated scores available at api.scorecard.dev. Weekly scans omit CI-Tests, Contributors, and Dependency-Update-Tool checks due to API costs.

### Badges
Projects can display auto-updating Scorecard badges.

## Platform Support

**Supported Repositories:** GitHub.com, GitHub Enterprise Server, GitLab.com, GitLab self-hosted

**Scans:** 1 million+ repositories weekly. Results published in BigQuery public dataset.

## Important Limitations

1. Not definitive: Aggregate scores obscure individual behaviors
2. Heuristic-based: False positives and negatives occur
3. Applicability varies: Not all checks apply equally to all project types
4. Weekly scan omissions: API-cost considerations exclude three checks from REST API results
5. GitHub-focused: Project list currently GitHub-only (expansion planned)
