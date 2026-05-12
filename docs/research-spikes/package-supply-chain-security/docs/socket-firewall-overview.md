# Socket Firewall Overview — Socket.dev Documentation

- **Source URL**: https://docs.socket.dev/docs/socket-firewall-overview
- **Retrieved**: 2026-05-12

## Core Functionality

Socket Firewall operates as an intelligent HTTP/HTTPS proxy that sits between package managers and registries. The system works by:

1. **Intercepting requests**: Captures package manager download attempts before installation
2. **Security analysis**: Queries Socket's security API against organizational policies
3. **Enforcement**: Blocks malicious packages at any dependency depth

## Two Deployment Models

**Socket Firewall Free** requires no API key or configuration, operates only in wrapper mode (command prefix), and protects against known malware for JavaScript, Python, and Rust ecosystems on public registries.

**Socket Firewall Enterprise** demands an API key, supports multiple deployment modes (wrapper and service/proxy), covers broader ecosystems (Go, Java, Ruby, .NET), and enables custom security policy configuration.

## Supported Package Managers

- **JavaScript**: npm, yarn, pnpm
- **Python**: pip, uv
- **Rust**: cargo
- **Enterprise adds**: Maven/Gradle (Java), gem/Bundler (Ruby), NuGet (.NET)

## Configuration & Policies

Enterprise deployments enable organizations to customize threat responses including AI-detected malware handling, CVE treatment, and unscanned package policies. The documentation does not detail specific quarantine or age-gating mechanisms, focusing instead on blocking decisions and allow-list overrides for approved packages.

## Integration Points

The system integrates with CI/CD pipelines, supports corporate proxy chaining, and provides dashboard visibility into installation activity and security events.

## Additional Details (from blog post, 2026-05-12)

- **Source**: https://socket.dev/blog/introducing-socket-firewall

Socket Firewall is described as "a free tool that blocks malicious packages at install time." It operates as an ephemeral HTTP proxy that "checks with the Socket API for safety before packages are fetched, extracted, and installed by your package manager," protecting both direct and transitive dependencies.

**Zero-Configuration Setup**: Users install globally via npm (`npm i -g sfw`), then prefix package manager commands with `sfw`. Works immediately without API keys or configuration. Example: `sfw uv pip install flask`.

**Free version limitations**: No custom registry support, AI-detected malware only generates warnings (not blocks), limited to four ecosystems, no allow-listing, cannot configure security policies.

**Enterprise additions**: Custom registry support, configurable blocking policies, extended ecosystem coverage, dashboard access, client/server deployment mode, configurable telemetry.

**Telemetry**: The free version collects anonymous usage data including machine identifiers, blocked package information, latency metrics, and GitHub organization names — never accessing source code or commit history.
