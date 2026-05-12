# GitHub Packages: Introduction

- **Source URL**: https://docs.github.com/en/packages/learn-github-packages/introduction-to-github-packages
- **Retrieved**: 2026-05-12

## Supported Registries & Formats

| Language | Format | Client |
|----------|--------|--------|
| JavaScript | `package.json` | npm |
| Ruby | `Gemfile` | gem |
| Java (Maven) | `pom.xml` | mvn |
| Java (Gradle) | `build.gradle(/.kts)` | gradle |
| .NET | `nupkg` | dotnet CLI |
| Containers | `Dockerfile` | Docker |

## Authentication

"You need an access token to publish, install, and delete private, internal, and public packages." Authentication uses personal access tokens (classic) with scope-based permissions. Within GitHub Actions, the `GITHUB_TOKEN` can publish packages for the workflow repository.

## Security & Permissions

Package permissions either inherit from the hosting repository or support granular user/organization-level controls (registry-dependent).

## Pricing Model

GitHub Packages usage is **free for public packages**. Private packages receive free storage and data transfer quotas based on account plan; overage requires valid payment methods.

## Key Limitation: No Upstream Proxying

**GitHub Packages does NOT support proxying or mirroring from upstream public registries.** It serves only as a registry for publishing and consuming packages. There is no transparent proxy functionality. It is NOT a replacement for tools like Artifactory, Nexus, or Verdaccio as a caching proxy.

## Management

REST API and (limited) GraphQL access available. Certain registries support only repository-scoped permissions.
