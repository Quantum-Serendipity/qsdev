<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/admin/templates/index.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Templates: Comprehensive Overview

## Foundational Concepts

Coder templates are infrastructure-as-code definitions built with Terraform that establish the underlying compute environments for all workspaces. As stated in the documentation, "Templates are written in Terraform and define the underlying infrastructure that all Coder workspaces run on."

## Getting Started

The recommended learning path involves two stages:

1. **Foundational Knowledge**: Users should first grasp Coder-specific template concepts by creating a basic template from scratch, supplemented by HashiCorp's Terraform tutorials for cloud provider fundamentals.

2. **Starter Templates**: Pre-configured templates for popular platforms (AWS, Kubernetes, Docker, etc.) provide sensible defaults, allowing teams to quickly import standardized infrastructure configurations.

## Customization & Extension

Templates become production-ready through targeted modifications:

- **Container Images**: Custom Docker images with pre-installed languages and development tools
- **Parameters**: Configurable options like disk size, instance type, and region selection
- **Development Tools**: Additional IDEs (JetBrains) and features (dotfiles, RDP support)

## Best Practices Framework

Organizations should adopt strategic approaches including:

- **Image Management**: Creating and publishing container images for workspace use
- **Dev Container Integration**: Native support via `@devcontainers/cli` and Docker
- **Resource Protection**: Template hardening to prevent accidental destruction of user data
- **Version Control**: GitOps-based change management with CI/CD pipelines
- **Access Control**: Role-based permissions governing template modification and use
- **Infrastructure Integration**: Connecting existing infrastructure through external workspace connections

This scalable approach allows deployments to evolve from universal baseline templates to specialized configurations serving distinct team requirements.
