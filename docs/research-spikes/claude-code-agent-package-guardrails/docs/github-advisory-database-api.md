<!-- Source: https://docs.github.com/en/rest/security-advisories/global-advisories -->
<!-- Retrieved: 2026-05-12 -->

# GitHub Global Security Advisories API Documentation

## Endpoints

**List Global Advisories:**
```
GET /advisories
```
Retrieves all security advisories matching specified parameters. By default, returns only reviewed advisories excluding malware.

**Get Single Advisory:**
```
GET /advisories/{ghsa_id}
```
Retrieves a specific advisory using its GitHub Security Advisory identifier.

## Authentication & Headers

All requests require:
- `Authorization: Bearer <YOUR-TOKEN>`
- `Accept: application/vnd.github+json`
- `X-GitHub-Api-Version: 2026-03-10`

## Query Parameters for Filtering

| Parameter | Type | Options |
|-----------|------|---------|
| `ghsa_id` | string | GitHub Security Advisory identifier |
| `cve_id` | string | Common Vulnerabilities and Exposures ID |
| `type` | string | `reviewed`, `malware`, `unreviewed` (default: reviewed) |
| `ecosystem` | string | rubygems, npm, pip, maven, nuget, composer, go, rust, erlang, actions, pub, other, swift |
| `severity` | string | unknown, low, medium, high, critical |
| `cwes` | string | Common Weakness Enumerations (e.g., cwes=79,284,22) |
| `affects` | string | Package names with optional versions (max 1000) |
| `is_withdrawn` | boolean | Filter withdrawn advisories |
| `published` | string | Date or date range |
| `updated` | string | Date or date range |
| `modified` | string | Date or date range |
| `epss_percentage` | string | EPSS score matching provided value |
| `epss_percentile` | string | EPSS relative rank |

## Pagination Parameters

- `per_page`: Results per page, max 100 (default: 30)
- `direction`: `asc` or `desc` (default: desc)
- `sort`: `updated`, `published`, `epss_percentage`, `epss_percentile` (default: published)
- `before`/`after`: Cursor-based pagination via Link header

## Response Format

Responses include advisory objects containing:
- Identifiers: GHSA ID, CVE ID
- Metadata: summary, description, type, severity
- Vulnerability details: affected packages, version ranges, patched versions
- CVSS scores (v3 and v4)
- EPSS metrics (percentage and percentile)
- CWE classifications
- Publication and update timestamps
- References and credits with user information

## HTTP Status Codes

- **200**: Successful request
- **404**: Advisory not found
- **422**: Validation failed or endpoint spammed
- **429**: Rate limit exceeded

## Example Request

```
curl -L -X GET https://api.github.com/advisories
```

## Key Relationships

GHSA IDs serve as the primary identifiers for GitHub advisories, with CVE IDs providing cross-reference to the National Vulnerability Database. The API enables systematic vulnerability tracking across multiple programming ecosystems.
