<!-- Source: https://ossf.github.io/osv-schema/ -->
<!-- Retrieved: 2026-05-12 -->

# OSV Schema Specification

The Open Source Vulnerability (OSV) format defines a JSON-based interchange structure for describing vulnerabilities in open source packages.

## Top-Level Fields

**Required:**
- `schema_version`: Version identifier following SemVer 2.0.0 (defaults to "1.0.0")
- `id`: Unique identifier in format `<DB>-<ENTRYID>` (e.g., "CVE-2021-3114")
- `modified`: RFC3339-formatted UTC timestamp of last modification

**Optional but Common:**
- `published`: RFC3339-formatted publication timestamp
- `withdrawn`: RFC3339-formatted withdrawal timestamp
- `summary`: One-line textual description (max ~120 characters recommended)
- `details`: CommonMark markdown providing additional context
- `aliases`: Array of IDs representing the same vulnerability in other databases
- `upstream`: Array of upstream vulnerability IDs
- `related`: Array of closely related vulnerability IDs
- `severity`: Array of severity objects with `type` and `score`
- `affected`: Array of affected package objects
- `references`: Array of reference objects with `type` and `url`
- `credits`: Array of credit objects
- `database_specific`: JSON object for database-defined custom fields

## Severity Objects

Each severity entry contains:
- `type`: "CVSS_V2", "CVSS_V3", "CVSS_V4", "Ubuntu", or custom
- `score`: String representation per the specified type

## Affected Array Structure

Each affected package object includes:

**Package Identification:**
```json
{
  "package": {
    "ecosystem": "string",   // Required
    "name": "string",        // Required
    "purl": "string"         // Optional, Package URL
  }
}
```

**Supported ecosystems:** npm, PyPI, RubyGems, Maven, NuGet, Go, Rust (crates.io), Linux, Android, Debian, Ubuntu, Alpine, Docker, Kubernetes, and 60+ others.

**Version Information:**
- `versions`: Array of specific affected version strings
- `ranges`: Array of version range objects

**Range Objects (type: SEMVER, ECOSYSTEM, or GIT):**
```json
{
  "ranges": [{
    "type": "SEMVER|ECOSYSTEM|GIT",
    "repo": "string",
    "events": [
      {"introduced": "string"},
      {"fixed": "string"},
      {"last_affected": "string"},
      {"limit": "string"}
    ],
    "database_specific": {}
  }]
}
```

## References Array

Each reference contains:
- `type`: ADVISORY, ARTICLE, DETECTION, DISCUSSION, REPORT, FIX, INTRODUCED, PACKAGE, EVIDENCE, WEB
- `url`: Fully-qualified URL

## Version Evaluation

A package is vulnerable if its version appears in the `versions` list OR falls within a `ranges` specification. The evaluation algorithm processes sorted range events sequentially, toggling vulnerable status based on introduced/fixed/last_affected markers.
