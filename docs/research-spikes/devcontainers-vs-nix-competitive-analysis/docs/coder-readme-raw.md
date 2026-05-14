<!-- Source: https://raw.githubusercontent.com/coder/coder/main/README.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder README Summary

## Product Overview
Coder is a self-hosted platform for establishing cloud development environments. Organizations can deploy development spaces through their public or private cloud infrastructure using Terraform definitions and secure Wireguard tunnels.

## Core Features
The platform offers three primary capabilities:

1. **Infrastructure as Code**: "Define cloud development environments in Terraform" supporting EC2 instances, Kubernetes pods, Docker containers, and similar resources.

2. **Cost Optimization**: Implements automatic shutdown of idle resources to reduce cloud spending.

3. **Developer Onboarding**: Accelerates team member setup from days to seconds.

## Installation Methods
Quick setup involves using the installation script:
```shell
curl -L https://coder.com/install.sh | sh
```

The script supports Linux, macOS, and Windows platforms, with options for dry-run testing and additional configuration flags.

## Deployment Options
Two deployment paths exist:

- **Local Development**: Single-command server startup caching data locally
- **Production**: Requires PostgreSQL (version 13+) and external access URL configuration

## Documentation Sections
The platform includes guides covering templates (Terraform-based infrastructure), workspaces (development environments with IDEs and dependencies), IDE integrations, and administrative operations.

## Official Integrations
Supported tools include VS Code Extension, JetBrains plugins (Toolbox and Gateway), Dev Container Builder, a private extension marketplace, and GitHub Actions support.

## Community & Support
Users can report issues on GitHub, join the Discord community, or explore community-contributed Terraform provisioning scripts and GitHub Actions.
