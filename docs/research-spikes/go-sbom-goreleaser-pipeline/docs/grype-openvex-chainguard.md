<!-- Source: https://www.chainguard.dev/unchained/vexed-then-grype-about-it-chainguard-and-anchore-announce-grype-supports-openvex -->
<!-- Retrieved: 2026-05-15 -->

# Grype's OpenVEX Support: Chainguard & Anchore Collaboration

## How It Works

Grype now consumes OpenVEX documents — machine-readable statements about software vulnerability status. This integration allows organizations to enrich scan findings and manage false positives.

## VEX Statements & Filtering

VEX statements operate as "assertions in time describing how the vulnerability affects (or not) a piece of software." Each statement requires:
- A container image (VEX product)
- Affected components (subcomponents like glibc-2.36-r3)
- A CVE identifier
- One of four status flags: `not_affected`, `affected`, `under_investigation`, or `fixed`
- When marking as `not_affected`, a predefined justification (e.g., `vulnerable_code_not_in_execute_path`)

## CLI Usage

```bash
grype [image] --vex-document [path-to-vex-document]
```

Suppressed vulnerabilities remain available in Grype's ignore list via `--output=json`.

## Industry Collaboration

Both Chainguard and Anchore jointly announced this feature and support OpenVEX through the Open Source Security Foundation, alongside Intel and Microsoft, promoting industry-wide adoption of vulnerability management standards.
