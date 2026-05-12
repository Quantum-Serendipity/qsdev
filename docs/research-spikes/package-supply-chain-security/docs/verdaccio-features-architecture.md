# Verdaccio: System Architecture and Key Features

- **Source URL**: https://deepwiki.com/verdaccio/verdaccio/1.1-key-features-and-use-cases
- **Retrieved**: 2026-05-12

## Core Features

**Proxy & Uplinks**
Supports multiple uplinks with fallback logic. Implements cache-first strategy: checks local storage, falls back to proxy on cache miss, then caches the response. The `@verdaccio/proxy` component uses HTTP/HTTPS with timeout and retry configuration.

**Caching Behavior**
Minimizes upstream requests through local caching. When packages are requested, the system performs local lookup first, returning manifests immediately if found. Missing packages trigger upstream fetches saved to local storage for future requests.

## Authentication & Authorization

Enforces three permission types: `allow_publish`, `allow_access`, and `allow_unpublish`. Default htpasswd plugin uses bcrypt hashing for credential storage. Supports custom authentication plugins implementing `IBasicAuth` or `IPluginAuth` interfaces.

## Security Features

**Signature Management**
JWT tokens created and verified through `@verdaccio/signature`.

**Supply Chain Considerations**
Audit middleware proxies npm audit requests to the upstream registry, enabling vulnerability scanning integration.

## Infrastructure Support

Supports Docker and Kubernetes deployment.

## Package Filter Plugin

Built-in `@verdaccio/package-filter` plugin (Verdaccio 6.x+) provides:
- `minAgeDays`: Hide versions published within the last N days
- `dateThreshold`: Serve only versions published before a specific date
- Block rules by scope, package name, or version range
- Allow rules (whitelist) that override all blocking
- Replace strategy: substitute blocked versions with nearest older safe version
- Automatic manifest cleanup after filtering

When both minAgeDays and dateThreshold are set, the earlier cutoff wins (more versions are filtered).
