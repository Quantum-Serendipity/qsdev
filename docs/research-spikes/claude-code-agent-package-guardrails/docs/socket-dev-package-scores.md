<!-- Source: https://docs.socket.dev/docs/package-scores -->
<!-- Retrieved: 2026-05-12 -->

# Socket.dev Package Scoring Documentation

## Score Dimensions

Socket evaluates packages across five primary categories:

1. **Supply Chain Risk** -- Detects malware, typosquatting, obfuscated code, unstable ownership, and dependency concerns
2. **Quality** -- Measures code size, popularity metrics, and documentation
3. **Maintenance** -- Tracks commit frequency, version releases, issue management, and contributor counts
4. **Vulnerabilities** -- CVE severity levels (Critical, High, Medium, Low)
5. **License** -- Identifies licensing issues and compliance problems

## Scoring Algorithm

The final score formula is:

**Si = 100 * min(max(0, min_j l_i,j), Sum_j w_j N_j(x_j) / Sum_j w_j)^gamma**

Where:
- **gamma** is a scaling exponent based on project size and popularity: gamma ~ 1/2 + c0 log(lines of code) + c1 log(popularity)
- Larger, more popular packages receive lower gamma values, softening penalty impacts
- Each metric has a weight (w_j) and normalization function (N_j)

## Alert Impact Tiers

| Alert Level | Normalization | Soft Cap | Effect |
|---|---|---|---|
| **Critical** | e^(-10x) | 0.25 (if present) | Most severe; limits score to ~33% |
| **High** | e^(-x) | max(0.25, 1 - x/10) | Significant decay; bottoms at 0.25+ alerts |
| **Medium** | e^(-x/20) | max(0.5, 1.15 - x/20) | Moderate impact; levels at ~0.5 |
| **Low** | e^(-x/40) | None | Minimal, gradual effect |

## Key Metrics (Sample)

**Supply Chain Risk metrics** include download count, dependency totals, and transitive dependency analysis.

**Maintenance signals** evaluate commit patterns across weekly/monthly/yearly intervals, version frequency, and issue management.

**Quality indicators** measure README documentation, bundle size, GitHub engagement (stars, forks, watchers), and lines of code.

## Detection Capabilities

Socket scans for:
- Typosquatting attacks
- Install scripts
- Dynamic code evaluation
- Shell/network access
- Obfuscated or minified code
- Environment variable exploitation
- Filesystem access patterns

**Note:** Socket acknowledges metrics are subject to continuous refinement and may not reflect exact current deployment.
