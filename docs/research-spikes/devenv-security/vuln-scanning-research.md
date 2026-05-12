# Vulnerability Scanning Integration for devenv.sh Projects

## Executive Summary

Six vulnerability scanning tools were evaluated for integration with devenv.sh: vulnix, sbomnix/vulnxscan, Trivy, Grype, nix-security-tracker, and flake-checker. The recommended scanning stack for a hardened devenv boilerplate is: **flake-checker** (fast, at commit/pre-push time) + **vulnxscan** (comprehensive, in CI) + **flake-checker + SBOM generation** (scheduled nightly). Trivy is unsuitable for direct Nix scanning and was itself compromised in a supply chain attack in March 2026. The nix-security-tracker has no public API, limiting programmatic integration. A fundamental gap exists: no tool can detect vulnerabilities in language-level packages (Python, Node) within Nix closures without first building a container image.

---

## 1. Vulnix -- Nix-Native CVE Scanner

### How It Works

Vulnix scans Nix store paths, derivations, or system profiles by matching package names and versions against the NIST National Vulnerability Database (NVD). The matching process:

1. Queries `nix-store` for live garbage collection roots or specified derivations
2. Converts `.drv` files into structured objects with name/version properties
3. Downloads and caches CVE data from NIST NVD
4. Matches Nix derivation `pname` and `version` against CVE Common Platform Enumeration (CPE) entries
5. Uses a heuristic: direct name match first, then variations with lowercase and underscore-for-hyphen substitution

**What it scans:**
- Store paths: `vulnix result/`
- Derivations directly: `vulnix -R /nix/store/*.drv`
- Current system: `vulnix --system`
- GC roots: `vulnix --gc-roots`
- User profiles: `vulnix --profile=PATH`
- Derivations from a file: `vulnix --from-file=PATH`
- Transitive dependencies included by default (`--requisites`); disable with `-R`

### False Positive Rate and Allowlisting

**False positive rate is high and well-documented.** The matching heuristic is acknowledged by the authors as "too simplistic." Known examples:
- The Nix `access` derivation matches Microsoft Access CVEs
- Generic package names collide with unrelated vendor products
- No version range matching -- any version match triggers a finding

**Allowlisting (whitelists) is well-supported** via TOML configuration:

```toml
# Allowlist specific CVEs for a specific package version
["openssl-3.0.12"]
cve = ["CVE-2024-0727"]
until = "2024-06-01"
comment = "Mitigated by NixOS patch"
issue_url = "https://github.com/NixOS/nixpkgs/issues/12345"

# Allowlist all CVEs for a package (any version)
["nss"]
comment = "False positive: NSS name collision"

# Allowlist a specific CVE globally
["*"]
cve = ["CVE-2023-99999"]
comment = "Not applicable to our architecture"
```

Whitelists can be loaded from local files or HTTP URLs (`-w` flag, repeatable). The `-W` flag auto-generates a whitelist from current findings. Rules have expiration dates (`until` field) and support issue tracker references.

### Maintenance Status

**Actively maintained.** Current maintainer: @henrirosten (also maintains sbomnix). Release cadence:
- v1.12.3 (Feb 2025) -- latest
- v1.12.0 (Aug 2024)
- v1.11.0 (Mar 2024)
- v1.10.0 (Jul 2023) -- added `--profile` scanning

Six releases in 18 months. Consistent development activity.

### Limitations

1. **Heuristic matching is coarse** -- high false positive rate, no CPE dictionary for Nix packages
2. **No language-level package scanning** -- cannot detect vulnerabilities in Python, Node, Rust packages within Nix closures; only scans Nix derivation-level metadata
3. **Requires Nix daemon access** -- needs `/nix/var/nix/db` readable, which may require permissions configuration in CI
4. **NVD-only data source** -- does not query GitHub Security Advisories, OSV, or vendor-specific feeds
5. **No flake-native interface** -- must point at store paths or derivations, not flake refs directly
6. **Initial NVD download is slow** -- first run downloads entire NVD cache (~5-10 min); subsequent runs use cached data

### devenv.nix Integration Example

