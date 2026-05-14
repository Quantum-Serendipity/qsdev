<!-- Source: https://github.com/roman/mcps.nix -->
<!-- Retrieved: 2026-05-14 -->

# mcps.nix: MCP Server Presets for Claude Code

## Overview

mcps.nix is "a curated library of MCP (Model Context Protocol) server presets for Claude Code that integrates with native Claude modules in devenv and Home Manager." It simplifies enabling popular MCP servers without manual JSON configuration.

## Core Features

- **Pre-configured servers**: Asana, GitHub, Buildkite, Git, Filesystem, LSP integrations, and 15+ others
- **Secure credential handling**: Reads API tokens from files rather than environment variables
- **Native integration**: Works with upstream Claude modules in both devenv and home-manager
- **Extensibility**: Allows custom MCP servers alongside presets

## Integration Approaches

### devenv Integration

Add to devenv configuration:

```nix
{
  imports = [ inputs.mcps.devenvModules.claude ];

  claude.code = {
    enable = true;
    mcps = {
      git.enable = true;
      filesystem = {
        enable = true;
        allowedPaths = [ "/path/to/your/project" ];
      };
    };
  };
}
```

### Home Manager: Two Options

**Option 1 - Native Integration** (recommended): MCP servers managed through Nix, stored in the Nix store:

```nix
{
  imports = [ inputs.mcps.homeManagerModules.claude ];

  programs.claude-code = {
    enable = true;
    mcps = {
      git.enable = true;
      asana = {
        enable = true;
        tokenFilepath = "/var/run/agenix/asana.token";
      };
    };
  };
}
```

**Option 2 - Script-based**: Uses Claude CLI to manage `~/.claude.json`, persisting configurations outside Nix for manual editing flexibility.

## Available MCP Servers

Notable presets include:

| Server | Purpose |
|--------|---------|
| **github** | GitHub API with configurable toolsets (repos, issues, code_security) |
| **filesystem** | Local file access with path restrictions |
| **git** | Version control operations |
| **lsp-*** | Language servers for Go, Nix, Python, Rust, TypeScript |
| **nixos** | NixOS configuration and package discovery |
| **obsidian** | Vault integration for notes |
| **grafana** | Monitoring and alerting management |
| **sequential-thinking** | Enhanced reasoning capabilities |

## Project Structure

- Written entirely in Nix (100%)
- 26 GitHub stars, 7 forks
- Licensed under MIT
- Contains example devenv configuration and test suite
