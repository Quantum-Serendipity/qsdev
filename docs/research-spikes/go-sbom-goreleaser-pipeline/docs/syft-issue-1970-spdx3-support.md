<!-- Source: https://github.com/anchore/syft/issues/1970 -->
<!-- Retrieved: 2026-05-15 -->

# SPDX 3.0 Support in Syft - Issue #1970

## Status
**Open** -- assigned to "SPDX 3" milestone, in Backlog status. No assigned developer.

## Request
Feature request to implement SPDX 3.0 component properties support, similar to existing CycloneDX functionality. Would allow expressing `pkg.Package.Metadata` as arbitrary properties.

## Key Takeaway
As of May 2026, Syft does NOT support SPDX 3.0 output. It remains in backlog with no timeline. Syft's SPDX output is SPDX 2.3 (spdx-json and spdx-tag-value formats).

This means the practical SPDX version for Go SBOM tooling is 2.3, not 3.0.