```nix
# devenv.nix
{ pkgs, ... }:
{
  packages = [
    pkgs.vulnix
  ];

  scripts.vuln-scan.exec = ''
    echo "Scanning devenv closure for known CVEs..."
    # Build the devenv shell derivation and scan its closure
    DEVENV_DRV=$(nix path-info --derivation .#devShells.x86_64-linux.default 2>/dev/null || echo "")
    if [ -n "$DEVENV_DRV" ]; then
      vulnix "$DEVENV_DRV" \
        -w .vulnix-allowlist.toml \
        --json 2>/dev/null | jq -r '.[] | "\(.pname) \(.version): \(.affected_by | join(", "))"'
    else
      echo "Warning: Could not resolve devenv derivation. Scanning system instead."
      vulnix --system -w .vulnix-allowlist.toml
    fi
  '';

  scripts.vuln-scan-full.exec = ''
    echo "Full vulnerability scan with details..."
    vulnix result/ -w .vulnix-allowlist.toml --json | jq .
  '';

  # Pre-push hook (moderate speed, <30s with cached NVD)
  git-hooks.hooks.vulnix-scan = {
    enable = true;
    name = "vulnix-scan";
    entry = "${pkgs.writeShellScript "vulnix-prepush" ''
      vulnix --gc-roots -w .vulnix-allowlist.toml -R 2>/dev/null
      EXIT=$?
      if [ $EXIT -eq 2 ]; then
        echo "WARNING: Vulnerabilities detected. Review with: devenv shell vuln-scan"
        # Exit 0 to warn but not block (change to exit 1 to enforce)
        exit 0
      fi
    ''}";
    stages = ["pre-push"];
  };
}
```

### Exit Code Semantics (Nagios-compatible)
- `0` -- no vulnerabilities
- `1` -- all findings whitelisted (with `--show-whitelisted`)
- `2` -- active vulnerabilities present
- `3` -- error condition

---

## 2. Sbomnix -- SBOM Generator for Nix

### SBOM Formats

Sbomnix produces SBOMs in three formats:
- **CycloneDX** (`sbom.cdx.json`) -- primary format, used by vulnxscan
- **SPDX** (`sbom.spdx.json`) -- alternative standard
- **CSV** (`sbom.csv`) -- tabular format for spreadsheet analysis

### Generating SBOMs from devenv Closures

```bash
# Generate SBOM for a specific package
nix run github:tiiuae/sbomnix#sbomnix -- github:NixOS/nixpkgs/nixos-unstable#wget

# Generate SBOM from a local flake output (e.g., devenv shell)
nix run github:tiiuae/sbomnix#sbomnix -- .#devShells.x86_64-linux.default

# Include buildtime dependencies (larger SBOM, more comprehensive)
nix run github:tiiuae/sbomnix#sbomnix -- .#devShells.x86_64-linux.default --buildtime

# Generate from an already-built store path
nix run github:tiiuae/sbomnix#sbomnix -- /nix/store/abc123-my-devenv-shell
```

**Runtime vs buildtime dependencies:**
- Runtime (default): transitive set of store paths actually needed at execution time. Requires the target to be built first.
- Buildtime (`--buildtime`): complete closure including compilers and build tools. Only requires derivation evaluation, not building.

For devenv security scanning, **runtime dependencies** are the priority (what actually runs on developer machines), but **buildtime** gives a more complete picture for supply chain compliance.

### Feeding SBOMs into Grype/Trivy

**Grype integration works but with caveats:**

```bash
# Generate CycloneDX SBOM, then scan with Grype
nix run github:tiiuae/sbomnix#sbomnix -- .#devShells.x86_64-linux.default
grype sbom:./sbom.cdx.json

# Or pipe directly
cat sbom.cdx.json | grype
```

**Critical limitation:** Grype matches vulnerabilities using Package URLs (PURLs) and CPEs. Nix is not yet defined in the PURL specification. This means:
- System-level packages (openssl, sqlite, zlib) are detected via CPE matching
- Language-level packages (Python, Node, Rust) are **not detected** because their PURLs use the `pkg:nix/` scheme which vulnerability databases don't recognize
- This is a fundamental gap in the Nix SBOM ecosystem, not a tool-specific bug

**Trivy cannot directly consume sbomnix SBOMs** for Nix-specific scanning (see Section 3).

### Vulnxscan -- The Recommended Approach

Sbomnix includes `vulnxscan`, which orchestrates multiple scanners against Nix targets:

