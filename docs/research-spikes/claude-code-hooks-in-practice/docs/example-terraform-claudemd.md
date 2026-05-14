# Example CLAUDE.md: Terraform Project (ArthurClune/claude-md-examples)
- **Source**: https://raw.githubusercontent.com/ArthurClune/claude-md-examples/main/terraform-CLAUDE.md
- **Retrieved**: 2026-03-27

---

# Terraform Development Instructions for Claude Code

## Overview
Defines how to work with Terraform code, emphasising multi-file coordination, linting with tflint, and security scanning with checkov.

## Critical Rules

### 1. Multi-File Awareness
- Always scan the entire module directory before making changes
- Check for dependencies across files: variables, outputs, data sources, and resources may be referenced anywhere
- When modifying a resource, search for all references to it across `.tf` files
- Consider module boundaries and inter-module dependencies

### 2. File Organisation
Standard Terraform conventions:
- `main.tf`, `variables.tf`, `outputs.tf`, `providers.tf`, `versions.tf`, `data.tf`, `locals.tf`

### 3. Change Coordination
1. Identify scope: List all files that might be affected
2. Search references: Use grep/search for resource names, variable names, output names
3. Update systematically: Make changes in dependency order
4. Verify completeness: Re-check all references after changes

## Linting with tflint

Setup, standard process, common issues to fix (deprecated syntax, invalid instance types, hardcoded credentials, missing providers, unused declarations)

## Security Scanning with checkov

Standard security scan commands with focus on:
- Encryption at rest/in transit
- Public access restrictions
- IAM least privilege
- Logging enabled
- Backup configured
- No hardcoded secrets

## Development Workflow

### Before Starting Changes
```bash
terraform init
terraform validate
tflint
checkov -d .
```

### After Changes
```bash
terraform fmt -recursive
terraform validate
tflint --recursive
checkov -d .
terraform plan -out=plan.out
```

## Common Patterns

Resource naming conventions, variable usage with validation blocks, output references with guidance on checking usage in parent modules, remote state, and external scripts.

## Notable Characteristics

- ~200 lines, highly structured
- Includes full bash commands for every tool
- Security scanning integrated as standard workflow step
- Multi-file awareness is first critical rule
- Includes concrete code examples for patterns
