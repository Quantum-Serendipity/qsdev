<!-- Source: https://github.com/tiiuae/sbomnix/blob/main/doc/vulnxscan.md -->
<!-- Retrieved: 2026-05-12 -->

# Vulnxscan Documentation

## Overview
Vulnxscan is a command-line tool demonstrating vulnerability scanning using SBOM as input. It primarily targets Nix packages but works with any CycloneDX-formatted SBOM.

## Command-Line Options

**Basic invocation:**
```
vulnxscan [target] [options]
vulnxscan --sbom sbom.cdx.json [options]
```

**Available flags:**
- `--whitelist=<file.csv>` — Apply vulnerability exclusion rules from CSV file
- `--buildtime` — Include buildtime dependencies alongside runtime dependencies
- `--triage` — Query repology.org for version information to classify vulnerabilities
- `--nixprs` — Search GitHub for relevant nixpkgs pull requests (slow due to rate limits)
- `--sbom <file>` — Use CycloneDX SBOM as input instead of Nix target

## Supported Vulnerability Scanners

**Three integrated scanners:**

1. **Vulnix** — Nix-specific scanner using NIST NVD; includes CVE patch auto-detection; produces more false positives through heuristic matching
2. **Grype** — Container-focused scanner supporting CycloneDX SBOM input; uses multiple public vulnerability sources
3. **OSV.py** — Custom OSV client (not Google's official scanner); queries without specifying ecosystem, resulting in potential false positives for Nix

Scanner selection is automatic based on input type. Vulnix is excluded when SBOM is the input source.

## Output Formats

**Console report:** Table displaying:
- Vulnerability ID (vuln_id)
- NVD/OSV URL
- Affected package name and version
- CVSS severity score
- Scanner agreement columns (grype, osv, vulnix, sum)

**CSV output files:**
- `vulns.csv` — Detailed report including whitelisted entries, comments, and analysis columns
- `vulns.triage.csv` — Generated when using `--triage` flag; includes version comparisons and classifications

## Whitelist Configuration

**CSV structure (required columns):**
- `vuln_id` — Regular expression pattern for matching vulnerabilities
- `comment` — Reason for exclusion or analysis notes

**Optional columns:**
- `package` — Strict package name match (narrows scope of rule)
- `whitelist` — Boolean (True/False) to record analysis without excluding findings

**Priority:** Rules listed first take precedence when multiple match.

## Usage Examples

**Scan runtime dependencies:**
```
vulnxscan github:NixOS/nixpkgs/nixos-unstable#git
```

**Scan with whitelist:**
```
vulnxscan github:NixOS/nixpkgs/nixos-unstable#git --whitelist=whitelist.csv
```

**From SBOM file:**
```
sbomnix github:NixOS/nixpkgs/nixos-unstable#git
vulnxscan --sbom sbom.cdx.json
```

**With triage classification:**
```
vulnxscan github:tiiuae/ghaf?ref=main#packages.x86_64-linux.generic-x86_64-release \
  --buildtime --whitelist=manual_analysis.csv --triage
```

## Key Features

- **Patch detection:** Identifies CVE patches in Nix derivations; excluded from reports
- **Multi-scanner consensus:** Sum column shows agreement across scanners
- **Classification categories:** When using `--triage`, vulnerabilities are classified as "fix_update_to_version_nixpkgs," "fix_update_to_version_upstream," or "fix_not_available"
- **Manual analysis tracking:** Non-whitelisting rules allow recording analysis without excluding findings
