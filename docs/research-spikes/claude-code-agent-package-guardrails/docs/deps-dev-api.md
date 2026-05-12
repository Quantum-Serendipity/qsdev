<!-- Source: https://docs.deps.dev/api/v3alpha/ -->
<!-- Retrieved: 2026-05-12 -->

# deps.dev API Documentation

## Overview
The deps.dev API provides programmatic access to Open Source Insights data, enabling queries about package versions, dependencies, licenses, and security information across multiple package ecosystems.

## Base URL
`https://api.deps.dev/v3alpha`

## Authentication
No authentication requirements mentioned in documentation. Appears to be free/open access.

## Access Methods
- **JSON over HTTP** (primary)
- **gRPC** (details at github.com/google/deps.dev)

## Supported Ecosystems
Go, RubyGems, npm, Cargo, Maven, PyPI, and NuGet.

## Core Endpoints

### Package Information
- **GetPackage**: `GET /v3alpha/systems/{system}/packages/{name}` - Lists versions and metadata
- **GetVersion**: `GET /v3alpha/systems/{system}/packages/{name}/versions/{version}` - Detailed version info
- **GetVersionBatch**: `POST /v3alpha/versionbatch` - Batch requests (max 5000)

### Dependency Data
- **GetRequirements**: Returns dependency constraints in system-specific formats
- **GetDependencies**: Returns resolved dependency graphs (npm, Cargo, Maven, PyPI)
- **GetDependents**: Provides dependent package counts

### Project & Advisory Information
- **GetProject**: `GET /v3alpha/projects/{id}` - GitHub/GitLab/Bitbucket data
- **GetAdvisory**: Security advisories from OSV
- **GetCapabilities**: Go package capability usage (Capslock)

### Search & Lookup
- **Query**: `GET /v3alpha/query` - Search by name or file hash
- **PurlLookup**: Search using Package URLs
- **GetSimilarlyNamedPackages**: Find packages with similar names

## Key Response Data

**Version Information includes:**
- License data (SPDX 2.1 expressions)
- Security advisories (OSV identifiers)
- Publication timestamps
- Deprecation status
- Links to repositories and homepages

**Project Data includes:**
- Repository metrics (stars, forks, issues)
- OpenSSF Scorecard results
- OSS-Fuzz coverage information

**Dependency Graphs contain:**
- Node relationships (direct/indirect)
- Resolved versions
- Error conditions

## Request Format Requirements

**Path/Query Parameters:** Special characters must be percent-encoded using URL-safe methods.

**Batch Methods:** Use HTTP POST with JSON body; limited to 5000 requests per batch.

**Data Parameters:** Hash values use base64 encoding. Example:
```
openssl sha1 -binary <file> | base64
```

## Package Naming Conventions
- Maven: `<groupID>:<artifactID>` format
- PyPI: Normalized per PEP 503
- NuGet: Lowercased with semantic versioning
- Others: Ecosystem-standard names preserved

## Rate Limits & Quotas
Not specified in provided documentation. Likely free with reasonable use limits.

## Coverage Limitations
- Go: Only modules fetched through proxy.golang.org
- PyPI: Wheels and sdists only
- Maven: Central, Jenkins, Gradle Plugins, Google registries
- Projects: GitHub, GitLab, Bitbucket only

## Response Pagination
Batch methods support pagination via `nextPageToken` when result sets exceed limits.
