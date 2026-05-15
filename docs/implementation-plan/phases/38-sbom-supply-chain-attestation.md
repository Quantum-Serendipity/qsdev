# Phase 38: SBOM Generation & Supply Chain Attestation

## Goal

Enhance the Phase 10 distribution pipeline with SBOM generation, cryptographic signing, vulnerability scanning, and supply chain attestation. At the end of this phase, every gdev release ships dual-format SBOMs (SPDX 2.3 + CycloneDX 1.5), cosign-signed checksums and SBOMs, govulncheck-generated OpenVEX documents, and GitHub attestations achieving SLSA Build Level 2. Consumers can verify provenance, inspect the full dependency tree, and scan for vulnerabilities with reachability-based false-positive suppression.

## Dependencies

Phase 10 complete (GoReleaser pipeline, GitHub Actions release workflow, install scripts, self-update, and shell completions all exist and functional).

## Phase Outputs

- GoReleaser `sboms:` configuration generating dual-format SBOMs per release archive
- GoReleaser `signs:` configuration with cosign keyless signing of checksums and SBOMs
- GitHub Actions workflow additions for Syft, cosign, govulncheck, and `actions/attest@v4`
- govulncheck CI integration with OpenVEX output and release-blocking policy
- OpenVEX documents shipped as release artifacts alongside SBOMs
- `scripts/verify-release.sh` consumer verification script
- `gdev version --sbom` command exposing embedded SBOM data
- SBOM signature verification integrated into `gdev self-update`
- Nix flake with SRI hashes pinned to signed GitHub Release artifacts

---

### Unit 38.1: Syft SBOM Generation in GoReleaser

**Description:** Add SBOM generation to the existing `.goreleaser.yaml` configuration, producing both SPDX 2.3 JSON and CycloneDX 1.5 JSON per release archive via Syft.

**Context:** GoReleaser provides first-class SBOM support through its `sboms:` configuration block (free/OSS since v1.2.0). The default generator is Syft (Anchore), which analyzes Go binaries via `debug/buildinfo` metadata to extract the full dependency tree including versions, hashes, and build settings. Syft supports multi-format output, so a single tool produces both SPDX and CycloneDX. SBOMs are generated after archiving but before checksums, meaning checksums.txt transitively covers all SBOM files. GoReleaser uses OpenSSF naming conventions when document templates are configured. Trivy is explicitly excluded due to its March 2026 supply chain compromise.

**Desired Outcome:** `goreleaser release --snapshot --clean` produces SPDX 2.3 JSON and CycloneDX 1.5 JSON SBOM files for every platform archive, with both files included in checksums.txt and uploaded as GitHub Release assets.

**Steps:**

1. Install Syft in the GitHub Actions release workflow by adding the `anchore/sbom-action/download-syft@v0` step before the GoReleaser step:
   ```yaml
   - uses: anchore/sbom-action/download-syft@v0
   ```

2. Add dual-format `sboms:` entries to `.goreleaser.yaml`. The first uses GoReleaser's default Syft args for SPDX; the second overrides args for CycloneDX. Both use OpenSSF-compliant naming:
   ```yaml
   sboms:
     - id: spdx
       artifacts: archive
       documents:
         - "{{ .ArtifactName }}.spdx.json"
       cmd: syft
       args:
         - "$artifact"
         - "--output"
         - "spdx-json=$document"
         - "--enrich"
         - "all"
       env:
         - SYFT_FILE_METADATA_CATALOGER_ENABLED=true

     - id: cyclonedx
       artifacts: archive
       documents:
         - "{{ .ArtifactName }}.cdx.json"
       cmd: syft
       args:
         - "$artifact"
         - "--output"
         - "cyclonedx-json=$document"
         - "--enrich"
         - "all"
       env:
         - SYFT_FILE_METADATA_CATALOGER_ENABLED=true
   ```

3. Verify that both SBOM entries use unique `id` values (GoReleaser requires this for multiple `sboms:` entries; colliding IDs cause a build error).

4. Confirm the GoReleaser pipeline order is preserved: Build -> Archive -> SBOM -> Checksum -> Sign -> Release. This ensures checksums.txt includes SBOM file hashes, and signing the checksum file transitively covers all SBOMs.

