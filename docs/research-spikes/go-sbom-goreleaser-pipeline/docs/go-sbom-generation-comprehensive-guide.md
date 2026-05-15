# Go SBOM Generation: Comprehensive Guide

- **Source**: https://sbomgenerator.com/guides/go
- **Retrieved**: 2026-05-15

## Core Concepts

Go's static linking creates unique SBOM challenges. "The source dependency graph can be very clear while the final deployed binary is highly optimized and statically linked," requiring teams to distinguish between source dependencies, build-time dependencies, and actual binary contents.

## Dependency Management Foundation

### go.mod and go.sum Files

The `go.mod` file serves as a semantic versioning-aware manifest declaring module requirements. The accompanying `go.sum` file provides cryptographic checksums ensuring reproducible builds. These files enable Go's Minimal Version Selection (MVS) algorithm to build consistent dependency graphs.

Key dependency categories:
- **Direct dependencies**: Explicitly listed in go.mod with version pinning
- **Transitive dependencies**: Automatically resolved by Go modules
- **Standard library**: Versioned with the Go compiler
- **CGO dependencies**: C libraries that bypass Go's module system
- **Vendored modules**: Local copies created via `go mod vendor`
- **Build-time dependencies**: Tools used during compilation

## SBOM Generation Tools

### Syft (Recommended Standard)

Syft has emerged as the de facto standard for Go SBOM generation due to its deep understanding of Go's packaging formats. It can analyze source modules, compiled binaries, and container images, even extracting dependency information from stripped Go binaries.

```bash
syft dir:. -o cyclonedx-json=sbom.json
syft ./myapp -o cyclonedx-json=binary-sbom.json
syft dir:. -o spdx-json=sbom.spdx.json
```

### CycloneDX CLI for Go

The `cyclonedx-gomod` tool offers tighter integration with the CycloneDX ecosystem:

```bash
cyclonedx-gomod mod -json -output sbom.json
cyclonedx-gomod mod -json -include-test -output sbom-with-tests.json
```

## Embedding SBOMs in Binaries

Using Go's `//go:embed` directive to compile SBOM data directly into the binary:

```go
//go:embed sbom.json
var sbomData []byte
```

This enables runtime SBOM endpoints without external file dependencies.

### Dynamic SBOM Generation

A `SBOMGenerator` struct demonstrates:
- Runtime SBOM generation with caching to minimize performance impact
- Context-aware timeout handling
- Runtime information gathering including Go version, OS, and architecture
- HTTP endpoints for on-demand SBOM access

## CI/CD Integration

### GitHub Actions Workflow

The workflow includes multi-platform builds (Linux, Windows, Darwin across amd64 and arm64), dependency verification, SBOM generation from both source and binaries, vulnerability scanning with Grype, and artifact archival.

## Docker Integration

### Multi-stage Build Strategy

- Separate builder stage for Go compilation
- Dedicated SBOM generation stage using Syft
- Minimal scratch runtime stage with embedded SBOMs
- Development stage with complete tooling

## Best Practices

1. **Module Management**: Regular updates via `go get -u`, verification with `go mod verify`, and vulnerability checking with govulncheck
2. **Build Reproducibility**: Using specific Go versions, pinned tool versions, and consistent build flags including `-trimpath` and `-buildid=`
3. **Quality Assurance**: PURL coverage, hash inclusion, license metadata completeness

## FAQ Highlights

- **Discrepancies between go.mod and binary SBOMs** occur because Go's compiler performs dead code elimination
- **Vendored dependencies** require using `syft dir:vendor` for accurate representation
- **CGO dependencies** necessitate multi-layer analysis combining Go dependency scans with system library detection
- **Private modules** require proper authentication configuration via `.netrc` or git credentials alongside GOPRIVATE
