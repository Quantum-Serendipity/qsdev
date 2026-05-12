<!-- Source: https://www.businesswire.com/news/home/20250106967344/en/Veracode-Acquires-Phylum-Inc.-Technology-to-Transform-Software-Supply-Chain-Security -->
<!-- Source: https://appsecsanta.com/veracode-sca -->
<!-- Retrieved: 2026-05-12 -->

# Phylum: Package Analysis Platform (Now Part of Veracode)

## Key Developments (2025-2026)

Veracode acquired malicious package analysis, detection, and mitigation technology from Phylum, Inc. in January 2025 to enhance software supply chain security.

## Phylum's Technology Capabilities (Pre-Acquisition)

Phylum brought ML-powered detection of malicious packages, typosquatting, dependency confusion, and compromised maintainer accounts. More specifically, Phylum's technology monitors for:
- Typosquatting (names similar to popular libraries)
- Dependency confusion (public packages mimicking internal names)
- Compromised maintainer accounts
- Malicious code injection in legitimate packages

## Detection Approach

Phylum analyzed packages using automated sandbox execution (dynamic analysis) combined with static analysis. Each package version was executed in a controlled environment to observe actual behavior: network connections, file operations, process spawning. This complemented static analysis of source code patterns.

## Integration Results

Following the acquisition, Veracode reports 60% more accurate malicious package detection. The package registry firewall for npm and PyPI blocks malicious packages before installation, preventing compromised dependencies from reaching the codebase.

## Current Status (2026)

Phylum's standalone product is no longer available. The technology has been integrated into Veracode SCA. The integration combines with Veracode SAST and DAST for correlated findings across scan types.

## Pre-Acquisition Capabilities

Phylum supported npm, PyPI, RubyGems, Maven, NuGet, Go, Cargo, and Packagist. It offered:
- CLI tool for local scanning
- GitHub App for PR analysis
- CI/CD integrations (GitHub Actions, GitLab CI, Jenkins)
- Package firewall / allowlist enforcement
- Risk scoring based on five domains: software vulnerabilities, license issues, author risk, engineering risk, malicious code

## Limitations (Current State)

- No longer available as standalone product
- Now requires Veracode enterprise licensing
- Technology focus narrowed to npm and PyPI in Veracode integration
- Open source components on GitHub (phylum-dev) are archived/unmaintained