5. Run `goreleaser release --snapshot --clean` locally and verify:
   - Two SBOM files exist per archive (e.g., `gdev_0.1.0_Linux_x86_64.tar.gz.spdx.json` and `gdev_0.1.0_Linux_x86_64.tar.gz.cdx.json`)
   - Both SBOM files appear in checksums.txt
   - SPDX file validates as SPDX 2.3 JSON (check `spdxVersion` field is `"SPDX-2.3"`)
   - CycloneDX file validates as CycloneDX 1.5 JSON (check `specVersion` field is `"1.5"`)
   - Both SBOMs contain the Go module dependency graph (module paths, versions, hashes)

6. Validate SBOM content depth by inspecting a generated SBOM:
   - Go compiler version is present
   - All direct dependencies from `go.mod` appear with correct versions
   - Build settings (GOOS, GOARCH, CGO_ENABLED) are captured
   - Package URLs (PURLs) use the `pkg:golang/` scheme

**Acceptance Criteria:**
- [ ] `goreleaser release --snapshot --clean` produces `.spdx.json` and `.cdx.json` files for every platform archive
- [ ] SPDX files conform to SPDX 2.3 JSON schema
- [ ] CycloneDX files conform to CycloneDX 1.5 JSON schema
- [ ] Both SBOM formats list all Go module dependencies with versions and hashes
- [ ] Both SBOM files appear as entries in checksums.txt
- [ ] SBOM files are uploaded as GitHub Release assets alongside binaries
- [ ] No GoReleaser Pro features are required (all `sboms:` config uses OSS-only artifact types)

**Research Citations:**
- `goreleaser-sbom-research.md` -- sections 1 (Configuration Reference), 2 (Supported Generators), 3 (How SBOMs Attach to GitHub Releases), 4 (GoReleaser Pro vs OSS), 9 (Real-World Examples), 11 (Recommended Configuration for qsdev)
- `generation-tools-research.md` -- section on Syft architecture, Go-specific strengths, CLI usage; section on Trivy disqualification
- `sbom-formats-research.md` -- sections 1-2 (SPDX vs CycloneDX), 5 (Shipping Both), 6 (Practical Recommendation)
- `distribution-research.md` -- section 1 (GitHub Release Assets), naming conventions

**Status:** Not Started

---

### Unit 38.2: Cosign Keyless Signing & SLSA Attestation

**Description:** Integrate cosign keyless signing and GitHub artifact attestations into the release workflow, signing checksums and SBOMs via Sigstore OIDC and generating SLSA Build Level 2 provenance.

**Context:** Cosign keyless signing uses GitHub Actions' OIDC identity provider to obtain short-lived (10-minute) X.509 certificates from Fulcio, eliminating all key management. Signatures are recorded in Rekor's append-only transparency log for non-repudiation. GoReleaser's `signs:` block with cosign v3's `--bundle` flag produces single `.sigstore.json` files containing signature, Fulcio certificate, and Rekor entry. The `actions/attest@v4` action (the canonical GitHub attestation action, replacing deprecated `attest-build-provenance` and `attest-sbom` wrappers) creates in-toto format attestations that achieve SLSA Build Level 2 in normal workflows. Level 3 is achievable via reusable workflows but is incompatible with GoReleaser as the build system -- Level 2 provides strong practical security for gdev's threat model.

**Desired Outcome:** Every release has cosign-signed checksums and SBOMs (`.sigstore.json` bundles), plus GitHub attestations for build provenance. Consumers can verify via `cosign verify-blob`, `gh attestation verify`, or manual SHA256 checksum comparison.

**Steps:**

1. Add cosign installation to the GitHub Actions release workflow:
   ```yaml
   - uses: sigstore/cosign-installer@v3
   ```

2. Add the required GitHub Actions permissions to the release workflow:
   ```yaml
   permissions:
     contents: write       # Upload release assets
     id-token: write       # OIDC token for Fulcio certificates
     attestations: write   # Upload GitHub attestations
     packages: write       # Container registry (if applicable)
   ```

