<!-- Source: https://raw.githubusercontent.com/OWASP/Amass/master/.goreleaser.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned a summary rather than raw content. Key details extracted. -->

# OWASP Amass .goreleaser.yaml Configuration

## Key Components

**Build Configuration**: Compiles AMASS for Windows, Linux, macOS, and FreeBSD across multiple architectures (amd64, 386, ARM, ARM64), with specific exclusions for unsupported platform combinations.

**Packaging**: Creates ZIP archives containing the binary, license, README, and example configuration files with a custom naming template that converts "darwin" to "macos" and "386" to "i386".

**SBOM Generation**: Uses **cyclonedx-gomod** (not syft) to create software bill-of-materials documents for each build artifact with environment variables set for the target OS and architecture.

The SBOM configuration likely looks something like:
```yaml
sboms:
  - documents:
      - "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.bom.json"
    artifacts: binary
    cmd: cyclonedx-gomod
    args: ["app", "-licenses", "-json", "-output", "$document", "../"]
    env:
      - GOARCH={{ .Arch }}
      - GOOS={{ .Os }}
```

**Distribution**: Generates checksums, manages GitHub releases under owasp-amass/amass, and publishes to a Homebrew tap repository.

**Changelog**: Filters Git history to exclude merge commits and tag references.

## Key Insight
OWASP Amass is a notable example of a project using cyclonedx-gomod instead of syft for SBOM generation. This demonstrates GoReleaser's SBOM tool-agnostic design -- any command that can generate an SBOM file can be used via the `cmd` field. The cyclonedx-gomod approach produces CycloneDX format SBOMs with license information included.
