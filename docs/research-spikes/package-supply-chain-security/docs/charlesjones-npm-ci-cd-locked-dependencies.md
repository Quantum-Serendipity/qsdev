# Supply Chain Security: Locked Dependencies in npm CI/CD Pipelines

- **Source**: https://charlesjones.dev/blog/npm-supply-chain-attacks-ci-cd-locked-dependencies
- **Retrieved**: 2026-05-12

## The Core Vulnerability

Modern build pipelines relying on `npm install` create a critical security gap. "Every time `npm install` runs, it resolves dependencies within semver ranges, potentially pulling in newer versions that could contain malicious code."

## Real-World Attack Example

The September 2025 incident: Over 180 npm packages compromised through a self-replicating worm mechanism. The malicious code "injected a bundle.js file that downloaded and ran TruffleHog credential scanner on developer machines, exfiltrating GitHub tokens, npm tokens, AWS access keys."

## Solution: npm ci with Locked Dependencies

**1. Generate and commit lock files:**
```bash
npm install
git add package-lock.json
git commit -m "chore: add package-lock.json"
```

**2. Replace npm install with npm ci in CI/CD:**
```bash
# Before
npm install
# After
npm ci
```

## Why npm ci Prevents Attacks

Four protective mechanisms:
- **Exact version enforcement:** Installs precisely what's in package-lock.json, ignoring semver ranges
- **Tampering detection:** Fails if package.json and lock file diverge
- **Audit trails:** Complete cryptographic hash verification in version control
- **Performance:** 2-10x faster than standard installation

## Advanced Defense: pnpm's minimumReleaseAge

The `minimumReleaseAge` setting creates a time-based buffer:

```
# .npmrc
minimumReleaseAge=1440  # 24 hours in minutes
```

## Implementation Checklist

1. Ensure lock files exist in all projects
2. Commit lock files to version control
3. Audit all deployment scripts (Dockerfiles, CI/CD configs, shell scripts)
4. Replace `npm install` with `npm ci` everywhere
5. Add automated security scanning: `npm audit --audit-level=high`
6. Review package-lock.json changes during code reviews

## Addressing Common Concerns

**Security updates:** Should be intentional, not automatic. Use `npm audit` to identify vulnerabilities, update deliberately in development, test thoroughly, and deploy on schedule.

**Lock file sync failures:** When `npm ci` fails due to mismatched files, this is protective -- it prevents builds from succeeding with potentially tampered configurations.

**Other package managers:** Equivalent commands exist: Yarn (`--frozen-lockfile`), pnpm (`--frozen-lockfile`), and Yarn 2+ (`--immutable`).