3. Add cosign signing entries to `.goreleaser.yaml` for both checksums and SBOMs:
   ```yaml
   signs:
     - id: cosign-checksums
       cmd: cosign
       signature: "${artifact}.sigstore.json"
       args:
         - "sign-blob"
         - "--bundle=${signature}"
         - "${artifact}"
         - "--yes"
       artifacts: checksum
       output: true

     - id: cosign-sboms
       cmd: cosign
       signature: "${artifact}.sigstore.json"
       args:
         - "sign-blob"
         - "--bundle=${signature}"
         - "${artifact}"
         - "--yes"
       artifacts: sbom
       output: true
   ```

4. Ensure the existing `.goreleaser.yaml` checksum configuration uses a stable filename for attestation compatibility:
   ```yaml
   checksum:
     name_template: "checksums.txt"
   ```

5. Add build provenance attestation after the GoReleaser step using `actions/attest@v4`:
   ```yaml
   - uses: actions/attest@v4
     with:
       subject-checksums: ./dist/checksums.txt
   ```

6. Add SBOM attestation to bind SBOMs to their corresponding artifacts:
   ```yaml
   - name: Attest SBOMs
     run: |
       for sbom in ./dist/*.spdx.json ./dist/*.cdx.json; do
         if [ -f "$sbom" ]; then
           artifact_name="${sbom%.spdx.json}"
           artifact_name="${artifact_name%.cdx.json}"
           if [ -f "$artifact_name" ]; then
             gh attestation attest "$artifact_name" \
               --sbom "$sbom" \
               -R ${{ github.repository }}
           fi
         fi
       done
     env:
       GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
   ```

7. Verify the complete release artifact set includes:
   ```
   gdev_1.0.0_Linux_x86_64.tar.gz
   gdev_1.0.0_Linux_x86_64.tar.gz.spdx.json
   gdev_1.0.0_Linux_x86_64.tar.gz.cdx.json
   gdev_1.0.0_Linux_x86_64.tar.gz.spdx.json.sigstore.json
   gdev_1.0.0_Linux_x86_64.tar.gz.cdx.json.sigstore.json
   checksums.txt
   checksums.txt.sigstore.json
   ```

8. Test consumer verification paths locally:
   ```bash
   # Path 1: Cosign verification
   cosign verify-blob \
     --bundle checksums.txt.sigstore.json \
     --certificate-identity "https://github.com/Quantum-Serendipity/gdev/.github/workflows/release.yml@refs/tags/v1.0.0" \
     --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
     checksums.txt

   # Path 2: GitHub attestation verification
   gh attestation verify ./gdev-linux-amd64 -R Quantum-Serendipity/gdev

   # Path 3: Manual checksum verification
   sha256sum -c checksums.txt --ignore-missing
   ```

**Acceptance Criteria:**
- [ ] Every release produces `.sigstore.json` bundles for checksums.txt and all SBOM files
- [ ] `cosign verify-blob` succeeds for checksums.txt using the workflow's certificate identity
- [ ] `cosign verify-blob` succeeds for each SBOM file
- [ ] `gh attestation verify` succeeds for release binaries (build provenance)
- [ ] SBOM attestations are visible in the GitHub attestation API
- [ ] No long-lived signing keys exist -- signing uses only OIDC-based ephemeral certificates
- [ ] Rekor transparency log entries are created for each signing event (public repo)
- [ ] Release workflow requires only `GITHUB_TOKEN` -- no additional secrets for signing

**Research Citations:**
- `signing-attestation-research.md` -- sections 1 (Sigstore Ecosystem), 2 (Keyless Signing), 4 (in-toto Attestation Framework), 5 (SLSA Provenance), 6 (SBOM-Specific Signing), 8 (GoReleaser Signing Integration), 11 (Recommended Architecture for qsdev)
- `github-integration-research.md` -- sections 4 (GitHub Artifact Attestations), 5 (SLSA Provenance via GitHub Actions), 6 (GitHub Actions Workflow Patterns)
- `goreleaser-sbom-research.md` -- sections 7 (Signing Integration), 11 (Recommended Configuration)

**Status:** Not Started

---

### Unit 38.3: govulncheck & OpenVEX Integration

**Description:** Integrate govulncheck into the CI pipeline to perform reachability-based vulnerability analysis, generate OpenVEX documents for false-positive suppression, and enforce a release-blocking policy on reachable vulnerabilities.

