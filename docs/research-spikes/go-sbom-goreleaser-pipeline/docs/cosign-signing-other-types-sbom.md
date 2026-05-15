# Signing Other Types with Cosign (including SBOMs)

- **Source**: https://docs.sigstore.dev/cosign/signing/other_types/
- **Retrieved**: 2026-05-15

## Overview

Cosign enables signing of various artifact types beyond container images. "Cosign can sign anything in a registry," including "Helm Charts, Tekton Pipelines, and anything else currently using OCI registries for distribution."

## OCI Artifacts

To sign custom OCI artifacts, first push them using ORAS:

```bash
oras push us-central1-docker.pkg.dev/user-vmtest2/test/artifact ./cosign
```

Then sign the pushed artifact by its digest:

```bash
cosign sign --key cosign.key us-central1-docker.pkg.dev/user-vmtest2/test/artifact@sha256:551e6cce7ed2e5c914998f931b277bc879e675b74843e6f29bc17f3b5f692bef
```

Verify using the public key:

```bash
cosign verify --key cosign.pub us-central1-docker.pkg.dev/user-vmtest2/test/artifact@sha256:551e6cce7ed2e5c914998f931b277bc879e675b74843e6f29bc17f3b5f692bef
```

## SBOMs (Software Bill Of Materials)

For SBOMs stored in OCI registries, use standard signing commands:

```bash
cosign sign --key cosign.key $SBOM_OCI_IMAGE
cosign verify --key cosign.pub $SBOM_OCI_IMAGE
```

Alternatively, attach SBOM metadata to container images using attestations:

```bash
echo '{"sbom_path": "example.com/...", "sbom_hash": "sha256:0a1b2c..."}' > sbom.predicate.json
cosign attest --type custom --predicate sbom.predicate.json $IMAGE
```

For disk-based files, substitute `sign-blob` or `attest-blob` for the OCI versions. The documentation cautions against SBOM predicate types, noting they embed the entire SBOM in signatures, requiring full downloads during verification.

## Tekton Bundles

Tekton Bundles can be managed and signed within OCI registries:

```bash
tkn bundle push us.gcr.io/user-vmtest2/pipeline:latest -f task-output-image.yaml
cosign sign --key cosign.key us.gcr.io/user-vmtest2/pipeline:latest
```
