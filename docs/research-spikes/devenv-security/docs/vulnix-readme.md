# Vulnix: Nix/NixOS Vulnerability Scanner
- **Source**: https://github.com/nix-community/vulnix
- **Retrieved**: 2026-05-12

## Overview
Vulnix is a security utility that identifies potentially vulnerable packages in Nix store environments. It "validates a Nix store for any packages that are reachable from live paths and likely to be affected by vulnerabilities listed in the NVD."

## Core Functionality

**CVE Detection Method:**
The tool pulls published vulnerabilities from NIST and matches Nix package names and versions against known CVE entries. Currently, matching relies on a straightforward heuristic: direct name matching first, then variations using lowercase letters and underscores replacing hyphens.

## System Requirements

Vulnix needs:
- Common Nix tools like `nix-store` in the PATH
- Access to the Nix store database at `/nix/var/nix/db`
- Nix version 1.10 or higher
- Proper locale settings (LANG environment variable)

## Command-Line Usage

Common invocations include:
- `vulnix --system` - scan current system
- `vulnix result/` - check build output and dependencies
- `vulnix -R /nix/store/*.drv` - check derivations without resolving requisites
- `vulnix --json` - generate machine-readable output

## Whitelist Feature

Users can exclude packages through TOML configuration files, specifying:
- Particular package versions or all versions
- Associated CVEs to filter
- Expiration dates (`until` field)
- Issue tracker references and comments

Whitelists load from local files or HTTP sources via the `-w` flag.

## Patch Auto-Detection

The scanner recognizes CVE identifiers embedded in patch filenames and automatically excludes matching vulnerabilities from reports.

## Limitations

The matching heuristic between Nix packages and NVD products is acknowledged as "too simplistic" and requires future improvement. The tool also depends on proper Nix daemon setup or appropriate user permissions.
