<!-- Source: https://google.github.io/osv.dev/api/ -->
<!-- Retrieved: 2026-05-12 -->

# OSV.dev API Documentation

## Overview

The OSV API enables programmatic vulnerability queries for open source packages.

## Endpoints

1. **POST /v1/query** — Query vulnerabilities for a project at a specific commit hash or version
2. **POST /v1/querybatch** — Batch queries for multiple package versions and commit hashes
3. **GET /v1/vulns/{id}** — Retrieve a Vulnerability object using an OSV ID
4. **POST /v1experimental/determineversion** — Identify probable C/C++ project versions (experimental)
5. **GET /v1experimental/importfindings** — Retrieve records failing quality checks by source (experimental)

## Query Format

```bash
curl -d '{
  "version": "2.4.1",
  "package": {
    "name": "jinja2",
    "ecosystem": "PyPI"
  }
}' "https://api.osv.dev/v1/query"
```

## Batch Query Format

```bash
curl -d '{
  "queries": [
    {"package": {"name": "jinja2", "ecosystem": "PyPI"}, "version": "2.4.1"},
    {"package": {"name": "express", "ecosystem": "npm"}, "version": "4.17.1"}
  ]
}' "https://api.osv.dev/v1/querybatch"
```

## Supported Ecosystems

npm, PyPI, crates.io, Go, Maven, NuGet, Packagist, RubyGems, Linux distributions (Debian, Ubuntu, Alpine, Rocky Linux, AlmaLinux, Chainguard/Wolfi, SUSE/openSUSE), and more.

## Rate Limiting

Currently there are NO limits on the API.

## Response Limits

- HTTP/1.1: 32MiB response size limit
- HTTP/2: No limit (recommended for large queries)

## Key Features

- Free, no authentication required
- Supports commit hash and version queries
- Batch queries for efficiency
- Covers vulnerabilities from GitHub Advisory Database, NVD, and ecosystem-specific databases
- OpenAPI specification available for download
