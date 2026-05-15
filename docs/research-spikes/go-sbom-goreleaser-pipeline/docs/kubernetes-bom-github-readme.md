<!-- Source: https://github.com/kubernetes-sigs/bom -->
<!-- Retrieved: 2026-05-15 -->

# BOM Project Analysis (Kubernetes SIG Release)

## Repository Metrics
- **GitHub Stars:** 455
- **Forks:** 65
- **Latest Release:** v0.7.1 (September 26, 2025)
- **Primary Language:** Go (99.7%)

## Project Purpose
BOM is described as "a utility that lets you create, view and transform Software Bills of Materials (SBOMs)." It generates SPDX-compliant manifests for software projects and is incubating within the Linux Foundation's Automating Compliance Tooling TAC.

## Key Capabilities

**Output Formats:**
- Tag-value format
- JSON format
- In-toto provenance attestations

**Go-Specific Features:**
- Golang dependency analysis via go.mod
- Support for filtering transient dependencies
- Built-in analysis of Go packages

**Main Subcommands:**
1. `bom generate` - Creates SPDX manifests from files, directories, container images, and archives
2. `bom document` - Visualizes and queries SBOM contents

## Notable Features
- 400+ SPDX license recognition capability
- Full .gitignore support for repository scanning
- Container image analysis with optional deep inspection using available analyzers
- Multi-source processing (combine files, images, and directories)

## Maintenance Status
With 12 releases tracked and recent activity (v0.7.1 in 2025), the project demonstrates active maintenance and ongoing development.
