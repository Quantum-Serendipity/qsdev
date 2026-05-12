# NixOS Discourse: Nix State of the SBOM
- **Source**: https://discourse.nixos.org/t/nix-state-of-the-sbom/73629
- **Retrieved**: 2026-05-12

## Current Tools and Capabilities

Several SBOM generation tools exist for Nix, including **Genealogous** and **bombon**. Both operate at the "Nix-level" rather than on derivations. Determinate Systems is also developing internal SBOM tooling for packages in their supported Nixpkgs subset, with plans to "publish our tooling in the next few months."

## Identified Gaps

**String Context Issues**: Tools struggle to access string contexts from Nix code, causing missing dependency detection. An upstream Nix issue (#4677) exists to improve this functionality.

**Metadata Deficiencies**: Major limitations include:
- Lack of package type classification (firmware, drivers, libraries, applications, etc.)
- Missing vulnerability and patch metadata
- Insufficient provenance information

**Artifact Duplication**: "It's ~impossible to automatically detect duplication/copies generally, since an arbitrary mutation can be performed," though marking and tagging could help.

## Community Perspectives

**Package Types**: One contributor noted that while SBOM formats require type properties, "most SBOM tools mostly ignore it," though accurate classification helps detect things more reliably.

**Vulnerability Handling**: The community debates whether `meta.knownVulnerabilities` belongs in SBOMs. Long-term support scenarios favor `meta.remediatedCVEs` instead, allowing distributions to handle legacy versions differently than rolling releases.

## Future Plans

- Enhanced metadata attributes for package types and vulnerability information
- Lock file integration for better dependency tracking
- Presentations scheduled at FOSDEM and Planet Nix conferences
