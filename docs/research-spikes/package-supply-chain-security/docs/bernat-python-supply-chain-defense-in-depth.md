# Python Supply Chain Security: Defense in Depth

- **Source**: https://bernat.tech/posts/securing-python-supply-chain/
- **Retrieved**: 2026-05-12

## Core Security Layers

The article outlines a "defense in depth" strategy where multiple security controls protect against different attack vectors. No single control is perfect — when one fails, others catch the threat.

## Dependency Pinning & Hash Verification

### The Problem with Unpinned Dependencies

When dependencies use loose version constraints like `flask>=2.0`, new malicious versions can be installed silently without code changes. The document illustrates this progression:

- **Unpinned** (`flask>=2.0`): Gets any version, vulnerable to compromise
- **Version pinned** (`flask==3.1.1`): Exact version, but no integrity verification
- **Hash pinned**: Cryptographic fingerprints prevent tampering

### Hash Pinning Implementation

The article emphasizes that "hash pinned" dependencies with SHA256 checksums create an "immutable record of exactly what you installed." Tools that implement this:

**uv (recommended):**
```bash
uv lock
uv sync
uv pip compile --generate-hashes requirements.in -o requirements.txt
```

**pip-tools:**
```bash
pip-compile --generate-hashes requirements.in > requirements.txt
```

The resulting files include entries like:
```
flask==3.1.1 \
    --hash=sha256:d667207822...
```

### Critical Enforcement Detail

"pip only enforces hash checking when every requirement in the file has a hash, or when you pass `--require-hashes` explicitly. A single unhashed line silently disables verification for that package."

This means:
- Either all dependencies must have hashes, OR
- Use `pip install --require-hashes -r requirements.txt` to force strict checking

## Vulnerability Scanning

### pip-audit Integration

The article recommends running `pip-audit` in CI to catch known CVEs automatically:

```bash
uvx pip-audit --requirement requirements.txt
uvx pip-audit --format json --requirements requirements.txt > report.json
```

The tool checks against multiple sources: PyPA Advisories, GitHub Advisories, and the National Vulnerability Database through OSV aggregation.

## Software Bill of Materials (SBOM)

### CycloneDX Generation

SBOMs enable rapid response when vulnerabilities are announced. Generate with:

```bash
uv pip install cyclonedx-bom
cyclonedx-py environment --output-file sbom.json
```

The SBOM includes package metadata with cryptographic hashes and links back to source repositories, enabling "are we affected?" answers in minutes rather than days.

## Secure Package Publishing

### Trusted Publishing with OIDC

The document contrasts approaches:

**Traditional (risky):** Long-lived PyPI API tokens stored indefinitely
**Modern (secure):** OIDC-generated short-lived credentials via Trusted Publishing

The workflow automatically generates attestations via Sigstore, "linking your packages back to source repos" with cryptographic proof of publishing identity and source commit SHA.

## Code Security

### Ruff Linting Configuration

The article recommends starting with security rules only:

```toml
[tool.ruff]
line-length = 120
lint.select = ["E", "F", "S"]
```

Key security checks (`S` prefix):
- **S105**: Hardcoded secrets detection
- **S324**: Weak cryptography (MD5, SHA1)
- **S113**: Missing request timeouts
- **S301**: Unsafe pickle deserialization
- **S608**: SQL injection via string formatting

## Development vs. Deployment Separation

For applications, maintain two files:

```toml
# pyproject.toml - flexible ranges
[project]
dependencies = ["flask>=2.0", "requests>=2.28"]
```

Auto-generate pinned deployment files:
```bash
uv export --format requirements-txt -o requirements.txt
```

This provides development flexibility while guaranteeing reproducible deployments.

## Time-Based Defenses

### Delayed Ingestion for Individual Developers

Modern tools support filtering by publication date:

```bash
# uv: exclude packages published after a date
uv pip compile --exclude-newer 2026-03-02 requirements.in -o requirements.txt

# pip v26+: equivalent
pip install --uploaded-prior-to 2026-03-02T00:00:00Z -r requirements.txt
```

This "provides a buffer period that can help catch obvious malicious packages before they reach your systems."

### Organizational Ingestion Control

Large organizations can run internal mirrors with mandatory delays (typically 6-7 days), allowing threat discovery before packages become available to developers. This requires dedicated security infrastructure but provides significant protection.

## Workflow Security

### GitHub Actions Auditing

The `zizmor` tool detects workflow vulnerabilities:

```bash
uvx zizmor .
uvx zizmor --gh-token $(gh auth token) .
```

Key audits: template injection, unpinned actions, excessive permissions, credential leaks, and detection of known vulnerable actions.

## Implementation Roadmap

**Phase 1 (1-2 days):** Ruff linting + uv dependency pinning

**Phase 2 (1 week):** pip-audit in CI, SBOM generation, Dependabot, Trusted Publishing

**Phase 3 (ongoing):** Delayed ingestion, SBOM tracking systems

## Key Principle

"Layer your defenses and don't trust any single control." The article emphasizes that "hash pinning stops tampering but won't save you from a malicious package you installed on day one. Scanning finds known CVEs but misses zero-days." Multiple overlapping controls compensate for individual limitations.
