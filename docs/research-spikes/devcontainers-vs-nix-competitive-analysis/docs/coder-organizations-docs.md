<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/admin/users/organizations.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Organizations: Comprehensive Overview

## Core Functionality
Organizations segment and isolate resources within a Coder deployment. The system includes a default organization named `coder` where all new users are automatically added.

## Isolation & Resource Scoping
Each additional organization maintains complete separation with "unique admins, users, templates, provisioners, groups, and workspaces." This architectural design ensures that resources don't bleed between organizational boundaries.

## Provisioner Requirements
A critical technical requirement states that "each organization must have at least one dedicated provisioner since the built-in provisioners only apply to the default organization." Organizations require Coder v2.16+ with a Premium license. Provisioners are organization-scoped and execute Terraform/OpenTofu infrastructure provisioning.

## Template & Workspace Management
Templates are organization-specific, meaning users see only templates associated with their organization. When creating templates, administrators select the target organization through a dedicated dropdown interface.

## User Management
Users can belong to multiple organizations simultaneously. Assignment occurs either through manual administration or automated "organization/role/group sync" from identity providers. Members gain visibility of their organization's templates upon addition.

## Administrative Controls
Owner-role users manage organizational settings including names, icons, descriptions, user rosters, and groups. Configuration happens via Admin Settings > Organizations interface.

## Key Requirement
"Organizations requires a Premium license" and must be explicitly enabled.
