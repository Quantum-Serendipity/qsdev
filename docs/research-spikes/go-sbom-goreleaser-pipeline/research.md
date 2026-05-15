# Research Summary: Go SBOM Generation & GoReleaser Pipeline

## Overview

Exhaustively explore the ecosystem of tools, libraries, GitHub integrations, and GoReleaser integrations for automatically generating thorough SBOMs (Software Bill of Materials) that ship alongside the qsdev binary. The goal is full visibility into qsdev's entire dependency tree so users can verify it is secure, up to date, and uncompromised.

Key areas: SBOM formats (SPDX vs CycloneDX), Go-native generation tools (syft, cyclonedx-gomod, trivy, etc.), GoReleaser's built-in SBOM support, GitHub Actions integration, attestation/signing (cosign, SLSA provenance), and distribution mechanisms (embedding in release assets, OCI artifacts, GitHub dependency graph).

## Topics

### SBOM Format Landscape (SPDX vs CycloneDX)
- **Status**: Complete
- **Report**: [`sbom-formats-research.md`](sbom-formats-research.md)
- **Finding**: SPDX 2.3 JSON is the recommended primary format for qsdev. It is GoReleaser's zero-configuration default (Syft produces SPDX JSON), GitHub's native SBOM export format, and the format shipped by Kubernetes and cosign. CycloneDX JSON is a strong secondary option with superior VEX/vulnerability integration but is not the ecosystem default. Shipping both formats is trivial with GoReleaser (two YAML entries). SPDX 3.0 tooling is immature -- Syft does not support it -- so target SPDX 2.3. Trivy was compromised in March 2026; use Syft exclusively.

### Go SBOM Generation Tools — Landscape Analysis
- **Status**: Complete
- **Report**: [`generation-tools-research.md`](generation-tools-research.md)
- **Finding**: Seven tools evaluated for Go SBOM generation. Syft (Anchore, 8.9k stars) is the de facto standard with strong binary analysis, multi-format output (CycloneDX + SPDX), and native GoReleaser integration. cyclonedx-gomod (CycloneDX, 183 stars) produces the most accurate Go-specific SBOMs via build-constraint-aware source analysis, but outputs CycloneDX only. Trivy (~35k stars) is technically capable but disqualified by the March 2026 supply chain compromise. All tools build on Go's `debug/buildinfo` foundation. Recommended: Syft for binary SBOM generation in the release pipeline, cyclonedx-gomod as a build-time complement for development auditing, govulncheck for vulnerability scanning with reachability analysis.

### GoReleaser SBOM Integration
- **Status**: Complete
- **Report**: [`goreleaser-sbom-research.md`](goreleaser-sbom-research.md)
- **Finding**: GoReleaser provides first-class, tool-agnostic SBOM generation through its `sboms:` configuration block, available in the free/OSS edition since v1.2.0 (December 2021). The default uses Syft to generate SPDX-JSON per archive, but any generator (cyclonedx-gomod, trivy, custom scripts) works via the `cmd` field. SBOMs are placed in the dist directory, included in checksums.txt, and uploaded as GitHub Release assets. The pipeline order (Build -> Archive -> SBOM -> Checksum -> Sign -> Release) enables transitive trust: signing the checksum file covers all SBOMs. Docker v2 (v2.12+) generates and attaches SBOMs to OCI images by default. Critical limitations: the `sboms:` block cannot catalog container images, and SBOMs do not flow to Homebrew/Scoop/Nix distributions. No Pro license needed for core SBOM features. Six real-world open-source projects documented with SBOM configs.

### SBOM Distribution Mechanisms
- **Status**: Complete
- **Report**: [`distribution-research.md`](distribution-research.md)
- **Finding**: SBOM distribution for qsdev requires a layered strategy across multiple channels. Tier 1 (must have): GitHub Release assets using OpenSSF naming conventions (`.cdx.json` / `.spdx.json` appended to artifact name) plus GitHub Attestations for cryptographic binding via Sigstore. Tier 2 (should have): Embedded source SBOM via Go's `//go:embed` directive, exposed through `qsdev version --sbom`, ensuring `go install` users have SBOM access. Tier 3 (nice to have): Channel-specific accommodations -- Homebrew generates its own bottle SBOMs (upstream SBOMs do not survive), Nix can include upstream SBOM in `$out/share/sbom/` with passthru attributes. OCI artifact attachment (ORAS/cosign) is low priority unless qsdev ships container images. CycloneDX JSON is recommended as primary format based on ecosystem adoption patterns. The Transparency Exchange API (TEA) is the most promising universal discovery standard but remains in Beta 2.

