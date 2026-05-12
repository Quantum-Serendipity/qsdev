# PyPI Trusted Publishers
- **Source**: https://docs.pypi.org/trusted-publishers/
- **Retrieved**: 2026-05-12

## What is OIDC?

OpenID Connect (OIDC) is a standard protocol that enables secure identity verification. "Trusted Publishing" uses OIDC to "exchange short-lived identity tokens between a trusted third-party service and PyPI."

## How It Works with PyPI

The process involves four key steps:

1. **Identity Providers**: Certain CI services (like GitHub Actions) can issue temporary credentials that third parties can verify came from that specific service, including which user and repository executed the code.

2. **Configuration Trust**: Projects configure PyPI to trust specific CI service setups as publishers for their packages.

3. **Token Exchange**: When releasing, the CI service submits an OIDC token to PyPI, which validates it against trusted configurations.

4. **Short-Lived API Token**: Upon successful verification, PyPI generates "a short-lived API token for those projects and returns it" — valid for only 15 minutes.

## Security & Usability Benefits

**Security advantages** include eliminating long-lived token exposure. Unlike traditional API tokens, these credentials "expire automatically," preventing extended compromise windows.

**Usability improvements** mean developers no longer need manual token creation and copy-pasting into CI systems — only initial publisher configuration is required.

## Supported CI/CD Providers

- GitHub Actions
- GitLab CI/CD
- Google Cloud Build
- ActiveState

## Attestations

PyPI's initiative yielded:
- **Trusted Publishing**: Uses OIDC to establish misuse-resistant credentials between PyPI and CI/CD systems
- **Attestations**: Uses machine identities and Sigstore to provide zero-setup package signing by default

Generating signed digital attestations for all distribution files and uploading them together is now on by default for all projects using Trusted Publishing.

## 2026 Context

As of March 2026, PyPI hosts over 743,000 packages. The technology continues to be actively adopted as the recommended security practice for Python package publishing.
