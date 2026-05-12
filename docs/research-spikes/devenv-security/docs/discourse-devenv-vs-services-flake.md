<!-- Source: https://discourse.nixos.org/t/devenv-vs-services-flake-vs/59074 -->
<!-- Retrieved: 2026-05-12 -->

# NixOS Discourse Thread: Devenv vs Services-Flake Comparison

## Original Question (freelock)

The user sought to create Nix-based development environments for Drupal projects, comparing alternatives to Docker solutions like ddev. Key requirements included:

- Single-command project setup with `nix run`
- Template-based project initialization
- Per-project language version control
- Multi-project simultaneous execution
- Darwin/Windows portability

## Devenv.sh Experience (nick_kadutskyi)

A PHP developer shared positive devenv usage on macOS, demonstrating workflow with `nix develop --impure` and custom startup commands. They confirmed devenv supports multiple language versions and simultaneous projects, though they used macOS's built-in Apache rather than a devenv-configured server.

## Drupal-Devenv Alternative (LiamMcDermott)

A Drupal shop created drupal-devenv targeting similar goals. Key points:

- Requires initial `devenv.yaml` configuration, then `devenv up`
- Supports language version switching
- Uses Caddy for multi-project handling without multiple ports
- Linux-only due to privilege escalation code
- Process management occasionally requires manual PID cleanup versus Docker Compose reliability

## Shift to Services-Flake (freelock)

After initial devenv experimentation, the user pivoted to services-flake, describing it as feeling "less clunky compared to a regular flake." They achieved their primary goal:

> "anyone with nix...can spin up a fresh Drupal CMS install with a single line"

Current implementation uses:
- High ports avoiding root requirements
- `*.ddev.site` DNS for localhost resolution
- Nginx (with future Caddy migration consideration)

## Technical Discussion

Community members suggested improvements including `php.buildComposerProject2` for better Drupal packaging and replacing Nginx with Caddy for enhanced localhost SSL handling. The devenv maintainer (domenkozar) expressed interest in addressing "clunkiness" concerns.
