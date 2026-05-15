<!-- Source: https://github.com/goreleaser/goreleaser/issues/2597 -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned limited content from this GitHub issue page. -->

# GoReleaser SBOM Generation Proposal - Issue #2597

## Original Request

The proposal, submitted by VinodAnandan on October 21, 2021, requests that GoReleaser publish "Software Bill of Materials" artifacts as part of each release.

## Metadata
- **Issue**: #2597
- **Status**: Closed
- **Label**: Enhancement
- **Milestone**: v1.2.0
- **Proposer**: VinodAnandan

The issue references the CycloneDX project as an example, specifically pointing to how cyclonedx-gomod implements SBOM generation for Go modules.

## Implementation
This proposal was implemented in PR #2648 by wagoodman (Alex Goodman from Anchore/Syft team), merged December 12, 2021, and released in GoReleaser v1.2.0.

## Timeline
- October 21, 2021: Proposal filed
- December 12, 2021: PR #2648 merged
- Late December 2021 / Early January 2022: Released in v1.2.0
