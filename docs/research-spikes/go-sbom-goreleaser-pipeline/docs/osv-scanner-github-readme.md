<!-- Source: https://github.com/google/osv-scanner -->
<!-- Retrieved: 2026-05-15 -->

# OSV-Scanner: Go & SBOM Capabilities

## SBOM Input Support

OSV-Scanner supports scanning SBOMs:
```bash
osv-scanner scan -L sbom.cdx.json
```

SPDX and CycloneDX SBOMs using Package URLs are supported.

## Go Module Scanning

OSV-Scanner supports Go modules as part of its "11+ language ecosystems and 19+ lockfile types." The tool can recursively scan directories for supported package files including `go.mod` files.

## Govulncheck Integration

OSV-Scanner uses the govulncheck library to analyze Go source code to identify called vulnerable functions. This provides reachability analysis similar to govulncheck itself.

## Vulnerability Database

Primary data source is OSV.dev. The tool provides "an officially supported frontend to the OSV database" and queries this API for vulnerability information. Each advisory originates from "an open and authoritative source (e.g. GitHub Security Advisories, RustSec Advisory Database, Ubuntu security notices)."

## CLI Commands

- `osv-scanner scan source` — directory scanning
- `osv-scanner scan image` — container image analysis
- `osv-scanner scan -L sbom.cdx.json` — SBOM scanning
- `osv-scanner --offline` — offline database scanning
- `osv-scanner --licenses` — license checking

## Guided Remediation Features

Experimental feature suggests package upgrades based on dependency depth, severity thresholds, fix strategy, and ROI. Currently supports npm (lockfile/manifest) and Maven (manifest) formats.

## Go-Specific Capabilities

Beyond `go.mod` and `go.sum` support, OSV-Scanner integrates govulncheck for reachability analysis of Go code.

## V2 (March 2025)

Added container scanning, guided remediation for Maven, and interactive HTML output. Latest release v2.3.5 (March 2026) enables transitive scanning for Python requirements.txt.
