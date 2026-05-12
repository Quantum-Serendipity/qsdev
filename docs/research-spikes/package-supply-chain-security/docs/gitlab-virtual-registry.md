# GitLab Virtual Registry

- **Source URL**: https://docs.gitlab.com/user/packages/virtual_registry/
- **Retrieved**: 2026-05-12

## Status & Availability

Generally available since GitLab 18.10. Available for Premium and Ultimate tier users.

## Core Functionality

Serves as a unified proxy allowing organizations to configure applications to use one virtual registry instead of multiple upstream registries. Consolidates access to multiple external package repositories behind a single endpoint.

### Key Capabilities
- Proxies and caches packages from multiple upstream sources
- Supports up to 20 upstream registries per virtual registry
- Handles both public and private external registries
- Stores credentials within upstream configurations (not in package manager settings)

## Supported Package Formats
- Maven packages
- Container images

## Authentication Methods

Requires tokens with `api` or `read_virtual_registry` scopes:
- Personal access tokens
- Group deploy tokens (top-level group)
- Group access tokens (top-level group)
- CI/CD job tokens
- OAuth 2.0 tokens

## Caching System

1. **Request Caching**: Stores responses for identical requests
2. **Priority Walking**: On cache miss, traverses upstream list in priority order
3. **Cache Validity Periods**: Configurable timeframes (default 24 hours, adjustable 0-365 days)

Checks if upstream response is identical to cache before refreshing expired entries. Cached content stored in object storage's `dependency_proxy` bucket, counts toward group storage limits.

## Upstream Registry Management

- Registries ordered by priority (queried sequentially)
- Private registries should be placed before public ones
- Higher-priority placement for registries with larger package collections
- Public registries as fallback entries

## Cleanup Policies

Scheduled jobs that identify cached entries unused for a specified retention period (1-365 days, default 7). Execute on configurable cadences (daily/weekly/monthly).

## Limitations

- Only supports Maven and container images currently
- Premium/Ultimate tier required
- Users must be direct members of the top-level group
