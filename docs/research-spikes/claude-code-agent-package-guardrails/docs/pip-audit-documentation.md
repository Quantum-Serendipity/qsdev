<!-- Source: https://github.com/pypa/pip-audit -->
<!-- Retrieved: 2026-05-12 -->

# pip-audit Documentation

## Purpose & Functionality

pip-audit audits Python environments, requirements files and dependency trees for known security vulnerabilities, and can automatically fix them. It analyzes installed packages against known vulnerability databases to identify and remediate security issues.

## Vulnerability Data Sources

The tool queries multiple vulnerability services:
- **PyPI**: Uses the Python Packaging Advisory Database via PyPI JSON API
- **OSV**: Supports querying the Open Source Vulnerabilities database
- **ESMS**: Also available as a vulnerability service option

## Installation & Requirements

- **Python version**: Requires Python 3.10 or newer
- **Primary installation**: `python -m pip install pip-audit`
- **CI/CD integration**: Official GitHub Action available at `pypa/gh-action-pip-audit`
- **Pre-commit support**: Integrates with pre-commit hooks

## CLI Usage & Key Options

Core command format: `pip-audit [options] [project_path]`

**Primary input modes:**
- Scan local environment: `pip-audit`
- Audit requirements file: `pip-audit -r requirements.txt`
- Audit project: `pip-audit ./path`
- Audit lockfiles: `pip-audit --locked ./path`

**Output formatting options:**
- `--format`: columns (default), json, cyclonedx-json, cyclonedx-xml, markdown
- `--desc`: Include vulnerability descriptions (on/off/auto)
- `--aliases`: Show CVE and GHSA IDs (on/off/auto)

**Performance & control flags:**
- `--dry-run`: Show what would be audited without performing the scan
- `--fix`: Automatically upgrade vulnerable packages
- `--no-deps`: Skip dependency resolution for fully pinned requirements
- `--timeout`: Set socket timeout (default: 15 seconds)

**Filtering options:**
- `--local`: Only include locally installed dependencies
- `--ignore-vuln ID`: Exclude specific vulnerabilities by ID or alias
- `--skip-editable`: Don't audit editable packages

## Exit Codes

- **0**: No vulnerabilities detected
- **1**: One or more vulnerabilities found

## JSON Output Format

The tool emits structured data containing:
- Package name and version
- Vulnerability ID (PYSEC, CVE, GHSA formats)
- Available fix versions
- Vulnerability descriptions and aliases
- Full SBOM support in CycloneDX XML/JSON formats

## Environment Variables

Configuration can be set via:
- `PIP_AUDIT_FORMAT`
- `PIP_AUDIT_VULNERABILITY_SERVICE`
- `PIP_AUDIT_DESC`
- `PIP_AUDIT_PROGRESS_SPINNER`
- `PIP_AUDIT_OUTPUT`

## Performance Considerations

The tool may require full dependency resolution, which can take roughly as long as `pip install` does for a project. Users can optimize by:
- Auditing pre-installed environments instead
- Using `--no-deps` with fully pinned requirements
- Employing `--require-hashes` for already-resolved dependencies

## Programmatic API

pip-audit provides programmatic access through modules:
- `Auditor` class with `audit()` method and `AuditOptions` dataclass
- `VulnerabilityService` base class with `PyPIService` and `OsvService` implementations
- `Dependency` and `VulnerabilityResult` data structures
- Custom OSV API URL can be specified

## Security Model

The tool performs best-effort full dependency resolution but is not a static code analyzer. It identifies known vulnerabilities, not all potential threats.

## Project Details

- Maintained by Trail of Bits with Google support
- Licensed under Apache 2.0
- Latest: v2.10.0 (December 2025)
