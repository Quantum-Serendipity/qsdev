# SLSA 3 Compliance with GitHub Actions and Sigstore for Go

- **Source URL**: https://github.blog/security/supply-chain-security/slsa-3-compliance-with-github-actions/
- **Retrieved**: 2026-05-15

---

## Core Purpose

The article explains how to achieve SLSA 3 compliance for Go modules using GitHub Actions integrated with Sigstore's signing tools. The approach addresses supply chain security by generating non-forgeable build provenance that proves software authenticity and origin.

## Key Components

**Sigstore consists of three projects:**
- Cosign for signing software
- Fulcio, a certificate authority providing short-lived certificates via OpenID Connect
- Rekor, a secure transparency log for signing events

**SLSA Framework**: A standards framework that helps organizations verify software provenance throughout its development lifecycle, responding to NIST recommendations.

## How It Works

The workflow leverages GitHub Actions' isolated virtual machines and OIDC tokens containing repository, branch, commit, and workflow metadata. This information becomes part of the provenance signature, allowing anyone to inspect, audit, or replicate a build.

**Notable advantage:** Open source projects can sign builds without managing private keys -- Fulcio exchanges OIDC tokens for temporary certificates automatically (keyless signing).

## Getting Started

Users can access the reusable workflow through the Actions tab in any GitHub repository. The process generates provenance metadata that can be verified using Rekor CLI commands.

## Important Update (December 2024)

The article includes an update directing readers to newer guidance on GitHub Artifact Attestations for current SLSA Level 3 implementation. GitHub's native attestation system is now the recommended path rather than the original slsa-github-generator approach.
