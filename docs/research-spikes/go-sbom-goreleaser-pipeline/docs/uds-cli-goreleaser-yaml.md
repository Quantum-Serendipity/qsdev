<!-- Source: https://raw.githubusercontent.com/defenseunicorns/uds-cli/main/.goreleaser.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned a summary rather than raw content. Key details extracted. -->

# Defense Unicorns UDS CLI .goreleaser.yaml Configuration

## Key Sections

**Build Configuration:**
- Universal macOS binary support
- Linux and Darwin (macOS) targets
- AMD64 and ARM64 architectures
- CGO disabled for cross-platform compatibility

**SBOM Configuration:**
- Generates SBOMs per BINARY (not archive) using custom naming:
  `"sbom_{{ .ProjectName }}_{{ .Tag }}_{{- title .Os }}_{{ .Arch }}.sbom"`
- This is a more detailed SBOM config than most projects use

**Distribution:**
- Homebrew tap integration with automated PR creation
- GitHub releases with auto-prerelease detection
- Two formula variants (generic and versioned)

**Notable Settings:**
- Version templating handles the "v" prefix for Homebrew compatibility
- Git tag sorting by creator date
- Ignores "nightly-unstable" tags

## Key Insight
UDS CLI generates SBOMs per binary rather than per archive, and uses custom document naming. This is relevant for projects that want binary-level SBOM granularity. No signing is configured in this file.
