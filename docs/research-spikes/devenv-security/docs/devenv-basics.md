<!-- Source: https://devenv.sh/basics/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv Basics - Complete Content

## Core Concept

The page introduces devenv as a tool for creating "Fast, Declarative, Reproducible, and Composable Developer Environments using Nix."

## Configuration Structure

The `devenv.nix` file functions as a Nix language file that accepts inputs. The documentation explains that "devenv.nix is a function with inputs. pkgs is an input passed as a special argument to the function." Users should employ a catch-all operator `...` to avoid listing every input explicitly.

The basic structure returns an attribute set—comparable to JSON objects—with nested attributes and values referencing inputs.

## Key Configuration Attributes

**env**: Sets environment variables within the development shell.

**packages**: Lists dependencies to make available (example: `pkgs.jq`).

**enterShell**: Executes bash code upon shell activation. The documentation notes: "For more complex setup operations, consider using tasks instead of enterShell. Tasks provide better control over execution order, dependencies, and can run in parallel."

## Command Usage

Activating the environment uses `devenv shell`, which builds and enters the configured shell environment.

## Environment Information

The `devenv info` command displays a comprehensive summary including:
- Environment variables (DEVENV_DOTFILE, DEVENV_ROOT, DEVENV_STATE, custom variables)
- Available packages
- Scripts
- Processes

## Learning Resources

The documentation recommends the "Nix language tutorial" for deeper understanding, suggesting it provides "a 1-2 hour deep dive" enabling users to comprehend Nix files generally.
