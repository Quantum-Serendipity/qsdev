<!-- Source: https://docs.deps.dev/api/v3/ -->
<!-- Retrieved: 2026-05-12 -->

# deps.dev API v3 Technical Overview

## Available Endpoints

The deps.dev API provides eight primary methods:

1. **GetPackage** - `GET /v3/systems/{system}/packages/{name}` — Returns package information including all available versions and default version designation.
2. **GetVersion** - `GET /v3/systems/{system}/packages/{name}/versions/{version}` — Retrieves detailed version data: licenses, security advisories, attestations, and project relationships.
3. **GetRequirements** - `GET /v3/systems/{system}/packages/{name}/versions/{version}:requirements` — Delivers system-specific dependency constraints in native format.
4. **GetDependencies** - `GET /v3/systems/{system}/packages/{name}/versions/{version}:dependencies` — Returns resolved dependency graphs showing actual installed versions.
5. **GetProject** - `GET /v3/projects/{projectKey.id}` — Provides project metadata from GitHub, GitLab, or Bitbucket, including OpenSSF Scorecard results and OSS-Fuzz data.
6. **GetProjectPackageVersions** - `GET /v3/projects/{projectKey.id}:packageversions` — Maps projects to their associated package versions (max 1500 results).
7. **GetAdvisory** - `GET /v3/advisories/{advisoryKey.id}` — Supplies security advisory details including CVSS scores and aliases.
8. **Query** - `GET /v3/query` — Enables multi-parameter searches by package name or file content hash (SHA1, SHA256, MD5, SHA512).

## Supported Ecosystems

Go, RubyGems, npm, Cargo, Maven, PyPI, and NuGet.

## Data Coverage Details

**Package sources:** Crates.io, Go Module Mirror, Maven Central Repository (plus Google, Jenkins, Gradle registries), npm Registry, NuGet, PyPI, and RubyGems.

**Project sources:** GitHub, GitLab, Bitbucket (matched to known packages only).

**Additional data:**
- Security advisories from OSV.dev
- OpenSSF Scorecard assessments
- OSS-Fuzz coverage metrics

## Key Data Points Returned

**Version Information:** Publication timestamps, deprecation status, SPDX 2.1 license identifiers

**Dependencies:** Unresolved requirements (ecosystem-specific constraints) via GetRequirements; Resolved dependency graphs via GetDependencies (available for npm, Cargo, Maven, PyPI only)

**Security:** Advisory identifiers, CVSS v3 scores/vectors, CVE aliases; Sigstore bundle verification status

**Attestations:** SLSA provenance (v0.2, v1), PyPI publish attestations; Cryptographic verification indicators, source repository and commit data

**Project Quality Indicators:** OpenSSF Scorecard with 14+ checks; OSS-Fuzz line coverage metrics; Stars, forks, open issues (GitHub/GitLab only)

## Query Feature

Unique hash query capability: look up the hash of a file's contents and find all package versions that contain that file. Useful for SBOMs, container analysis, incident response, and forensics.

## Rate Limits & Caching

Clients are expressly permitted to cache data. No specific rate limits documented publicly.

## License & Terms

Generated data available under CC-BY 4.0. Commercial and research applications supported.
