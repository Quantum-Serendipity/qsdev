<!-- Source: https://github.blog/security/supply-chain-security/slsa-3-compliance-with-github-actions/ -->
<!-- Retrieved: 2026-05-15 -->

# Achieving SLSA 3 Compliance with GitHub Actions and Sigstore for Go Modules

## Overview

GitHub provides tools to achieve SLSA 3 compliance for Go modules by combining GitHub Actions with Sigstore's signing infrastructure. This approach addresses supply chain security concerns by enabling developers to prove software authenticity and build provenance.

## Key Components

**Sigstore** comprises three projects:
- **Cosign**: Signs software and includes provenance metadata
- **Fulcio**: A certificate authority providing short-lived certificates via OpenID Connect
- **Rekor**: A secure transparency log documenting signing events

**SLSA Framework** establishes levels for improving software artifact integrity throughout development. It responds to NIST recommendations that users verify their software dependencies' provenance.

## How It Works

The workflow leverages GitHub Actions' isolated virtual machines and OpenID Connect tokens to create non-forgeable build metadata. The provenance information comes from the Actions OIDC token, which contains information specific to your run of an Actions workflow.

This metadata includes:
- Repository and branch information
- Specific commit hash
- Exact Actions workflow used
- Build environment details

## Implementation

GitHub offers a reusable workflow accessible through the Actions tab. The process:

1. Creates signed Go module builds automatically
2. Generates provenance metadata during the build
3. Records signing events in Rekor's transparency log
4. Allows verification of build integrity through CLI commands

## Verification

The generated provenance data can be verified by querying Rekor using simple command-line tools, enabling both internal teams and external parties to audit build integrity and authenticity.
