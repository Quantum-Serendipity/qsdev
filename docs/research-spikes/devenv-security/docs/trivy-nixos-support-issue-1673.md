<!-- Source: https://github.com/aquasecurity/trivy/issues/1673 -->
<!-- Retrieved: 2026-05-12 -->

# NixOS Support in Trivy - Issue #1673

## Original Request
User @06kellyjac opened this feature request on February 3, 2022, seeking support for scanning NixOS systems. Trivy failed to detect the operating system when scanning a NixOS rootfs, resulting in: "OS is not detected and vulnerabilities in OS packages are not detected."

## Technical Challenges Identified

1. **Vulnerability Data Sources**: Unlike Alpine, Amazon, and RHEL (which have official vulnerability trackers), NixOS would need to rely on NVD data. This approach has accuracy limitations since NixOS applies patches that NVD entries don't reflect.

2. **Symlink Handling**: NixOS heavily uses symlinks and it was questioned whether Trivy follows symbolic links during scanning. This functionality might need to be optional or NixOS-specific.

3. **Existing Alternatives**: The only comparable tool mentioned was Vulnix, which exclusively uses NVD data.

## Current Status

- **Status**: Closed
- **Labels**: "kind/feature" and "lifecycle/stale"
- No implementation visible
- Closed as stale without NixOS scanning being implemented

## Conclusion
Trivy does not natively support NixOS/Nix store path scanning. The issue was closed without resolution, meaning NixOS scanning remains unsupported in Trivy's core.
