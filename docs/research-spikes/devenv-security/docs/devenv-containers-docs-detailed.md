# devenv Container Generation Documentation
- **Source**: https://devenv.sh/containers/
- **Retrieved**: 2026-05-12

## Overview

devenv enables creating OCI containers from development environments using the command `devenv container build <name>`. This feature became available in version 0.6.

## Supported Container Types

Two container types are predefined:

1. **shell**: "Generate a container and start the environment, equivalent of using `devenv shell`"
2. **processes**: "Generate a container and start processes, equivalent of using `devenv up`"

Custom containers can also be created by defining them in `devenv.nix`.

## Building Containers

The basic build command is straightforward:

```bash
devenv container build shell
```

This outputs a JSON specification file (e.g., `/nix/store/...-image-devenv.json`).

**macOS Requirement**: Container generation on macOS requires a remote Linux builder. The documentation suggests three setup approaches: Nixcademy tutorials, official Nixpkgs documentation, or nix-darwin's linux-builder module.

## Running Containers

Once built, containers can be executed:

```bash
devenv container run shell
```

This initiates the containerized development environment with available tools (e.g., Python interpreter).

## Configuration: copyToRoot

The `copyToRoot` parameter controls what filesystem contents are included. Setting it to `null` creates minimal containers:

```nix
containers."processes".copyToRoot = null;
```

This exclusion "make the container smaller" by omitting the source repository.

Alternatively, specify custom artifact paths:

```nix
containers."prod" = {
  copyToRoot = ./dist;
  startupCommand = "/mybinary serve";
};
```

## Custom Container Configuration

Define custom containers with specific startup commands:

```nix
containers."serve" = {
  name = "myapp";
  startupCommand = config.processes.serve.exec;
};
```

## Registry Integration

Push containers to registries using:

```bash
devenv container --registry docker://ghcr.io/ copy <name>
```

For deployment platforms like fly.io, configure declaratively:

```nix
containers."processes" = {
  registry = "docker://registry.fly.io/";
  defaultCopyArgs = [ "--dest-creds" "x:\"$(...)\"" ];
};
```

## Environment Conditionals

Control package inclusion based on build context:

```nix
packages = [ pkgs.openssl ]
  ++ lib.optionals (!config.container.isBuilding) [ pkgs.git ];
```

This approach allows "openssl package to native and container environments, but `git` only for native environments."

## Devcontainer Integration

The documentation references "Devenv Container" and "Codespaces / devcontainer" integrations but provides no detail on `.devcontainer.json` generation in the provided content.

## Container Isolation

The documentation does not specify isolation mechanisms or security boundaries provided by these OCI containers. Containers are built using Nix's streamLayeredImage (not Docker), producing rootless OCI images. Isolation depends on the container runtime used to execute them (Docker, Podman, etc.).
