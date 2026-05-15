# Team-Level Reporting & Multi-Repo Aggregation

## Problem

A consulting firm manages 10-50 client projects simultaneously. The engineering lead needs to answer: "Which projects have degraded security posture? Which are missing defenses? Are all projects on the latest gdev version?" This requires aggregating per-project posture data into an organizational view.

## Prior Art: Multi-Repo Aggregation Tools

### OpenSSF Scorecard Monitor
- GitHub Action that tracks Scorecard results across organizations
- JSON database with historical scores per repo
- Markdown reports with score comparison over time
- Auto-generates GitHub issues when scores drop
- Used by Node.js Security WG for organizational oversight

### DefectDojo
- Open-source vulnerability management platform (OWASP Flagship)
- Ingests results from 200+ security tools
- Normalizes, deduplicates, prioritizes across all projects
- Product-level and organization-level dashboards
- Overkill for gdev's use case but demonstrates the aggregation pattern

### GitLab Security Dashboard
- Native multi-project vulnerability aggregation
- Per-group and per-instance security dashboards
- Requires GitLab Ultimate (enterprise pricing)
- Demonstrates the "zero-config aggregation via CI" model

## Architecture Options

### Option A: CI Artifact Aggregation (Recommended)

Each project's CI pipeline generates `qsdev status --json > posture.json` as a build artifact. A separate aggregation job collects artifacts across repos and generates the team report.

```yaml
# Per-project CI (GitHub Actions)
- name: Generate posture report
  run: qsdev status --json > posture.json
- name: Upload posture artifact
  uses: actions/upload-artifact@v4
  with:
    name: gdev-posture
    path: posture.json
```

```yaml
# Aggregation repo (runs weekly via cron)
- name: Collect posture reports
  run: |
    for repo in $(cat repos.txt); do
      gh run download --repo "$repo" --name gdev-posture -D "reports/$repo"
    done
- name: Aggregate
  run: qsdev team-report --input-dir reports/ --output team-posture.md
```

**Pros:** No new infrastructure. Uses existing CI. Each project controls its own scan schedule. Reports are git-committed for audit trail.

**Cons:** Depends on CI running recently. Aggregation is pull-based (must enumerate repos). Requires cross-repo artifact access permissions.

### Option B: Git-Based Collection (Scorecard Monitor Pattern)

A central repo contains a scope file listing all tracked projects. A scheduled action clones each repo, runs `qsdev status --json`, and stores results in a JSON database.

```json
// scope.json
{
  "projects": [
    {"repo": "org/client-a-api", "branch": "main"},
    {"repo": "org/client-a-frontend", "branch": "main"},
    {"repo": "org/client-b-app", "branch": "develop"},
    {"repo": "org/internal-tools", "branch": "main"}
  ]
}
```

```json
// database.json (scorecard-monitor pattern)
{
  "org/client-a-api": {
    "previous": [
      {"score": 78, "grade": "B", "date": "2026-05-05"},
      {"score": 75, "grade": "B", "date": "2026-04-28"}
    ],
    "current": {"score": 82, "grade": "B+", "date": "2026-05-12"}
  }
}
```

**Pros:** Self-contained. Historical tracking built-in. Works for repos without CI.

**Cons:** Requires cloning each repo (slow, disk-heavy). Central repo needs read access to all tracked repos. Harder to scale.

### Option C: Push-Based Webhook (DefectDojo Pattern)

Each project's CI POSTs its posture JSON to a central endpoint. Dashboard aggregates in real-time.

**Pros:** Real-time updates. No polling.

**Cons:** Requires running a server. Overkill for a consulting firm. Creates an infrastructure dependency.

### Recommendation

**Option A (CI artifact aggregation) for MVP.** It requires no new infrastructure, works with existing GitHub Actions, and the aggregation script is a simple Go program or shell script. Option B as a future enhancement for historical tracking.

## Team Report Design

### Summary Dashboard (Markdown)

```markdown
# Team Security Posture — 2026-05-12

## Overview
| Metric | Value |
|--------|-------|
| Projects tracked | 12 |
| Average score | 79/100 (B) |
| Projects at baseline | 10/12 (83%) |
| Projects at enhanced | 6/12 (50%) |
| Total critical vulns | 0 |
| Total high vulns | 7 |
| Projects needing update | 3 |

## Project Scores

| Project | Score | Grade | Baseline | Enhanced | Vulns (C/H) | Last Scan |
|---------|-------|-------|----------|----------|-------------|-----------|
| client-a-api | 92 | A | PASS | PASS | 0/0 | 1h ago |
| client-a-frontend | 85 | B+ | PASS | FAIL | 0/2 | 1h ago |
| client-b-app | 82 | B+ | PASS | FAIL | 0/1 | 3h ago |
| client-c-monorepo | 71 | C+ | FAIL | FAIL | 0/3 | 6h ago |
| internal-tools | 65 | C | FAIL | FAIL | 0/1 | 12h ago |

## Attention Required

### Critical Issues (0)
None.

### High Priority (3)
- **client-c-monorepo**: Baseline FAIL — lock file missing for Python
- **internal-tools**: Baseline FAIL — pre-commit hooks not installed
- **client-a-frontend**: 2 high vulns in npm dependencies

### Score Changes This Week
| Project | Previous | Current | Change |
|---------|----------|---------|--------|
| client-a-api | 88 | 92 | +4 (improved) |
| internal-tools | 72 | 65 | -7 (degraded) |
```

