<!-- Source: https://github.com/mchmarny/s3cme -->
<!-- Retrieved: 2026-05-15 -->

# s3cme: Supply Chain Security Template Repository for Go

## Overview

Template Go application repository demonstrating best practices for software supply chain security (S3C). Automated workflows for testing, building, and releasing container images with cryptographic verification.

## Workflow Pipelines

### On-Push (PR Qualification)
- Static code vulnerability scanning with Trivy
- CodeQL-based security alerts via SARIF reports

### On-Tag (Release Pipeline)
- Container image construction using ko (with automatic SBOM generation)
- Image vulnerability assessment using Trivy with configurable severity thresholds
- Image signing and attestation via cosign
- SLSA provenance generation through slsa-framework/slsa-github-generator
- Provenance verification using slsa-verifier and CUE policies

### On-Schedule (Maintenance)
- Semantic code analysis runs every four hours using CodeQL

## Key Security Artifacts

The release pipeline generates four OCI artifacts in the container registry:
1. **Container image** - the actual application
2. **`.sig` artifact** - cosign cryptographic signature
3. **`.att` artifact** - SLSA attestations (supply chain metadata)
4. **`.sbom` artifact** - Software Bill of Materials in SPDX v2.3 format

## Provenance Verification

### Manual Verification Command

```bash
cosign verify-attestation --type slsaprovenance \
  --certificate-identity-regexp "^https://github.com/slsa-framework/..." \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --policy policy/provenance.cue $digest
```

The verification output includes:
- Cosign claims validation
- Transparency log verification
- Certificate authority confirmation
- Workflow metadata (trigger, SHA, name, repository, ref)

### In-Cluster Verification

Kubernetes deployments can enforce SLSA provenance policies using sigstore's admission controller. The setup requires:
- Policy configuration specifying trusted image registries
- Identity verification against GitHub Actions OIDC issuer
- CUE-based policy definitions checking predicateType

Labeled namespaces will reject deployments lacking valid SLSA attestations matching the configured policy.
