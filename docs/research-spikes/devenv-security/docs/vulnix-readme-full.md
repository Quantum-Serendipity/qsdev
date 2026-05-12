<!-- Source: https://raw.githubusercontent.com/nix-community/vulnix/master/README.rst -->
<!-- Retrieved: 2026-05-12 -->

# Vulnix: Nix(OS) Vulnerability Scanner - Full README

## Overview
This utility validates Nix store packages for potential vulnerabilities by cross-referencing against the National Institute of Standards and Technology (NIST) National Vulnerability Database (NVD).

The tool "pulls all published CVEs from NIST and caches them locally," matching package names and versions against known security advisories. It supports both CLI inspection and Sensu monitoring integration.

## System Requirements
- Common Nix utilities like `nix-store` in PATH
- Access to Nix store database at `/nix/var/nix/db`
- Nix version 1.10 or later (including 2.x)
- Locale settings configured (e.g., `LANG=C.UTF-8`)

## Command-Line Usage

**Check current system vulnerabilities:**
```shell
vulnix --system
```

**Scan build output with dependencies:**
```shell
vulnix result/
```

**Analyze derivations without requisite determination:**
```shell
vulnix -R /nix/store/*.drv
```

**Machine-readable output:**
```shell
vulnix --json /nix/store/my-derivation.drv
```

## Whitelist Configuration
Whitelists filter results using TOML format files from local or remote sources:

```shell
vulnix -w /path/to/whitelist.toml -w https://example.org/whitelist.toml
```

**Whitelist options include:**
- `cve`: List of CVE identifiers to exclude
- `until`: Expiration date (YYYY-MM-DD format)
- `issue_url`: Issue tracker reference
- `comment`: Free-form documentation

## CVE Patch Auto-Detection
The scanner automatically recognizes patches addressing specific CVEs when filenames contain CVE identifiers, suppressing those vulnerabilities from reports.
