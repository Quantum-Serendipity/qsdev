<!-- Source: https://github.com/anchore/syft -->
<!-- Retrieved: 2026-05-15 -->

# Syft GitHub Repository Analysis

## Repository Metrics
- **GitHub Stars:** 8.9k
- **Latest Release:** v1.44.0 (May 1, 2026)
- **Primary Language:** Go (98.9%)

## Supported Output Formats
The tool generates SBOMs in multiple formats including:
- CycloneDX
- SPDX
- Syft JSON
- Additional formats with conversion capabilities between SBOM formats

## Supported Packaging Ecosystems
Syft detects packages from numerous ecosystems: Alpine (apk), Debian (dpkg), RPM, Go, Python, Java, JavaScript, Ruby, Rust, PHP, .NET, and numerous others (full list available in documentation).

## Go-Specific Capabilities
The documentation mentions Go as one of the supported packaging ecosystems for dependency detection. However, the page content does not detail specific capabilities regarding binary analysis, source code analysis, or module detection for Go applications.

## Key CLI Commands for Go Usage
Basic scanning examples shown:
- `syft alpine:latest` (container image analysis)
- `syft ./my-project` (directory scanning)
- `syft <image> -o cyclonedx-json` (SBOM output)

## Limitations
The provided content does not explicitly mention Go-specific limitations in Syft's scanning capabilities.