**Context:** SBOM-based vulnerability scanning produces a 97.5% false positive rate because it operates at the package level without call-graph analysis. govulncheck (Go Security Team) solves this for Go projects by tracing function call chains to determine whether vulnerable functions are actually reachable. Its `-format openvex` flag produces OpenVEX documents that categorize each vulnerability as `affected` (reachable) or `not_affected` with justification `vulnerable_code_not_in_execute_path` (unreachable). Both Grype and Trivy natively consume OpenVEX to suppress unreachable vulnerabilities. The Go vulnerability database (vuln.go.dev) is uniquely valuable because it includes symbol-level information enabling this reachability analysis. govulncheck's exit code is non-zero when reachable vulnerabilities are found, making it a natural CI gate.

**Desired Outcome:** Every PR and main-branch push runs govulncheck as a CI check. Every release includes an OpenVEX document. Releases are blocked if govulncheck detects reachable vulnerabilities.

**Steps:**

1. Add govulncheck to the CI workflow (`.github/workflows/ci.yml`) as a check on every PR and push to main:
   ```yaml
   - name: Install govulncheck
     run: go install golang.org/x/vuln/cmd/govulncheck@latest

   - name: Run vulnerability check
     run: govulncheck ./...
   ```

2. Add OpenVEX generation to the release workflow (`.github/workflows/release.yml`), running before GoReleaser so the VEX document is available as a release artifact:
   ```yaml
   - name: Install govulncheck
     run: go install golang.org/x/vuln/cmd/govulncheck@latest

   - name: Generate OpenVEX document
     run: |
       govulncheck -format openvex ./... > dist/gdev.vex.json || true
       # Also run in blocking mode to fail on reachable vulns
       govulncheck ./...
   ```

   The two-step approach is deliberate: the first invocation with `-format openvex` always produces a VEX document (even when reachable vulns exist, documenting all findings). The second invocation without flags uses govulncheck's default exit-code behavior to block the release if reachable vulnerabilities are found.

3. Configure the VEX output filename to include the version for release artifact naming:
   ```yaml
   - name: Generate OpenVEX document
     run: |
       VERSION="${GITHUB_REF_NAME#v}"
       govulncheck -format openvex ./... > "dist/gdev_${VERSION}.vex.json" || true
       govulncheck ./...
   ```

4. Ensure the VEX document is included as a GitHub Release asset. Add it via GoReleaser's `extra_files:` configuration or as an additional upload step after GoReleaser:
   ```yaml
   - name: Upload VEX document
     if: startsWith(github.ref, 'refs/tags/v')
     run: |
       VERSION="${GITHUB_REF_NAME#v}"
       gh release upload "${GITHUB_REF_NAME}" \
         "dist/gdev_${VERSION}.vex.json" \
         --clobber
     env:
       GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
   ```

5. Document the consumer scanning workflow in release notes and README:
   ```bash
   # Download SBOM and VEX from a release
   gh release download v1.0.0 --pattern '*.cdx.json' --pattern '*.vex.json'

   # Scan with reachability-aware false-positive suppression
   grype sbom:./gdev_1.0.0_Linux_x86_64.tar.gz.cdx.json \
     --vex ./gdev_1.0.0.vex.json

   # CI gate: fail on high+ severity reachable vulnerabilities
   grype sbom:./gdev_1.0.0_Linux_x86_64.tar.gz.cdx.json \
     --vex ./gdev_1.0.0.vex.json \
     --fail-on high --only-fixed
   ```

6. Add a govulncheck binary analysis step to the release workflow for validation of the compiled artifacts (complementing the source-mode analysis):
   ```yaml
   - name: Validate binary (govulncheck binary mode)
     run: |
       for bin in dist/gdev_linux_amd64*/gdev; do
         govulncheck -mode binary "$bin" || true
       done
   ```

