<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/user-guides/devcontainers/index.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder's Dev Containers Integration: Complete Overview

## Core Integration Architecture

Coder seamlessly integrates dev containers through the `@devcontainers/cli` and Docker, enabling developers to "define your development environment as code using a `devcontainer.json` file." The system treats each dev container as a sub-agent within the workspace, providing isolated environments with their own applications, SSH access, and port forwarding capabilities.

## Configuration and File Placement

Developers can position their configuration files in three locations:
- `.devcontainer/devcontainer.json` (recommended approach)
- `.devcontainer.json` (repository root)
- `.devcontainer/<folder>/devcontainer.json` (for multiple configurations in monorepos)

A minimal configuration requires just a name and image specification, though the full Dev Container specification supports extensive customization options.

## Automatic Discovery and Startup

The integration includes "automatic dev container detection from repositories" with dashboard visibility. When workspaces initialize, Coder:

1. Pre-creates sub-agents if the template defines resources like `coder_app`, `coder_script`, or `coder_env`
2. Initializes the Docker environment
3. Scans repositories for dev container configurations
4. Displays discovered containers in the dashboard
5. Automatically builds and starts containers if configured via `coder_devcontainer` or autostart settings
6. Creates or updates the sub-agent for running containers

Users can manually initiate containers lacking auto-start configuration through dashboard buttons.

## Sub-Agent Management

Each dev container receives its own agent name derived from the workspace folder path. For instance, a container at `/home/coder/my-app` becomes agent `my-app`. Names undergo sanitization to "contain only lowercase alphanumeric characters and hyphens," with options for custom naming in `devcontainer.json`.

## Connection Methods

Running containers support multiple access approaches:
- Web terminal through the Coder dashboard
- SSH using `coder ssh <workspace>.<agent>` syntax
- VS Code Desktop integration via dashboard buttons

## Supported Features

The integration provides:
- "Change detection with outdated status indicator"
- On-demand rebuilding through dashboard controls
- Template-defined apps, scripts, and environment variables via Terraform
- Integrated VS Code experience
- Direct SSH container access
- Automatic port detection

## Technical Requirements

Prerequisites include:
- Coder version 2.24.0 or later
- Docker availability within the workspace
- `@devcontainers/cli` installation

The integration is "enabled by default" for compatible workspaces.

## Critical Limitations

**Platform Restrictions:** Dev Containers function exclusively on Linux; "Windows or macOS workspaces" lack support.

**Configuration Changes:** Modifications to `devcontainer.json` necessitate manual dashboard-initiated rebuilds.

**Port Forwarding:** The `forwardPorts` property doesn't support `host:port` syntax for Docker Compose sidecars, though single-container environments can utilize `coder port-forward` for direct port access.

**Advanced Features:** Some sophisticated dev container capabilities have restricted or limited support levels.

## Alternative Approach for Non-Docker Environments

For workspaces without Docker access, administrators can deploy Envbuilder, which "builds the workspace image itself from your dev container configuration" as an alternative solution.
