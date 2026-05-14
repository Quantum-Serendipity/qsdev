<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/tutorials/best-practices/organizations.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Organizations Best Practices Summary

## Core Structure
Organizations function as hierarchical parents for templates, groups, and provisioners. "Users can belong to multiple organizations while templates and provisioners cannot." Each organization maintains separate templates, provisioners, groups, and workspaces to prevent resource sharing across organizational boundaries.

## When to Implement Organizations
Organizations should only be deployed when necessary due to increased maintenance overhead. Implementation is appropriate when "a separate group of users needs to manage their own templates and underlying infrastructure" and they have a dedicated platform team capable of maintaining these resources.

### Ideal Use Cases:
- **Mergers & Acquisitions**: Teams with independent cloud accounts, platform teams, and existing infrastructure pipelines
- **Cloud-Native Teams**: Groups managing their own Kubernetes clusters and namespaces
- **Distributed Contractors**: Offshore teams requiring data sovereignty and low-latency considerations
- **Specialized Platforms**: ML teams and data platforms with custom infrastructure needs

### Poor Fit Scenarios:
- Java monoliths supported by central teams
- Developer groups with varied regional needs
- Teams requiring specific development tools or repositories

## Migration Strategy
Templates cannot be moved between organizations, so deprecation is recommended. "Users can use a file transfer tool such as rsync to migrate their files from one workspace to another."

## Security & Isolation

**Provisioner Isolation**: Provisioners operate in separate infrastructure with isolated authentication keys, accessing cloud resources inaccessible to the control plane. The control plane submits simple provisioner jobs rather than direct infrastructure access.

**Identity Provider Sync**: Manual user assignment is discouraged. "Instead, we recommend syncing the state from your identity provider such as Okta" using claims like `memberOf` for role and organization mapping.

## Access Control
Custom organization-scoped roles can restrict access, including:
- Provisioner Admins (deploy only)
- Template Editors (manage templates only)
- Template Pushers (system accounts with limited permissions)

## Scale Management
For multi-organization deployments, use the Coderd Terraform provider to automate onboarding, quota management, and SSO synchronization rather than manual configuration.
