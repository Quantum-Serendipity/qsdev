# SBOM Format Landscape: SPDX vs CycloneDX for Go Binaries

## Research Question

What SBOM format(s) should a Go CLI tool (qsdev) use for automatically generated SBOMs shipped alongside release binaries?

## Executive Summary

**Recommendation: Ship SPDX JSON as the primary format. Optionally ship CycloneDX JSON as a second artifact if downstream consumers need it.** GoReleaser defaults to SPDX JSON via Syft, GitHub natively exports SPDX, and the largest Go projects (Kubernetes, cosign) ship SPDX. CycloneDX offers superior VEX/vulnerability integration but is not required for qsdev's use case. Generating both formats is trivial with GoReleaser and costs nothing at build time.

---

## 1. SPDX: The Compliance-First Standard

### Version History

| Version | Date | Key Changes |
|---------|------|-------------|
| 2.0 | 2015 | First major release with package-level SBOM support |
| 2.2 | May 2020 | Added SPDX-lite; satisfies NTIA minimum SBOM elements |
| 2.2.1 | Aug 2021 | Published as **ISO/IEC 5962:2021** -- first ISO-standardized SBOM format |
| 2.3 | Aug 2022 | Added security-related fields, improved interoperability with CycloneDX, explicit NTIA compliance guidance (Annex K.2) |
| 3.0 | Apr 2024 | Major rewrite: profile-based architecture (Core, Software, Security, AI, Build, Dataset), element-based linked-data model, JSON-LD serialization |
| 3.0.1 | Aug 2025 | Bugfix release; 3.1 RC available |

### Standardization

- **ISO/IEC 5962:2021** -- the only SBOM format with full ISO standardization
- Maintained by the Linux Foundation
- The SPDX License List is the industry-standard license identifier catalog used by both SPDX and CycloneDX

### Serialization Formats

- **SPDX 2.3**: JSON, Tag-Value, RDF/XML, YAML, spreadsheet
- **SPDX 3.0**: JSON-LD alongside traditional formats
- **Practical default for Go tooling**: SPDX 2.3 JSON (`spdx-json`)

### NTIA Minimum Elements Compliance

SPDX 2.3 directly maps to all seven NTIA minimum elements:

| NTIA Element | SPDX 2.3 Field |
|---|---|
| Supplier Name | `PackageSupplier` |
| Component Name | `PackageName` |
| Version | `PackageVersion` |
| Unique Identifiers | `DocumentNamespace`, `SPDXID` |
| Dependency Relationship | `Relationship` (CONTAINS) |
| Author of SBOM Data | `Creator` |
| Timestamp | `Created` |

