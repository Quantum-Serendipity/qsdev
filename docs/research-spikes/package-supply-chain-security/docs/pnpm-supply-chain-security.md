# Mitigating Supply Chain Attacks — pnpm Documentation

- **Source URL**: https://pnpm.io/supply-chain-security
- **Retrieved**: 2026-05-12

## Core Security Strategies

**Postinstall Script Control**
pnpm v10+ disables automatic execution of `postinstall` scripts by default. Users should employ the `allowBuilds` setting to explicitly permit scripts only from trusted dependencies, rather than using the broader `dangerouslyAllowAllBuilds` option.

**Transitive Dependency Restrictions**
The `blockExoticSubdeps` setting, when enabled, prevents transitive dependencies from sourcing code through unconventional channels like git repositories or direct tarball URLs, ensuring reliance on verified package registries.

**Temporal Release Delays**
The `minimumReleaseAge` setting establishes a waiting period before newly published versions become installable. In pnpm v11, this defaults to "1440 (1 day), meaning newly published packages will not be resolved until they are at least 1 day old." Users can customize this value or disable it entirely.

**Trust Policy Enforcement**
When configured to `no-downgrade`, the `trustPolicy` setting prevents installing packages whose trust credentials have declined relative to earlier versions. The `trustPolicyExclude` and `trustPolicyIgnoreAfter` settings provide exceptions for specific packages or older releases respectively.

**Dependency Locking**
The documentation emphasizes committing lockfiles to repositories to prevent unintended version updates.

## Detection Partnerships

pnpm references security firms including Socket, Snyk, and Aikido that identify compromised packages rapidly after publication.

## Additional Notes (second fetch, 2026-05-12)

- `allowBuilds` in pnpm v10 uses an explicit allowlist model — rather than globally enabling builds, only trusted dependencies are listed
- `blockExoticSubdeps` ensures all transitive dependencies come from trusted registries
- `minimumReleaseAge` in pnpm v11 defaults to 1440 minutes (24 hours)
- `trustPolicy` with "no-downgrade" mode protects against compromised maintainers by preventing trust downgrades
