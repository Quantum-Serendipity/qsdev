# DevEnv Auto-Activation
- **Source**: https://devenv.sh/auto-activation/
- **Retrieved**: 2026-05-12

## Core Functionality

DevEnv's auto-activation feature, introduced in version 2.1, automatically activates developer environments when entering project directories. As stated in the documentation, "devenv includes a built in shell hook that automatically activates your developer environment when you cd into a project directory. No external tools required."

## Trust Model & Security

The system implements explicit trust verification before activation occurs. Users must navigate to a project and execute `devenv allow` to authorize auto-activation. This prevents unauthorized environment modifications. The documentation notes: "This is a security measure that prevents untrusted projects from modifying your shell."

Trust operates at the project directory level rather than individual files. Once revoked via `devenv revoke`, a project no longer auto-activates.

## How Activation Works

The activation process follows these steps:

1. Scans upward from the current directory for `devenv.yaml`
2. Validates the project against the trust database
3. Spawns a subshell running `devenv shell` if authorized

Importantly, "The hook only detects projects that have a devenv.yaml file. Projects with only devenv.nix (without devenv.yaml) are not detected."

## Setup Requirements

Users add one line to their shell configuration file (bash, zsh, fish, or nushell) to enable the hook -- no external tool installations are necessary.

## Automatic Deactivation & Nesting Prevention

The environment automatically exits when leaving the project directory. The system prevents nested environments within the same project, maintaining the current shell when navigating subdirectories.
