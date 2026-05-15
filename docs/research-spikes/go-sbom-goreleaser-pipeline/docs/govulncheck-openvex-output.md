<!-- Source: https://pkg.go.dev/golang.org/x/vuln/internal/openvex + https://github.com/golang/go/issues/62486 -->
<!-- Retrieved: 2026-05-15 -->

# Govulncheck OpenVEX Output

## Overview

Govulncheck supports the Vulnerability EXchange (VEX) output format via `-format openvex`, following the OpenVEX specification at https://github.com/openvex/spec.

## VEX Document Structure

Package vex defines the Vulnerability EXchange Format (VEX) types supported by govulncheck. OpenVEX documents are minimal JSON-LD files that capture the minimal requirements for VEX as defined by the VEX working group organized by CISA.

## Document Components

The OpenVEX output from govulncheck includes:
- A top-level struct with author, timestamp, version, tooling, and statements
- Each OSV emitted by govulncheck has a corresponding statement
- Status is either "not_affected" or "affected"
- If status is not_affected, justification is "vulnerable_code_not_in_execute_path" (the VEX justification that best matches govulncheck's reachability filtering)

## Key Insight

This means govulncheck can directly produce VEX documents that complement SBOMs. A Go project can:
1. Generate an SBOM (via syft/cyclonedx-gomod)
2. Run `govulncheck -format openvex ./...`
3. Ship the VEX alongside the SBOM

Consumers scanning the SBOM with grype or trivy can then use the VEX to automatically suppress vulnerabilities that govulncheck determined are not reachable through the call graph.

## Exit Behavior

Govulncheck exits successfully (exit code 0) when `-format openvex` is provided, regardless of detected vulnerabilities. This makes it safe for CI pipeline usage where the VEX document is the desired output rather than a pass/fail gate.

## Kubernetes sig-security Usage

The Kubernetes sig-security team has explored generating VEX documents from govulncheck output for their projects, validating this as a real-world workflow pattern.
