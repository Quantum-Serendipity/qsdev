# GitHub REST API: Repository Attestations

- **Source URL**: https://docs.github.com/en/rest/repos/attestations
- **Retrieved**: 2026-05-15

---

## Overview
The GitHub REST API provides endpoints to manage artifact attestations associated with repositories. Most endpoints require standard headers: `Authorization: Bearer <YOUR-TOKEN>`, `Accept: application/vnd.github+json`, and `X-GitHub-Api-Version: 2026-03-10`.

## Endpoints

### 1. Create an Attestation
**Endpoint:** `POST /repos/{owner}/{repo}/attestations`

**Purpose:** Store and associate an artifact attestation with a repository.

**Authentication:** Requires write permission to the repository and, for fine-grained tokens, the `attestations:write` permission.

**Path Parameters:**
- `owner` (string, required): Repository account owner (case-insensitive)
- `repo` (string, required): Repository name without .git extension (case-insensitive)

**Request Body:**
- `bundle` (object, required): The attestation's Sigstore Bundle
  - `mediaType` (string)
  - `verificationMaterial` (object)
  - `dsseEnvelope` (object)

**Response:**
- Status 201: Created
- Status 403: Forbidden
- Status 422: Validation failed or endpoint spammed

**Example Request:**
```bash
curl -L -X POST https://api.github.com/repos/OWNER/REPO/attestations \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer <YOUR-TOKEN>" \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  -d '{
    "bundle": {
      "mediaType": "application/vnd.dev.sigstore.bundle.v0.3+json",
      "verificationMaterial": {...},
      "dsseEnvelope": {...}
    }
  }'
```

### 2. List Attestations
**Endpoint:** `GET /repos/{owner}/{repo}/attestations/{subject_digest}`

**Purpose:** Retrieve artifact attestations with a specified subject digest linked to a repository.

**Authentication:** Requires read access; fine-grained tokens need `attestations:read` permission.

**Path Parameters:**
- `owner` (string, required): Repository account owner
- `repo` (string, required): Repository name
- `subject_digest` (string, required): Attestation subject's SHA256 digest (format: `sha256:HEX_DIGEST`)

**Query Parameters:**
- `per_page` (integer): Results per page; maximum 100, default 30
- `before` (string): Cursor for pagination
- `after` (string): Cursor for pagination
- `predicate_type` (string): Optional filter for provenance, sbom, release, or custom types

**Response:**
- Status 200: OK
- Returns `attestations` array containing:
  - `repository_id` (integer)
  - `bundle_url` (string)
  - `initiator` (string)

**Example Request:**
```bash
curl -L -X GET https://api.github.com/repos/OWNER/REPO/attestations/SUBJECT_DIGEST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer <YOUR-TOKEN>" \
  -H "X-GitHub-Api-Version: 2026-03-10"
```

## Security Notes
Attestations require cryptographic verification of signatures and timestamps, plus validation of signer identity. Use the GitHub CLI `attestation verify` command for standard verification.
