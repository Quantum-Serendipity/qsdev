# Research Log: Go SBOM Generation & GoReleaser Pipeline

## 2026-05-15 12:00 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Goal: exhaustively explore the ecosystem, tools, libraries, GitHub integrations, and GoReleaser integrations for thorough, automatically generated SBOMs that ship alongside the qsdev binary. Users should be able to verify the full dependency tree is secure, up to date, and uncompromised.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-15 14:30 — SBOM Formats Landscape Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GoReleaser SBOM docs](https://goreleaser.com/customization/sbom/) → `docs/goreleaser-sbom-configuration.md`
  - [sbomify format comparison](https://sbomify.com/2026/01/15/sbom-formats-cyclonedx-vs-spdx/) → `docs/sbomify-cyclonedx-vs-spdx-comparison.md`
  - [Anchore SBOM standards](https://anchore.com/sbom/key-things-to-know-about-sboms-and-sbom-standards/) → `docs/anchore-sbom-standards-overview.md`
  - [Go SBOM generation guide](https://sbomgenerator.com/guides/go) → `docs/sbomgenerator-go-guide.md`
  - [SPDX NTIA HOWTO](https://spdx.github.io/spdx-ntia-sbom-howto/) → `docs/spdx-ntia-sbom-howto.md`
  - [GitHub SBOM export docs](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/exporting-a-software-bill-of-materials-for-your-repository) → `docs/github-sbom-export-docs.md`
  - [arXiv SBOM tool ecosystems study](https://arxiv.org/abs/2512.21781) → `docs/arxiv-sbom-tool-ecosystems-study.md`
  - [OpenSSF tool selection](https://openssf.org/blog/2025/06/05/choosing-an-sbom-generation-tool/) → `docs/openssf-choosing-sbom-generation-tool.md`
  - [GoReleaser CycloneDX issue #2808](https://github.com/goreleaser/goreleaser/issues/2808) → `docs/goreleaser-issue-2808-cyclonedx-support.md`
  - [GoReleaser example-supply-chain](https://github.com/goreleaser/example-supply-chain) → `docs/goreleaser-example-supply-chain-overview.md`
  - [Syft SPDX 3.0 issue #1970](https://github.com/anchore/syft/issues/1970) → `docs/syft-issue-1970-spdx3-support.md`
  - [Cosign .goreleaser.yml](https://raw.githubusercontent.com/sigstore/cosign/main/.goreleaser.yml) → `docs/cosign-goreleaser-yml-analysis.md`
  - [Kubernetes bom tool](https://github.com/kubernetes-sigs/bom) → `docs/kubernetes-bom-spdx-tool.md`
  - [CycloneDX VEX usage](https://cyclonedx.org/capabilities/vex/) → `docs/cyclonedx-vex-practical-usage.md`
- **Summary**: Deep investigation of SPDX vs CycloneDX for Go binaries. SPDX 2.3 JSON is the recommended primary format: it's GoReleaser's zero-config default (via Syft), GitHub's native export format, and what Kubernetes and cosign ship. CycloneDX offers superior VEX/vulnerability integration but is secondary. Both can be shipped trivially. SPDX 3.0 tooling is immature (Syft doesn't support it). Trivy was compromised in Mar 2026 -- use Syft. Full report at `sbom-formats-research.md`.
- **Next**: Continue with remaining Phase 1 tasks (Go SBOM generation tools, GoReleaser integration, GitHub integration, signing/attestation, distribution, vuln scanning).

## 2026-05-15 15:45 — Go SBOM Generation Tools Landscape Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Syft GitHub](https://github.com/anchore/syft) → `docs/syft-github-readme.md`
  - [Syft Go cataloging architecture](https://deepwiki.com/anchore/syft/3.2-go-package-cataloging) → `docs/syft-go-package-cataloging-deepwiki.md`
  - [Syft scanning pipeline](https://anchore.com/blog/how-syft-scans-software-to-generate-sboms/) → `docs/anchore-syft-scanning-architecture.md`
  - [cyclonedx-gomod GitHub](https://github.com/CycloneDX/cyclonedx-gomod) → `docs/cyclonedx-gomod-github-readme.md`
  - [Trivy SBOM docs](https://trivy.dev/docs/latest/supply-chain/sbom/) → `docs/trivy-sbom-docs.md`
  - [Trivy supply chain compromise](multiple sources) → `docs/trivy-supply-chain-compromise-march-2026.md`
  - [Kubernetes bom GitHub](https://github.com/kubernetes-sigs/bom) → `docs/kubernetes-bom-github-readme.md`
  - [spdx-sbom-generator GitHub](https://github.com/opensbom-generator/spdx-sbom-generator) → `docs/spdx-sbom-generator-github-readme.md`
  - [govulncheck docs](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) → `docs/govulncheck-official-docs.md`
  - [Go vulnerability management](https://go.dev/doc/security/vuln/) → `docs/go-vulnerability-management.md`
  - [go version -m details](https://appliedgo.net/spotlight/get-build-information-of-a-go-binary/) → `docs/go-version-m-build-info.md`
  - [debug/buildinfo package](https://pkg.go.dev/debug/buildinfo) → `docs/debug-buildinfo-package.md`
  - [Go SBOM guide](https://sbomgenerator.com/guides/go) → `docs/sbomgenerator-go-guide.md`
  - [Tool comparison Jan 2026](https://sbomify.com/2026/01/26/sbom-generation-tools-comparison/) → `docs/sbomify-tools-comparison-2026.md`
  - [Syft vs Trivy vs CycloneDX](https://secure-pipelines.com/ci-cd-security/sbom-tools-compared-syft-trivy-cyclonedx-cli/) → `docs/sbom-tools-compared-syft-trivy-cyclonedx.md`
  - [OpenSSF tool guidance](https://openssf.org/blog/2025/06/05/choosing-an-sbom-generation-tool/) → `docs/openssf-choosing-sbom-tool.md`
- **Summary**: Comprehensive landscape analysis of 7 Go SBOM generation tools. Syft (8.9k stars, v1.44.0) is the de facto standard with strong Go binary analysis via pluggable catalogers, 4-tier license resolution, and native GoReleaser integration. cyclonedx-gomod (183 stars, v1.10.0) is the most accurate for Go specifically — its `app` mode evaluates build constraints (GOOS/GOARCH), producing precision SBOMs no other tool matches. Trivy (~35k stars) is technically capable but disqualified by the March 2026 supply chain compromise. cdxgen (917 stars) requires Node.js and is better for polyglot projects. bom (455 stars) is Kubernetes-niche. spdx-sbom-generator was archived Jan 2025. govulncheck complements SBOM tools with reachability-based vulnerability analysis and VEX output. All tools build on the same foundation: Go's `debug/buildinfo` embedded in compiled binaries. Full report at `generation-tools-research.md`.
- **Next**: Continue with GoReleaser SBOM integration, GitHub integration, signing/attestation tasks.

## 2026-05-15 16:30 — SBOM Distribution Mechanisms Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [OpenSSF SBOM naming conventions](https://sbom-catalog.openssf.org/sbom-naming.html) → `docs/openssf-sbom-naming-conventions.md`
  - [GitHub Releases SBOM problems](https://sbom-insights.dev/posts/github-releases-are-where-sboms-goto-die/) → `docs/github-releases-sbom-distribution-problems.md`
  - [GoReleaser SBOM config](https://goreleaser.com/customization/sbom/) → `docs/goreleaser-sbom-configuration.md`
  - [GoReleaser supply chain example](https://github.com/goreleaser/goreleaser-example-supply-chain) → `docs/goreleaser-supply-chain-example.md`
  - [GitHub Actions attest-sbom](https://github.com/actions/attest-sbom) → `docs/github-actions-attest-sbom.md`
  - [OCI reference types](https://oras.land/docs/concepts/reftypes/) → `docs/oci-reference-types-attached-artifacts.md`
  - [Cosign signing other types](https://docs.sigstore.dev/cosign/signing/other_types/) → `docs/cosign-signing-other-types-sbom.md`
  - [Homebrew 4.3.0 SBOM support](https://brew.sh/2024/05/14/homebrew-4.3.0/) → `docs/homebrew-4.3-sbom-attestation.md`
  - [Nix State of the SBOM](https://arnout.engelen.eu/blog/nix-state-of-the-sbom/) → `docs/nix-state-of-the-sbom.md`
  - [Bombon Nix CycloneDX tool](https://github.com/nikstur/bombon) → `docs/bombon-nix-cyclonedx-sbom.md`
  - [sbomnix tool](https://github.com/tiiuae/sbomnix) → `docs/sbomnix-nix-sbom-generation.md`
  - [Go binary build info](https://appliedgo.net/spotlight/get-build-information-of-a-go-binary/) → `docs/go-binary-build-information.md`
  - [Go SBOM comprehensive guide](https://sbomgenerator.com/guides/go) → `docs/go-sbom-generation-comprehensive-guide.md`
  - [Transparency Exchange API](https://sbomify.com/2026/03/01/why-were-bullish-on-tea/) → `docs/transparency-exchange-api-tea.md`
  - [PyPI SBOM adoption / PEP 770](https://sbomify.com/2026/03/12/pypi-sbom-analysis/) → `docs/pypi-sbom-adoption-pep770.md`
  - [SBOM in Go - ByteSizeGo](https://www.bytesizego.com/blog/understanding-sbom-in-go) → `docs/understanding-sbom-in-go-bytesizego.md`
- **Summary**: Comprehensive analysis of SBOM distribution across 8 channels. Recommended layered strategy: (1) GitHub Release assets with OpenSSF naming (`.cdx.json`) as canonical source, (2) GitHub Attestations for cryptographic binding, (3) embedded source SBOM via `//go:embed` for `go install` users, (4) channel-specific accommodations for Homebrew (accept its own bottle SBOMs) and Nix (include in `$out/share/sbom/` + passthru attribute). CycloneDX JSON recommended as primary format. OCI artifact attachment is low priority for CLI tools. TEA is the most promising universal discovery standard but not yet production-ready. Full report at `distribution-research.md`.
- **Next**: Continue with remaining Phase 1 tasks (signing/attestation, vulnerability scanning).

## 2026-05-15 17:00 — GoReleaser SBOM Integration Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GoReleaser SBOM docs](https://goreleaser.com/customization/sbom/) -> `docs/goreleaser-sbom-configuration-docs.md`
  - [GoReleaser supply chain blog](https://goreleaser.com/blog/supply-chain-security/) -> `docs/goreleaser-supply-chain-security-blog.md`
  - [Example supply chain repo](https://github.com/goreleaser/goreleaser-example-supply-chain) -> `docs/goreleaser-example-supply-chain-repo.md`
  - [Example .goreleaser.yaml](https://raw.githubusercontent.com/goreleaser/goreleaser-example-supply-chain/main/.goreleaser.yaml) -> `docs/goreleaser-example-supply-chain-yaml.md`
  - [PR #2648 adding SBOM support](https://github.com/goreleaser/goreleaser/pull/2648) -> `docs/goreleaser-sbom-pr-2648.md`
  - [GoReleaser's own config](https://raw.githubusercontent.com/goreleaser/goreleaser/main/.goreleaser.yaml) -> `docs/goreleaser-full-goreleaser-yaml.md`
  - [Docker v2 docs](https://goreleaser.com/customization/package/dockers_v2/) -> `docs/goreleaser-docker-v2-docs.md`
  - [Signing docs](https://goreleaser.com/customization/sign/sign/) -> `docs/goreleaser-signing-docs.md`
  - [Pro features list](https://goreleaser.com/pro/) -> `docs/goreleaser-pro-features-list.md`
  - [SBOM proposal #2597](https://github.com/goreleaser/goreleaser/issues/2597) -> `docs/goreleaser-sbom-proposal-issue-2597.md`
  - [CycloneDX request #2808](https://github.com/goreleaser/goreleaser/issues/2808) -> `docs/goreleaser-cyclonedx-sbom-issue-2808.md`
  - [Syft .goreleaser.yaml](https://raw.githubusercontent.com/anchore/syft/main/.goreleaser.yaml) -> `docs/syft-goreleaser-yaml.md`
  - [k8sgpt config](https://raw.githubusercontent.com/k8sgpt-ai/k8sgpt/main/.goreleaser.yaml) -> `docs/k8sgpt-goreleaser-yaml.md`
  - [UDS CLI config](https://raw.githubusercontent.com/defenseunicorns/uds-cli/main/.goreleaser.yaml) -> `docs/uds-cli-goreleaser-yaml.md`
  - [OWASP Amass config](https://raw.githubusercontent.com/OWASP/Amass/master/.goreleaser.yaml) -> `docs/owasp-amass-goreleaser-yaml.md`
  - [ContainerInfra GitLab blog](https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/) -> `docs/containerinfra-goreleaser-gitlab-sbom-blog.md`
- **Summary**: Exhaustive investigation of GoReleaser's SBOM capabilities. The `sboms:` block is tool-agnostic (defaults to syft, supports cyclonedx-gomod, trivy, custom scripts), available in OSS since v1.2.0. Pipeline order is Build->Archive->SBOM->Checksum->Sign->Release, enabling transitive trust via checksum signing. Docker v2 (v2.12+) has built-in SBOM attachment ON by default. Critical limitation: `sboms:` block cannot catalog container images. SBOMs do not flow to Homebrew/Scoop/Nix. Found 6+ real open-source projects with SBOM configs. Recommended qsdev config: `sboms: [{artifacts: archive}]` with cosign checksum signing. Full report at `goreleaser-sbom-research.md`.
- **Next**: Continue with remaining Phase 1 tasks (GitHub integration, signing/attestation, vulnerability scanning).

## 2026-05-15 18:00 — GitHub Integration Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GitHub Dependency Submission API docs](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/using-the-dependency-submission-api) → `docs/github-dependency-submission-api-docs.md`
  - [GitHub SBOM REST API endpoints](https://docs.github.com/en/rest/dependency-graph/sboms) → `docs/github-sbom-rest-api-endpoints.md`
  - [GitHub Artifact Attestations docs](https://docs.github.com/en/actions/concepts/security/artifact-attestations) → `docs/github-artifact-attestations-docs.md`
  - [Dependabot-based dependency graphs for Go (Dec 2025)](https://github.blog/changelog/2025-12-09-dependabot-dgs-for-go/) → `docs/github-dependabot-go-dependency-graphs.md`
  - [SLSA GitHub Generator README](https://github.com/slsa-framework/slsa-github-generator/blob/main/README.md) → `docs/slsa-github-generator-readme.md`
  - [actions/attest-build-provenance](https://github.com/actions/attest-build-provenance) → `docs/github-attest-build-provenance-action.md`
  - [actions/attest-sbom](https://github.com/actions/attest-sbom) → `docs/github-attest-sbom-action.md`
  - [actions/attest (canonical action)](https://github.com/actions/attest) → `docs/github-actions-attest-action.md`
  - [GitHub blog: SLSA 3 compliance for Go](https://github.blog/security/supply-chain-security/slsa-3-compliance-with-github-actions/) → `docs/github-blog-slsa3-go-compliance.md`
  - [Anchore SBOM Action](https://github.com/anchore/sbom-action) → `docs/anchore-sbom-action-readme.md`
  - [GitHub SBOM export docs](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/exporting-a-software-bill-of-materials-for-your-repository) → `docs/github-sbom-export-docs.md`
  - [Using artifact attestations guide](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations) → `docs/github-using-artifact-attestations-guide.md`
  - [GoReleaser example-supply-chain repo](https://github.com/goreleaser/goreleaser-example-supply-chain) → `docs/goreleaser-example-supply-chain.md`
  - [GoReleaser example-supply-chain .goreleaser.yaml](https://raw.githubusercontent.com/goreleaser/goreleaser-example-supply-chain/main/.goreleaser.yaml) → `docs/goreleaser-example-supply-chain-config.md`
  - [GoReleaser example-supply-chain workflow](https://raw.githubusercontent.com/goreleaser/goreleaser-example-supply-chain/main/.github/workflows/release.yml) → `docs/goreleaser-example-supply-chain-workflow.md`
  - [GoReleaser SBOM customization](https://goreleaser.com/customization/sbom/) → `docs/goreleaser-sbom-customization-docs.md`
  - [GoReleaser attestations docs](https://goreleaser.com/customization/publish/attestations/) → `docs/goreleaser-attestations-docs.md`
  - [GitHub attestation REST API](https://docs.github.com/en/rest/repos/attestations) → `docs/github-attestation-rest-api.md`
  - [actions/go-dependency-submission](https://github.com/actions/go-dependency-submission/blob/main/README.md) → `docs/go-dependency-submission-action-readme.md`
  - [SLSA Go builder docs](https://raw.githubusercontent.com/slsa-framework/slsa-github-generator/main/internal/builders/go/README.md) → `docs/slsa-go-builder-readme.md`
  - [Offline attestation verification](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/verifying-attestations-offline) → `docs/github-offline-attestation-verification.md`
- **Summary**: Deep investigation of GitHub's full supply chain security stack for Go projects. Key findings: (1) GitHub now uses Dependabot-powered dynamic resolution for Go dependency graphs (Dec 2025), providing accurate transitive deps without Actions minutes. (2) `actions/attest@v4` is the canonical attestation action — both `attest-build-provenance` and `attest-sbom` are deprecated wrappers. (3) GitHub attestations achieve SLSA L2 in normal workflows, L3 via reusable workflows. (4) `slsa-github-generator` provides stronger L3 isolation but is incompatible with GoReleaser as a build system. (5) GoReleaser integrates with attestations via `subject-checksums` pointing to `checksums.txt`. (6) Attestations stored in GitHub's API with Sigstore transparency log for public repos. (7) Private repo attestations require GitHub Enterprise Cloud. Full report at `github-integration-research.md`.
- **Next**: Continue with signing/attestation and vulnerability scanning tasks.

## 2026-05-15 19:30 — Vulnerability Scanning from SBOMs Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Grype GitHub README](https://github.com/anchore/grype) → `docs/grype-github-readme.md`
  - [Grype CycloneDX/SPDX support](https://anchore.com/sbom/grype-support-cyclonedx-spdx/) → `docs/grype-cyclonedx-spdx-support.md`
  - [Grype filtering & VEX](https://oss.anchore.com/docs/guides/vulnerability/filter-results/) → `docs/grype-filter-results.md`
  - [Grype OpenVEX (Chainguard)](https://www.chainguard.dev/unchained/vexed-then-grype-about-it-chainguard-and-anchore-announce-grype-supports-openvex) → `docs/grype-openvex-chainguard.md`
  - [Trivy SBOM scanning](https://trivy.dev/docs/latest/target/sbom/) → `docs/trivy-sbom-scanning.md`
  - [Trivy Go language support](https://trivy.dev/docs/latest/guide/coverage/language/golang/) → `docs/trivy-go-language-support.md`
  - [Trivy VEX local files](https://trivy.dev/docs/latest/supply-chain/vex/file/) → `docs/trivy-vex-local-files.md`
  - [OSV-Scanner GitHub](https://github.com/google/osv-scanner) → `docs/osv-scanner-github-readme.md`
  - [Go vulnerability management](https://go.dev/doc/security/vuln/) → `docs/go-vulnerability-management.md`
  - [Govulncheck reference](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) → `docs/govulncheck-reference.md`
  - [Govulncheck OpenVEX output](https://pkg.go.dev/golang.org/x/vuln/internal/openvex) → `docs/govulncheck-openvex-output.md`
  - [CycloneDX VEX capabilities](https://cyclonedx.org/capabilities/vex/) → `docs/cyclonedx-vex-capabilities.md`
  - [OpenVEX spec](https://github.com/openvex/spec) → `docs/openvex-spec.md`
  - [VEX comprehensive overview](https://www.aquasec.com/cloud-native-academy/vulnerability-management/vulnerability-exploitability-exchange/) → `docs/vex-comprehensive-overview.md`
  - [Dependency-Track overview](https://dependencytrack.org/) → `docs/dependency-track-overview.md`
  - [Dependency-Track detailed review](https://appsecsanta.com/dependency-track) → `docs/dependency-track-detailed-review.md`
  - [Vulnerability database reconciliation (15 DBs)](https://dev.to/benzsevern/reconciling-15-oss-vulnerability-databases-what-they-actually-cover-19fl) → `docs/vulnerability-database-reconciliation.md`
  - [Vulnerability database comparison (GitGuardian)](https://blog.gitguardian.com/open-source-vulnerability-databases-comparison/) → `docs/vulnerability-database-comparison-gitguardian.md`
  - [SBOM false positive rate study (arXiv)](https://arxiv.org/html/2511.20313v1) → `docs/sbom-vuln-management-false-positives-study.md`
  - Enterprise SBOM consumer workflow (multiple sources) → `docs/enterprise-sbom-consumer-workflow.md`
- **Summary**: Comprehensive investigation of SBOM-based vulnerability scanning and VEX integration for Go projects. Key findings: (1) Three major scanners (Grype, Trivy, OSV-Scanner) consume CycloneDX/SPDX SBOMs. (2) SBOM-based scanning alone has a 97.5% false positive rate due to package-level matching without reachability. (3) Govulncheck's `-format openvex` output is the critical bridge — generates VEX with reachability-based suppressions that both Grype and Trivy consume natively. (4) OpenVEX is the recommended VEX format: govulncheck produces it, both major scanners consume it, and it's SBOM-format-agnostic. (5) Go vulndb uniquely includes symbol-level vulnerability info. (6) Dependency-Track is the enterprise standard for continuous SBOM monitoring. (7) Trivy has degraded accuracy for third-party SBOMs; Grype is better for this use case. (8) Recommended: ship CycloneDX SBOM + OpenVEX document as release artifacts.
- **Next**: Remaining Phase 1 tasks: signing/attestation.

## 2026-05-15 20:30 — Signing & Attestation Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Sigstore cosign signing overview](https://docs.sigstore.dev/cosign/signing/overview/) → `docs/cosign-signing-overview-sigstore.md`
  - [Sigstore keyless verification deep dive](https://dev.to/kanywst/sigstore-deep-dive-unmasking-the-magic-behind-keyless-verification-lmh) → `docs/sigstore-keyless-verification-deep-dive.md`
  - [Cosign GitHub README](https://github.com/sigstore/cosign) → `docs/cosign-readme-github.md`
  - [GoReleaser signing configuration](https://goreleaser.com/customization/sign/sign/) → `docs/goreleaser-signing-configuration.md`
  - [GoReleaser supply chain security blog](https://goreleaser.com/blog/supply-chain-security/) → `docs/goreleaser-supply-chain-security-blog-signing.md`
  - [SLSA 3 compliance for Go (GitHub blog)](https://github.blog/security/supply-chain-security/slsa-3-compliance-with-github-actions/) → `docs/slsa-3-compliance-github-actions-go.md`
  - [SLSA GitHub Generator](https://github.com/slsa-framework/slsa-github-generator) → `docs/slsa-github-generator-go-builder.md`
  - [SLSA provenance hands-on tutorial](https://dev.to/kanywst/slsa-provenance-hands-on-generate-with-github-actions-verify-with-slsa-verifier-56ka) → `docs/slsa-provenance-hands-on-github-actions.md`
  - [SLSA levels spec v1.1](https://slsa.dev/spec/v1.1/levels) → `docs/slsa-levels-spec-v1.1.md`
  - [SLSA FAQ spec v1.1](https://slsa.dev/spec/v1.1/faq) → `docs/slsa-faq-spec-v1.1.md`
  - [in-toto attestation framework](https://github.com/in-toto/attestation) → `docs/in-toto-attestation-framework-readme.md`
  - [GitHub Actions attest-sbom](https://github.com/actions/attest-sbom) → `docs/github-actions-attest-sbom.md`
  - [GitHub artifact attestations guide](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds) → `docs/github-artifact-attestations-provenance.md`
  - [Rekor transparency log overview](https://docs.sigstore.dev/logging/overview/) → `docs/rekor-transparency-log-overview.md`
  - [Syft + Sigstore SBOM attestations](https://anchore.com/sbom/creating-sbom-attestations-using-syft-and-sigstore/) → `docs/syft-sigstore-sbom-attestations.md`
  - [s3cme template Go supply chain repo](https://github.com/mchmarny/s3cme) → `docs/s3cme-template-go-supply-chain.md`
  - [GoReleaser + GitLab + cosign + SBOM](https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/) → `docs/goreleaser-gitlab-cosign-sbom.md`
- **Summary**: Comprehensive investigation of SBOM signing, attestation, and cryptographic verification for Go binaries. Key findings: (1) Cosign keyless signing via GitHub Actions OIDC is the recommended approach — eliminates all key management, Fulcio issues 10-minute certs bound to workflow identity, Rekor provides non-repudiation via Merkle-tree-based inclusion proofs. (2) in-toto attestation framework wraps typed predicates (SBOM, SLSA provenance) in DSSE envelopes, binding metadata to artifact digests — fundamentally different from simple signing which only proves identity. (3) SLSA L3 achievable via slsa-github-generator's Go builder but incompatible with GoReleaser as a build system; GoReleaser + cosign achieves L2 with strong practical security. (4) Three consumer verification paths: `cosign verify-blob` (Sigstore-native), `gh attestation verify` (simplest), `slsa-verifier` (SLSA provenance). (5) Both artifact signing AND SBOM signing are needed — they answer different questions (who built it vs what's inside). (6) GoReleaser's `signs:` block with cosign v3 `--bundle` flag produces single `.sigstore.json` files. (7) Real-world examples: goreleaser/example-supply-chain, mchmarny/s3cme, slsa-verifier itself. Full report at `signing-attestation-research.md`.
- **Next**: All Phase 1 tasks complete. Ready for Phase 2 synthesis.

## 2026-05-15 21:00 — Phase 3 Synthesis Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all 7 research reports into actionable conclusions in research.md. Reconciled the SPDX-vs-CycloneDX primary format tension: formats research favored SPDX (GoReleaser default, GitHub native), distribution research favored CycloneDX (VEX support, ecosystem adoption). Resolution: ship both (trivial with GoReleaser), CycloneDX as the actionable format for vulnerability workflows, SPDX for compliance. Wrote full pipeline architecture, key decisions table, and consumer verification paths. All depth checklist items satisfied across all 7 topic reports. 117 source documents saved to docs/. Spike ready for completion.
- **Next**: Run /complete-spike or proceed to implementation.

## 2026-05-15 21:30 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. 7 topic reports (all passing depth checklist 42/42), 117 source documents, 9 completed tasks across 2 phases. Conclusions synthesize a complete SBOM pipeline for qsdev: Syft via GoReleaser generating both SPDX 2.3 and CycloneDX 1.5, cosign keyless signing with SLSA L2, govulncheck OpenVEX for false-positive suppression, layered distribution across GitHub Releases / go install / Homebrew / Nix. Cross-report format tension (SPDX vs CycloneDX primary) resolved: ship both, CycloneDX for VEX workflows, SPDX for compliance. Trivy excluded due to March 2026 supply chain compromise. 1 follow-on candidate flushed to proposed-spikes.md (nix-sbom-provenance-composition). 3 remaining open questions are implementation decisions, not research gaps.
