# NixOS Security Wiki Page
- **Source**: https://wiki.nixos.org/wiki/Security
- **Retrieved**: 2026-05-12

## Core Nix Security Features

The system includes built-in protections: "effort to avoid rpath collisions across users" through isolated runtime search paths, mandatory multi-user installation preventing privilege escalation for package installation, SHA256-verified packages with GPG signatures, and store path obscurity that complicates some malware discovery methods.

## Supported Security Technologies

**Encryption**: NixOS supports LUKS partition-level disk encryption for data protection.

**Isolation mechanisms** include:
- Flatpak sandboxing (though bundled dependencies create additional risks)
- Linux Containers with unprivileged mode for security
- Docker containers using namespace controls
- Virtual machines offering robust process isolation
- Systemd service hardening via options like `PrivateNetwork=yes`

**Networking**: A stateful firewall is enabled by default, blocking unexpected incoming connections.

## Awaiting Implementation

The wiki identifies three major security technologies still needing proper NixOS integration:
- **Secure Boot** (experimental support via Lanzaboote exists)
- **SELinux** (work revived in 2025 but not yet integrated)
- **AppArmor** (available but unintegrated as of April 2026)

## Additional Resources

The page references complementary tools: `vulnix` for CVE scanning, Spectrum OS for compartmentalization-focused design, and external Linux hardening guides for broader security practices.
