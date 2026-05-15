# Go SBOM Generation Tools: Comprehensive Landscape Analysis

## Executive Summary

The Go SBOM generation landscape in mid-2026 is dominated by **Syft** (Anchore) as the de facto general-purpose tool and **cyclonedx-gomod** (CycloneDX) as the Go-specific precision tool. Trivy's viability was severely damaged by a supply chain compromise in March 2026. The remaining tools (bom, spdx-sbom-generator, cdxgen) serve niche roles or are deprecated. All tools ultimately build on the same foundation: Go's built-in `debug/buildinfo` metadata embedded in compiled binaries, accessible via `go version -m`.

For the qsdev binary specifically, the recommended approach is **syft for binary SBOM generation** (broadest format support, GoReleaser integration, active maintenance) with **cyclonedx-gomod as a build-time complement** for maximum dependency accuracy.

---

## Foundation: `go version -m` and `debug/buildinfo`

Every Go SBOM tool for binary analysis ultimately reads the same data: the build information embedded by the Go compiler in every module-aware binary. Understanding this foundation is essential for evaluating tool accuracy.

### What Go Embeds in Binaries

Since Go 1.13+ (with improvements through 1.18+), the Go compiler embeds structured build metadata accessible via:
- **CLI**: `go version -m <binary>`
- **Programmatic**: `debug/buildinfo.ReadFile()` or `runtime/debug.ReadBuildInfo()`

The embedded data includes:

| Field | Example | Notes |
|-------|---------|-------|
| Go version | `go1.21.5` | Compiler version, critical for stdlib vuln tracking |
| Module path | `github.com/org/repo` | Main module only — **no version for root module** |
| Dependencies | `dep github.com/pkg/foo v1.2.3 h1:abc=` | Path, version, and h1 hash for each dep |
| Build settings | `CGO_ENABLED=0`, `GOARCH=arm64`, `GOOS=linux` | Build environment |
| VCS info | `vcs.revision`, `vcs.time`, `vcs.modified` | Git commit, timestamp, dirty flag |
| Build flags | `-buildmode=exe`, `-compiler=gc` | Compilation configuration |

### Key Limitations of the Built-in Data

1. **Root module version is missing**: The compiled binary carries dependency versions and hashes but only the package name (not version) for the root module itself. SBOM tools must infer the root version from VCS info or ldflags.
2. **No license information**: Build info contains zero license data — tools must resolve licenses separately.
3. **No transitive dependency graph**: Dependencies are a flat list with no parent-child relationships.
4. **CGO dependencies invisible**: C libraries linked via CGO bypass Go's module system entirely and are not captured.
5. **Build tags affect inclusion**: Dead code elimination means the binary's actual dependency set differs from go.mod's declared set (binary is a strict subset).
6. **Pre-Go 1.18 binaries**: Have minimal build info (no settings, limited module info).

### Source vs. Binary SBOM: A Critical Distinction

| Aspect | Source SBOM (go.mod/go.sum) | Binary SBOM (go version -m) |
|--------|----------------------------|----------------------------|
| **Dependency scope** | All potential deps (including test, unused) | Only actually compiled deps |
| **Accuracy for deployment** | Over-reports (includes dead code paths) | Precise (reflects actual binary content) |
| **Transitive deps** | Full graph via `go mod graph` | Flat list only |
| **Checksums** | go.sum has h1 hashes for all | h1 hashes for compiled deps only |
| **Build constraints** | Not evaluated | Fully evaluated (GOOS, GOARCH, tags) |
| **CGO deps** | Not captured | Not captured |
| **Availability** | Requires source checkout | Works on any Go binary |

**Recommendation**: For shipping SBOMs alongside release binaries, binary analysis is preferred because it reflects the actual artifact contents. Source analysis is useful for development-time auditing.

---

## Tool-by-Tool Analysis

### 1. Syft (Anchore)

