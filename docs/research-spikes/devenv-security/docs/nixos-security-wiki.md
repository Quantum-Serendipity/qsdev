<!-- Source: https://wiki.nixos.org/wiki/Security -->
<!-- Retrieved: 2026-05-12 -->

# NixOS Security Wiki Summary

## Core Security Features

**Isolation of Runtime Search Paths**: NixOS actively works to prevent rpath collisions across users, reducing potential privilege escalation vectors.

**Multi-User Installation**: "NixOS is automatically installed in Multi-User mode," enabling isolated store environments where users can install applications without requiring root access through delegated build users.

**Cryptographic Verification**: Installation resources include "SHA256 checksums which are GPG signed by the Nix team" for authenticity verification, with all packages validated against local checksums traceable to signed materials.

**Store Obscurity**: The Nix store replaces typical Linux filesystem hierarchy, potentially frustrating malware expecting tools in standard locations—though this represents only minor security through obscurity.

## Supported Security Technologies

The wiki documents support for:
- **LUKS encryption** for disk-level protection
- **Process isolation** via Flatpak, Linux Containers, Docker, and virtual machines
- **Systemd service hardening** with options like `PrivateNetwork=yes`
- **Stateful firewall** enabled by default, blocking unexpected incoming connections

## Sandboxing & Isolation Mechanisms

### Flatpaks
Described as "sandboxed" applications requiring explicit permission declarations for access beyond their own paths. However, the wiki cautions that bundled dependencies introduce security risks, and most application flatpaks "do not make meaningful use of the sandbox."

### Linux Containers (LXC)
Presented as chroot environments with resource constraints (cgroups). Unprivileged containers are necessary to prevent container root from becoming system root.

### Docker Containers
Implemented similarly to LXC on Linux, using "namespacing controls" similar to unprivileged containers by default.

### Virtual Machines
Described as "generally one of the most robust tools available for process isolation," though with performance overhead.

## Security Gaps

The wiki identifies three major unresolved areas:

1. **Secure Boot**: Development ongoing; experimental Lanzaboote implementation available
2. **SELinux**: "Proper integration does not exist" despite being technically possible
3. **AppArmor**: Available but "not yet been properly integrated"
