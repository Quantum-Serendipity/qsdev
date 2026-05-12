# devenv.sh Basics - Writing devenv.nix
- **Source**: https://devenv.sh/basics/
- **Retrieved**: 2026-05-12

## Core Concept

The page explains how to write a basic `devenv.nix` configuration file. According to the documentation, "devenv.nix is a function with inputs. pkgs is an input passed as a special argument to the function."

## Hello World Example Structure

The foundational example demonstrates three key components:

1. **Function Declaration**: The file begins with `{ pkgs, ... }:` which accepts inputs, with `pkgs` providing access to Nix packages and `...` serving as a catch-all.

2. **Attribute Set Return**: The function returns "an attribute set, similar to an object in JSON."

3. **Key Configuration Options**:
   - `env.GREET = "hello"` - Sets environment variables
   - `packages = [ pkgs.jq ]` - Declares required packages
   - `enterShell` - Executes bash code upon shell activation

## Practical Usage

Running `devenv shell` builds and enters the environment, executing the `enterShell` bash block which echoes the greeting and displays the jq version.

## Environment Information

The `devenv info` command displays a comprehensive summary including:
- Environment variables (DEVENV_DOTFILE, DEVENV_ROOT, DEVENV_STATE, GREET)
- Installed packages with versions
- Scripts and processes configuration

## Best Practice Recommendation

The documentation advises: "For more complex setup operations, consider using tasks instead of enterShell. Tasks provide better control over execution order, dependencies, and can run in parallel."

The page references the Nix language tutorial for deeper learning and links extensively to related documentation sections covering packages, scripts, tasks, and language-specific configurations.
