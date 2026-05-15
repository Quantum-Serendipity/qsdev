# Tasks: Go SBOM Generation & GoReleaser Pipeline

## Phase 3: Synthesis & Review

### Pending

### Active

### Completed
- [x] **Cross-report reconciliation** — Resolve SPDX vs CycloneDX primary format tension between formats and distribution reports
  - Outcome: success
  - Completed: 2026-05-15
  - Notes: Both formats recommended — ship both. CycloneDX for VEX/scanning workflows, SPDX for GitHub/regulatory compliance.
- [x] **Write conclusions** — Synthesize all 7 reports into actionable architecture recommendation
  - Outcome: success
  - Completed: 2026-05-15
  - Notes: Full pipeline design, key decisions table, consumer verification paths documented in research.md

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Go SBOM generation tools** — syft, cyclonedx-gomod, trivy, `go version -m`, bom, spdx-sbom-generator: capabilities, accuracy, Go-specific strengths, maintenance status
  - Priority: high
  - Estimate: large
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `generation-tools-research.md`
  - Finding: Syft (8.9k stars) is the de facto standard with strong Go binary analysis and native GoReleaser integration. cyclonedx-gomod is the most accurate for Go-specific SBOMs (build-constraint-aware). Trivy disqualified by March 2026 supply chain compromise. All tools build on Go's `debug/buildinfo` foundation.
- [x] **Signing & attestation** — cosign, Sigstore, in-toto, SLSA provenance levels, SBOM signing workflows, keyless vs key-based
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `signing-attestation-research.md`
  - Finding: Cosign keyless signing via GitHub Actions OIDC is the recommended approach for qsdev. Fulcio issues 10-minute certs bound to workflow identity; Rekor provides non-repudiation. in-toto attestations (DSSE envelopes) wrap typed predicates (SBOM, SLSA provenance), binding metadata to artifact digests. SLSA L3 achievable via slsa-github-generator Go builder but incompatible with GoReleaser; GoReleaser + cosign achieves L2. Three consumer verification paths: `cosign verify-blob`, `gh attestation verify`, `slsa-verifier`. Both artifact AND SBOM signing needed. GoReleaser `signs:` block with cosign v3 `--bundle` produces `.sigstore.json` bundles. 17 sources documented.
- [x] **Vulnerability scanning from SBOMs** — How SBOMs feed grype, trivy, osv-scanner for downstream consumers; VEX integration
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `vulnerability-scanning-research.md`
  - Finding: Three major scanners (Grype, Trivy, OSV-Scanner) consume CycloneDX/SPDX SBOMs but produce 97.5% false positives due to package-level matching. The critical mitigation is govulncheck's `-format openvex` output, which provides reachability-based VEX that both Grype and Trivy consume natively. OpenVEX is the recommended VEX format (govulncheck produces, major scanners consume, SBOM-format-agnostic). Go vulndb uniquely includes symbol-level info. Dependency-Track is the enterprise standard for continuous monitoring. Recommended: ship CycloneDX SBOM + OpenVEX document; run govulncheck in CI.
- [x] **GitHub integration** — Dependency graph submission API, Dependabot SBOM export, GitHub Attestations, SLSA provenance, Actions workflow patterns
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `github-integration-research.md`
  - Finding: GitHub provides layered supply chain security for Go. Dependency graph now uses Dependabot-powered dynamic resolution (Dec 2025). `actions/attest@v4` is the canonical attestation action (SLSA L2, L3 via reusable workflows). GoReleaser integrates via `subject-checksums: ./dist/checksums.txt`. `slsa-github-generator` provides stronger L3 isolation but is incompatible with GoReleaser as build system. Private repo attestations require Enterprise Cloud. 16 sources documented.
- [x] **Distribution mechanisms** — Release asset packaging, OCI artifact attachment, Nix derivation considerations, how consumers discover and verify SBOMs
  - Priority: medium
  - Estimate: small → medium (scope expanded to cover 8 distribution channels + cross-ecosystem comparison)
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `distribution-research.md`
  - Finding: Layered distribution strategy recommended. Tier 1: GitHub Release assets with OpenSSF naming (`.cdx.json`) + GitHub Attestations. Tier 2: Embedded source SBOM via `//go:embed` for `go install` users. Tier 3: Channel-specific (Homebrew generates its own bottle SBOMs; Nix can include upstream SBOM in `$out/share/sbom/` with passthru attribute). OCI artifact attachment is low priority for CLI tools. CycloneDX JSON recommended as primary format. TEA is most promising universal discovery standard but not yet production-ready (Beta 2, target late 2026/2027).
- [x] **SBOM formats landscape** — SPDX vs CycloneDX: Go ecosystem adoption, tooling support, format maturity, which is standard for Go binaries
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `sbom-formats-research.md`
  - Finding: SPDX 2.3 JSON is the recommended primary format (GoReleaser default, GitHub native, Kubernetes/cosign precedent). CycloneDX JSON is a viable secondary format for VEX/vulnerability workflows. Ship both if downstream consumers need it -- trivial with GoReleaser.
- [x] **GoReleaser SBOM integration** — Built-in `sboms:` config, supported generators, release asset attachment, Nix/Homebrew/Docker artifact coverage
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: `goreleaser-sbom-research.md`
  - Finding: GoReleaser provides tool-agnostic SBOM generation via `sboms:` block (free/OSS since v1.2.0). Defaults to Syft generating SPDX-JSON per archive. SBOMs are GitHub Release assets covered by checksums.txt. Docker v2 generates/attaches SBOMs to OCI images by default (v2.12+). SBOMs do NOT flow to Homebrew/Scoop/Nix. Container images cannot be cataloged by the `sboms:` block -- only via `dockers_v2:` built-in or hooks. No Pro license needed for core SBOM features. Found 6+ real-world examples including GoReleaser itself, Syft, k8sgpt, OWASP Amass (cyclonedx-gomod), UDS CLI, and FluxCD.
