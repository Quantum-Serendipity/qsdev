# Artifact Keeper — Universal Artifact Registry
> Source: https://github.com/artifact-keeper/artifact-keeper
> Retrieved: 2026-05-12

## Overview
Open-source universal artifact registry. Drop-in Artifactory/Nexus alternative.

## Key Stats
- 45+ package formats
- Built in Rust with Axum framework
- 6,400+ unit tests
- 69 PostgreSQL migrations

## Supported Package Formats
**Languages:** Maven, NPM, PyPI, NuGet, Cargo, Go, RubyGems, Hex, Composer, Pub, CocoaPods, Swift, CRAN, SBT
**Containers/Infra:** Docker/OCI, Helm, Terraform, Vagrant
**System Packages:** RPM, Debian, Alpine (APK), Conda, OPKG
**Other:** Chef, Puppet, Ansible, HuggingFace, VS Code, JetBrains, Protobuf/BSR, Conan, Git LFS, Bazel, P2, Generic
**Extensible:** Custom formats via WASM plugin system

## Security Scanning Pipeline
1. Deduplication via SHA-256 hashing
2. Dual scanner: Trivy (filesystem/container) + Grype (dependency trees)
3. Vulnerability scoring: grades A through F
4. Policy engine: blocks or quarantines failing artifacts
5. Artifact signing: GPG/RSA for Debian, RPM, Alpine, Conda

## Authentication
- JWT, OpenID Connect, LDAP, SAML 2.0, API tokens

## Container Security
- DISA STIG-approved Red Hat UBI 9 base images
- Non-root execution
- No shell or package manager in runtime

## CI/CD Integration
- ci.yml: lint, unit/integration tests, smoke E2E on every push
- docker-publish.yml: multi-arch Docker images to ghcr.io
- e2e.yml: full E2E across 10 native client formats with stress/failure injection
- release.yml: gated releases with cross-platform binaries
- scheduled-tests.yml: nightly vulnerability and dependency checks

## Relevance to gdev
- Could serve as the hosted artifact cache/registry for team devenvs
- Built-in vulnerability scanning means packages are checked before caching
- Policy engine can enforce organizational standards
- WASM plugin system allows custom format support (e.g., Nix binary cache)
- Self-hostable, no vendor lock-in