**Acceptance Criteria:**
- [ ] govulncheck runs on every PR and push to main in the CI workflow
- [ ] CI fails (non-zero exit) when reachable vulnerabilities are detected
- [ ] Every release includes a `gdev_<version>.vex.json` OpenVEX document as a release asset
- [ ] VEX document contains `not_affected` entries with `vulnerable_code_not_in_execute_path` justification for unreachable dependencies
- [ ] VEX document contains `affected` entries for any genuinely reachable vulnerabilities
- [ ] `grype sbom:./gdev.cdx.json --vex ./gdev.vex.json` suppresses unreachable vulnerability findings
- [ ] Release workflow blocks on reachable vulnerabilities (govulncheck exit code != 0)
- [ ] govulncheck binary-mode analysis runs against at least one compiled binary per release

**Research Citations:**
- `vulnerability-scanning-research.md` -- sections 2 (Govulncheck: The Go-Authoritative Complement), 3 (VEX), 6 (False Positive Management), 7 (Recommendations for qsdev)
- `generation-tools-research.md` -- section 7 (govulncheck as complementary tool)
- `research.md` -- Conclusions section on vulnerability mitigation strategy

**Status:** Not Started

---

### Unit 38.4: Verification & Consumer Toolchain

**Description:** Build consumer-facing verification tooling: a release verification script, `gdev version --sbom` for embedded SBOM access, and SBOM signature verification integrated into `gdev self-update`.

**Context:** Consumers range from basic (manual checksum) to compliance-focused (full cosign + attestation verification). The verification script automates the full verification flow. Embedding a source SBOM via `//go:embed` ensures `go install` users have SBOM access even without GitHub Release assets. Integrating SBOM verification into self-update means every binary update verifies supply chain integrity. The embedded SBOM is necessarily a source SBOM (generated pre-build from `go.mod`), complementing the build SBOM in release assets.

**Desired Outcome:** `scripts/verify-release.sh` fully verifies a downloaded release. `gdev version --sbom` outputs the embedded CycloneDX SBOM. `gdev self-update` verifies SBOM signatures before applying updates.

**Steps:**

1. Create `scripts/verify-release.sh` that automates the full verification workflow for a downloaded release:
   ```bash
   #!/usr/bin/env bash
   set -euo pipefail

   # Usage: ./verify-release.sh <version> [--skip-cosign] [--skip-attestation]
   # Example: ./verify-release.sh v1.0.0

   VERSION="${1:?Usage: verify-release.sh <version>}"
   REPO="Quantum-Serendipity/gdev"
   RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

   # Detect platform
   OS="$(uname -s)"
   ARCH="$(uname -m)"
   # ... map to GoReleaser naming ...

   # Step 1: Download artifacts
   echo "Downloading release artifacts..."
   curl -fsSLO "${RELEASE_URL}/gdev_${VERSION#v}_${OS}_${ARCH}.tar.gz"
   curl -fsSLO "${RELEASE_URL}/checksums.txt"
   curl -fsSLO "${RELEASE_URL}/checksums.txt.sigstore.json"
   curl -fsSLO "${RELEASE_URL}/gdev_${VERSION#v}_${OS}_${ARCH}.tar.gz.cdx.json"
   curl -fsSLO "${RELEASE_URL}/gdev_${VERSION#v}_${OS}_${ARCH}.tar.gz.cdx.json.sigstore.json"

   # Step 2: Verify checksum signature (cosign)
   if command -v cosign &>/dev/null && [[ "${SKIP_COSIGN:-}" != "true" ]]; then
     echo "Verifying checksum signature..."
     cosign verify-blob \
       --bundle checksums.txt.sigstore.json \
       --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
       --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
       checksums.txt
     echo "Checksum signature verified."
   fi

   # Step 3: Verify file checksum
   echo "Verifying file integrity..."
   sha256sum -c checksums.txt --ignore-missing

   # Step 4: Verify SBOM signature (cosign)
   if command -v cosign &>/dev/null && [[ "${SKIP_COSIGN:-}" != "true" ]]; then
     echo "Verifying SBOM signature..."
     cosign verify-blob \
       --bundle "gdev_${VERSION#v}_${OS}_${ARCH}.tar.gz.cdx.json.sigstore.json" \
       --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
       --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
       "gdev_${VERSION#v}_${OS}_${ARCH}.tar.gz.cdx.json"
     echo "SBOM signature verified."
   fi

   # Step 5: Verify GitHub attestation
   if command -v gh &>/dev/null && [[ "${SKIP_ATTESTATION:-}" != "true" ]]; then
     echo "Verifying GitHub build attestation..."
     gh attestation verify "gdev_${VERSION#v}_${OS}_${ARCH}.tar.gz" -R "${REPO}"
     echo "Build attestation verified."
   fi

   echo "All verification checks passed."
   ```

