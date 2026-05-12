# Azure Artifacts: Safeguard Against Malicious Public Packages

- **Source URL**: https://learn.microsoft.com/en-us/azure/devops/artifacts/concepts/upstream-behavior?view=azure-devops
- **Retrieved**: 2026-05-12

## Allow External Versions Feature

Controls whether package versions from public registries (NuGet.org, npmjs.com) can be consumed. By default, this option is **disabled**, adding security by reducing exposure to potentially malicious packages from public registries.

When enabled for a specific package, versions from the public registry become available to be saved to the feed. Changing this setting does not affect package versions already saved to the feed.

Must be a **Feed Owner** to enable.

## Supported Package Types for Upstream Control

- NuGet
- npm (including scoped packages)
- Python (PyPI)
- Maven
- Cargo

## Key Security Scenarios

### Public Versions Blocked
- If a private package is later made public, the feed blocks new versions with the same name from public sources
- When a team uses both private and public packages, the feed blocks new public versions

### Public Versions Allowed
- All packages are private (no impact)
- All packages are public (no impact)
- Public package made private (no impact)

## Configuration

Available via Azure DevOps UI (per-package toggle) and REST API (per-package type endpoints). Changes may take up to 3 hours to propagate.

## Pricing

Every organization gets **2 GiB free storage**. Additional storage: tiered rates from $2/GiB decreasing to $0.25/GiB at scale. Unlimited users at no extra charge.

## Supported Upstream Sources

NuGet (.NET), npm (Node.js), Maven (Java), Python (PyPI), Cargo (Rust), Universal Packages.

## Limitations

- No built-in vulnerability scanning or malware detection
- No age-gating or quarantine features
- Security is limited to the allow/block external versions mechanism
- Per-package configuration can be tedious at scale