```bash
# Scan a flake output with all available scanners
nix run github:tiiuae/sbomnix#vulnxscan -- .#devShells.x86_64-linux.default

# Scan with allowlist
nix run github:tiiuae/sbomnix#vulnxscan -- .#devShells.x86_64-linux.default \
  --whitelist=allowlist.csv

# Scan from pre-generated SBOM (vulnix excluded in this mode)
nix run github:tiiuae/sbomnix#vulnxscan -- --sbom sbom.cdx.json

# Include buildtime deps + triage classification
nix run github:tiiuae/sbomnix#vulnxscan -- .#devShells.x86_64-linux.default \
  --buildtime --whitelist=allowlist.csv --triage
```

**Vulnxscan integrates three scanners:**
1. **Vulnix** -- Nix-native, NVD matching (higher false positives, better Nix coverage)
2. **Grype** -- CycloneDX SBOM input, multiple vulnerability databases (lower false positives, misses Nix-specific packages)
3. **OSV.py** -- Custom OSV client, queries without ecosystem specification (broad but noisy)

**Output:** Console table + `vulns.csv` with columns for each scanner's findings and a `sum` column showing scanner agreement. Higher `sum` = higher confidence the finding is real.

**Allowlist format (CSV):**
```csv
vuln_id,comment,package,whitelist
CVE-2024-.*openssl.*,Under review by NixOS security team,,False
CVE-2023-99999,False positive: not applicable,nss,True
```

### devenv.nix Integration Example

```nix
# devenv.nix
{ pkgs, ... }:
{
  scripts.generate-sbom.exec = ''
    echo "Generating SBOM for devenv closure..."
    nix run github:tiiuae/sbomnix#sbomnix -- \
      .#devShells.${pkgs.system}.default \
      --cdx sbom.cdx.json --spdx sbom.spdx.json --csv sbom.csv
    echo "SBOM files generated: sbom.cdx.json, sbom.spdx.json, sbom.csv"
  '';

  scripts.vuln-scan-full.exec = ''
    echo "Running multi-scanner vulnerability scan..."
    nix run github:tiiuae/sbomnix#vulnxscan -- \
      .#devShells.${pkgs.system}.default \
      --whitelist=.vulnxscan-allowlist.csv
  '';

  scripts.vuln-triage.exec = ''
    echo "Running vulnerability scan with triage..."
    nix run github:tiiuae/sbomnix#vulnxscan -- \
      .#devShells.${pkgs.system}.default \
      --whitelist=.vulnxscan-allowlist.csv --triage
    echo "Triage output: vulns.triage.csv"
  '';
}
```

---

## 3. Trivy -- Container/Filesystem Scanner

### Can Trivy Scan Nix Store Paths?

