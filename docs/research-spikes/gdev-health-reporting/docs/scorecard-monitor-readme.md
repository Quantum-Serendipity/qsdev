<!-- Source: https://github.com/ossf/scorecard-monitor -->
<!-- Retrieved: 2026-05-12 -->

# OpenSSF Scorecard Monitor Documentation

## Overview

The scorecard-monitor is a GitHub Action that centralizes OpenSSF Scorecard tracking across organizations. It "Simplifies OpenSSF Scorecard tracking in your organization with automated markdown and JSON reports, plus optional GitHub issue alerts."

## Core Functionality

**Data Aggregation**: The tool scans specified organizations for repositories already analyzed by OpenSSF Scorecard, either through the official Action, CLI, or automatic weekly tracking via cronjob.

**Multi-Format Reporting**: Results are exported in two formats:
- **Markdown reports** with essential metadata (hash, date, score) and comparative analysis against prior scores
- **JSON database** storing current and historical records locally within the repository

**Change Tracking**: The system maintains score history, enabling it to identify improvements or declines and compare current values against previous captures.

**Issue Generation**: When scores change, the action can automatically create GitHub issues with updated information, including assignees, labels, and links to remediation tools like StepSecurity.

## Configuration Options

Key inputs include:

| Option | Purpose |
|--------|---------|
| `scope` | Defines tracked repositories |
| `database` | JSON storage path for scores |
| `report` | Markdown output location |
| `auto-commit` / `auto-push` | Automation controls |
| `generate-issue` | Enable issue creation |
| `discovery-enabled` | Auto-scan organizations |
| `discovery-orgs` | Target organization list |
| `max-request-in-parallel` | API concurrency control |
| `report-tags-enabled` | Embed reports within existing content |
| `results-path` | Direct Scorecard JSON input |

## Output Examples

**Database Structure** (JSON):
```json
{
  "github.com": {
    "organization": {
      "repository": {
        "previous": [{"score": 6.7, "date": "2022-08-21"}],
        "current": {"score": 4.4, "date": "2022-11-28"}
      }
    }
  }
}
```

**Scope Structure** (JSON):
```json
{
  "github.com": {
    "included": {"org": ["repo1", "repo2"]},
    "excluded": {"org": ["repo3"]}
  }
}
```

**Action Outputs**: Score data is exported as JSON, accessible via `${{ steps.id.outputs.scores }}` in workflows.

## Advanced Features

- **Discovery Mode**: Lists all tracked repositories across organizations
- **Visualization Integration**: Links to Scorecard Visualizer and deps.dev tools
- **Comparison Tools**: Supports scorecard comparison between commits
- **Branch Protection Compatibility**: Can generate pull requests instead of direct commits
- **Tag-Based Embedding**: Uses custom comment tags to update reports without affecting surrounding content
- **Custom Remediation Links**: Provides direct pathways to StepSecurity for score improvement

## Real-World Usage

Notable implementations include Node.js Security Working Group, One Beyond, and NodeSecure, which use the tool to generate organizational security reports and track compliance metrics across multiple repositories.
