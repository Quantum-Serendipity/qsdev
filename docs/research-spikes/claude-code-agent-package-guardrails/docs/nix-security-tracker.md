<!-- Source: https://github.com/NixOS/nix-security-tracker -->
<!-- Retrieved: 2026-05-12 -->

# Nix Security Tracker

## Core Purpose
A web service for managing information on vulnerabilities in software distributed through Nixpkgs. Intended to help with solving the record linkage problem of matching packages in the CVE database and Nixpkgs.

## Primary Audiences
- **NixOS security team**: Review CVEs and associate them with affected packages
- **Nixpkgs maintainers**: Receive vulnerability notifications for their packages
- **Nixpkgs users**: Subscribe to alerts for packages of interest

## Deployment
The service operates at https://tracker.security.nixos.org

## Development History
- **2023**: Prototype funded through Sovereign Tech Fund's "Contribute Back Challenge"
- **2024**: Production deployment delayed; team completed demo to NixOS security leadership
- **2025**: Continued development funded by Tweag; actively used by security team to publish issues

## Technical Stack
- Python (82.7%)
- HTML (7.2%)
- Nix (6.7%)
- CSS (3.1%)

## Current Status
The security team has begun productive use, publishing and addressing numerous security issues within the Nixpkgs repository, indicating operational deployment.

## Limitations
No documented public API for programmatic access. Appears to be primarily a web interface for the NixOS security team rather than a service that can be queried from hooks or automation.