**No.** Trivy does not natively support NixOS or Nix store path scanning. The feature was requested in [Issue #1673](https://github.com/aquasecurity/trivy/issues/1673) (Feb 2022) but was closed as stale without implementation. When scanning a NixOS rootfs, Trivy reports: "OS is not detected and vulnerabilities in OS packages are not detected."

**Technical barriers:**
- NixOS's symlink-heavy filesystem structure is not handled by Trivy's OS detection
- No NixOS vulnerability data source integrated into Trivy's database
- NixOS applies patches that NVD entries don't reflect, causing accuracy issues

### Workaround: Container Scanning

Trivy can scan Nix-built container images:

```bash
# Build a container with dockerTools, then scan
nix build .#container
docker load < result
trivy image my-nix-container:latest
```

This works because Trivy recognizes OS packages within container filesystems. However, it still misses Nix-specific packages that don't appear in standard OS package databases.

### Comparison to Vulnix for Nix-Specific Scanning

| Aspect | Trivy | Vulnix |
|--------|-------|--------|
| Nix store scanning | Not supported | Native support |
| NixOS detection | Not supported | Native support |
| Data sources | NVD, vendor feeds, GitHub Advisories, EPSS | NVD only |
| Container scanning | Excellent | Not supported |
| Language packages | Good (in containers) | Not supported |
| False positive handling | Minimal | TOML allowlists with expiry |
| Maintenance | Active (but compromised in 2026) | Active |

### Trivy Supply Chain Compromise (March 2026)

**Critical context for any recommendation involving Trivy:** On March 19, 2026, Trivy itself was compromised in a sophisticated supply chain attack. Attackers (TeamPCP) pushed a malicious v0.69.4 tag that distributed credential-stealing malware. The attack:

- Compromised GitHub Actions tags (75 of 76 version tags in trivy-action)
- Exfiltrated CI/CD secrets, SSH keys, cloud credentials from runner memory
- Planted persistent backdoors and spread a self-propagating worm across npm packages
- Took 3-4 hours to detect despite high-visibility repositories

**Lesson for devenv integration:** Pin all GitHub Actions to immutable SHA hashes, never mutable version tags. This applies to ALL actions, not just Trivy. The compromise demonstrates that security scanning tools are themselves high-value supply chain targets.

### devenv Integration (Limited)

Trivy is only useful for devenv projects that produce container images:

```nix
# devenv.nix -- only for projects using devenv's container generation
{ pkgs, ... }:
{
  packages = [ pkgs.trivy ];

  # Only useful if your project builds containers
  scripts.trivy-scan-container.exec = ''
    if [ -f result ]; then
      echo "Scanning container image..."
      docker load < result
      trivy image --severity HIGH,CRITICAL my-app:latest
    else
      echo "No container image found. Build with: nix build .#container"
      echo "For Nix closure scanning, use vulnxscan instead."
    fi
  '';
}
```

---

## 4. Grype -- Vulnerability Scanner

### SBOM Consumption

Grype accepts multiple SBOM formats:
- **CycloneDX** (JSON and XML)
- **SPDX** (JSON)
- **Syft JSON** (native format)

```bash
# Scan a CycloneDX SBOM from sbomnix
grype sbom:./sbom.cdx.json

# Pipe SBOM directly
cat sbom.cdx.json | grype

# Scan with severity filter
grype sbom:./sbom.cdx.json --only-fixed --fail-on critical
```

### Comparison to Vulnix for Nix Packages

| Aspect | Grype | Vulnix |
|--------|-------|--------|
| Input | SBOM files, containers, filesystems | Nix store paths, derivations |
| Nix awareness | None (generic SBOM consumer) | Native Nix understanding |
| Data sources | NVD, GitHub Advisories, vendor feeds, EPSS/KEV | NVD only |
| PURL matching | Standard PURLs only (not `pkg:nix/`) | Name/version heuristic |
| Language packages | Detects in containers; misses via Nix SBOMs | Does not detect |
| False positives | Lower (more data sources for disambiguation) | Higher (coarse heuristic) |
| Patch detection | No Nix patch awareness | Auto-detects CVE patches in derivations |

### Key Finding: Grype + sbomnix SBOM Has a Fundamental Gap

When Grype consumes a sbomnix-generated CycloneDX SBOM, it can match system-level packages (openssl, zlib, sqlite) via CPE entries but **cannot match Nix-specific packages** because:
1. Nix is not in the PURL specification
2. Grype does not use CPE data from CycloneDX SBOMs for matching (only PURLs)
3. This means a significant portion of the SBOM is invisible to Grype

**Vulnxscan solves this** by running both Vulnix (which understands Nix naming) AND Grype (which has broader vulnerability databases) and merging results.

### devenv Integration

```nix
{ pkgs, ... }:
{
  packages = [ pkgs.grype ];

  scripts.grype-scan.exec = ''
    if [ -f sbom.cdx.json ]; then
      echo "Scanning SBOM with Grype..."
      grype sbom:./sbom.cdx.json --only-fixed --fail-on critical
    else
      echo "No SBOM found. Generate with: devenv shell generate-sbom"
    fi
  '';
}
```

---

## 5. Nix-Security-Tracker

### How It Works

The [Nixpkgs security tracker](https://tracker.security.nixos.org/) is a Django web application that solves the "record linkage problem" -- matching CVEs from the NVD to specific Nixpkgs derivations. The workflow:

1. **CVE ingestion**: `manage ingest_bulk_cve` pulls CVE data from NVD
2. **Automatic matching**: `django-pgpubsub` triggers asynchronous matching of ingested CVEs against Nixpkgs package metadata via database listeners
3. **Triage pipeline**: Matches land as "untriaged suggestions" requiring human review
4. **Publication**: Reviewed vulnerabilities become "published issues" with persistent identifiers (e.g., NIXPKGS-2026-1471) and auto-created GitHub issues for maintainer notification

**Issue states:** Untriaged -> Dismissed | Accepted -> Published (with GitHub issue)

**Technology:** Python/Django, PostgreSQL, `django-pgpubsub` for async messaging, Nix for deployment.

### Is There an API?

**No public REST API exists.** The tracker is a web-first application with HTML views. There is:
- No documented REST API
- No GraphQL endpoint
- No JSON export endpoints visible on the web interface
- No programmatic query interface for external tools

**Workarounds for programmatic access:**
1. Scrape the published issues page at `tracker.security.nixos.org/issues/`
2. Query the GitHub issues auto-created by the tracker in the NixOS/nixpkgs repository
3. Use the tracker's management commands (`manage run_evaluation <commit>`) if running a local instance

### Can Projects Query It for Their Pinned nixpkgs Commit?

**Not directly.** The tracker evaluates specific nixpkgs commits internally (`manage run_evaluation <commit>`) but this is an admin command, not a user-facing query. There is no endpoint like "given commit X, what CVEs affect it."

**Practical alternative:** Use vulnxscan with `--triage` flag, which queries Repology for version information and classifies findings by fixability. This gives similar information (what vulnerabilities exist, whether upstream fixes are available) without needing the tracker's API.

### Integration with devenv

The tracker is not directly integrable as a scanning tool. Its value is as a **data source** that other tools can benefit from indirectly:
- Vulnix uses the same NVD data that feeds the tracker
- The tracker's published issues inform nixpkgs maintainers to apply patches, which then flow into nixpkgs channels that devenv consumes

**Best integration point:** Use flake-checker (Section 6) to ensure your nixpkgs pin is recent enough to include patches prompted by tracker findings.

---

## 6. Flake-Checker (Determinate Systems)

### What It Checks

Three validation checks, all enabled by default:

1. **Supported release branches**: Verifies nixpkgs inputs use officially supported branches (`nixos-25.05`, `nixos-unstable`, etc.). Flags unsupported branches whose release branches stop receiving security updates ~7 months after release.

2. **Currency (recency)**: Ensures nixpkgs inputs have been updated within the last 30 days. Older revisions miss community security patches.

3. **Upstream verification**: Confirms nixpkgs inputs originate from the official `NixOS` GitHub organization. Flags forks or untrusted variants that could introduce vulnerabilities.

### Configuration

```bash
# Run with defaults
nix run github:DeterminateSystems/flake-checker

# Custom policy via CEL (Common Expression Language)
nix run github:DeterminateSystems/flake-checker -- \
  --condition "supportedRefs.contains(gitRef) && numDaysOld < 30 && owner == 'NixOS'"

# Disable specific checks
nix run github:DeterminateSystems/flake-checker -- \
  --check-outdated=false

# Point at specific lock file
nix run github:DeterminateSystems/flake-checker /path/to/flake.lock
```

**CEL variables available:** `gitRef`, `numDaysOld`, `owner`, `supportedRefs`, `refStatuses`

### How to Integrate as a devenv Script or CI Step

```nix
# devenv.nix
{ pkgs, ... }:
{
  scripts.check-flake-health.exec = ''
    echo "Checking flake input health..."
    nix run github:DeterminateSystems/flake-checker -- \
      --no-telemetry \
      --condition "supportedRefs.contains(gitRef) && numDaysOld < 30 && owner == 'NixOS'"
    echo "Flake health check passed."
  '';

  # Fast pre-commit hook (<5s)
  git-hooks.hooks.flake-checker = {
    enable = true;
    name = "flake-checker";
    entry = "${pkgs.writeShellScript "flake-check" ''
      if [ -f flake.lock ]; then
        nix run github:DeterminateSystems/flake-checker -- --no-telemetry 2>&1 || true
      fi
    ''}";
    stages = ["pre-commit"];
    pass_filenames = false;
  };
}
```

### Limitations

- Only checks nixpkgs inputs, not other flake inputs (e.g., custom overlays, private flakes)
- Does not perform CVE scanning -- only checks recency, branch support, and provenance
- Sends telemetry by default (disable with `--no-telemetry` or `FLAKE_CHECKER_NO_TELEMETRY=true`)
- The 30-day default may be too aggressive for projects that intentionally pin older nixpkgs for stability

### Important Note on devenv-nixpkgs

Flake-checker flags non-NixOS-owned nixpkgs forks. **devenv's default package source is `devenv-nixpkgs/rolling`**, a Cachix-maintained fork. Flake-checker will flag this, which is actually a useful security signal -- teams should consciously decide whether to accept Cachix's fork or switch to upstream nixpkgs.

---

## 7. CI Pipeline Examples

### GitHub Actions: Full Security Pipeline

```yaml
name: "Security Scan"
on:
  pull_request:
  push:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # Nightly at 2 AM

jobs:
  flake-health:
    name: "Flake Health Check"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: cachix/install-nix-action@v31
      # Pin to SHA, never mutable tags (lesson from Trivy compromise)
      - uses: DeterminateSystems/flake-checker-action@<pin-to-sha>
        with:
          send-statistics: false

  vuln-scan:
    name: "Vulnerability Scan"
    runs-on: ubuntu-latest
    needs: flake-health
    steps:
      - uses: actions/checkout@v5
      - uses: cachix/install-nix-action@v31
      - uses: cachix/cachix-action@v16
        with:
          name: devenv

      - name: Install devenv
        run: nix profile install nixpkgs#devenv

      # Generate SBOM for compliance/auditing
      - name: Generate SBOM
        run: |
          nix run github:tiiuae/sbomnix#sbomnix -- \
            .#devShells.x86_64-linux.default \
            --cdx sbom.cdx.json --spdx sbom.spdx.json

      # Run multi-scanner vulnerability scan
      - name: Vulnerability Scan (vulnxscan)
        run: |
          nix run github:tiiuae/sbomnix#vulnxscan -- \
            .#devShells.x86_64-linux.default \
            --whitelist=.vulnxscan-allowlist.csv 2>&1 | tee vuln-report.txt

      # Upload artifacts
      - name: Upload Security Artifacts
        uses: actions/upload-artifact@v4
        with:
          name: security-reports
          path: |
            sbom.cdx.json
            sbom.spdx.json
            vuln-report.txt
            vulns.csv

  # Nightly: full scan with triage and buildtime deps
  nightly-deep-scan:
    name: "Nightly Deep Scan"
    if: github.event_name == 'schedule'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: cachix/install-nix-action@v31
      - name: Deep Vulnerability Scan
        run: |
          nix run github:tiiuae/sbomnix#vulnxscan -- \
            .#devShells.x86_64-linux.default \
            --buildtime --whitelist=.vulnxscan-allowlist.csv --triage
      - name: Upload Triage Report
        uses: actions/upload-artifact@v4
        with:
          name: nightly-triage
          path: |
            vulns.csv
            vulns.triage.csv
```

### GitLab CI: Security Pipeline

```yaml
stages:
  - check
  - scan
  - report

variables:
  NIX_CONFIG: "experimental-features = nix-command flakes"

.nix-base:
  image: nixos/nix:latest
  before_script:
    - nix-channel --update

flake-health:
  extends: .nix-base
  stage: check
  script:
    - nix run github:DeterminateSystems/flake-checker -- --no-telemetry
  allow_failure: true  # Warn but don't block

vulnerability-scan:
  extends: .nix-base
  stage: scan
  script:
    # Generate SBOM
    - nix run github:tiiuae/sbomnix#sbomnix --
        .#devShells.x86_64-linux.default
        --cdx sbom.cdx.json --spdx sbom.spdx.json
    # Run vulnerability scan
    - nix run github:tiiuae/sbomnix#vulnxscan --
        .#devShells.x86_64-linux.default
        --whitelist=.vulnxscan-allowlist.csv
  artifacts:
    paths:
      - sbom.cdx.json
      - sbom.spdx.json
      - vulns.csv
    reports:
      # GitLab can consume CycloneDX for its dependency scanning dashboard
      cyclonedx: sbom.cdx.json
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH

nightly-deep-scan:
  extends: .nix-base
  stage: scan
  script:
    - nix run github:tiiuae/sbomnix#vulnxscan --
        .#devShells.x86_64-linux.default
        --buildtime --whitelist=.vulnxscan-allowlist.csv --triage
  artifacts:
    paths:
      - vulns.csv
      - vulns.triage.csv
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
```

### Local Pre-Push Hook (Lightweight)

```nix
# devenv.nix -- lightweight pre-push security checks
{ pkgs, ... }:
{
  git-hooks.hooks = {
    # Fast: flake input health (<5s)
    flake-checker = {
      enable = true;
      name = "flake-health";
      entry = "${pkgs.writeShellScript "flake-health" ''
        if [ -f flake.lock ]; then
          ${pkgs.lib.getExe (pkgs.writeShellScriptBin "fc" ''
            nix run github:DeterminateSystems/flake-checker -- --no-telemetry 2>&1 || true
          '')}
        fi
      ''}";
      stages = ["pre-push"];
      pass_filenames = false;
    };

    # Moderate: quick vulnix scan of GC roots (<30s with cached NVD)
    vuln-quick = {
      enable = true;
      name = "vuln-quick-check";
      entry = "${pkgs.writeShellScript "vuln-quick" ''
        if command -v vulnix &>/dev/null; then
          echo "Quick vulnerability check..."
          vulnix --gc-roots -R -w .vulnix-allowlist.toml 2>/dev/null
          EXIT=$?
          if [ $EXIT -eq 2 ]; then
            echo "WARNING: Known vulnerabilities in GC roots. Run 'devenv shell vuln-scan-full' for details."
          fi
        fi
      ''}";
      stages = ["pre-push"];
      pass_filenames = false;
    };

    # Secrets detection (always fast, <2s)
    ripsecrets = {
      enable = true;
    };
  };
}
```

---

## 8. Scanning Strategy -- What Should Run When?

### At Commit Time (fast, <5s)

| Check | Tool | Purpose |
|-------|------|---------|
| Secret detection | ripsecrets | Prevent credential leaks |
| Flake input health | flake-checker | Ensure nixpkgs is recent, supported, and from NixOS org |
| Shellcheck | shellcheck | Catch shell injection patterns in scripts |

These are **informational/warning only** at commit time to avoid developer friction. They should not block commits.

### At Pre-Push (moderate, <30s)

| Check | Tool | Purpose |
|-------|------|---------|
| Quick CVE scan | vulnix (GC roots, no requisites) | Fast check against cached NVD |
| Flake health | flake-checker | Repeated check (may have changed since commit) |
| License compliance | nix-license-check (custom) | Verify no disallowed licenses in closure |

Pre-push hooks should **warn** but optionally **block** (configurable per team). The vulnix scan with `-R --gc-roots` is fast because it skips transitive dependency resolution.

### In CI (thorough, minutes OK)

| Check | Tool | Purpose |
|-------|------|---------|
| Multi-scanner vuln scan | vulnxscan | Comprehensive scan with Vulnix + Grype + OSV consensus |
| SBOM generation | sbomnix | CycloneDX + SPDX for compliance and auditing |
| Flake health | flake-checker-action | Block merge if nixpkgs is stale or from fork |
| Container scan (if applicable) | Trivy/Grype | Scan Nix-built container images |
| Allowlist audit | custom script | Verify no expired allowlist entries |

CI scans should **block merges** on critical/high severity findings not in the allowlist.

### On Schedule (nightly CVE database refresh)

| Check | Tool | Purpose |
|-------|------|---------|
| Deep scan with triage | vulnxscan --buildtime --triage | Full closure including build deps, with fix availability classification |
| Three-version comparison | Custom (ghafscan pattern) | Compare current pin vs updated nixpkgs vs unstable |
| Allowlist expiry audit | custom script | Flag allowlist entries past their `until` date |
| SBOM refresh | sbomnix | Update SBOM artifacts for compliance dashboard |

Nightly scans catch newly published CVEs against the existing pinned nixpkgs. The ghafscan three-version comparison pattern is particularly valuable: it shows whether updating nixpkgs would fix known vulnerabilities.

### Scanning Tier Summary

```
Commit (pre-commit)          Pre-Push                CI (on PR)              Nightly
========================     ====================    ====================    ========================
ripsecrets        <1s        vulnix (quick)  <30s    vulnxscan     2-5min    vulnxscan --buildtime
flake-checker     <3s        flake-checker   <5s     sbomnix       1-3min      --triage        5-15min
shellcheck        <2s                                flake-checker <5s       3-version compare 10-20min
                                                     container scan 2-5min   allowlist audit   <1min
─────────────────────        ────────────────────    ────────────────────    ────────────────────────
Total: <5s                   Total: <30s             Total: 5-15min          Total: 15-40min
Warn only                    Warn (configurable)     Block on critical       Report + notify
```

---

## Tool Comparison Matrix

| Feature | Vulnix | Vulnxscan (sbomnix) | Trivy | Grype | Flake-Checker | Security Tracker |
|---------|--------|---------------------|-------|-------|---------------|-----------------|
| Nix-native | Yes | Yes | No | No | Yes (flake.lock) | Yes |
| Scans store paths | Yes | Yes | No | No | N/A | N/A |
| Scans flake refs | No (needs path) | Yes | No | No | Yes | Internal only |
| CVE sources | NVD | NVD + GitHub Advisories + OSV | NVD + vendor + GitHub | NVD + GitHub + vendor + EPSS/KEV | N/A | NVD |
| Language packages | No | No | Yes (containers only) | Yes (containers only) | N/A | No |
| SBOM generation | No | Yes (CycloneDX, SPDX) | Yes | No (uses Syft) | No | No |
| Allowlisting | TOML with expiry | CSV with regex | .trivyignore | .grype.yaml | CEL conditions | N/A |
| False positive rate | High | Medium (consensus) | Low (but no Nix support) | Low | N/A | N/A |
| CI integration | Exit codes | Exit codes + CSV | Actions, GitLab template | Actions | Actions, CLI | No public API |
| Speed (cached) | 10-30s | 2-5min | 1-3min | 30s-2min | <5s | N/A |
| Maintenance | Active | Active | Active (compromised 3/2026) | Active | Active | Active |
| devenv fit | Good | Best | Poor (Nix) | Moderate | Excellent | Not integrable |

---

## Key Findings and Recommendations

### 1. Use vulnxscan as the primary CI scanner
Vulnxscan's multi-scanner consensus approach (vulnix + grype + OSV) provides the best balance of coverage and false positive management for Nix targets. It is the only tool that combines Nix-native understanding with broader vulnerability databases.

### 2. Use flake-checker for fast local checks
Flake-checker is the only tool fast enough for commit-time hooks (<5s) and provides critical hygiene checks (nixpkgs recency, provenance, branch support) that no vulnerability scanner covers.

### 3. Do not rely on Trivy for Nix scanning
Trivy does not support NixOS/Nix store scanning and its March 2026 supply chain compromise makes it a cautionary tale. Only use Trivy if your devenv project produces container images and you need to scan those containers.

### 4. The language-package gap is real and unsolved
No tool can detect vulnerabilities in Python, Node, Rust, or other language-level packages within Nix closures via SBOM-based scanning. The only workaround is building a container image and scanning it with Syft+Grype. This is a fundamental limitation of the Nix SBOM ecosystem until Nix is added to the PURL specification.

### 5. Pin GitHub Actions to SHA hashes
The Trivy compromise demonstrated that mutable version tags in GitHub Actions are a critical supply chain vector. All CI pipeline examples should use SHA-pinned actions.

### 6. The nix-security-tracker is not programmatically accessible
Despite being the canonical source for nixpkgs vulnerability data, the tracker has no public API. Its value flows indirectly through nixpkgs maintainer patches. Teams cannot query "is my pinned nixpkgs commit affected by CVE-X" programmatically.

### 7. Ghafscan provides the reference architecture
The [ghafscan](https://github.com/tiiuae/ghafscan) project demonstrates production-grade daily vulnerability scanning for Nix flakes with three-version comparison, allowlist management, and automated reporting. This is the closest thing to a reference implementation for Nix vulnerability scanning CI.

---

## Sources

All raw source material saved to `docs/`:
- `docs/vulnix-readme-full.md` -- vulnix README
- `docs/vulnix-manpage.md` -- vulnix command-line reference
- `docs/vulnix-whitelist-format.md` -- allowlist TOML format
- `docs/vulnix-releases.md` -- release history and maintenance status
- `docs/vulnix-flyingcircus-introduction.md` -- design philosophy and matching algorithm
- `docs/sbomnix-readme-full.md` -- sbomnix README
- `docs/sbomnix-vulnxscan-docs.md` -- vulnxscan usage and scanner integration
- `docs/trivy-nixos-support-issue-1673.md` -- Trivy NixOS support request (closed/stale)
- `docs/trivy-supply-chain-compromise-2026.md` -- March 2026 Trivy compromise details
- `docs/grype-readme.md` -- Grype SBOM consumption and scanning
- `docs/flake-checker-readme-full.md` -- flake-checker checks, configuration, CI integration
- `docs/nix-security-tracker-interface.md` -- tracker web interface and workflow
- `docs/nix-security-tracker-contributing.md` -- tracker architecture and CVE matching process
- `docs/discourse-vuln-scanning-nix-sboms.md` -- community experience with SBOM-based scanning
- `docs/ghafscan-daily-vuln-scanning.md` -- reference implementation for daily Nix vuln scanning
- `docs/devenv-github-actions-integration.md` -- devenv GitHub Actions workflow examples
