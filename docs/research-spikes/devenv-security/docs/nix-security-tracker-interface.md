<!-- Source: https://tracker.security.nixos.org/ -->
<!-- Retrieved: 2026-05-12 -->

# Nixpkgs Security Tracker - Interface & Features

## Available Pages/Sections
- **Untriaged suggestions**: automatically generated matches between a CVE Record and Nixpkgs derivations
- **Dismissed suggestions**: CVEs classified as not affecting Nixpkgs
- **Accepted suggestions**: Vulnerabilities pending publication but potentially requiring edits
- **Published issues**: Persistent identifiers linked to GitHub issues where maintainers are notified and mitigation is coordinated
- **Notifications page**: For maintainers and subscribers

## Data & Linking Mechanism
The core function addresses "the record linkage problem of matching packages in the CVE database and Nixpkgs." The system automatically matches CVE records to Nixpkgs package derivations, then routes them through human review stages.

## User Roles & Access
- **Nixpkgs committers** can edit suggestions
- **Maintainers** receive notifications about relevant packages
- **Users** can subscribe to vulnerability notifications

## Technology Stack (from GitHub)
- Python (82.7%), HTML (7.2%), Nix (6.7%), CSS (3.1%)
- MIT license
- Funded: Sovereign Tech Fund (2023), Tweag (2025+)

## API
No explicit REST API documentation visible. No query interface for programmatic access documented on the web interface or GitHub README. The tracker is primarily a web dashboard for the NixOS security team workflow, not an API-first service.
