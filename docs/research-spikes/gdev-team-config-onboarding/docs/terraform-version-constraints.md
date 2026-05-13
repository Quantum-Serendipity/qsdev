<!-- Source: https://developer.hashicorp.com/terraform/language/expressions/version-constraints -->
<!-- Retrieved: 2026-05-12 -->

# Terraform Version Constraint System

## Constraint Syntax
Version constraints use string literals with operators and version numbers separated by commas. The format is `"<operator> <version>"`. For example: `">= 1.2.0, < 2.0.0"` accepts versions from 1.2.0 up to (but not including) 2.0.0.

## Operators
Terraform supports these constraint operators:

- **Exact match**: `=` or no operator (only one version)
- **Exclusion**: `!=` (excludes specific version)
- **Comparison**: `>`, `>=`, `<`, `<=` (standard numeric comparison)
- **Pessimistic**: `~>` (rightmost component increments only; e.g., `~> 1.0.4` allows 1.0.5 but not 1.1.0)

## Runtime Compatibility Checking
"Terraform consults version constraints to determine whether it has acceptable versions of itself, any required provider plugins, and any required modules." When constraints are unsatisfied, Terraform attempts automatic downloads. If no acceptable version exists, Terraform halts and prevents any plan, apply, or state operations.

## Pre-release Handling
Pre-release versions (containing dash-suffixes like `1.2.0-beta`) only match with exact `=` operators. "Terraform does not match pre-release versions on `>`, `>=`, `<`, `<=`, or `~>` operators."

## Recommended Practices
Root modules should use `~>` constraints for both lower and upper bounds, while reusable modules should constrain only minimum versions (e.g., `>= 0.12.0`) to maximize downstream flexibility.
