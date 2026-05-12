<!-- Source: https://github.com/nix-community/vulnix/blob/master/doc/vulnix-whitelist.5.md -->
<!-- Retrieved: 2026-05-12 -->

# Vulnix Whitelist Documentation

## Overview
Vulnix uses whitelist files to exclude specific program versions from vulnerability reports. Whitelists consist of rules defining matching criteria for derivations.

## Key Components

**Matching Elements:**
- "program": The derivation name portion before the first dash-digit combination
- "version": The numeric part following the dash
- "cve": One or more CVE identifiers

## TOML Format Rules

Three rule types are supported:

1. **Specific version**: `["program-version"]` — Matches exact derivation combinations
2. **Any version**: `["program"]` — Matches all versions of a program (when no more specific rule exists)
3. **Wildcard**: `["*"]` — Catches unmatched derivations (requires `cve` field)

## Available Fields

Each rule can include:

- **cve**: List of CVE identifiers; rule becomes invalid if additional CVEs are discovered
- **until**: Date specification; derivations flagged after this date
- **comment**: Notes explaining the rule (accepts string or list)
- **issue_url**: Tracker link for fix development (accepts string or list)

## Application Logic

Rules apply in decreasing specificity order. Multiple whitelists merge by:
- Treating version-specific and generic rules as distinct
- Concatenating and deduplicating CVE lists
- Converting comments and URLs into consolidated lists

## Legacy Support

YAML format remains available but deprecated, supporting `name`, `version`, `cve`, `comment`, and `status` fields (note: `issue_url` unsupported in YAML).