An [NTIA conformance checker](https://github.com/spdx/ntia-conformance-checker) exists for automated validation.

### SPDX 3.0 Adoption Reality

SPDX 3.0 is architecturally ambitious but **tooling lags significantly**:
- Syft does NOT support SPDX 3.0 output (issue #1970 is in backlog with no timeline)
- GoReleaser's default Syft integration produces SPDX 2.3
- Most generation tools still target 2.3
- The graph-based model is more powerful but introduces complexity

**Practical implication for qsdev**: Target SPDX 2.3, not 3.0. When Syft adds 3.0 support, upgrading will be a config change.

### Strengths

- ISO standardization carries weight in regulated industries
- Richer license metadata (the SPDX License List is the canonical reference)
- File-level and snippet-level analysis support
- Mature ecosystem: 470 tools (vs 171 for CycloneDX per December 2025 study)
- GitHub's native SBOM export is SPDX format

### Weaknesses

- More verbose document structure with more mandatory fields
- No native vulnerability/VEX support in 2.3 (requires external references; 3.0 adds Security profile but tooling isn't ready)
- Slower release cadence (2-3 years between majors)
- SPDX 3.0's linked-data model adds complexity without clear tooling benefits yet

---

## 2. CycloneDX: The Security-First Standard

### Version History

| Version | Date | Key Changes |
|---------|------|-------------|
| 1.4 | Jan 2022 | Added vulnerability disclosure, VEX support |
| 1.5 | Jun 2023 | ML-BOM, Manufacturing BOM, SaaS BOM |
| 1.6 | Apr 2024 | Attestation support, Cryptography BOM; **ECMA-424** standardization |
| 1.7 | Oct 2025 | Patent/IP metadata, data provenance citations, enhanced crypto transparency |

### Standardization

- **ECMA-424** -- Ecma International standard (less weight than ISO in government procurement, but growing)
- Maintained by OWASP Foundation
- Rapid release cadence (~annual major versions)

### Serialization Formats

- JSON, XML, Protocol Buffers
- **Practical default for Go tooling**: CycloneDX JSON (`cyclonedx-json`)

### VEX Integration

CycloneDX's killer feature: **native VEX (Vulnerability Exploitability eXchange) support**. A single CycloneDX document can contain both the component inventory and exploitability status for known vulnerabilities.

This is particularly relevant for Go because:
- Go's dead code elimination means many `go.mod` dependencies may not be compiled into the binary
- A VEX document can communicate that a vulnerable dependency exists in go.mod but the vulnerable code path is unreachable in the compiled binary
- Industry data suggests up to 85% of flagged vulnerabilities in open-source libraries aren't reachable in production

Grype has initial support for consuming CycloneDX VEX documents.

### Go-Specific Tooling

- **cyclonedx-gomod**: Purpose-built Go module SBOM generator, tighter Go module system integration than Syft
- **cyclonedx-go**: Go library for consuming/producing CycloneDX SBOMs programmatically
- Can be embedded directly into Go applications for runtime SBOM generation

### Strengths

- Simpler, flatter document model -- fewer mandatory fields, easier to parse
- Native VEX and vulnerability disclosure support
- Faster iteration with annual releases
- Higher developer engagement metrics per recent research
- EU CRA compliance explicitly accepts CycloneDX 1.6+

### Weaknesses

- Smaller tool ecosystem (171 vs 470 tools)
- ECMA standardization carries less weight than ISO in some regulated contexts
- Not GitHub's native export format
- Less mature license metadata compared to SPDX's License List heritage

---

## 3. Head-to-Head Comparison for Go Ecosystem

### Tool Defaults

| Tool | Default SBOM Format | Notes |
|------|---------------------|-------|
| **GoReleaser** | SPDX JSON (via Syft) | Default args: `["$artifact", "--output", "spdx-json=$document", "--enrich", "all"]` |
| **Syft** | Syft native JSON | Supports both SPDX and CycloneDX output; SPDX JSON is GoReleaser's configured default |
| **Trivy** | CycloneDX | Supports both; **compromised in March 2026 supply chain attack -- avoid in CI/CD** |
| **cyclonedx-gomod** | CycloneDX JSON | CycloneDX-only tool |
| **bom (Kubernetes)** | SPDX 2.3 | Built specifically for Kubernetes release SBOMs |
| **GitHub Export** | SPDX | Native dependency graph export is SPDX; CycloneDX via submission actions only |
| **GitHub Dependency Submission API** | Both | Accepts SPDX and CycloneDX uploads |

### What Real Go Projects Ship

| Project | SBOM Format | Tool | Notes |
|---------|-------------|------|-------|
| **Kubernetes** | SPDX | bom (custom) | Built dedicated SPDX tooling; largest Go project |
| **cosign (Sigstore)** | SPDX JSON | Syft via GoReleaser | Uses GoReleaser defaults (`sboms: - artifacts: binary`) |
| **GoReleaser example-supply-chain** | SPDX JSON | Syft | Official example repo; demonstrates the default path |
| **CycloneDX projects** | CycloneDX | cyclonedx-gomod | Naturally use their own format |

**Pattern**: Go projects using GoReleaser overwhelmingly ship SPDX JSON because that's the zero-configuration default. Projects with specific security/VEX requirements may add CycloneDX.

### GitHub Integration

- **Native SBOM export**: SPDX format only (UI export, REST API)
- **Dependency submission**: Accepts both formats via Actions
- **Artifact attestations** (`gh attestation verify`): Format-agnostic; works with any SBOM attached as a release asset
- **Dependabot**: Does not consume release SBOMs; uses its own dependency graph analysis

### Downstream Vulnerability Scanning

Both formats work with major vulnerability scanners:
- **Grype**: Accepts both SPDX and CycloneDX; has additional CycloneDX VEX support
- **osv-scanner**: Accepts both formats
- **govulncheck**: Works from Go source, not SBOMs (complementary tool)

---

## 4. Regulatory and Industry Landscape

### US Federal Requirements

- **EO 14028 (May 2021)**: Requires SBOMs for federal software procurement. Format-agnostic -- accepts SPDX, CycloneDX, and SWID.
- **NTIA Minimum Elements (Jul 2021)**: Defines required data fields. Both formats satisfy all requirements.
- **CISA 2025 Minimum Elements**: Updated guidance; still format-agnostic.
- **EO 14144 (Jan 2025)**: Strengthened attestation requirements including mandatory SBOMs. **EO 14306 (Jun 2025)**: Rescinded key portions including the SBOM-as-artifact mandate and CISA validation role.

**Net effect**: US federal landscape accepts both formats equally. The political pendulum has swung away from mandatory SBOMs, but the technical infrastructure remains.

### EU Cyber Resilience Act

The most prescriptive requirement: **BSI TR-03183-2 specifies CycloneDX 1.6+ or SPDX 3.0.1+ in JSON or XML**. Since SPDX 3.0 tooling is immature, CycloneDX has a practical advantage for EU CRA compliance today.

### Industry Consensus

Every major comparison source reaches the same conclusion: the formats are converging in capability, and most organizations should either pick the one their toolchain defaults to or ship both. As Anchore puts it: "There is equally strong demand for the two formats."

---

## 5. Can You Ship Both? Should You?

### Shipping Both is Trivial with GoReleaser

```yaml
sboms:
  - id: spdx
    artifacts: binary
    # Uses GoReleaser defaults: syft with spdx-json output

  - id: cyclonedx
    cmd: syft
    args: ["$artifact", "--output", "cyclonedx-json=$document"]
    artifacts: binary
    documents:
      - "${artifact}.cyclonedx.json"
```

This produces two SBOM files per binary artifact. Syft runs twice but is fast (seconds per binary). Both files are automatically attached as release assets.

### Cost-Benefit Analysis

| Factor | Ship SPDX only | Ship CycloneDX only | Ship both |
|--------|----------------|---------------------|-----------|
| GoReleaser config | Zero-config default | Custom args needed | Two entries |
| GitHub native compat | Full | Submission API only | Full + submission |
| Vulnerability scanning | Grype, osv-scanner | Grype + VEX, osv-scanner | Best of both |
| Regulatory coverage | US federal, ISO | EU CRA (today) | Complete |
| Build time cost | ~seconds | ~seconds | ~2x seconds (negligible) |
| Consumer confusion | None | None | Two files to choose from |
| Maintenance burden | Minimal | Minimal | Two files to validate |

---

## 6. Practical Recommendation for qsdev

### Primary: SPDX JSON (via GoReleaser defaults)

1. **Zero-configuration path**: GoReleaser + Syft produces SPDX JSON out of the box
2. **Ecosystem alignment**: Kubernetes, cosign, and most GoReleaser-based Go projects ship SPDX
3. **GitHub compatibility**: Native dependency graph integration
4. **NTIA compliance**: Full coverage of all minimum elements
5. **Industry standard**: ISO/IEC 5962:2021

### Optional: Add CycloneDX JSON as a second artifact

Add it when any of these apply:
- Downstream consumers explicitly request CycloneDX
- You want VEX integration for vulnerability triage
- EU CRA compliance becomes relevant (CycloneDX has better tooling support today than SPDX 3.0)
- You want to maximize compatibility at negligible cost

### What NOT to do

- **Don't use SPDX 3.0**: Tooling isn't ready. Syft doesn't support it. Stick with 2.3.
- **Don't use Trivy**: Compromised in March 2026 supply chain attack. Use Syft.
- **Don't use Tag-Value format**: JSON is the interoperable choice. Tag-Value is human-readable but less tooling support.
- **Don't convert between formats**: Native generation preserves all metadata. Format conversion loses information at boundary edges.

### Recommended Starting Configuration

```yaml
# .goreleaser.yaml
sboms:
  - artifacts: binary
```

That's it. One line. GoReleaser handles the rest: Syft generates SPDX 2.3 JSON for each platform binary, and the files are attached as release assets.

When you're ready to add CycloneDX:

```yaml
sboms:
  - id: spdx
    artifacts: binary

  - id: cyclonedx
    cmd: syft
    args: ["$artifact", "--output", "cyclonedx-json=$document"]
    artifacts: binary
    documents:
      - "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.cdx.json"
```

---

## 7. Depth Checklist

- [x] **Underlying mechanism explained**: Both format structures, serialization options, and document models covered
- [x] **Key tradeoffs and limitations identified**: SPDX verbosity vs CycloneDX simplicity; VEX gap in SPDX 2.3; SPDX 3.0 tooling immaturity; Trivy compromise
- [x] **Compared to alternatives**: SPDX vs CycloneDX head-to-head across seven dimensions
- [x] **Failure modes and edge cases**: Format conversion data loss; SPDX 3.0 tooling gaps; Trivy supply chain attack; EU CRA version requirements
- [x] **Concrete examples**: Kubernetes (SPDX), cosign (SPDX via GoReleaser defaults), GoReleaser example-supply-chain, CycloneDX-gomod projects
- [x] **Standalone-readable**: Complete with GoReleaser configs, tool comparison tables, and actionable recommendation

## Sources

All raw sources saved to `docs/`:
- `goreleaser-sbom-configuration.md` -- GoReleaser SBOM docs
- `sbomify-cyclonedx-vs-spdx-comparison.md` -- Format comparison (sbomify, Jan 2026)
- `anchore-sbom-standards-overview.md` -- Anchore format overview
- `sbomgenerator-go-guide.md` -- Go SBOM generation guide
- `spdx-ntia-sbom-howto.md` -- SPDX NTIA compliance guide
- `github-sbom-export-docs.md` -- GitHub SBOM export documentation
- `arxiv-sbom-tool-ecosystems-study.md` -- Academic study of tool ecosystems (Dec 2025)
- `openssf-choosing-sbom-generation-tool.md` -- OpenSSF tool selection guide
- `goreleaser-issue-2808-cyclonedx-support.md` -- GoReleaser CycloneDX support history
- `goreleaser-example-supply-chain-overview.md` -- Official GoReleaser supply chain example
- `syft-issue-1970-spdx3-support.md` -- Syft SPDX 3.0 support status
- `cosign-goreleaser-yml-analysis.md` -- Cosign's own GoReleaser config
- `kubernetes-bom-spdx-tool.md` -- Kubernetes bom tool
- `cyclonedx-vex-practical-usage.md` -- CycloneDX VEX practical usage
