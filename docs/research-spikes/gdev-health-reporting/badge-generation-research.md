# Security Posture Badge Generation

## Problem

Consulting engineers want a visual indicator of security posture in project READMEs -- analogous to Snyk's vulnerability badges but covering gdev's broader defense posture rather than just dependency vulnerabilities.

## Prior Art

### Snyk Badges
- URL pattern: `https://snyk.io/test/github/{owner}/{repo}/badge.svg`
- Shows vulnerability count (green = 0 vulns, red = N vulns)
- Requires Snyk account and project monitoring
- Per-manifest targeting via `targetFile` query param

### OpenSSF Scorecard Badges
- URL pattern: `https://api.scorecard.dev/projects/github.com/{owner}/{repo}/badge`
- Shows aggregate score (0-10)
- Auto-updates when `publish_results: true` in GitHub Action
- No account required (public repos)

### shields.io Endpoint Badge
- URL pattern: `https://img.shields.io/endpoint?url=<encoded-json-url>`
- Custom JSON endpoint returns: `{schemaVersion: 1, label, message, color}`
- Supports custom styling (flat, flat-square, plastic, for-the-badge)
- 15-minute cache by default, configurable via `cacheSeconds`

## Badge Design for gdev

### Primary Badge: Security Posture Score

```
[gdev security | 82/100 B+]  (green background)
```

JSON endpoint:
```json
{
  "schemaVersion": 1,
  "label": "gdev security",
  "message": "82/100 B+",
  "color": "green"
}
```

### Alternative Badge: Conformance Status

```
[gdev baseline | PASS]    (brightgreen)
[gdev enhanced | FAIL]    (orange)
```

### Alternative Badge: Defense Coverage

```
[defenses | 8/10 enabled]  (green)
```

### Color Mapping

| Score Range | Grade | Color | shields.io Color |
|-------------|-------|-------|-----------------|
| 90-100 | A | Bright green | `brightgreen` |
| 75-89 | B | Green | `green` |
| 60-74 | C | Yellow | `yellow` |
| 45-59 | D | Orange | `orange` |
| 0-44 | F | Red | `red` |

For conformance badges:
- PASS: `brightgreen`
- FAIL: `red`
- PARTIAL: `yellow`

## Generation Methods

### Method 1: Static File in Repo (Recommended)

CI job generates badge JSON and commits to repo:

```yaml
# .github/workflows/posture.yml
name: Security Posture
on:
  push:
    branches: [main]
  schedule:
    - cron: '0 6 * * 1'  # Weekly Monday 6am

jobs:
  posture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Generate badge
        run: gdev status --format badge > .gdev/badge.json
      - name: Commit badge
        run: |
          git config user.name "gdev-bot"
          git config user.email "gdev@noreply"
          git add .gdev/badge.json
          git diff --cached --quiet || git commit -m "Update security posture badge"
          git push
```

README usage:
```markdown
[![gdev security](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/org/repo/main/.gdev/badge.json)](https://github.com/org/repo/actions)
```

**Pros:** No server needed. Works with any git host. Badge updates on every push to main. Fully self-contained.

**Cons:** Requires CI job. Small commit noise. Badge freshness depends on CI frequency. Raw URL must be public.

### Method 2: GitHub Pages Endpoint

Generate badge JSON to a `gh-pages` branch:

```yaml
- name: Deploy badge
  uses: peaceiris/actions-gh-pages@v4
  with:
    github_token: ${{ secrets.GITHUB_TOKEN }}
    publish_dir: .gdev/
    destination_dir: badges/
    keep_files: true
```

README usage:
```markdown
[![gdev security](https://img.shields.io/endpoint?url=https://org.github.io/repo/badges/badge.json)]
```

**Pros:** Clean URL. No commit noise on main branch. Can host multiple badge variants.

**Cons:** Requires GitHub Pages enabled. Only works for GitHub.

### Method 3: CLI-Generated SVG (Offline)

`gdev status --format svg` generates an SVG badge file directly, without relying on shields.io:

```go
func generateBadgeSVG(label, message, color string) string {
    // Use go-shields or badger library to generate SVG
    // Or embed a simple template:
    return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" ...>
      <rect width="%d" fill="#555"/>
      <rect x="%d" width="%d" fill="%s"/>
      <text>%s</text>
      <text>%s</text>
    </svg>`, labelWidth, labelWidth, messageWidth, color, label, message)
}
```

**Pros:** Fully offline. No external service dependency. Can be committed directly.

**Cons:** SVG rendering is tricky to get right (font metrics, sizing). Maintaining badge SVG templates is ongoing work. Better to use shields.io unless offline is a hard requirement.

### Recommendation

**Method 1 (static file) for most projects.** Zero infrastructure, works everywhere. Method 2 for organizations that already use GitHub Pages. Method 3 only if offline badge generation is specifically needed.

## Multiple Badges

A project might display several badges:

```markdown
[![gdev security](https://img.shields.io/endpoint?url=...badge.json)](...)
[![gdev baseline](https://img.shields.io/endpoint?url=...baseline-badge.json)](...)
[![gdev defenses](https://img.shields.io/endpoint?url=...defense-badge.json)](...)
```

`gdev status --format badge` generates the primary score badge. Additional badge variants:

```
gdev status --format badge                    # Score badge (default)
gdev status --format badge --badge-type conformance  # Baseline conformance
gdev status --format badge --badge-type defense      # Defense coverage
gdev status --format badge --badge-type vulns        # Vulnerability count
```

Or generate all at once:
```
gdev status --format badge --all-badges --output-dir .gdev/badges/
```

## Integration with Team Reporting

The team report can include a badge summary table:

```markdown
| Project | Score | Baseline | Vulns |
|---------|-------|----------|-------|
| client-a-api | ![](badge-url) | ![](baseline-url) | ![](vulns-url) |
```

This creates a visual dashboard in the team report markdown.

## Tradeoffs

**shields.io dependency:** Using shields.io endpoint badges means depending on an external service. For public repos this is standard practice (millions of badges served daily). For private/air-gapped environments, Method 3 (local SVG) is needed.

**Badge freshness:** Badges reflect the last CI run, not real-time state. This is fine for README display but should be clearly timestamped in the badge tooltip or link.

**Score gaming:** A visible score badge might incentivize gaming (enabling tools without configuring them properly just to raise the score). The conformance track (PASS/FAIL) is harder to game than the numeric score.

**Badge proliferation:** More than 3 badges in a README is visual noise. Recommend one primary badge (score) with conformance as optional second.

## Depth Checklist

- [x] Underlying mechanism explained: Three generation methods, shields.io endpoint protocol, color mapping
- [x] Key tradeoffs and limitations identified: External dependency, freshness, score gaming, proliferation
- [x] Compared to at least one alternative: Snyk badges, Scorecard badges, self-hosted shields.io
- [x] Failure modes and edge cases: Private repos, air-gapped environments, stale badges
- [x] Concrete examples or reference implementations: CI pipeline YAML, JSON schema, README markdown, SVG sketch
- [x] Report is standalone-readable: Complete badge implementation guide
