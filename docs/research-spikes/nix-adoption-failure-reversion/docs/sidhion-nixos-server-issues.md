# NixOS is a good server OS, except when it isn't
- **Source URL**: https://sidhion.com/blog/nixos_server_issues/
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Daniel Sidhion

## Use Case
NixOS for custom server images for cloud deployment (DigitalOcean), worker machines running microVMs, infrastructure requiring deterministic, reproducible configurations across multiple servers.

## Primary Pain Points

### System Size Issues
Minimal, headless NixOS configurations consumed approximately 900MB — unacceptably large for server use case. Author wanted "a thin, locked-down server with the single purpose of running the software I declare, and not a single extra tool in it."

### Specific Problems
1. **Unnecessary Nix Installation** (~179MB): Complete Nixpkgs source copy included by default through flake registry configurations
2. **Perl & Python Dependencies** (~242MB): Perl through activation scripts and system configuration tools; Python through bootloader installation scripts
3. **Redundant Systemd Variants**: Both `systemd` and `systemd-minimal` present simultaneously due to circular dependencies
4. **Security Infrastructure Overhead**: Security wrappers, FUSE, sudo, and capability-setting binaries designed for interactive systems added unnecessary complexity
5. **Default Assumptions**: "NixOS is built with the perspective of an interactive OS, to be used as daily drivers by humans," making server optimization fundamentally misaligned

## Outcome
**Stayed with NixOS** but abandoned deep optimization. Achieved ~447MB (50% reduction) through disabling Nix, using the perlless profile, and removing udev/lvm/sudo. But explicitly stated: "I concluded that trying to mold NixOS into the shape I wanted just isn't the way to go...doing it on top of NixOS currently feels like a bad path to take."

## Team/Company Context
None evident — individual infrastructure work.
