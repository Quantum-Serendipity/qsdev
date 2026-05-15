<!-- Source: https://anchore.com/sbom/creating-sbom-attestations-using-syft-and-sigstore/ -->
<!-- Retrieved: 2026-05-15 -->

# Creating SBOM Attestations with Syft and Sigstore

## Overview

Syft v0.40.0 introduced the ability to create cryptographically signed SBOM attestations using Project Sigstore, enabling secure validation that SBOMs originate from trusted sources in the software supply chain.

## What Are Attestations?

"An attestation is a cryptographically signed 'statement' that claims something (a 'predicate') is true about another thing (a 'subject')." In SBOM contexts, the SBOM serves as the predicate while the container image is the subject. This signing mechanism allows consumers to verify data integrity and authenticate the source.

## Why Attestations Matter

Attestations enable downstream users to safely rely on SBOM data without performing their own analysis. They provide tamper detection and establish trust based on the signer's identity, making them essential when SBOMs cross organizational boundaries.

## Complete Workflow

### Step 1: Generate Keys
```
$ cosign generate-key-pair
```

Store your password in the `COSIGN_PASSWORD` environment variable.

### Step 2: Create SBOM Attestation
Generate and sign the attestation using Syft's attest command:
```
$ syft attest --key ./cosign.key <my-image> -o cyclonedx-json > ./my-image-sbom.att.json
```

Supports CycloneDX JSON, SPDX JSON, and Syft's native JSON formats.

### Step 3: Attach to Registry
Upload the attestation to a container registry:
```
$ cosign attach attestation <my-image> --attestation ./my-image-sbom.att.json
```

### Step 4: Verify Attestation
Anyone with your public key can verify:
```
$ cosign verify-attestation <my-image> --key ./cosign.pub
```

## Technical Implementation

Syft creates attestations using the in-toto attestation framework, with attestation data structured as JSON. The framework's flexibility allows predicates in multiple formats while maintaining verification capabilities.

## Future Enhancements

Planned integrations include:
- Support for Sigstore's keyless workflow
- Attestation capabilities in Grype for vulnerability scan verification
- Deeper integration between Syft, Grype, and Sigstore ecosystems
