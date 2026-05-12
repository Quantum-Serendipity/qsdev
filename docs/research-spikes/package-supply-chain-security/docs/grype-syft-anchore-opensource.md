<!-- Source: https://anchore.com/opensource/ -->
<!-- Source: https://github.com/anchore/grype -->
<!-- Source: https://github.com/anchore/syft -->
<!-- Retrieved: 2026-05-12 -->

# Syft & Grype: Open Source Container Security Tools (Anchore)

## How They Work Together

Syft and Grype are complementary tools in Anchore's open source security suite. Syft generates Software Bills of Materials (SBOMs), while Grype performs vulnerability scanning. Combining these tools enables faster scans by leveraging Syft's SBOM output as input for Grype's analysis.

## Syft: SBOM Generation Capabilities

**Primary Function:** CLI tool for generating a Software Bill of Materials (SBOM) from container images and filesystems.

**Key Capabilities:**
- Automatic SBOM generation within CI/CD pipelines
- Discovery of direct and transitive dependencies
- File-level visibility into container contents
- Multiple output formats: JSON, SPDX, and CycloneDX

**Ecosystems Supported:** Alpine (apk), Debian (dpkg), RPM, Go, Python, Java, JavaScript, Ruby, Rust, PHP, .NET, and many more. Both OS-level and language-specific packages.

## Grype: Vulnerability Scanning

**Core Functionality:** Vulnerability scanning tool for container images and filesystems that generates a list of known vulnerabilities from an SBOM, container image, or project directory.

**Scanning Features:**
- Detects OS and language-specific packages
- Provides optimized results across vulnerability sources
- Cross-references against comprehensive vulnerability database aggregating NVD, GitHub, and distribution-specific feeds (Red Hat, Debian, Ubuntu, etc.)
- Integrates with CI/CD automation

## CI/CD Integration

Both tools support pipeline automation with integrations for:
- GitHub Actions
- GitLab CI
- Azure DevOps
- Jenkins
- CircleCI
- Bitbucket

## Open Source vs. Enterprise

Both Syft and Grype are fully open source (Apache 2.0 license).

**Anchore Enterprise** builds upon these foundations, adding:
- Continuous compliance and security solutions
- Multi-team and multi-toolchain pipeline management
- Policy controls and visibility for security teams
- Enterprise-grade compliance enforcement

## Latest Versions
- Syft v1.2+ with enhanced heuristics for symbol-table scanning in binary-only environments
- Both tools available on GitHub under Anchore's repositories
