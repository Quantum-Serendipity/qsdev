# Firewall Quarantine — Sonatype Nexus Documentation

- **Source URL**: https://help.sonatype.com/en/firewall-quarantine.html
- **Retrieved**: 2026-05-12

## Overview

The Repository Firewall quarantine mechanism automatically isolates components that violate organizational security policies. When a newly requested component breaches policy requirements, it enters quarantine, preventing download through the proxy repository while generating an error message with violation details.

## How Quarantine Works

**Core Function:** The system places components into quarantine when they violate policies configured with a `FAIL` action at the `PROXY` stage. "When a requested component violates your open-source policy, the component is put into quarantine while returning an error message."

**Scope Limitation:** "Repository Firewall only quarantines newly requested components." Pre-existing components in the repository receive auditing but not quarantine enforcement to avoid disrupting existing build pipelines.

## Configuration Steps

1. **Access Policy Settings:** Log into IQ Server and navigate to "Repos and Policies"
2. **Select Target Policy:** Choose the policy requiring quarantine enforcement
3. **Set Action:** Select the `Fail` radio button under the `Proxy` column in the Actions section
4. **Apply Changes:** Click Update to implement the configuration

## Proxy Stage Actions

The quarantine system offers three action levels:
- **Fail:** "Quarantines any newly requested components that violate the policy"
- **Warn:** Triggers email notifications for policy violations
- **No Action:** Default setting displaying violations only in audit reports

## Component Evaluation Process

- Components already stored in proxy repositories bypass quarantine but undergo auditing
- Deleted components requested again may trigger quarantine if they violate current policies
- The system "fails closed" during service unavailability, immediately quarantining new component requests

## Viewing Quarantined Components

Security teams access quarantined components through multiple interfaces:
- **Firewall Dashboard:** Centralized view of all quarantined components with component detail links
- **Repository Results View:** Displays audit reports for specific proxy repositories
- **Quarantined Component View:** Individual component assessment available via command-line requests
- **REST API:** Programmatic access for third-party tool integration

## Release and Remediation Strategies

**Release Process:** Components exit quarantine when failing policy violations receive waivers or when violations are resolved.

**Remediation Options:**
1. **Version Selection:** Choose alternative component versions without violations
2. **Component Substitution:** Select different components addressing the same requirements
3. **Violation Waiver:** Accept risk through formal waivers

## Waiver Considerations

- Waivers should target specific repositories or implement short expiration windows
- Once expired, violations re-trigger but previously downloaded components remain available
- Time-window waivers prevent re-violation incidents during active waiver periods

## Technical Constraints

- NuGet clients hardcode authentication failures and return HTTP 409 status codes for quarantined components instead of the standard 403 response
- Repository Firewall explicitly does not support legacy violations in policy enforcement decisions

## Operational Safeguards

Organizations should educate development teams about quarantine processes, particularly regarding ambiguous 403 error messages in build logs. "Fail closed" behavior during service outages prioritizes security by maintaining quarantine until evaluation completion.
