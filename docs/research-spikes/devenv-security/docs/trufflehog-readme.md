<!-- Source: https://github.com/trufflesecurity/trufflehog -->
<!-- Retrieved: 2026-05-12 -->

# TruffleHog: Secret Detection and Verification Tool

## Core Functionality

TruffleHog is a security tool designed to "Find, verify, and analyze leaked credentials." Four key operations:

**Discovery**: Scans multiple sources including Git repositories, cloud storage (S3, GCS), Docker images, CI/CD platforms, and filesystems for potential secrets.

**Classification**: Classifies over 800 secret types, mapping them back to the specific identity they belong to. This enables identification of AWS keys, Stripe tokens, database passwords, SSL keys, and similar credentials.

**Verification**: For detected credentials, TruffleHog validates whether they're active by testing against target APIs. Results fall into three categories: verified (confirmed valid), unverified (detected but unconfirmed), and unknown (verification failed due to network/API errors).

**Analysis**: For commonly-leaked credential types, performs deeper analysis to determine permissions and resource access.

## Detection Methods

Multiple detection approaches:

- **Regular expressions**: Pattern matching for credential formats
- **Entropy filtering**: Shannon entropy analysis to reduce false positives
- **API verification**: Programmatic testing against service APIs (e.g., AWS GetCallerIdentity calls)
- **Custom detectors**: Support for organization-specific regex patterns with webhook-based verification

## Performance and Integration

Supports pre-commit hook integration. Available as a GitHub Action, GitLab CI integration, Docker container, and standalone binary. Performance varies by source — GitHub organization scans benefit from authentication tokens to bypass rate limits. Significantly slower than regex-only tools due to API verification step.

## Nix/devenv Integration

No mentions of Nix or devenv integration. Available in nixpkgs as `pkgs.trufflehog`.
