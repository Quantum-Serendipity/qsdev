<!-- Source: https://tracker.security.nixos.org/ -->
<!-- Retrieved: 2026-05-12 -->

# Nixpkgs Security Tracker

## Core Purpose
Manages vulnerability information for software in Nixpkgs and NixOS, addressing the "record linkage problem" of matching CVE database entries to actual packages.

## Issue Tracking Structure
Four states:

1. **Untriaged suggestions** - automatically generated CVE-to-package matches awaiting review
2. **Dismissed suggestions** - CVEs determined not to affect Nixpkgs
3. **Accepted suggestions** - slated to be published, but might need further refinement
4. **Published issues** - permanent identifiers (NIXPKGS-YYYY-NNNN format) linked to GitHub issues with "1.severity: security" labels

## Stakeholders & Access
- **Contributors**: Nixpkgs committers can edit suggestions
- **Maintainers**: encouraged to check notifications
- **Users**: can subscribe to published vulnerability notifications

## Machine-Readable Access
No documented API or machine-readable endpoints. The tracker is primarily a web interface for the NixOS security team. URL paths like `/suggestions/untriaged/` exist but no API specification is published.

## Technical Details
- Running on Python (82.7%), HTML, Nix, CSS
- Deployed at https://tracker.security.nixos.org
- Funded through Sovereign Tech Fund and Tweag
