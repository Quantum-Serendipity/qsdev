# Nix-Security-Tracker README
- **Source**: https://github.com/NixOS/nix-security-tracker/blob/main/README.md
- **Retrieved**: 2026-05-12

## Purpose
The Nixpkgs security tracker is a web service designed to manage vulnerability information for software distributed through Nixpkgs. It addresses security gaps in the Nix ecosystem by centralizing CVE tracking and remediation.

## Key Audiences
The platform serves three distinct groups:
- **NixOS security team**: Reviews incoming CVEs and links them to affected packages
- **Nixpkgs maintainers**: Receives vulnerability notifications for their maintained packages
- **Nixpkgs users**: Can subscribe to notifications for packages of interest

## Deployment
The service is deployed at https://tracker.security.nixos.org and became operationally active in 2025 after initial development delays.

## Project Timeline
- **2023**: Prototype funded through Sovereign Tech Fund's "Contribute Back Challenge"
- **2024**: Production deployment delayed; demo completed by year-end
- **2025**: Resumed development with Tweag sponsorship; NixOS security team began active use, publishing numerous security issues

## Current Status
The security team is productively using the tracker to identify and address vulnerabilities, demonstrating its effectiveness as critical infrastructure for the Nix ecosystem's robustness.

## Support
Financial backing comes from the Sovereign Tech Agency, acknowledging the importance of securing this foundational open-source infrastructure.
