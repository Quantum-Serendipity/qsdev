<!-- Source: https://github.com/sirwart/ripsecrets (via web search) -->
<!-- Retrieved: 2026-05-12 -->

# ripsecrets: Rust-Based Secret Scanner

## Overview

ripsecrets is a command-line tool to prevent committing secret keys into your source code. It's focused on pre-commit, since it's a lot cheaper to prevent secrets from getting committed in the first place than dealing with the consequences once a secret has been committed to your repository.

## Performance

Written in Rust, which means there's no interpreter startup time. Claims to be 95 times faster or more than other tools. Performance was benchmarked on the Sentry repo on an M1 air laptop. Most of the time, your pre-commit will be running on a small number of files, so those runtimes are not typical, but when working with large commits that touch a lot of files, the runtime can become noticeable.

## Detection Methods

1. **Known patterns**: Can find secrets with known patterns that can be matched, such as API keys from services like Stripe and Slack that have a predefined prefix.
2. **Random string detection**: Detects random strings assigned to secret variables. For secrets like AWS secret access keys that don't have a known pattern, ripsecrets looks for variables assigned with words like "token", "secret", and "password", and checks if a random string is assigned to it by calculating how likely it is to have occurred by random chance.

## Privacy

Designed to be the best "local only" tool and will never send data off of your computer.

## git-hooks.nix Integration

ripsecrets is a built-in hook in git-hooks.nix with settings:
- `additionalPatterns` (listOf types.str): Additional regex patterns used to find secrets

## Nix Availability

Available in nixpkgs as `pkgs.ripsecrets`. First-class support in devenv via `git-hooks.hooks.ripsecrets.enable = true`.
