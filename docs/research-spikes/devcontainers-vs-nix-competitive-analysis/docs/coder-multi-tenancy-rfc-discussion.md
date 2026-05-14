<!-- Source: https://github.com/coder/coder/discussions/7638 -->
<!-- Retrieved: 2026-03-20 -->

# Coder Multi-Tenancy RFC Discussion #7638

## Overview
This RFC proposes implementing multi-tenancy in Coder to enable multiple isolated organizations within a single deployment, addressing scalability limitations and enabling SaaS deployment models.

## Core Problem Statement
"Coder deployment scale is frequently blocked by our lack of user management features." The current workaround — deploying multiple Coder instances — creates administrative burden and licensing enforcement challenges.

## Proposed Multi-Tenant Architecture

### Organization-Level Isolation
- Multiple organizations can coexist within a single Coder deployment
- Each organization maintains isolated user groups, templates, and administrators
- Strong isolation prevents organizations from interacting with or discovering each other

### Key Technical Challenges

**Configuration Management:**
Control plane settings like `--disable-password-auth` require conversion from global to organization-scoped configuration.

**Authentication Providers:**
"OIDC providers are globally registered, whereas in a multi-tenant system, they would need to be registered per organization." The architecture may retain concept of global providers (GitHub, Google) alongside org-specific configurations.

**User Account Scope:**
The RFC identifies a critical design decision: whether user accounts span multiple organizations (site-namespaced) or remain limited to single organization (org-namespaced).

### Provisioner Isolation
According to the discussion, "Templates in one organization cannot use the same provisioner as templates in another organization." This ensures infrastructure credential separation and prevents cross-tenant job contamination.

Within organizations, "provisioner groups" provide additional isolation layers.

### Credential Management
Organizations require secure, isolated credential injection for:
- IaaS API access
- Template-specific secrets
- Organization-scoped authentication providers

## Implementation Status
The discussion tracks:
- User/group management UI/CLI/API exposure
- Engineering work documentation
- Feedback collection on multi-tenancy demand

## Alternative Perspectives
Community members suggested Coder maintain API-first architecture, allowing clients to implement custom user management and multi-tenancy at their application layer rather than within Coder itself.
