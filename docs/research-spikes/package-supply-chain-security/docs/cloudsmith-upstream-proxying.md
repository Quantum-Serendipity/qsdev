# Cloudsmith Upstream Proxying and Caching

- **Source URL**: https://docs.cloudsmith.com/repositories/upstreams
- **Retrieved**: 2026-05-12

## Core Concepts

**Proxying** allows transparent access to packages from upstream repositories, which appear as part of your Cloudsmith repository. **Caching** extends this by permanently storing requested upstream packages in Cloudsmith, protecting against outages and security breaches.

## Indexing Mechanisms

Three approaches:
1. **Ahead-of-Time**: Determines all available packages before upstream is used, ensuring deterministic performance
2. **Just-in-Time**: Learns about packages when first cached, then maintains synchronization
3. **Real-Time**: Queries upstreams on each request (least performant)

## Supported Package Formats (27+)

Alpine, Cargo (Rust), Conda, Composer, CRAN, Dart, Debian, Docker, Generic, Go, Gradle, Helm, Hex, Hugging Face, Maven, npm, NuGet, Python, RedHat (RPM), Ruby, sbt, Swift, Cocoapods, Conan, LuaRocks, OCI, Terraform, Unity, Vagrant.

## Priority and Resolution

Priority determines upstream evaluation order (1..n). Lower numbers = higher priority. Avoid duplicate priorities for optimal performance.

## Authentication and Security

Optional credentials for private upstream repositories. Optional per-request headers. SSL certificate verification recommended for all public sources. Docker Hub requires authentication credentials due to rate-limiting.

## Configuration Methods

- Quick Configure Wizard (pre-configured canonical registries)
- Cloudsmith CLI
- Manual web-based configuration
