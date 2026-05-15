<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/cli/cli.js -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned a summary rather than verbatim content. The file was too large for full extraction. -->

# @lateos/npm-scan — cli/cli.js (Summary)

Node.js command-line application built with Commander.js providing npm supply chain security scanning.

## Core Commands

### scan
Analyzes individual packages for security vulnerabilities:
- Local tarball scanning via `--file`
- Multiple output formats (SARIF, CSV, SBOM)
- Policy-based filtering
- Risk scoring (0-10 scale)
- Audit logging
- FIPS 140-2/3 crypto mode
- Offline caching with configurable TTL and size limits

### scan-lockfile
Examines dependency lockfiles (npm, yarn, pnpm):
- Watch mode with debounce capability
- Monorepo support for multiple lockfiles
- Severity-based failure thresholds
- Real-time monitoring with Ctrl+C exit handling

### report
Generates compliance and analysis reports:
- NIST 800-161, EU CRA, STIG compliance formats
- SIEM integration (CEF, ECS, Sentinel, QRadar)
- PDF export (premium feature)
- HTML and text output options

### serve
Launches an HTTP API server (premium):
- Health check endpoint
- `/scan` POST endpoint for remote scanning
- SIEM and PDF endpoints with feature gating

## Key Implementation Details
- License enforcement via `requirePremium()` function checking `NPM_SCAN_LICENSE_KEY` environment variable
- Output supports JSON, structured reporting, and piping to external systems
- Error handling uses exit codes (0 for success, 1 for policy violations or findings above severity thresholds)
