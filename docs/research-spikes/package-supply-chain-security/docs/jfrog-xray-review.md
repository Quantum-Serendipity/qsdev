# JFrog Xray Review 2026: Binary-Level SCA in Artifactory

- **Source URL**: https://appsecsanta.com/jfrog-xray
- **Retrieved**: 2026-05-12

## What It Scans

JFrog Xray performs binary-level Software Composition Analysis (SCA) by examining compiled artifacts stored in Artifactory rather than source manifests. It "scans compiled artifacts in JFrog Artifactory — Docker images, JAR files, and installed packages — rather than source manifests."

The tool uses deep recursive scanning to analyze complete dependency graphs, tracing through all layers and transitive dependencies that surface-level approaches might miss.

## Artifactory Integration

Xray operates exclusively as an integrated component of the JFrog Platform and requires Artifactory. The tool scans artifacts at the repository level and cannot function standalone. This tight coupling enables binary-level analysis of actual deployment artifacts rather than theoretical dependencies.

## Supported Package Types

- **Java**: Maven (JAR, WAR, EAR), Gradle
- **JavaScript**: npm, yarn
- **Python**: PyPI (wheel, sdist)
- **Go**: Go modules
- **.NET**: NuGet
- **Ruby**: RubyGems
- **PHP**: Composer
- **Rust**: Cargo
- **C/C++**: Conan
- **Containers**: Docker, OCI, Helm charts
- **OS packages**: Alpine (APK), Debian (DEB), RPM
- **Generic**: ZIP, TAR, binaries

## Vulnerability Databases

Draws from NVD, GitHub Advisories, and JFrog Security Research data (the team has catalogued more than 2.8 million malicious artifacts to date). Implements Contextual Analysis—proprietary applicability rules that determine whether detected vulnerabilities are actually exploitable within your specific codebase.

## Policy Enforcement

Policies can trigger actions based on: CVE severity thresholds, specific CVE IDs, CVSS scores, license type restrictions, component age. When violations occur, Xray can block downloads, fail builds, trigger alerts, or fire webhooks.

## Pricing & Licensing

- **Pro X**: Starting at $150/month (entry point, 25 GB included)
- **Enterprise X**: Starting at $950/month (125 GB included)
- **Enterprise+**: Custom pricing
- Self-Managed: Pro X from $27,000/year, Enterprise X from $51,000/year

All tiers require Artifactory subscription; no standalone Xray option exists.
