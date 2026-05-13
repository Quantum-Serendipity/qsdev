<!-- Source: https://github.com/ossf/scorecard -->
<!-- Retrieved: 2026-05-12 -->

# OpenSSF Scorecard Security Checks and Features

## Security Checks Overview

OpenSSF Scorecard performs 23 automated security assessments. Here's the complete list:

| Check Name | Purpose | Risk Level |
|---|---|---|
| Binary-Artifacts | Detects checked-in binaries | High |
| Branch-Protection | Validates branch protection settings | High |
| CI-Tests | Confirms CI/CD test execution | Low |
| CII-Best-Practices | Verifies OpenSSF badge status | Low |
| Code-Review | Ensures code review practices | High |
| Contributors | Checks multi-organization involvement | Low |
| Dangerous-Workflow | Identifies unsafe GitHub Actions patterns | Critical |
| Dependency-Update-Tool | Validates automated dependency updates | High |
| Fuzzing | Confirms fuzz testing implementation | Medium |
| License | Verifies license declaration | Low |
| Maintained | Assesses project activity (90+ days) | High |
| Pinned-Dependencies | Checks dependency pinning | Medium |
| Packaging | Validates official package publishing | Medium |
| SAST | Detects static analysis tool usage | Medium |
| Security-Policy | Confirms security policy presence | Medium |
| Signed-Releases | Verifies cryptographic release signatures | High |
| Token-Permissions | Ensures read-only workflow tokens | High |
| Vulnerabilities | Scans for unfixed CVEs via OSV | High |
| Webhooks | Validates webhook token authentication | Critical |

## Scoring System

Scorecard calculates weighted aggregate scores using risk-based multipliers:
- **Critical**: 10x weight
- **High**: 7.5x weight
- **Medium**: 5x weight
- **Low**: 2.5x weight

Individual checks score 0-10, producing a final aggregated metric.

## Output Formats

The document explicitly mentions two supported formats:
- **Default** (text-based output)
- **JSON** (structured data format)

## Command-Line Flags

Key usage options include:
- `--repo`: Target repository URL
- `--checks`: Run specific checks (comma-separated)
- `--show-details`: Display detailed failure reasons
- `--show-annotations`: Display maintainer context
- `--format`: Specify output format (json/default)
- `--npm`, `--pypi`, `--rubygems`, `--nuget`: Package ecosystem lookups

## Badge Features

Projects can embed dynamic badges displaying their Scorecard rating using markdown:
```
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/{owner}/{repo}/badge)](https://scorecard.dev/viewer/?uri=github.com/{owner}/{repo})
```

Badges auto-update with repository changes when `publish_results: true` is enabled in GitHub Actions.

## Authentication

Scorecard requires GitHub/GitLab tokens to bypass API rate limits via environment variables: `GITHUB_AUTH_TOKEN`, `GITHUB_TOKEN`, `GITLAB_AUTH_TOKEN`, or GitLab App credentials.
