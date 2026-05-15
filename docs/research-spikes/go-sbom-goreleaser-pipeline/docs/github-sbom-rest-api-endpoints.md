# GitHub SBOM REST API Endpoints

- **Source URL**: https://docs.github.com/en/rest/dependency-graph/sboms
- **Retrieved**: 2026-05-15

---

## Authentication
All requests require three standard headers:
- `Authorization: Bearer <YOUR-TOKEN>`
- `Accept: application/vnd.github+json`
- `X-GitHub-Api-Version: 2026-03-10`

## Endpoint 1: Export SBOM

**Method & URL:**
```
GET /repos/{owner}/{repo}/dependency-graph/sbom
```

**Description:** Exports the software bill of materials (SBOM) for a repository in SPDX JSON format.

**Required Parameters:**
- `owner` (string): Repository account owner (case-insensitive)
- `repo` (string): Repository name without .git extension (case-insensitive)

**Response Codes:** 200 (OK), 403 (Forbidden), 404 (Not found)

**Example Request:**
```bash
curl -L -X GET \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer <YOUR-TOKEN>" \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  https://api.github.com/repos/OWNER/REPO/dependency-graph/sbom
```

**Response Schema (200):** Contains SPDX document with:
- `sbom` object: SPDXID, spdxVersion, creationInfo, name, dataLicense, documentNamespace
- `packages` array: name, versionInfo, downloadLocation, licenseConcluded, copyrightText, externalRefs
- `relationships` array: relationshipType, spdxElementId, relatedSpdxElement

## Endpoint 2: Fetch Generated SBOM

**Method & URL:**
```
GET /repos/{owner}/{repo}/dependency-graph/sbom/fetch-report/{sbom_uuid}
```

**Description:** Fetches a previously generated software bill of materials (SBOM) for a repository. Returns a 302 redirect to temporary download URL; reports retained up to one week.

**Required Parameters:**
- `owner`, `repo` (same as above)
- `sbom_uuid` (string): Unique SBOM export identifier

**Response Codes:** 202 (Processing), 302 (Redirect to download), 403 (Forbidden), 404 (Not found)

**Example Request:**
```bash
curl -L -X GET \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer <YOUR-TOKEN>" \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  https://api.github.com/repos/OWNER/REPO/dependency-graph/sbom/fetch-report/SBOM_UUID
```

## Endpoint 3: Request SBOM Generation

**Method & URL:**
```
GET /repos/{owner}/{repo}/dependency-graph/sbom/generate-report
```

**Description:** Triggers a job to generate a software bill of materials (SBOM) for a repository in SPDX JSON format.

**Required Parameters:**
- `owner`, `repo` (same as above)

**Response Codes:** 201 (Created), 403 (Forbidden), 404 (Not found)

**Example Request:**
```bash
curl -L -X GET \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer <YOUR-TOKEN>" \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  https://api.github.com/repos/OWNER/REPO/dependency-graph/sbom/generate-report
```

**Response Schema (201):**
- `sbom_url` (string): URL for retrieving the generated report
