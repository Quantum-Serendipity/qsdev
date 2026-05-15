<!-- Source: https://github.com/CycloneDX/cyclonedx-gomod -->
<!-- Retrieved: 2026-05-15 -->

# cyclonedx-gomod: SBOM Generation Tool for Go Modules

## Repository Metrics
- **GitHub Stars:** 183
- **Latest Release:** v1.10.0 (January 31, 2026)
- **Language:** Go (99.3%)
- **License:** Apache 2.0

## Core Functionality

cyclonedx-gomod generates Software Bill of Materials (SBOMs) in CycloneDX format from Go module dependencies. The tool creates comprehensive component inventories that document software composition for security and compliance purposes.

## Three Primary Subcommands

**`app`** – Generates SBOMs for compiled applications, including only dependencies actually used in the binary. Supports build constraint configuration via environment variables (GOARCH, GOOS, CGO_ENABLED, GOFLAGS) to reflect specific build targets.

**`mod`** – Produces SBOMs for Go modules themselves, capturing the aggregate dependency graph. Optionally incorporates test dependencies. Suited for library distribution and inventory tracking rather than binary-specific analysis.

**`bin`** – Analyzes compiled binaries to extract embedded module information, enabling SBOM generation without source code access.

## Key Features

- **Output Formats:** XML and JSON, with support for CycloneDX specification versions 1.0 through 1.6
- **License Detection:** Optional automated license identification with results reported as evidence rather than assertions
- **Package-Level Analysis:** Can include individual packages and their constituent files within modules
- **Build Constraint Support:** Respects Go's environment variables to generate architecture and platform-specific SBOMs

## Installation Methods

- Prebuilt binaries via GitHub releases
- Homebrew: `brew install cyclonedx/cyclonedx/cyclonedx-gomod`
- Source installation (requires Go 1.25+): `go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest`

## Known Limitations

**Vendoring:** Projects using Go's vendor directory cannot generate accurate component hashes, and license detection may fail due to incomplete file copying.

**Version Detection:** Only Git repositories receive automated version detection; other VCS systems require manual version specification.

**Pseudo Versions:** Limited repository clone depth may prevent accurate pseudo-version generation if previous version history is unavailable.
