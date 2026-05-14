<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/admin/users/index.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder User Management Summary

## Roles
- **Owner** — Full platform control, manages organizations and global settings
- **Member** — "Typically, most developers in your organization have the `Member` role, allowing them to create workspaces"
- **Template Admin** — Manages templates across the deployment
- **User Admin** — Can manage user suspension/activation and password resets
- **Auditor** — Has "administrative capabilities such as auditing"

"Roles determine which actions users can take within the platform."

## Authentication Methods
For production deployments, Coder recommends using an SSO authentication provider with multi-factor authentication (MFA).

Supported options include:
- **OpenID Connect** (Okta, KeyCloak, PingFederate, Azure AD)
- **GitHub** (including GitHub Enterprise)
- **Password authentication** (default, not recommended for production)

The documentation references "Group & Role Sync" and "idp-sync" capabilities for automated user provisioning.

## User Statuses
Three account states exist:
- **Active** — Full platform access (default)
- **Dormant** — No login for 90+ days; doesn't consume license seats
- **Suspended** — Deactivated by administrators; doesn't consume license seats
