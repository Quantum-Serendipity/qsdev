<!-- Source: https://github.com/coder/coder -->
<!-- Retrieved: 2026-03-20 -->

# Coder Repository Overview

## Project Description

Coder is a platform that "enables organizations to set up development environments in their public or private cloud infrastructure." The system allows teams to define cloud development environments using Terraform, connect them through secure Wireguard tunnels, and automatically shut down idle resources to reduce costs.

## Key Features

- **Infrastructure-as-Code**: Development environments defined in Terraform supporting EC2 VMs, Kubernetes Pods, and Docker Containers
- **Cost Optimization**: Automatic shutdown of idle resources to minimize cloud spending
- **Developer Onboarding**: Significantly reduces setup time from days to seconds
- **Secure Connectivity**: High-speed encrypted tunnel technology for remote access
- **Multi-IDE Support**: Integration with VS Code, JetBrains Gateway, and JetBrains Toolbox

## Technology Stack

The repository composition reveals:
- **Go**: 76.0% of codebase
- **TypeScript**: 21.7%
- **Shell**: 1.0%
- **HCL**: 0.4% (Terraform configurations)
- **PLpgSQL**: 0.3%
- **Makefile**: 0.2%

## Installation Methods

The quickstart involves a simple installation script: `curl -L https://coder.com/install.sh | sh`. Production deployments require PostgreSQL 13+ and can use either automatic URL setup or manual configuration with `--postgres-url` and `--access-url` flags.

## Notable Statistics

- **12.6k stars** on GitHub
- **1.2k forks**
- **71 watchers**
- **287 releases** (latest v2.30.4)
- **13,092 commits** in main branch

## Licensing

Dual licensing model: AGPL-3.0 for open source and an enterprise license available separately.

## Community & Support

Active Discord community, GitHub issue tracking, and comprehensive documentation covering templates, workspaces, IDEs, and administration. Multiple official integrations exist for popular development tools and platforms.
