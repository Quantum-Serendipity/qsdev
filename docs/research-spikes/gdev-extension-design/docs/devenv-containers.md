# devenv.sh Containers
- **Source**: https://devenv.sh/containers/
- **Retrieved**: 2026-05-12

## Overview

DevEnv provides container generation capabilities through the `devenv container` command. As stated in the documentation, "Use `devenv container build <name>` to generate an OCI container from your development environment."

## Key Commands

The primary operations include:

- **Build**: `devenv container build shell` - Creates a container for entering the development environment
- **Run**: `devenv container run <name>` - Executes a container using Docker
- **Copy**: `devenv container --registry docker://ghcr.io/ copy <name>` - Transfers containers to registries

## Container Types

Two predefined containers exist by default:
1. **shell** - Launches the development environment interactively
2. **processes** - Starts configured processes automatically

## Configuration Examples

### Basic Environment Container
A simple Python environment generates via this configuration:
```nix
{
  name = "simple-python-app";
  languages.python.enable = true;
}
```

### Running Processes
Multiple processes can execute within containers by defining them in the configuration and setting `copyToRoot = null` to reduce image size.

### Custom Startup Commands
Individual processes can serve as container entry points using the `startupCommand` property, allowing selective command execution rather than entering the shell.

### Production Artifacts
Containers can include only compiled binaries from `./dist` directories, excluding development dependencies through selective `copyToRoot` paths.

## Registry Deployment

Containers support multiple registry destinations. The documentation mentions "Any arguments passed to `--copy-args` are forwarded to skopeo copy," enabling authentication and other transfer options.

## Conditional Configuration

Environment variables can be conditionalized using `config.container.isBuilding` to provide package-specific inclusions based on build context.

## Platform Requirements

macOS users require a remote Linux builder to generate containers due to platform constraints.
