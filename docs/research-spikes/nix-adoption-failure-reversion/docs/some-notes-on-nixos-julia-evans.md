<!-- Source: https://jvns.ca/blog/2024/01/01/some-notes-on-nixos/ -->
<!-- Retrieved: 2026-03-20 -->

# Some notes on NixOS

**Author:** Julia Evans
**Date:** January 1, 2024

## Motivation for NixOS
Evans previously used Ansible for server provisioning but made ad-hoc changes that created an unmaintainable system. She wanted a more reliable approach for managing a server running personal Go services.

## Why NixOS Over Ansible
She found NixOS advantageous because: "NixOS is the operating system. It has full control over all your users and services and packages." This comprehensive control prevents configuration drift better than Ansible's partial system management.

## Implementation Steps
1. Installation: Used nixos-infect to convert a Hetzner Ubuntu instance to NixOS
2. Configuration management: Copied generated Nix files to a local Git repository
3. Flakes: Created a flake.nix wrapper, noting the counterintuitive requirement to git add files without committing them
4. Deployment: Used nixos-rebuild switch with remote build/target host options
5. SSH setup: Configured agent forwarding for private Git repository access
6. Service configuration: Built a single-file approach combining Go package definition and systemd service setup using Caddy as reverse proxy

## Pain Points & Confusions

### Nix language syntax
Evans acknowledged struggling with Nix syntax, admitting: "I still don't really understand the nix language syntax that well" and planned to copy-paste templates indefinitely.

### Debugging difficulties
She encountered cryptic errors like fetchTree requires a locked input, which required Discord assistance to resolve. She discovered that Nix truncates stack traces and caches errors, making diagnosis frustrating.

### Outstanding questions
She remained uncertain about what nixos-rebuild validates and how to streamline deployment workflows for updated service versions.

## What Worked Well
- DynamicUser & StateDirectory: Appreciated systemd features for creating service-specific users and persistent directories without boilerplate
- All-in-one configuration: Preferred consolidating service definition and configuration in single files
- Caddy integration: Used Caddy's automatic Let's Encrypt setup with custom configuration rather than learning Nix-specific syntax

## Overall Assessment
Despite difficulty with Nix's complexity, Evans found the approach "more reliable than the approach I was taking with Ansible." She remained cautiously optimistic after one week, comparing Nix engagement to missing "linux evenings," though she expressed uncertainty about long-term adoption.