### GitHub SBOM & Supply Chain Security Integration
- **Status**: Complete
- **Report**: [`github-integration-research.md`](github-integration-research.md)
- **Finding**: GitHub provides a layered supply chain security ecosystem for Go projects. The dependency graph now uses Dependabot-powered dynamic resolution for Go (Dec 2025), providing accurate transitive dependency tracking without consuming Actions minutes. `actions/attest@v4` is the canonical attestation action (both `attest-build-provenance` and `attest-sbom` are deprecated wrappers), achieving SLSA Build Level 2 in normal workflows or Level 3 via reusable workflows. For qsdev, the recommended approach combines GoReleaser's Syft-based SBOM generation with `actions/attest@v4` for build provenance attestation via `subject-checksums: ./dist/checksums.txt`, plus optional SBOM attestation binding SBOMs to specific artifacts. The `slsa-github-generator` Go builder provides stronger L3 isolation but is incompatible with GoReleaser as a build system -- use it only if formal SLSA L3 certification is required. Attestations are stored in GitHub's API with Sigstore transparency log entries for public repos; private repos require GitHub Enterprise Cloud. Consumers verify with `gh attestation verify`. 16 sources documented across GitHub docs, GoReleaser docs, and SLSA framework docs.

### Vulnerability Scanning & VEX Integration
- **Status**: Complete
- **Report**: [`vulnerability-scanning-research.md`](vulnerability-scanning-research.md)
- **Finding**: Three major open-source scanners (Grype, Trivy, OSV-Scanner) consume CycloneDX/SPDX SBOMs for vulnerability scanning, but SBOM-based scanning alone produces a 97.5% false positive rate because it performs package-level matching without reachability analysis. For Go projects, the critical mitigation is govulncheck's `-format openvex` output, which generates VEX documents with call-graph-based reachability assessments. Both Grype and Trivy natively consume OpenVEX to suppress unreachable vulnerabilities. OpenVEX is the recommended VEX format because govulncheck produces it, major scanners consume it, and it is SBOM-format-agnostic. The Go vulnerability database (vuln.go.dev) is uniquely valuable among vulnerability databases because it includes symbol-level information enabling reachability analysis. For enterprise consumers, Dependency-Track (OWASP) is the standard platform for continuous SBOM monitoring. Recommended for qsdev: ship a CycloneDX SBOM plus an OpenVEX document generated by govulncheck as release artifacts, and run govulncheck in CI.

### SBOM Signing, Attestation & Cryptographic Verification
- **Status**: Complete
- **Report**: [`signing-attestation-research.md`](signing-attestation-research.md)
- **Finding**: Cosign keyless signing via GitHub Actions OIDC is the recommended approach for qsdev, eliminating all key management while providing strong cryptographic provenance. Fulcio issues 10-minute X.509 certificates bound to the GitHub Actions workflow identity, and Rekor's append-only transparency log provides non-repudiation through Merkle-tree-based inclusion proofs. The in-toto attestation framework wraps typed predicates (SBOM as CycloneDX/SPDX, SLSA provenance) in DSSE envelopes, cryptographically binding metadata to specific artifact digests -- fundamentally different from simple signing, which only proves identity. SLSA Build Level 3 is achievable via the `slsa-github-generator` Go builder but is incompatible with GoReleaser as a build system; GoReleaser with cosign keyless signing achieves Level 2, which provides strong practical security for most threat models. Both artifact signing and SBOM signing are needed (they answer different questions: who built it vs what's inside). GoReleaser's `signs:` block with cosign v3's `--bundle` flag produces single `.sigstore.json` files containing signature, Fulcio certificate, and Rekor entry. Consumers have three verification paths: `cosign verify-blob` (Sigstore-native, most flexible), `gh attestation verify` (simplest for GitHub-hosted projects), and `slsa-verifier verify-artifact` (SLSA provenance). Real-world examples documented: goreleaser/example-supply-chain, mchmarny/s3cme, slsa-verifier itself. 17 sources documented.

## Cross-Report Tension: Primary Format

The formats research recommends SPDX 2.3 as primary (GoReleaser default, GitHub native export, Kubernetes/cosign precedent). The distribution research recommends CycloneDX as primary (native VEX support, 100% PyPI adopter choice, ECMA-424 standardization, better Go tooling via cyclonedx-gomod). Both are correct from different angles — they serve different consumers. The resolution: **ship both formats** (trivial with GoReleaser, two YAML entries), with CycloneDX as the actionable format for vulnerability/VEX workflows and SPDX for GitHub/regulatory compliance. If embedding one format via `//go:embed` for `go install` users, CycloneDX is the better choice because downstream scanners (Grype, Trivy) consume it alongside OpenVEX documents.

