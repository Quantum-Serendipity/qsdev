<!-- Source: https://cyclonedx.org/capabilities/vex/ -->
<!-- Retrieved: 2026-05-15 -->

# CycloneDX VEX Capabilities

## Core Purpose

CycloneDX VEX is designed to "convey the exploitability of vulnerable components in the context of the product in which they're used." The capability focuses on whether vulnerabilities can actually be exploited in specific contexts rather than simply disclosing their existence.

## Key Benefits

- Communicates exploitability status in machine-readable format
- Helps prioritize remediation by assessing actual risk
- Integrates with broader system inventories for contextual analysis
- Reduces unnecessary patching efforts

## Integration Approach

CycloneDX supports VEX both as embedded data within the SBOM and as standalone VEX documents. The CycloneDX VEX format is part of the broader CycloneDX specification (ECMA-424).

## Vulnerability Status Categories

CycloneDX VEX uses vulnerability analysis states:
- **Not Affected**: The vulnerability is not exploitable in this context
- **Affected**: The vulnerability is exploitable
- **Fixed**: The vulnerability has been remediated
- **Under Investigation**: Assessment is ongoing

## Analysis Fields

The vulnerability analysis section includes:
- State (the four categories above)
- Justification (why not affected, e.g., code_not_reachable, protected_by_mitigating_control)
- Response (actions taken)
- Detail (human-readable explanation)

## BOM-Link

CycloneDX supports BOM-Link to reference external BOMs and VEX documents, enabling standalone VEX documents to reference specific SBOMs.

Note: The fetched page was an overview. Full technical specification details are in the ECMA-424 standard and CycloneDX specification docs.
