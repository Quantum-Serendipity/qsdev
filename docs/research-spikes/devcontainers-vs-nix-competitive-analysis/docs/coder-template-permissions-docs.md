<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/admin/templates/template-permissions.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Template Permissions Summary

## Feature Availability
Template permissions are a **Premium feature** requiring a licensed Coder administrator account.

## Permission Roles

Two distinct roles are available:

1. **Use Role**: Grants permission to use the template for creating workspaces
2. **Admin Role**: Provides complete control over all template aspects

## Access Control

**Who can be granted permissions:**
- Individual users
- User groups

**Default Configuration:**
The "Everyone" group is automatically assigned to all templates, meaning any Coder user can utilize templates by default.

## Key Capability

As noted in the documentation: "This offers a way to elevate the privileges of ordinary users for specific templates without granting them the site-wide role of `Template Admin`."

## Restriction Option

Administrators can disable the "Allow everyone to use the template" setting during template creation to prevent universal access and implement more restrictive permission models.

## Technical Summary

The permission system operates on a simple two-tier model where administrators assign users or groups either read-level access (Use) or full control (Admin), with granular control over default public access through toggle settings.
