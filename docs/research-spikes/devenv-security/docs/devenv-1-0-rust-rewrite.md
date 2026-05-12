<!-- Source: https://devenv.sh/blog/2024/03/20/devenv-10-rewrite-in-rust/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv 1.0: Rust Rewrite - Complete Blog Post Content

## Overview

The devenv team released version 1.0, which involved rewriting the CLI from Python to Rust. The post explains the rationale, new features, and migration requirements.

## Why the Rewrite Occurred

The author explains that during documentation of the Python rewrite, they "came up with only excuses as to why it is not fast and realized that we were simply breaking our promise." This motivated moving to Rust for performance improvements.

A secondary reason involved the Nix ecosystem. The tvix project has been rewriting Nix itself in Rust, creating future opportunities for devenv to "use the same Rust libraries and tooling."

## Major New Features

**Process Management**: The default process manager changed to process-compose, which "handles dependencies between processes and provides a nice ncurses interface to view the processes and their logs."

**Testing Infrastructure**: A new `enterTest` attribute enables defining testing logic directly in `devenv.nix`. When running `devenv test`, the system executes the test command and "will be started and stopped" if processes exist.

**Python Support**: Significant effort addressed native library handling. The implementation now allows "use native libraries in Python without any extra effort" through configuration options like `languages.python.libraries`.

**Container Security**: Generated containers now "run as a plain user—improving security and unlocking the ability to run software that forbids root."

**Environment Variable Handling**: The `DEVENV_RUNTIME` variable was introduced to address socket path limits, pointing to `$XDG_RUNTIME_DIR` by default.

**Testing Coverage**: A new project called devenv-nixpkgs runs "around 300 tests across different languages and processes."

## CLI Enhancements

New commands include:
- `devenv inputs add <name> <url>` for adding inputs
- `devenv update <input>` for single input updates
- `devenv build languages.rust.package` for building attributes
- `devenv shell --clean EDITOR,PAGER` for clean environment execution

Performance defaults were adjusted: cores defaulted to 2 and max-jobs to half the CPU count, as excessive parallelism caused memory issues.

## Breaking Changes and Migrations

The `.env` file prefix requirement became mandatory. More significantly, hermetic defaults changed—the `--impure` flag is no longer necessary by default, but "you will have to use `pkgs.stdenv.system`" instead of `builtins.currentSystem`.

The `devenv.lock` format changed, making "newly-generated lockfiles cannot be used with older versions of devenv."

Commands were renamed: `devenv container --copy` became `devenv container copy`, and `devenv ci` became `devenv test`.

## Future Directions

Planned features include container-based execution via `devenv shell --in-container`, macOS container generation support, executing `enterShell` during container generation, and automatic dependency mapping from language-specific package managers.