2. Generate a pre-build source SBOM for embedding. Add a build step (Makefile target or CI step) that generates a CycloneDX SBOM from `go.mod` before compilation:
   ```makefile
   .PHONY: generate-embedded-sbom
   generate-embedded-sbom:
   	syft dir:. -o cyclonedx-json=internal/embedded/sbom.cdx.json
   ```

3. Create `internal/embedded/sbom.go` with the embedded SBOM:
   ```go
   package embedded

   import _ "embed"

   //go:embed sbom.cdx.json
   var SBOM []byte
   ```

4. Add `--sbom` flag to the existing `gdev version` command:
   ```go
   // In cmd/gdev/version.go or equivalent
   var sbomFlag bool

   func init() {
       versionCmd.Flags().BoolVar(&sbomFlag, "sbom", false,
           "Output the embedded CycloneDX SBOM for the running binary")
   }

   func runVersion(cmd *cobra.Command, args []string) {
       if sbomFlag {
           fmt.Println(string(embedded.SBOM))
           return
       }
       // ... existing version output ...
   }
   ```

5. Integrate SBOM signature verification into `gdev self-update` (in `internal/selfupdate/`). After downloading the new binary and checksums, also download and verify the SBOM signature before applying the update:
   ```go
   func (u *Updater) verifySupplyChain(release *Release) error {
       // 1. Download checksums.txt.sigstore.json
       // 2. Verify checksum signature via cosign CLI or sigstore-go library
       // 3. Verify binary hash against checksums.txt
       // 4. Download SBOM .sigstore.json
       // 5. Verify SBOM signature
       // 6. Log verification results
       return nil
   }
   ```

   For the initial implementation, shell out to `cosign verify-blob` if cosign is available on the system. If cosign is not installed, fall back to SHA256 checksum verification only (with a warning that full supply chain verification requires cosign). A future enhancement could use the `sigstore-go` library for native verification without requiring cosign.

6. Add `--skip-verify` flag to `gdev self-update` for environments where cosign/network access to Rekor is unavailable, but log a warning when used.

**Acceptance Criteria:**
- [ ] `scripts/verify-release.sh v1.0.0` downloads, verifies checksums, verifies cosign signatures, and verifies GitHub attestations for a release
- [ ] Script degrades gracefully when cosign or gh CLI is not installed (skips those checks with a warning)
- [ ] `gdev version --sbom` outputs valid CycloneDX JSON to stdout
- [ ] Embedded SBOM contains the Go module dependency graph from the build
- [ ] `gdev self-update` verifies SHA256 checksums before applying updates
- [ ] `gdev self-update` verifies cosign signatures when cosign is available
- [ ] `gdev self-update --skip-verify` skips supply chain verification with a logged warning
- [ ] `gdev version --sbom | syft convert -` produces valid output (SBOM is machine-parseable)

**Research Citations:**
- `signing-attestation-research.md` -- section 7 (Verification Workflows for Consumers), section 11.3 (Consumer Verification Options)
- `distribution-research.md` -- section 5 (Embedded SBOMs), section 6 (`go install` Path)
- `github-integration-research.md` -- section 8 (Attestation Storage and Consumer Verification)
- `research.md` -- Conclusions section on consumer verification and distribution layers

**Status:** Not Started

---

### Unit 38.5: Nix Distribution with Integrity

**Description:** Create or update the Nix flake for gdev to pin against signed GitHub Release artifacts with SRI hashes, include SBOM metadata, and support `nix run` with verified integrity.

**Context:** Nix has its own reproducibility guarantees but does not automatically consume upstream SBOMs or signatures. Go packages in nixpkgs use `buildGoModule` which vendors dependencies via `go mod vendor` -- individual Go modules do not appear as separate Nix derivations. The upstream SBOM must be explicitly included in the Nix package output. SRI (Subresource Integrity) hashes in the Nix flake ensure that fetched source archives match expected content. The Nix SBOM ecosystem is immature compared to container-focused tooling, but `$out/share/sbom/` and `passthru` attributes provide a pragmatic approach for including upstream SBOM metadata.

