<!-- Source: https://sbomgenerator.com/guides/go -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: This is an AI-summarized version of the page content. May need re-fetch for full detail. -->

# Go SBOM Generation: Comprehensive Content Summary

## Core Concepts

### Go's Dependency Model
Go's approach differs significantly from other languages. The ecosystem includes several dependency categories:

- **Direct dependencies**: Modules explicitly listed in go.mod, representing packages your code directly imports
- **Transitive dependencies**: Dependencies of dependencies, automatically resolved
- **Standard library**: Go's extensive standard library, which is versioned with the Go compiler itself
- **CGO dependencies**: C libraries that bypass Go's module system entirely
- **Vendor dependencies**: Local copies created via `go mod vendor`
- **Build-time dependencies**: Tools used during compilation

### go.mod and go.sum Files

The go.mod file serves as a semantic versioning-aware manifest. The go.sum file provides cryptographic checksums ensuring reproducible builds. Go employs Minimal Version Selection (MVS) algorithm to build consistent dependency graphs.

## Primary SBOM Generation Tools

### Syft (The Go Standard)

Syft has emerged as the de facto standard for Go SBOM generation. It can analyze:
- Source code and compiled binaries
- Container images
- Stripped Go binaries for dependency extraction

Key commands include:
```
syft dir:. -o cyclonedx-json=sbom.json
syft ./myapp -o cyclonedx-json=binary-sbom.json
syft dir:. -o spdx-json=sbom.spdx.json
```

### CycloneDX CLI for Go

The cyclonedx-gomod tool provides tighter integration with the CycloneDX ecosystem and enables "runtime SBOM generation" embedded directly in Go applications. It generates both JSON and XML output formats.

### SPDX Tools

The document recommends "generate SPDX directly with Syft instead of relying on ecosystem-specific conversion helpers."

## Binary vs. Source Analysis

**Critical Difference**: "The go.mod file lists all potential dependencies, but the actual binary only includes code that's actually used." Go's compiler performs dead code elimination, making binary analysis more accurate for final deployments.

SBOMs generated from binaries show fewer components than source analysis because:
- Build tags and platform-specific code affect inclusion
- Unused functions and entire packages are removed
- Dead code elimination removes unused functions

## Vendored Dependencies

For vendored code, use `syft dir:. --exclude 'vendor/**'` to exclude vendor directories from source scanning, or `syft dir:vendor` for vendor-specific analysis.

## CGO Handling

CGO dependencies require multi-layer SBOM generation since they bypass the module system. The document recommends scanning system libraries separately using tools like `ldd`.

## CI/CD Integration

### GitHub Actions
Comprehensive workflow covering:
- Multi-version Go testing (1.20, 1.21)
- Cross-platform builds (Linux, Windows, Darwin)
- Multi-architecture support (amd64, arm64)
- Vulnerability scanning with Grype
- SBOM validation and artifact uploading

### GitLab CI
Pipeline includes:
- Dependency verification
- SBOM generation from source and binaries
- Security scanning with Grype and Nancy
- GoVulnCheck integration

## SBOM Embedding

The document shows using Go's `//go:embed` directive to embed SBOMs in binaries, enabling runtime SBOM exposure via HTTP endpoints.

## Key Recommendations

1. **Analyze binaries over source** for production deployments due to actual component inclusion
2. **Generate multiple format SBOMs** (CycloneDX and SPDX) for broader compatibility
3. **Integrate scanning early** in CI/CD pipelines with Grype or equivalent tools
4. **Document build conditions** affecting component inclusion
5. **Include Go version** in SBOM metadata for standard library vulnerability tracking