## Open Questions

1. **SLSA L3 vs GoReleaser tradeoff**: If formal SLSA L3 certification becomes required, should qsdev split the pipeline (SLSA Go builder for binaries, GoReleaser for packaging/distribution)?
2. **VEX as attestation**: Should govulncheck's OpenVEX output also be signed and attached as a typed attestation alongside SBOMs?
3. **Nix derivation signing**: How should SBOMs and provenance compose with Nix's own reproducibility guarantees?
4. **Consumer verification script**: Should qsdev ship a `verify.sh` or Makefile target for easy local verification?

## Conclusions

### Recommended Architecture for qsdev

The research converges on a clear pipeline design with no significant open trade-offs blocking implementation:

**Generation**: Syft is the primary SBOM generator, integrated via GoReleaser's `sboms:` block (OSS, zero Pro dependency). It produces both SPDX 2.3 JSON and CycloneDX 1.5 JSON per release artifact. cyclonedx-gomod supplements as a build-time development tool for precision SBOMs with build-constraint awareness. Trivy is explicitly excluded due to its March 2026 supply chain compromise.

**Signing & Attestation**: Cosign keyless signing via GitHub Actions OIDC eliminates all key management. GoReleaser's `signs:` block with cosign v3 `--bundle` produces `.sigstore.json` bundles containing signature, Fulcio certificate, and Rekor transparency log entry. `actions/attest@v4` adds GitHub-native attestations for both build provenance and SBOMs. This achieves SLSA Build Level 2. Level 3 is achievable via `slsa-github-generator` but is incompatible with GoReleaser as a build system — only pursue if formal L3 certification becomes a requirement.

**Vulnerability Mitigation**: govulncheck runs in CI with `-format openvex` to produce reachability-based VEX documents. These ship alongside SBOMs as release assets, reducing the 97.5% false positive rate of package-level SBOM scanning to only genuinely reachable vulnerabilities. Grype is the recommended downstream scanner (better than Trivy for third-party SBOM consumption).

**Distribution**: Layered across four channels:
- **GitHub Releases**: SBOM files with OpenSSF naming (`.cdx.json`, `.spdx.json`) as release assets, covered by checksums.txt, signed via cosign, attested via `actions/attest@v4`
- **`go install`**: Embedded CycloneDX SBOM via `//go:embed`, exposed through `qsdev version --sbom`
- **Homebrew**: Accept that Homebrew generates its own bottle SBOMs; upstream SBOMs do not survive
- **Nix**: Include upstream SBOM in `$out/share/sbom/` with passthru attributes

**Consumer Verification**: Three paths available — `gh attestation verify` (simplest), `cosign verify-blob` (most flexible), `slsa-verifier` (SLSA-specific). Document all three in release notes.

### Pipeline Summary

```
go build → Syft scans binary → SPDX + CycloneDX SBOMs
                              → govulncheck → OpenVEX
GoReleaser: Build → Archive → SBOM → Checksum → Sign (cosign) → Release
GitHub Actions: → actions/attest@v4 (build provenance + SBOM attestation)
```

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary generator | Syft | De facto standard, GoReleaser native, multi-format output |
| Formats | Both SPDX 2.3 + CycloneDX 1.5 | Different consumers need different formats; trivial to ship both |
| Embedded format | CycloneDX JSON | VEX compatibility, scanner ecosystem preference |
| Signing approach | Cosign keyless (OIDC) | Zero key management, Fulcio + Rekor trust chain |
| SLSA level | L2 (GoReleaser + cosign) | L3 requires abandoning GoReleaser; L2 is sufficient |
| VEX format | OpenVEX (govulncheck) | Go-native, scanner-compatible, reachability-based |
| Trivy | Excluded | March 2026 supply chain compromise |

### What This Enables

Users who download qsdev can:
1. Verify the binary was built by the official CI pipeline (`gh attestation verify`)
2. Inspect the full dependency tree (SBOM in release assets or `qsdev version --sbom`)
3. Scan for vulnerabilities with false-positive suppression (SBOM + OpenVEX → Grype)
4. Ingest into enterprise platforms (Dependency-Track) for continuous monitoring
5. Verify cryptographic provenance without trusting GitHub (cosign + Rekor)