| Attribute | Value |
|-----------|-------|
| **GitHub** | [anchore/syft](https://github.com/anchore/syft) |
| **Stars** | ~8,900 |
| **Latest release** | v1.44.0 (May 1, 2026) |
| **Language** | Go (98.9%) |
| **License** | Apache 2.0 |
| **Output formats** | CycloneDX (JSON, XML), SPDX (JSON, tag-value), Syft JSON |
| **Maintenance** | Very active — monthly releases, 219+ contributors |

#### Architecture

Syft uses a pluggable cataloger architecture. Rather than a monolithic scanner, it delegates to ecosystem-specific catalogers. For Go, there are two:

1. **Go Module Binary Cataloger** (`go-module-binary-cataloger`): Extracts `debug/buildinfo` from compiled Go executables. Uses a three-tiered version detection strategy:
   - ldflags parsing (highest priority) — recognizes `-X main.version=1.0.0` patterns
   - Binary content regex scanning (medium priority)
   - VCS pseudo-versions from build settings (lowest priority)

2. **Go Module File Cataloger** (`go-module-file-cataloger`): Parses go.mod files directly. Can optionally use `golang.org/x/tools/go/packages` for deep source analysis (requires Go toolchain at scan time).

#### Go-Specific Strengths

- **Binary analysis is a first-class capability**: Can scan stripped Go binaries and still extract dependency information from the embedded buildinfo section.
- **License resolution**: Four-tier system — embedded in scan target > local mod cache (`$GOPATH/pkg/mod`) > vendor directory > remote proxies. Respects GOPROXY/GONOPROXY configuration.
- **PURL generation**: Produces correct `pkg:golang/` Package URLs with namespace and subpath extraction.
- **Build metadata capture**: Extracts GOARCH, GOOS, CGO_ENABLED, and other build settings into the SBOM.
- **Container image scanning**: Can analyze Go binaries inside Docker images, extracting deps from the binary even when source isn't present.

#### Go-Specific Weaknesses

- **Source dependency resolution can lag language-specific tools**: Anchore's own blog acknowledges Go (and Rust) support needs further development. The OpenSSF notes Syft "sometimes miss[es] dependencies found by other tools."
- **No build constraint evaluation for source scanning**: When scanning go.mod (not binaries), Syft doesn't evaluate GOOS/GOARCH constraints, potentially over-reporting.
- **Requires Go toolchain for deep source analysis**: Without it, falls back to file parsing only.

#### CLI Usage for Go

```bash
# Binary SBOM (most accurate for releases)
syft ./myapp -o cyclonedx-json=sbom.cdx.json

# Source directory SBOM
syft dir:. -o cyclonedx-json=sbom.cdx.json

# Container image SBOM
syft ghcr.io/org/app:latest -o spdx-json=sbom.spdx.json

# Multiple output formats simultaneously
syft ./myapp -o cyclonedx-json=sbom.cdx.json -o spdx-json=sbom.spdx.json
```

#### CI/CD Integration

- **GitHub Action**: `anchore/sbom-action` — official, well-maintained
- **GoReleaser**: Native `sboms:` configuration block, syft is the default generator
- **Grype pairing**: `anchore/grype` consumes Syft SBOMs for vulnerability scanning

---

### 2. cyclonedx-gomod (CycloneDX Official)

| Attribute | Value |
|-----------|-------|
| **GitHub** | [CycloneDX/cyclonedx-gomod](https://github.com/CycloneDX/cyclonedx-gomod) |
| **Stars** | ~183 |
| **Latest release** | v1.10.0 (January 31, 2026) |
| **Language** | Go (99.3%) |
| **License** | Apache 2.0 |
| **Output formats** | CycloneDX only (JSON and XML, spec versions 1.0-1.6) |
| **Maintenance** | Active — regular releases, smaller but focused team |

#### Architecture

cyclonedx-gomod integrates directly with Go's toolchain rather than reimplementing module parsing. It offers three distinct subcommands for different use cases:

**`app` subcommand** — The most precise mode. Generates SBOMs that include only modules the target application actually depends on. Build constraints are evaluated via environment variables (GOARCH, GOOS, CGO_ENABLED, GOFLAGS), enabling architecture and platform-specific SBOMs. This is the closest you can get to a "what's actually in my binary" SBOM from source.

**`mod` subcommand** — Captures the aggregate dependency graph of a Go module, optionally including test dependencies. Does NOT evaluate build constraints, giving a "whole picture" view. Best for library distribution and inventory tracking.

**`bin` subcommand** — Analyzes compiled binaries to extract embedded module information (using the same `debug/buildinfo` foundation). Enables SBOM generation without source code access, but produces "rudimentary" SBOMs compared to the `app` subcommand.

#### Go-Specific Strengths

- **Build constraint awareness** (app mode): The only tool that evaluates GOOS/GOARCH/CGO_ENABLED during source-based SBOM generation, producing the most accurate representation of what will actually be compiled.
- **License detection**: Optional automated license identification, reported as "evidence" rather than assertions (more legally precise).
- **Package-level granularity**: Can include individual packages and their constituent files within modules.
- **CycloneDX spec compliance**: Native CycloneDX output with support for latest spec versions (1.0-1.6).
- **Go-native**: Written in Go, uses Go toolchain directly, no external dependencies for core functionality.

#### Go-Specific Weaknesses

- **CycloneDX only**: No SPDX output. If you need SPDX, you must use a separate conversion tool.
- **Vendoring limitations**: Projects using `go mod vendor` cannot generate accurate component hashes because Go doesn't copy all module files to the vendor directory. License detection may also fail.
- **VCS requirement**: Only Git repositories get automated version detection; other VCS systems require manual version specification.
- **Pseudo-version sensitivity**: Shallow clones (common in CI) may prevent accurate pseudo-version generation.
- **Small community**: 183 stars means less community testing and slower issue resolution than Syft.

#### CLI Usage for Go

```bash
# Application SBOM (most accurate for specific binary)
cyclonedx-gomod app -json -output sbom.json ./cmd/myapp

# With build constraints
GOOS=linux GOARCH=amd64 cyclonedx-gomod app -json -output sbom.json ./cmd/myapp

# Module-level SBOM (includes test deps)
cyclonedx-gomod mod -json -output sbom.json -test .

# Binary analysis
cyclonedx-gomod bin -json -output sbom.json ./myapp

# With license detection
cyclonedx-gomod app -json -licenses -output sbom.json ./cmd/myapp
```

#### CI/CD Integration

- **GitHub Action**: `CycloneDX/gh-gomod-generate-sbom` — official action
- **Homebrew**: `brew install cyclonedx/cyclonedx/cyclonedx-gomod`
- **Go install**: `go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest`

---

### 3. Trivy (Aqua Security)

| Attribute | Value |
|-----------|-------|
| **GitHub** | [aquasecurity/trivy](https://github.com/aquasecurity/trivy) |
| **Stars** | ~35,000 |
| **Latest release** | v0.70.0 (April 17, 2026) |
| **Language** | Go |
| **License** | Apache 2.0 |
| **Output formats** | CycloneDX (JSON only), SPDX (JSON, tag-value), Trivy JSON |
| **Maintenance** | Active releases, but **trust severely damaged** |

#### CRITICAL: March 2026 Supply Chain Compromise

On March 19, 2026, Trivy suffered a devastating supply chain attack by the "TeamPCP" threat actor. The attack simultaneously compromised:
- The core Trivy binary (malicious v0.69.4 published)
- The `trivy-action` GitHub Action (76 of 77 version tags force-pushed to malicious commits)
- The `setup-trivy` GitHub Action (all 7 tags hijacked)

The payload ran silently before legitimate scans, stealing CI/CD secrets. A second wave on March 22 pushed additional malicious Docker Hub images. The root cause was a misconfigured GitHub Actions environment that leaked a privileged access token, compounded by incomplete credential rotation.

**Multiple security vendors (Microsoft, Palo Alto, CrowdStrike, Wiz) published incident response guidance.** This is now widely cited as a reason to avoid Trivy in automated CI/CD pipelines.

#### SBOM Capabilities (Technical Merits)

Setting aside the trust issue, Trivy's SBOM generation works as follows:
- SBOM generation is triggered by `--format cyclonedx` or `--format spdx-json`, which **disables vulnerability scanning by default**
- Supports scanning containers, filesystems, rootfs, VM images, and Kubernetes clusters
- Can auto-detect embedded SBOM files (`.spdx`, `.cdx`) in container images
- CycloneDX JSON only (no XML); SPDX in both JSON and tag-value

#### Go-Specific Assessment

- Go module analysis is supported but with "moderate" binary analysis depth
- Multiple comparison sources rank Trivy's Go binary analysis below Syft's
- The tool is primarily a vulnerability scanner with SBOM as a secondary capability

#### Recommendation

**Do not use Trivy for SBOM generation in CI/CD pipelines.** Even if technical capabilities recover, the trust damage from the March 2026 compromise is disqualifying for a supply chain security tool. Use Syft for the same use cases with better Go support and no trust baggage.

---

### 4. cdxgen (CycloneDX/OWASP)

| Attribute | Value |
|-----------|-------|
| **GitHub** | [cdxgen/cdxgen](https://github.com/cdxgen/cdxgen) |
| **Stars** | ~917 |
| **Latest release** | v12.3.0 (2026) |
| **Language** | JavaScript (Node.js required) |
| **License** | Apache 2.0 |
| **Output formats** | CycloneDX only (JSON) |
| **Maintenance** | Active — part of GitHub Secure Open Source Fund |

#### Overview

cdxgen is the official CycloneDX SBOM generation tool for multi-language projects. For Go, it uses `go mod why` to identify required packages and generates CycloneDX output.

#### Go-Specific Assessment

- Supports Go module analysis with transitive dependency resolution
- Deep dependency resolution with evidence-based SBOMs
- Call graph analysis capabilities for some ecosystems
- **Requires Node.js runtime** — adds a dependency for Go-only projects
- CycloneDX output only (no SPDX)

#### When to Consider

cdxgen makes sense for polyglot projects where you need consistent CycloneDX output across multiple language ecosystems in a single tool. For a pure Go project, cyclonedx-gomod is more appropriate (no Node.js dependency, deeper Go integration).

---

### 5. bom (Kubernetes SIG Release)

| Attribute | Value |
|-----------|-------|
| **GitHub** | [kubernetes-sigs/bom](https://github.com/kubernetes-sigs/bom) |
| **Stars** | ~455 |
| **Latest release** | v0.7.1 (September 26, 2025) |
| **Language** | Go (99.7%) |
| **License** | Apache 2.0 |
| **Output formats** | SPDX only (tag-value, JSON), in-toto provenance |
| **Maintenance** | Low-moderate — built for Kubernetes release process |

#### Overview

bom was created specifically to generate SBOMs for the Kubernetes project. It produces SPDX-compliant manifests from files, images, directories, and archives.

#### Go-Specific Assessment

- Go dependency analysis via go.mod
- Filtering of transient dependencies
- 400+ SPDX license recognition
- Container image analysis with deep inspection

#### When to Consider

bom is purpose-built for the Kubernetes release workflow. It's functional for SPDX generation from Go projects but lacks the breadth of Syft or the Go-specific depth of cyclonedx-gomod. Its small community (455 stars) and Kubernetes-specific focus make it a niche choice. **Not recommended as a primary SBOM tool for general Go projects.**

---

### 6. spdx-sbom-generator (OpenSSF) — DEPRECATED

| Attribute | Value |
|-----------|-------|
| **GitHub** | [opensbom-generator/spdx-sbom-generator](https://github.com/opensbom-generator/spdx-sbom-generator) |
| **Stars** | ~425 |
| **Status** | **Archived January 13, 2025** |
| **Last release** | v0.0.15 (July 12, 2022) |
| **Output formats** | SPDX v2.2 only |

#### Status

Officially deprecated due to maintainer unavailability. The repository recommends Syft, Trivy, or Parlay as alternatives. **Do not use for new projects.**

---

### 7. govulncheck (Go Team) — Complementary Tool

| Attribute | Value |
|-----------|-------|
| **Repository** | [golang/vuln](https://github.com/golang/vuln) |
| **Maintainer** | Go Security Team (Google) |
| **Latest releases** | Regular updates throughout 2025-2026 |
| **Language** | Go |
| **Output formats** | Text, JSON (streaming), SARIF, OpenVEX |

#### Why It's Not an SBOM Tool (But Matters)

govulncheck is a vulnerability scanner, not an SBOM generator. However, it's deeply relevant to the SBOM story because:

1. **Reachability analysis**: Unlike SBOM-based vulnerability scanners (grype, etc.) that flag all deps with known CVEs, govulncheck performs call graph analysis to determine if vulnerable functions are actually reachable. This dramatically reduces false positives.

2. **Binary analysis mode**: `govulncheck -mode binary ./myapp` scans compiled binaries using symbol tables, complementing binary SBOM generation.

3. **VEX output**: `govulncheck -format openvex` produces Vulnerability Exploitability eXchange documents — the standard complement to SBOMs for communicating "this CVE doesn't affect us because the vulnerable code path is unreachable."

4. **SARIF output**: `govulncheck -format sarif` produces results compatible with GitHub Code Scanning.

#### How It Fits the Pipeline

The recommended pattern is:
1. **SBOM generation** (syft/cyclonedx-gomod) — documents what's in the binary
2. **Vulnerability scanning** (grype consuming the SBOM, or govulncheck directly) — identifies known vulns
3. **VEX generation** (govulncheck -format openvex) — communicates exploitability context
4. **Attestation** (cosign/sigstore) — signs all of the above

---

## Comparative Analysis

### Accuracy Ranking for Go

| Rank | Tool | Why |
|------|------|-----|
| 1 | **cyclonedx-gomod (app mode)** | Evaluates build constraints, uses Go toolchain directly, includes only actually-compiled deps |
| 2 | **Syft (binary scanning)** | Strong binary analysis with ldflags version detection, license resolution |
| 3 | **cyclonedx-gomod (bin mode)** | Same binary data as Syft but less version inference logic |
| 4 | **Syft (source scanning)** | Parses go.mod but doesn't evaluate build constraints — over-reports |
| 5 | **cdxgen** | Good transitive resolution but Go is not its primary ecosystem |
| 6 | **Trivy** | Moderate Go binary depth, trails Syft in edge cases |
| 7 | **bom** | Basic go.mod parsing, adequate but not specialized |

### Feature Matrix

| Feature | Syft | cyclonedx-gomod | Trivy | cdxgen | bom |
|---------|------|-----------------|-------|--------|-----|
| **Binary analysis** | Strong | Rudimentary (bin) | Moderate | No | No |
| **Source analysis** | Yes (go.mod) | Yes (app, mod) | Yes | Yes | Yes (go.mod) |
| **Build constraint eval** | No (source) | Yes (app mode) | No | Partial | No |
| **CycloneDX output** | JSON, XML | JSON, XML | JSON only | JSON | No |
| **SPDX output** | JSON, tag-value | No | JSON, tag-value | No | JSON, tag-value |
| **License detection** | 4-tier resolution | Evidence-based | Basic | Yes | 400+ SPDX IDs |
| **Vendored deps** | Yes (with caveats) | Partial (no hashes) | Yes | Yes | Unknown |
| **Build toolchain version** | Yes | Yes | Yes | Yes | Unknown |
| **Container image scan** | Yes | No | Yes | Yes | Yes |
| **GoReleaser integration** | Native (default) | Manual config | Manual config | No | No |
| **GitHub Action** | anchore/sbom-action | gh-gomod-generate-sbom | trivy-action (compromised) | cdxgen-action | None official |
| **Node.js required** | No | No | No | Yes | No |
| **Trust status (2026)** | High | High | Damaged | High | Moderate |

### Maintenance & Community Health

| Tool | Stars | Last Release | Release Cadence | Contributors |
|------|-------|-------------|-----------------|-------------|
| **Trivy** | ~35,000 | Apr 2026 | Monthly | Large team |
| **Syft** | ~8,900 | May 2026 | Monthly | 219+ |
| **cdxgen** | ~917 | 2026 | Frequent | ~90 |
| **bom** | ~455 | Sep 2025 | Quarterly | Small team |
| **spdx-sbom-generator** | ~425 | Jul 2022 | **ARCHIVED** | Dead |
| **cyclonedx-gomod** | ~183 | Jan 2026 | Semi-annual | Small team |

Note: Trivy's high star count reflects its popularity as a vulnerability scanner, not SBOM quality. Its trust damage from the March 2026 compromise significantly diminishes its effective value despite the numbers.

---

## Handling Special Cases

### Vendored Dependencies

- **Syft**: Can scan vendor directories (`syft dir:vendor`), but recommend excluding with `--exclude 'vendor/**'` for source scans to avoid double-counting.
- **cyclonedx-gomod**: Partial support — cannot generate accurate component hashes from vendor directories because Go doesn't copy all module files. License detection may also fail.
- **Best practice**: Generate SBOM from binary (post-compilation) rather than source when using vendoring. The binary contains the correct resolved dependencies regardless of vendoring strategy.

### CGO Dependencies

No Go SBOM tool captures CGO dependencies (C libraries linked via CGO). These bypass Go's module system entirely. For projects with CGO:
- Scan system libraries separately using `ldd` on the binary
- Consider multi-layer SBOM generation (Go deps + system deps)
- Document CGO dependencies manually if few in number

### Build Toolchain Version

All tools that perform binary analysis capture the Go compiler version from `debug/buildinfo`. This is critical for tracking standard library vulnerabilities (which are versioned with the compiler).

### Private Module Registries

- **Syft**: Respects GOPROXY/GONOPROXY for license resolution from remote proxies
- **cyclonedx-gomod**: Requires standard Go authentication (.netrc, git credentials, GOPRIVATE)
- Both tools need appropriate credentials configured in CI for private module support

---

## Recommendations for qsdev

### Primary Tool: Syft

**Why**: Broadest format support (CycloneDX + SPDX), strongest binary analysis for Go, native GoReleaser integration, most actively maintained, high trust, and the overwhelming community standard.

**Usage pattern**: Generate SBOM from the compiled binary during the GoReleaser release process using the built-in `sboms:` configuration.

### Complementary Tool: cyclonedx-gomod (app mode)

**Why**: Build constraint-aware source analysis produces the most accurate pre-build dependency inventory. Useful for development-time auditing and comparing against the binary SBOM.

### Vulnerability Scanning: govulncheck + grype

**Why**: govulncheck provides Go-native reachability analysis (fewest false positives). grype consumes Syft SBOMs for broader ecosystem vulnerability matching. Together they cover both precision and breadth.

### Do NOT Use

- **Trivy**: Supply chain compromise makes it unsuitable for a supply chain security tool
- **spdx-sbom-generator**: Archived/deprecated
- **bom**: Too niche, too small a community for a primary tool

---

## Sources

All raw source material is saved in `docs/`:
- `docs/syft-github-readme.md` — Syft repository overview
- `docs/syft-go-package-cataloging-deepwiki.md` — Syft Go cataloger architecture
- `docs/anchore-syft-scanning-architecture.md` — Syft scanning pipeline
- `docs/cyclonedx-gomod-github-readme.md` — cyclonedx-gomod repository overview
- `docs/trivy-sbom-docs.md` — Trivy SBOM documentation
- `docs/trivy-supply-chain-compromise-march-2026.md` — Trivy compromise details
- `docs/kubernetes-bom-github-readme.md` — bom repository overview
- `docs/spdx-sbom-generator-github-readme.md` — spdx-sbom-generator (deprecated)
- `docs/govulncheck-official-docs.md` — govulncheck capabilities
- `docs/go-vulnerability-management.md` — Go vuln management overview
- `docs/go-version-m-build-info.md` — go version -m output format
- `docs/debug-buildinfo-package.md` — debug/buildinfo package details
- `docs/sbomgenerator-go-guide.md` — Go SBOM generation comprehensive guide
- `docs/sbomify-tools-comparison-2026.md` — Multi-tool comparison (Jan 2026)
- `docs/sbom-tools-compared-syft-trivy-cyclonedx.md` — Three-tool comparison
- `docs/openssf-choosing-sbom-tool.md` — OpenSSF tool selection guidance
