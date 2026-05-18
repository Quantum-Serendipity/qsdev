# Security Policy

## Reporting Security Vulnerabilities

**Do NOT open a public issue for security vulnerabilities.**

Please report security issues through [GitHub's private vulnerability reporting](https://github.com/Quantum-Serendipity/qsdev/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge receipt within 48 hours and provide a resolution timeline within 7 days.

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | Yes                |
| < latest | No (upgrade)      |

## Security Practices

- Releases are built with [SLSA Level 3](https://slsa.dev/) provenance
- All release artifacts are signed with Cosign (Sigstore)
- Dependencies are monitored with Dependabot and govulncheck
- Code is scanned with CodeQL on every PR
- Branch protection enforces peer review on `main`
- All CI actions are pinned to commit SHAs
