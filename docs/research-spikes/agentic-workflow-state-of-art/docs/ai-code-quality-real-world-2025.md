# AI-Generated Code Quality: Real-World Metrics and Production Data (2025-2026)
- **Sources**:
  - https://www.coderabbit.ai/blog/state-of-ai-vs-human-code-generation-report
  - https://jellyfish.co/blog/2025-ai-metrics-in-review/
  - https://www.veracode.com/blog/genai-code-security-report/
  - https://www.secondtalent.com/resources/ai-generated-code-quality-metrics-and-statistics-for-2026/
  - https://www.greptile.com/benchmarks
- **Retrieved**: 2026-03-15
- **Note**: Composite summary from multiple industry reports

## Adoption Rates (2025)
- Coding review agent adoption grew from 14.8% (Jan) to 51.4% (Oct)
- Nearly half of companies have 50%+ AI-generated code (up from 20% at start of year)
- 24% of production code is now AI-written (29% in US, 21% in Europe)

## Code Acceptance Metrics
- GitHub Copilot: 46% code completion rate, ~30% accepted by developers
- Enterprise (Zoominfo): 33% acceptance for suggestions, 20% for lines
- Retained rate: 88% of accepted code stays in final submissions
- Copilot Chat code review: 70% of comments accepted

## Productivity Gains
- PRs per engineer: +113% increase (1.36 to 2.9)
- Cycle time: 24% reduction (16.7 to 12.7 hours)
- PR time: 75% reduction (9.6 to 2.4 days) in some studies
- Tasks completed 55% faster with Copilot (4,800 developer study)

## Code Quality Issues
- AI-generated PRs: 10.83 issues each vs 6.45 for human PRs (1.7x more)
- AI PRs contain 1.4x more critical issues and 1.7x more major issues
- Maintainability errors: 1.64x higher
- Logic/correctness errors: 1.75x more frequent
- Security findings: 1.57x more

## Security Vulnerability Data
- 45% of AI code samples failed security tests (OWASP Top 10) — Veracode, 100+ models tested
- 62% of AI-generated solutions contain design flaws or known vulnerabilities
- AI code blamed for 1 in 5 breaches (Aikido Security 2026)
- 7 in 10 respondents found AI-introduced vulnerabilities; 1 in 5 had serious incidents

### Specific Vulnerability Types
- 86% of samples failed to defend against XSS (CWE-80)
- 88% vulnerable to log injection (CWE-117)
- 1.88x more improper password handling
- 1.91x more insecure object references
- 2.74x more XSS vulnerabilities
- 1.82x more insecure deserialization

## AI Code Review Tool Benchmarks (Greptile 2025)
- Greptile: 82% bug catch rate
- Cursor: 58% bug catch rate
- Traditional static analyzers: <20% catch rate
- AI review tools detect 42-48% of real-world runtime bugs

## Developer Trust
- 46% of developers actively distrust AI output accuracy
- 33% trust it
- Only 3% report "highly trusting"

## Production Quality Concerns
- As teams push more AI code to production, subtle defects surface later in release cycle
- Even "passing" AI code: average 1.45 static analysis issues per successful task
- Quality gap persists even with frontier models
