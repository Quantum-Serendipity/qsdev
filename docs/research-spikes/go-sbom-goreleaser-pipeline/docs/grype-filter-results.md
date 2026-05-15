<!-- Source: https://oss.anchore.com/docs/guides/vulnerability/filter-results/ -->
<!-- Retrieved: 2026-05-15 -->

# Grype Vulnerability Filtering Capabilities

## VEX Document Support

Grype supports two VEX formats for filtering vulnerabilities:

1. **OpenVEX**: A "compact JSON format with minimal required fields"
2. **CSAF VEX**: An "OASIS standard" format with comprehensive metadata

Both formats work identically within Grype, following CISA minimum requirements.

## VEX Flag Usage

```bash
grype alpine:latest --vex vex-report.json
grype alpine:latest --vex vex-1.json,vex-2.json
```

Alternatively, specify VEX documents in `.grype.yaml`:

```yaml
vex-documents:
  - vex-report.json
  - vex-findings.json
```

## VEX Status Values

**Automatic filtering statuses:**
- `not_affected` — Product unaffected by vulnerability
- `fixed` — Vulnerability remediated

**Augmenting statuses** (require explicit enablement):
- `affected` — Product affected
- `under_investigation` — Impact assessment ongoing

Enable augmentation via configuration:

```yaml
vex-add: ["affected", "under_investigation"]
```

## Ignore Rules Configuration

Define ignore rules in `.grype.yaml`:

```yaml
ignore:
  - vulnerability: CVE-2008-4318
  - package:
      name: libcurl
  - package:
      name: openssl
      version: 1.1.1g
  - package:
      type: npm
      location: "/usr/local/lib/node_modules/**"
  - vulnerability: CVE-2020-1234
    fix-state: not-fixed
```

Valid fix-state values: `fixed`, `not-fixed`, `wont-fix`, `unknown`

## Core Filtering Flags

```bash
grype alpine:latest --only-fixed
grype alpine:3.10 --only-notfixed
grype alpine:3.10 --ignore-states unknown,wont-fix
grype alpine:3.10 --fail-on high
grype alpine:3.10 --fail-on high --only-fixed
```

## Output Display Options

```bash
# Show suppressed vulnerabilities in table output
grype alpine:3.10 --only-fixed --show-suppressed

# Inspect in JSON output
grype alpine:3.10 --only-fixed -o json | jq '{matches, ignoredMatches}'
```

## VEX Document Creation with vexctl

```bash
vexctl create \
  --product="pkg:oci/alpine@sha256:4b7ce..." \
  --subcomponents="pkg:apk/alpine/[email protected]" \
  --vuln="CVE-2024-58251" \
  --status="not_affected" \
  --justification="vulnerable_code_not_present" \
  --file="vex.json"

grype alpine:3.22.2 --vex vex.json
```

## VEX Justifications (not_affected)

- `component_not_present`
- `vulnerable_code_not_present`
- `vulnerable_code_not_in_execute_path`
- `vulnerable_code_cannot_be_controlled_by_adversary`
- `inline_mitigations_already_exist`

## Matching Strategy

Grype matches VEX statements using container digests (most reliable), image tags, or package-level PURLs. VEX-filtered results behave identically to configuration-based ignore rules in output.
