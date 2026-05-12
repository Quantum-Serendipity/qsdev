<!-- Source: https://github.com/tiiuae/ghafscan -->
<!-- Retrieved: 2026-05-12 -->

# Ghafscan: Automated Daily Vulnerability Scanning for Nix Flakes

## Project Overview
Ghafscan automates vulnerability scanning for the Ghaf Framework, a security-focused operating system. Demonstrates "running automatic vulnerability scans for nix flake projects" with daily-updated reports across multiple hardware targets.

## How It Works

**Scanning Tool**: Leverages vulnxscan (from sbomnix), which scans Nix store paths without requiring builds. Enables rapid vulnerability detection for uncompiled flake outputs.

**Three-Version Comparison Strategy**: Compares vulnerabilities across:
1. **Current**: The production pinned version
2. **Lock Updated**: After running `nix flake lock --update-input nixpkgs`
3. **Nix Unstable**: Using the unstable channel instead

This identifies:
- Fixes available upstream but not yet integrated
- Patches in nix-unstable awaiting backport to current release channel

## CI/CD Pipeline
GitHub Action runs on a daily schedule. Reports auto-generated and committed to the repository for selected branches and targets:
- System76 Darp11 (x86_64-linux)
- Lenovo X1 Carbon Gen11 (x86_64-linux)
- NVIDIA Jetson Orin NX (aarch64-linux)

## Reporting
Results include manual analysis data merged with automated scans, organized as markdown reports with vulnerability categorization by fixability and timeline.

## Technologies
Python (75.5%), Nix (20.9%), Makefile (2.9%), Shell scripts. Apache-2.0 license.
