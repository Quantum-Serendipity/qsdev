# Verdaccio Package-Filter Plugin

- **Source URL**: https://github.com/verdaccio/verdaccio/blob/8.x/packages/plugins/package-filter/README.md
- **Retrieved**: 2026-05-12

## Purpose

The `@verdaccio/package-filter` plugin controls manifest visibility by removing or replacing package versions matching configurable rules. Built-in filter for Verdaccio 6.x+ that intercepts manifest responses before reaching clients.

## Core Use Cases
- Supply-chain security (blocking malicious packages)
- Version quarantine (hiding recently published code for review periods)
- Date freezes (point-in-time registry snapshots)
- Emergency response (immediate compromise blocking)

## How It Works

Pipeline: Clone manifest -> Apply block/replace rules -> Apply date-based filtering -> Clean up orphaned data -> Recalculate "latest" tag.

"Filtered versions are removed from the manifest metadata only. Tarballs already downloaded or cached are not affected."

## Configuration Options

### Time-Based Filtering

**minAgeDays**: Hide versions published within the last N days.
```yaml
filters:
  '@verdaccio/package-filter':
    minAgeDays: 30
```

**dateThreshold**: Serve only versions published before a specific date.
```yaml
filters:
  '@verdaccio/package-filter':
    dateThreshold: '2024-01-01'
```

When both set, the earlier cutoff wins.

### Block Rules

By scope: `block: [{ scope: '@evilscope' }]`
By package: `block: [{ package: 'malicious-pkg' }]`
By version range: `block: [{ package: '@coolauthor/stolen', versions: '>2.0.1' }]`

### Replace Strategy

Substitute blocked versions with nearest older safe version instead of removal:
```yaml
block:
  - package: '@coolauthor/stolen'
    versions: '>2.0.1'
    strategy: replace
```

### Allow Rules (Whitelisting)

Take precedence over all blocking including age/date thresholds:
```yaml
allow:
  - scope: '@my-company-scope'
  - package: '@coolauthor/not-stolen'
  - package: semver
    versions: '7.7.3'
```

## Manifest Cleanup

Post-filtering automatic cleanup: removes orphaned dist-tags, recalculates "latest" tag, deletes timestamps for removed versions, removes unused _distfiles entries.
