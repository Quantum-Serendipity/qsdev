<!-- Source: https://trivy.dev/docs/latest/guide/coverage/language/golang/ -->
<!-- Retrieved: 2026-05-15 -->

# Trivy's Go Language Support

## Scannable Artifacts

Trivy handles two primary Go scanning types:

1. **Go Modules** — Requires `go.mod` (and `go.sum` for Go <1.17). Supports SBOM generation, vulnerability scanning, and license detection.

2. **Go Binaries** — Scans compiled binaries containing embedded Go version and dependency information. Note: "It doesn't work with UPX-compressed binaries."

## Vulnerability Data Sources

Trivy leverages the Go Vulnerability Database for standard library detection and GitHub Advisory Database for other Go modules.

## Standard Library Handling

Detection occurs only in `--detection-priority comprehensive` mode for Go 1.21+. The tool identifies the minimum version between `go` and `toolchain` directives in `go.mod`. However, "Trivy does not know if or how you use stdlib functions, therefore it is possible that stdlib vulnerabilities are not applicable to your use case."

## Known Limitations

**Main Module Gaps**: Trivy scans only project dependencies, not the application itself. Binaries built without `go install` show `(devel)` versions; Trivy attempts extraction via `-ldflags` or ELF symbol tables, potentially resulting in empty versions.

**False Positives**: Stdlib scanning may report inapplicable vulnerabilities. Mitigation strategies include using `govulncheck` for reachability analysis or applying VEX suppression files.

**Dev Dependencies**: Go modules include dev dependencies in scanning; binaries exclude them.

## SBOM and Dependency Graph

Both features require pre-downloaded modules (via `go mod download`, `go mod tidy`, or `vendor` directory). Trivy prioritizes the `vendor` directory, then `$GOPATH/pkg/mod`.
