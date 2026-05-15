<!-- Source: https://learn.microsoft.com/en-us/defender-endpoint/attack-surface-reduction-rules-deployment-test -->
<!-- Retrieved: 2026-05-15 -->

# Test Attack Surface Reduction Rules - Microsoft Defender for Endpoint

Testing Microsoft Defender for Endpoint attack surface reduction rules helps you determine if rules impede line-of-business operations before you enable rules. By starting with a small, controlled group, you can limit potential work disruptions as you expand your deployment across your organization.

## Step 1: Test attack surface reduction rules using Audit

Begin the testing phase by turning on the attack surface reduction rules with the rules set to Audit, starting with your champion users or devices in ring 1. Typically, the recommendation is that you enable all the rules (in Audit) so that you can determine which rules are triggered during the testing phase.

Rules that are set to Audit don't generally impact functionality of the entity or entities to which the rule is applied but do generate logged events for the evaluation; there's no effect on end users.

### Configure attack surface reduction rules using Intune

- **Policy type**: Attack surface reduction
- **Platform**: Windows 10, Windows 11, and Windows Server
- **Profile**: Attack Surface Reduction Rules
- **Configuration settings**: Set all rules to **Audit mode** to assess impact before enforcement

### PowerShell Configuration

To enable an attack surface reduction rule in audit mode:

```PowerShell
Add-MpPreference -AttackSurfaceReductionRules_Ids <rule ID> -AttackSurfaceReductionRules_Actions AuditMode
```

To enable all the added attack surface reduction rules in audit mode:

```PowerShell
(Get-MpPreference).AttackSurfaceReductionRules_Ids | Foreach {Add-MpPreference -AttackSurfaceReductionRules_Ids $_ -AttackSurfaceReductionRules_Actions AuditMode}
```

## Step 2: Understand the attack surface reduction rules reporting page

The reporting page has three tabs:

- Detections: 30-day timeline of detected audit and blocked events
- Configuration: Per-computer aggregate state (Off, Audit, Block)
- Add exclusions: Select detected entities (false positives) for exclusion

### Event IDs

| Event ID | Description |
| --- | --- |
| 5007 | Event when settings are changed |
| 1121 | Event when an attack surface reduction rule fires in block mode |
| 1122 | Event when an attack surface reduction rule fires in audit mode |

## Key Deployment Principles

- Per-rule configuration: Each rule can independently be set to Off, Audit, Warn, or Block
- Per-rule exclusions: Specific exclusions can be configured for individual rules
- Ring-based deployment: Start with champion users (ring 1), expand to broader groups
- Recommended audit duration: Microsoft recommends reviewing audit data before transitioning to block mode
- Three-phase approach: Audit -> analyze detections -> build exclusions -> enable Block