**Desired Outcome:** `nix run github:Quantum-Serendipity/gdev` installs gdev with integrity verification. The Nix package includes the upstream SBOM in a standard location and exposes it via `passthru` attributes.

**Steps:**

1. Create or update `flake.nix` at the repository root with a gdev package definition using `buildGoModule`:
   ```nix
   {
     description = "gdev - secure developer environment bootstrapper";

     inputs = {
       nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
       flake-utils.url = "github:numtide/flake-utils";
     };

     outputs = { self, nixpkgs, flake-utils }:
       flake-utils.lib.eachDefaultSystem (system:
         let
           pkgs = nixpkgs.legacyPackages.${system};
         in {
           packages.default = pkgs.buildGoModule {
             pname = "gdev";
             version = "0.1.0";  # Updated per release
             src = self;
             vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
             # Updated per release with: nix-prefetch { ... }

             ldflags = [
               "-s" "-w"
               "-X main.version=${self.shortRev or "dev"}"
               "-X main.commit=${self.rev or "none"}"
             ];

             postInstall = ''
               install -Dm644 ${./internal/embedded/sbom.cdx.json} \
                 $out/share/sbom/gdev.cdx.json
             '';

             passthru = {
               sbom = ./internal/embedded/sbom.cdx.json;
             };

             meta = with pkgs.lib; {
               description = "Secure developer environment bootstrapper";
               homepage = "https://github.com/Quantum-Serendipity/gdev";
               license = licenses.mit;
               maintainers = [];
               mainProgram = "gdev";
             };
           };

           apps.default = flake-utils.lib.mkApp {
             drv = self.packages.${system}.default;
           };
         }
       );
   }
   ```

2. For release-pinned installations (fetching pre-built binaries from GitHub Releases rather than building from source), create an overlay or alternative package that fetches the release archive with SRI hash verification:
   ```nix
   packages.gdev-bin = pkgs.stdenv.mkDerivation {
     pname = "gdev-bin";
     version = "1.0.0";

     src = pkgs.fetchurl {
       url = "https://github.com/Quantum-Serendipity/gdev/releases/download/v1.0.0/gdev_1.0.0_Linux_x86_64.tar.gz";
       hash = "sha256-BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=";
     };

     sbom = pkgs.fetchurl {
       url = "https://github.com/Quantum-Serendipity/gdev/releases/download/v1.0.0/gdev_1.0.0_Linux_x86_64.tar.gz.cdx.json";
       hash = "sha256-CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=";
     };

     installPhase = ''
       install -Dm755 gdev $out/bin/gdev
       install -Dm644 ${sbom} $out/share/sbom/gdev.cdx.json
     '';

     passthru = {
       inherit sbom;
     };
   };
   ```

3. Add a CI step or Makefile target to compute and update SRI hashes after each release:
   ```bash
   nix-prefetch-url --type sha256 --unpack \
     "https://github.com/Quantum-Serendipity/gdev/releases/download/v1.0.0/gdev_1.0.0_Linux_x86_64.tar.gz"
   ```

4. Verify the flake builds and runs correctly:
   ```bash
   nix build .#default
   nix run .#default -- version
   nix run .#default -- version --sbom | head -20

   # Verify SBOM is in the output
   ls ./result/share/sbom/gdev.cdx.json
   ```

5. Add the flake to CI for build validation:
   ```yaml
   - name: Nix build check
     run: nix build .#default --no-link
   ```

6. Document the Nix installation method in the README:
   ```
   # Run directly
   nix run github:Quantum-Serendipity/gdev

   # Install to profile
   nix profile install github:Quantum-Serendipity/gdev

   # Access the upstream SBOM
   cat $(nix build github:Quantum-Serendipity/gdev --print-out-paths)/share/sbom/gdev.cdx.json
   ```

