<!-- Source: https://github.com/goreleaser/goreleaser/blob/main/.goreleaser.yaml -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser's Own .goreleaser.yaml Configuration

## Core Configuration
- **Version**: 2 (pro version enabled)
- **Builds**: Multi-platform support including Linux, Darwin, Windows across architectures (386, amd64, arm, arm64, loong64, ppc64, riscv64)
- **Go Modules**: Proxy enabled with size reporting

## SBOM Configuration
```yaml
sboms:
  - artifacts: archive
```

This is the simplest possible SBOM configuration -- GoReleaser itself uses the defaults:
- Default cmd: syft
- Default artifacts: archive
- Default args generate SPDX-JSON format
- One SBOM per archive artifact

## Signing & Attestation
The configuration includes multiple security features:

**Code Signing**: Uses Cosign for cryptographic verification with `signature: ${artifact}.sigstore.json`

**Docker Image Signing**: Implements container image attestation via `docker_signs` with Cosign

## Distribution Channels
Packages are published to:
- Docker registries (DockerHub, GitHub Container Registry)
- Homebrew, Nix, Scoop, Winget package managers
- AUR (Arch Linux)
- NPM registry
- Linux distributions (APK, DEB, RPM, Archlinux)
- Snapcraft and Flatpak

## Release Announcements
Configured to post release notifications across:
- Mastodon at fosstodon.org
- Discord
- Telegram (@goreleasernews)
- OpenCollective community updates

## Notable Features
The configuration enforces semantic versioning, generates man pages and shell completions, includes checksums, and maintains detailed changelog formatting with contributor attribution.

## Key Insight
GoReleaser itself (a major Go project) uses the minimal SBOM config (`artifacts: archive`), suggesting this is the recommended starting point. The SBOM files are included in the GitHub Release alongside other assets, covered by the checksums.txt file, and that checksums file is signed with Cosign.
