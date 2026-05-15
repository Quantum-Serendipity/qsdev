# Container Security Fundamentals Part 5: AppArmor and SELinux

- **Source URL**: https://securitylabs.datadoghq.com/articles/container-security-fundamentals-part-5/
- **Retrieved**: 2026-05-15

## Overview of Mandatory Access Control Systems

AppArmor and SELinux represent Mandatory Access Control (MAC) systems that differ fundamentally from Discretionary Access Control (DAC) approaches. Unlike file permissions where owners can modify access rules, MAC systems enforce restrictions that "even an otherwise privileged process cannot access" certain resources, with constraints that apply across user boundaries.

## AppArmor Implementation

AppArmor operates through process profiles that restrict access to files, network traffic, and Linux capabilities. The system includes Docker's default profile, though this "isn't as locked down as it could be, making it necessary to create more restrictive profiles" for enhanced protection.

### Custom Profile Example

The article demonstrates creating a profile blocking write access to `/etc`:

```
deny /etc/** wl,
```

This prevents write operations even for root users within containers, demonstrating MAC's enforcement strength beyond standard permission models.

## SELinux Configuration and Modes

SELinux operates differently, labeling resources rather than applying discrete profiles. The system supports three operational modes:

- **Enforcing**: Actively blocks policy violations
- **Permissive**: Logs violations without blocking (useful for setup)
- **Disabled**: Security subsystem inactive

The article shows SELinux using "targeted mode" in the example, meaning "SELinux will be applied to specific processes chosen by the distribution provider" rather than all processes.

## Critical Gap: Subsystem Failure Handling

The article does not address what happens when the security subsystem itself fails. This represents a significant omission regarding production deployment resilience.

## Production Deployment Considerations

The article acknowledges that customization "requires effort to learn" and represents "a significant undertaking" at scale, recommending organizations "assess their risk profile to determine if it makes sense to use them."

Tools like Bane (AppArmor) and udica (SELinux) can ease policy management, but the article provides limited explicit best practices for production environments.
