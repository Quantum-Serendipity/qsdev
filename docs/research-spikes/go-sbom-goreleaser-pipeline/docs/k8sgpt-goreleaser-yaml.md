<!-- Source: https://raw.githubusercontent.com/k8sgpt-ai/k8sgpt/main/.goreleaser.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned a summary rather than raw content. Key details extracted. -->

# k8sgpt .goreleaser.yaml Configuration

## Key Sections

- **before**: Executes `go mod tidy` and `go generate ./...`
- **builds**: Compiles for Linux, Windows, and Darwin with CGO disabled
- **nfpms**: Generates DEB, RPM, and APK packages
- **sboms**: Creates SBOMs for archive artifacts (simple config, defaults to syft)
- **archives**: Produces tar.gz (and zip for Windows)
- **brews**: Publishes to Homebrew repository
- **checksum**: Generates checksums.txt
- **snapshot**: Names development versions
- **announce**: Posts to Slack channel #general with custom messaging

## Notable Observations

The file does NOT include `signs:`, `docker_signs:`, or distribution sections beyond Homebrew.
The SBOM configuration uses the simple default: `artifacts: archive`.
k8sgpt is a popular Kubernetes debugging AI tool (Apache-2.0 licensed).

## Key Insight
k8sgpt demonstrates a common pattern: SBOM generation enabled but without signing. Many projects adopt SBOMs as a first step before adding signing infrastructure.
