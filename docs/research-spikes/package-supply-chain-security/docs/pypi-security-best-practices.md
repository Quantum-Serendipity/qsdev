# PyPI Security Best Practices

- **Source**: https://github.com/lirantal/pypi-security-best-practices
- **Retrieved**: 2026-05-12

This GitHub repository compiles 17 evidence-based security practices for Python package management, organized across three domains: installation security, local development hardening, and maintainer responsibilities.

## Core Installation Security Practices

**Binary-Only Installs**: Avoiding source distributions (sdists) that execute arbitrary `setup.py` code. Implementation uses `uv pip install --only-binary :all:` or pip's equivalent environment variable configuration to enforce wheel-only installations.

**Dependency Cooldowns**: A critical defense against freshly-published malicious packages. The LiteLLM incident (119,000+ compromised downloads in 2.5 hours) motivated this practice. uv implements `exclude-newer` settings accepting durations like "7 days" or ISO 8601 formats. pip v26.1+ supports `uploaded-prior-to` with relative time windows. The recommendation spans 3-30 days depending on deployment sensitivity.

**Cryptographic Hash Verification**: uv's lockfile (`uv.lock`) automatically includes SHA-256 checksums. For requirements-based workflows, `uv pip compile --generate-hashes` or `pip-compile --generate-hashes` generate pinned dependencies with embedded hashes. Installation with `--require-hashes` enforces verification.

**Deterministic Installations**: Production deployments should use `uv sync --frozen` to reject outdated lockfiles and abort if inconsistencies exist, preventing unexpected version resolution at install time.

## Dependency Security Hardening

**Dependency Confusion Prevention**: uv's "first-match" index strategy prevents public packages shadowing internal ones. Configuration via `[[tool.uv.index]]` establishes priority, with optional `explicit = true` for per-package index pinning using `[tool.uv.sources]`.

**Vulnerability Scanning**: 
- `pip-audit` queries the OSV database (aggregating PyPA, GitHub, NVD advisories)
- `uv-secure` analyzes lockfiles with configurable severity thresholds
- Integration into CI/CD pipelines catches CVEs before deployment

**Install-Time Firewall**: Socket Firewall (`sfw`) wraps package manager commands, blocking packages flagged for malware, obfuscated code, typosquatting, and suspicious network behavior using deep package analysis.

## Maintainer-Side Protections

**Account Security**: Two-factor authentication (preferably hardware security keys/WebAuthn over TOTP) required for all development accounts. PyPI enforced 2FA for publishers in 2024.

**Trusted Publishing (OIDC)**: Eliminates long-lived API tokens via OpenID Connect, using short-lived workflow-scoped credentials expiring within 15 minutes. GitHub Actions integration requires dedicated environments with approval gates.

**Package Attestations**: PEP 740 implementations using Sigstore provide cryptographic proof linking published artifacts to source repositories and build workflows — over 132,000 PyPI packages include these.

**CI/CD Hardening**: 
- Pin GitHub Actions to commit SHAs (not tags)
- Audit workflows with `zizmor` tool
- Start with empty permissions, expanding only as needed
- Avoid insecure triggers like `pull_request_target`
- Disable caching during release builds

## Package Health Assessment

**Dependency Minimization**: Leverage Python's standard library; each external dependency increases attack surface.

**SBOM Generation**: CycloneDX format SBOMs enable rapid vulnerability impact assessment via tools like `cyclonedx-py`.

**Pre-Adoption Verification**: Consult Snyk Security Database and OSV database before adopting packages. Inspect actual published wheel/sdist contents against GitHub source to detect build-pipeline compromises.
