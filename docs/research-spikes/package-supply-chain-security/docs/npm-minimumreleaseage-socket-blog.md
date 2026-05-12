# npm Introduces minimumReleaseAge and Bulk OIDC Configuration — Socket.dev Blog

- **Source URL**: https://socket.dev/blog/npm-introduces-minimumreleaseage-and-bulk-oidc-configuration
- **Retrieved**: 2026-05-12

## Overview

npm introduced `minimumReleaseAge` in CLI 11.x releases, allowing teams to enforce a delay before newly published package versions can be installed. This feature aims to reduce exposure to malicious packages by preventing rapid automated consumption before detection.

## How It Works

The setting creates a time-based gate that blocks installation of packages below a specified age threshold. "By enforcing a minimum age threshold before a version can be installed, the feature reduces exposure to malicious packages that rely on rapid, automated consumption before detection or takedown."

## Key Limitations

Unlike pnpm's implementation, npm's initial version lacks built-in exclusion mechanisms. This creates operational friction for teams managing mixed dependency portfolios. "The current implementation conflicts with a common workflow: being strict with third-party dependencies while remaining more lenient with internally maintained packages, since there is no way yet to exclude internal packages from the cooldown."

An open GitHub issue proposes adding flexible exclusions to address this gap.

## Ecosystem Context

All major Node.js package managers now offer release-age gating:
- **pnpm**: v10.16+ with exclusion rule support
- **Yarn**: 4.10.0+ with npmMinimalAgeGate
- **Bun**: v1.3+ with configurable minimumReleaseAge

This represents rapid ecosystem convergence on defensive install controls within a single year.
