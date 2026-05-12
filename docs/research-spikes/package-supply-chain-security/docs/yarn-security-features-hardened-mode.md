# Yarn Security Features

- **Source**: https://yarnpkg.com/features/security
- **Retrieved**: 2026-05-12

## Audits

Yarn doesn't execute audits automatically during `yarn install`. Instead, audits should be run as scheduled tasks using `yarn npm audit`. Unlike npm, Yarn's audit implementation focuses on direct dependencies by default, with options to expand scope using `-A,--all` and `-R,--recursive` flags. The `--environment production` flag excludes development dependencies from vulnerability reports.

## Postinstalls

As of version 4.14, Yarn disables postinstall scripts by default for security reasons. Users must explicitly enable them either:
- Globally via `enableScripts: true` in `.yarnrc.yml`
- Per-package using `dependenciesMeta` in `package.json`

## Age Gate

Yarn 4.12 introduced `npmMinimalAgeGate` to restrict installations to packages published at least N days prior. The `npmPreapprovedPackages` setting allows bypassing this check for specific, trusted packages.

## Hardened Mode

Hardened mode automatically activates when Yarn detects execution in a pull request from a public GitHub repository. It can be manually configured via the `enableHardenedMode` setting or the `YARN_ENABLE_HARDENED_MODE` environment variable.

**Key protections:** This mode automatically enables `--check-resolutions` and `--refresh-lockfile` flags during installation, addressing vulnerabilities related to "lockfile poisoning" where malicious modules could be injected through corrupted lockfiles.

**Performance trade-off:** The mode significantly slows installation since Yarn verifies lockfile accuracy against the registry. For multi-job CI pipelines, selectively enabling hardened mode in one job minimizes performance impact.