### JSON Aggregation Format

```json
{
  "schemaVersion": "1.0.0",
  "generatedAt": "2026-05-12T15:00:00Z",
  "summary": {
    "projectCount": 12,
    "averageScore": 79,
    "baselinePassRate": 0.83,
    "enhancedPassRate": 0.50,
    "totalCriticalVulns": 0,
    "totalHighVulns": 7,
    "projectsNeedingUpdate": 3
  },
  "projects": [
    {
      "name": "client-a-api",
      "repo": "org/client-a-api",
      "score": {"total": 92, "grade": "A", "defense": 95, "config": 100, "depHealth": 82},
      "conformance": {"baseline": true, "enhanced": true},
      "vulns": {"critical": 0, "high": 0},
      "gdevVersion": "1.2.0",
      "lastScan": "2026-05-12T14:00:00Z"
    }
  ],
  "trends": [
    {"project": "client-a-api", "scores": [{"date": "2026-05-05", "score": 88}, {"date": "2026-05-12", "score": 92}]},
    {"project": "internal-tools", "scores": [{"date": "2026-05-05", "score": 72}, {"date": "2026-05-12", "score": 65}]}
  ],
  "alerts": [
    {"project": "client-c-monorepo", "severity": "high", "message": "Baseline FAIL: lock file missing for Python"},
    {"project": "internal-tools", "severity": "high", "message": "Baseline FAIL: pre-commit hooks not installed"},
    {"project": "internal-tools", "severity": "medium", "message": "Score degraded by 7 points this week"}
  ]
}
```

## Team Commands

```
qsdev team-report                       # Generate team posture report
qsdev team-report --input-dir reports/  # Aggregate from directory of posture JSONs
qsdev team-report --scope scope.json    # Use scope file for project list
qsdev team-report --format md           # Markdown output (default)
qsdev team-report --format json         # JSON output
qsdev team-report --threshold 75        # Highlight projects below score threshold
qsdev team-report --trend               # Include historical trend data
```

### Alternative: `qsdev status --multi`

For smaller setups, support scanning multiple local project directories:

```
qsdev status --multi ~/projects/client-a ~/projects/client-b ~/projects/internal-tools
```

This runs `qsdev status` in each directory and aggregates the results. Simpler than CI-based collection for local-only workflows.

## GitHub Issue Generation (Scorecard Monitor Pattern)

When scores drop or conformance fails, auto-create GitHub issues:

```
Title: [gdev] Security posture degraded: internal-tools (65/100, -7)

Body:
## Security Posture Alert

Project **internal-tools** security posture has degraded.

| Metric | Previous | Current |
|--------|----------|---------|
| Score | 72 | 65 |
| Grade | C+ | C |
| Baseline | PASS | FAIL |

### Issues Found
- Pre-commit hooks not installed
- 1 high-severity vulnerability in npm dependencies

### Recommended Actions
1. Run `qsdev hooks install` to restore pre-commit hooks
2. Run `npm audit fix` to address vulnerability
3. Run `qsdev status` to verify posture

Labels: security, gdev-posture
Assignee: @team-lead
```

## Scaling Considerations

**10 projects:** Local multi-directory scan works fine. CI aggregation is straightforward.

**50 projects:** CI artifact aggregation is the practical approach. Scoring is instant (just JSON parsing). The bottleneck is artifact collection (one API call per repo).

**100+ projects:** Consider a lightweight server (Option C) or batch API calls. GitHub's REST API rate limits may require a PAT with elevated limits.

For a consulting firm, 10-50 projects is the realistic range. CI artifact aggregation handles this comfortably.

## Tradeoffs

**Centralization vs decentralization:** A central dashboard server is powerful but creates an infrastructure dependency and operational burden. CI artifact aggregation is decentralized (each project owns its scan) with lightweight centralized collection. For a consulting firm without dedicated DevOps staff, decentralized is better.

**Freshness vs cost:** More frequent scanning gives fresher data but costs CI minutes. Weekly aggregation is probably sufficient for a team overview; critical findings should be caught by per-project CI on every PR.

**Historical depth:** Storing score history enables trend analysis but grows the database. The scorecard-monitor pattern (JSON file in git) is self-limiting -- git compression handles the growth. Keep 90 days of weekly snapshots.

**Cross-client visibility:** A consulting firm might not want all projects visible in one dashboard (client confidentiality). The scope file should support grouping by client with per-group access controls.

## Depth Checklist

- [x] Underlying mechanism explained: Three architecture options with tradeoffs, CI artifact pipeline, aggregation logic
- [x] Key tradeoffs and limitations identified: Centralization, freshness, historical depth, client confidentiality
- [x] Compared to at least one alternative: Scorecard Monitor, DefectDojo, GitLab Security Dashboard
- [x] Failure modes and edge cases: Missing artifacts, stale scans, rate limits, cross-repo permissions
- [x] Concrete examples or reference implementations: Full markdown report mockup, JSON schema, CI pipeline examples, issue template
- [x] Report is standalone-readable: Complete team reporting architecture with scaling guidance