**Acceptance Criteria:**
- [ ] `nix build .#default` succeeds and produces a working gdev binary
- [ ] `nix run github:Quantum-Serendipity/gdev -- version` works from a clean environment
- [ ] The Nix package output contains `$out/share/sbom/gdev.cdx.json`
- [ ] `passthru.sbom` attribute is accessible for downstream Nix tooling
- [ ] SRI hashes in the flake are correct and reproducible
- [ ] Binary-fetching variant (`gdev-bin`) pins to signed GitHub Release artifacts
- [ ] Nix build is validated in CI

**Research Citations:**
- `distribution-research.md` -- section 4 (Nix Derivation Considerations), passthru attributes, sbomnix/bombon discussion
- `research.md` -- Conclusions section on Nix distribution channel
- `sbom-formats-research.md` -- section 6 (recommending CycloneDX as embedded format)

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### GoReleaser Pipeline Order

The existing Phase 10 `.goreleaser.yaml` already handles builds, archives, and checksums. Phase 38 inserts SBOM generation between Archive and Checksum in the pipeline, and signing between Checksum and Release. The full pipeline becomes:

```
Build -> Archive -> SBOM (Unit 38.1) -> Checksum -> Sign (Unit 38.2) -> Release
```

This ordering is not configurable -- GoReleaser enforces it automatically based on the presence of `sboms:` and `signs:` blocks. The key consequence is that checksums.txt includes SBOM file hashes, and signing the checksum file transitively covers all artifacts including SBOMs.

### Syft Installation in CI

Syft is not bundled with GoReleaser -- it must be separately installed. The `anchore/sbom-action/download-syft@v0` GitHub Action handles this. It must run before the `goreleaser/goreleaser-action` step.

### Cosign Keyless Signing in GitHub Actions

Cosign keyless signing works automatically in GitHub Actions because the runner has access to the OIDC token endpoint (`https://token.actions.githubusercontent.com`). The only requirement is `permissions.id-token: write` in the workflow. No secrets are needed beyond `GITHUB_TOKEN`. The `--yes` flag in cosign args skips interactive confirmation prompts that would block CI.

### Embedded SBOM Circular Dependency

The embedded SBOM (Unit 38.4) is a source SBOM generated from `go.mod`/`go.sum` before compilation. It cannot include the binary's own hash (chicken-and-egg problem). The release-asset SBOMs generated by Syft post-compilation are build SBOMs that accurately reflect the compiled binary. Both are valuable but serve different purposes: embedded for `go install` users, release-asset for download verification.

### VEX Document Scope

The govulncheck OpenVEX output (Unit 38.3) covers all Go module vulnerabilities in vuln.go.dev. It does not cover vulnerabilities tracked only in NVD or GHSA without Go vulndb cross-references. Consumers doing comprehensive scanning should use both the VEX document (for Go-native reachability data) and their own scanner's database (for broader coverage).

### Phase 10 Compatibility

This phase does not modify any Phase 10 acceptance criteria. The existing GoReleaser configuration, install scripts, self-update mechanism, and shell completions continue to work unchanged. Phase 38 adds new configuration blocks (`sboms:`, `signs:`), new workflow steps, and new CLI flags (`--sbom`, `--skip-verify`) without altering existing behavior.

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] `goreleaser release --snapshot --clean` produces SPDX and CycloneDX SBOMs for all platform archives
- [ ] Checksums and SBOMs are cosign-signed with keyless signatures (`.sigstore.json` bundles)
- [ ] `gh attestation verify` succeeds for release binaries against the gdev repository
- [ ] govulncheck runs in CI and blocks releases on reachable vulnerabilities
- [ ] OpenVEX document is included as a release asset
- [ ] `grype sbom:./gdev.cdx.json --vex ./gdev.vex.json` suppresses unreachable vulnerability findings
- [ ] `scripts/verify-release.sh` successfully verifies a complete release
- [ ] `gdev version --sbom` outputs valid CycloneDX JSON
- [ ] `gdev self-update` verifies checksums and cosign signatures before applying updates
- [ ] Nix flake builds, runs, and includes the upstream SBOM at `$out/share/sbom/`
- [ ] No Phase 10 acceptance criteria are broken by Phase 38 additions
- [ ] Release workflow requires no secrets beyond `GITHUB_TOKEN` for signing and attestation
