# Go Module Checksum Database: Technical Design Proposal
- **Source**: https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md
- **Retrieved**: 2026-05-12

## Overview

The proposal introduces a transparent log called the Go checksum database (hosted at `https://sum.golang.org/`) to authenticate public Go module downloads. Rather than trusting initial downloads, the `go` command verifies module versions against this cryptographically-secured database.

## Merkle Tree & Transparent Log Structure

The system employs a transparent log — a data structure enabling clients to verify records exist within it while preventing the server from removing entries retroactively. The database serves content in "tile" format:

- `/tile/H/L/K[.p/W]` endpoints provide log tiles containing multiple leaf hashes
- Tiles support partial variants with `.p/W` suffixes for bandwidth optimization
- Data tiles at `/tile/H/data/K[.p/W]` serve the actual `go.sum` entries corresponding to leaf hashes

This architecture makes caching efficient while obscuring lookup patterns from the server.

## Verification Protocol

The `go` command performs verification through these steps:

1. Fetches `/latest` for a signed tree size and hash
2. Queries `/lookup/M@V` for a specific module version, receiving log position, data, and authenticated tree hash
3. Validates the tree hash cryptographically
4. Maintains a local timeline of observed tree states to detect inconsistencies

The client caches signed tree heads at `$GOPATH/pkg/sumdb/<sumdb-name>/latest`, preserving them across `go clean -modcache` to detect fork attacks.

## Configuration Variables

**Privacy and security controls:**

- `GOSUMDB=<verifier-key>` specifies the database and its public key
- `GONOSUMDB=prefix1,prefix2` excludes module patterns from database verification (for private modules)
- `GOPROXY` and `GONOPROXY` control proxy usage independently from checksum verification
- `GONOSUMDB=*` disables the database entirely (not recommended)

Critically, database unavailability causes build failures unless explicitly disabled, preventing silent downgrade attacks.

## Integration with go.sum

The proposal distinguishes two authentication phases:

- **Future builds**: Authenticated by existing `go.sum` entries
- **Initial downloads**: Authenticated by the checksum database for new dependencies

When adding or upgrading dependencies, the `go` command fetches corresponding entries from the database before recording them in `go.sum`. This eliminates the "trust on first use" weakness where initial downloads were unverified.

## Threat Model & Attack Prevention

**Attacks addressed:**

- **Man-in-the-middle substitution**: Transparent logging forces consistent serving or detection via consistency checks
- **Forked log attacks**: Multiple proxies and gossip mechanisms make victim-specific log forks unsustainable
- **Silent substitution by code hosts**: Moving GitHub, GitLab, and other hosts outside the trusted computing base
- **Proxy compromises**: Proxies no longer need trust; their content is cryptographically verified

**Design principles prevent these attacks by:**

- Requiring all entries be permanently logged (auditors can detect violations)
- Using transparency for detection rather than prevention
- Enabling multiple organizations to operate proxies without creating new trust dependencies
- Making victim-identification during fork attacks computationally impractical with tile-based serving

## Privacy Mechanisms

1. **Private module path exposure**: Misconfigured lookups fail loudly, forcing corrections via `GONOSUMDB` configuration
2. **Public module usage signals**: Database contact occurs only when updating `go.sum`; results are cached to minimize repeated lookups
3. **Proxy/bulk download alternatives**: Organizations can route through proxies or download complete copies

## Proxy Support

Proxies can optionally proxy database endpoints using the pattern `<proxyURL>/sumdb/<databaseURL>/`. Before proxying, clients check `<proxyURL>/sumdb/<sumdb-name>/supported`. This allows corporate proxies to enforce privacy while simpler proxies remain stateless.

## Implementation

Implemented in Go 1.13 with the checksum database enabled by default. The system is maintained by Google as an ecosystem service.
