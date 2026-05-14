---
source: https://github.com/jetify-com/devbox
retrieved: 2026-03-20
---

# Devbox: Comprehensive Information Extract

## Project Overview
**Devbox** is a command-line utility developed by Jetify that facilitates the creation of isolated development environments. The repository shows **11.4k stars** and **307 forks** with **102 contributors** across 194 releases (latest: 0.17.0, March 2026).

## Core Features

Devbox provides several key capabilities:

1. **Environment Isolation**: Creates sandboxed shells without polluting the system
2. **Package Management**: Accesses "over 400,000 package versions from the Nix Package Registry"
3. **Team Consistency**: Enables declarative environments via `devbox.json` for unified team setups
4. **Version Conflict Resolution**: Manages multiple projects requiring different versions of identical tools
5. **Environment Portability**: Supports multiple deployment models including local shells, devcontainers, Dockerfiles, and cloud environments
6. **Speed**: Operates "without an extra-layer of virtualization slowing your file system"

## How Devbox Wraps Nix

The project is "internally powered by `nix`" and leverages Nix's isolation capabilities. Users declare packages in configuration files, and Devbox translates these into isolated Nix-managed environments. The tool abstracts away Nix complexity while preserving its functionality.

## Installation

The standard installation method is:
```
curl -fsSL https://get.jetify.com/devbox | bash
```

## devbox.json Format

The configuration file structure is minimal:
```json
{
  "packages": [
    "python@3.10"
  ]
}
```

This declarative approach allows version pinning and reproducible environments across team members.

## Plugin System & Services

The repository contains a `/plugins` directory indicating extensibility support, though specific plugin documentation isn't detailed in the README. Services integration appears supported based on the project structure, but explicit examples weren't provided in the main documentation.

## Shell Support

Devbox supports multiple shell environments including bash and Nushell (documented in `NUSHELL.md`). The tool integrates with existing shell configurations and environment variables.

## Quickstart Examples

**Basic workflow**:
```
devbox init           # Initialize project
devbox add python@3.10  # Add packages
devbox shell          # Enter isolated environment
python --version      # Use tools
exit                  # Exit environment
```

The prompt changes to indicate active Devbox shell status.

## Community Information

- **Discord**: Active community server at discord.gg/jetify with dedicated #devbox channel
- **Issue Tracking**: 395 open issues managed on GitHub
- **Pull Requests**: 17 active PRs
- **Social**: Updates available via Jetify's Twitter (@jetify_com)
- **Documentation**: Comprehensive docs at jetify.com/devbox/docs

## Technical Stack

**Language Composition**:
- Go: 95.0%
- TypeScript: 2.1%
- Go Template: 1.5%
- Nix: 0.7%
- Shell: 0.5%
- Dockerfile: 0.2%

## Contributing

The project welcomes contributions under the **Apache 2.0 License**. Contributors must review `CONTRIBUTING.md` before submitting pull requests. Development documentation is available in `devbox.md`.

## Additional Resources

- Main documentation portal: www.jetify.com/devbox/
- CLI reference guide available at jetify.com/devbox/docs/cli_reference/devbox/
- Contributor quickstart: jetify.com/devbox/docs/contributor-quickstart/
- Code of Conduct: Documented in `CODE_OF_CONDUCT.md`
