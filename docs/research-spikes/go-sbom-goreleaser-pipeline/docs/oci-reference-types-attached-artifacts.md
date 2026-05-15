# OCI Reference Types and Attached Artifacts

- **Source**: https://oras.land/docs/concepts/reftypes/
- **Retrieved**: 2026-05-15

## Core Concept

Reference types enable linking between OCI artifacts through a "subject" relationship. Artifacts can be tied to one another by defining one as a subject of another.

## Creating Reference Artifacts

The process involves two steps:

1. **Push the subject artifact** (Artifact A) to the registry using standard distribution APIs
2. **Push the referring artifact** (Artifact B) with a reference to Artifact A's digest

The specification allows pushing reference artifacts without requiring the subject's presence in the registry beforehand.

### Manifest Example

A reference artifact manifest includes a `subject` field pointing to the target:

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/example",
  "config": {
    "mediaType": "application/vnd.oci.empty.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2
  },
  "subject": {
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "digest": "sha256:5e140a61e16155b30356685a6801e5250339bfb11370e70573d28d4ff2dc89cf",
    "size": 477
  }
}
```

## Listing Referrers (Discovery Mechanism)

### API Endpoint

The OCI Distribution Specification defines: `/v2/<name>/referrers/<digest>`

### List All Referrers

Clients request all artifacts referring to a specific artifact, with pagination support for large result sets.

### Filter by Artifact Type

```
GET /v2/hello-world/referrers/sha256:5e140a6..?artifactType=application%2Fexample HTTP/1.1

Response includes: OCI-Filters-Applied: artifactType header
```

The filtered response example:

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:a10d2cf503458e48996a3e3030c75fb5f5cd967e21a38ca69ce6c1e1dee5fd27",
      "size": 677,
      "artifactType": "application/example"
    }
  ]
}
```

### Fallback: Referrers Tag Schema

If the registry lacks referrers API support, use the referrers tag schema for a client-created non-dynamic index.

## Practical Applications

- Discovery and distribution of artifacts like SBOMs or signatures for supply chain
- Movement of a graph of OCI content across environments
- Content management of a graph of artifacts by archiving, deleting or moving them together

SBOMs and signatures attach to container images as reference artifacts through this mechanism, enabling supply chain security and artifact tracking.
