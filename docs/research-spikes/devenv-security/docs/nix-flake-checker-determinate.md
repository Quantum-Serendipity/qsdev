<!-- Source: https://determinate.systems/blog/flake-checker/ -->
<!-- Retrieved: 2026-05-12 -->

# Nix Flake Checker: Ensuring Flake Security and Best Practices

## Overview

Determinate Systems created the Nix Flake Checker tool to help developers maintain security and follow best practices in flake-enabled Nix projects. The tool automatically validates Nixpkgs dependencies in flake inputs.

## Three Core Validation Checks

The checker performs three essential security and maintainability checks:

1. **Supported Release Branches**: Verifies that Git branches like `nixpkgs-unstable` are officially supported. The tool flags unsupported branches since "release branches stop receiving updates roughly 7 months after release," potentially leaving projects vulnerable.

2. **Currency Requirement**: Ensures Nixpkgs inputs have been updated within the last 30 days. Older revisions miss community security patches and improvements.

3. **Upstream Verification**: Confirms GitHub-hosted Nixpkgs inputs originate from the official NixOS organization, preventing use of forks or untrusted variants that could introduce "security vulnerabilities and unexpected behaviors."

## Usage Methods

Developers can run checks locally via command line:
```
nix run "github:DeterminateSystems/flake-checker"
```

For continuous integration, the Determinate Flake Checker Action integrates with GitHub Actions workflows, automatically scanning flake.lock files and providing detailed summaries with remediation guidance.

## Security Impact

By encouraging dependency pinning alongside regular updates, the checker helps projects benefit from Nix's "killer feature" while maintaining the supply chain security measures provided by upstream Nixpkgs' testing and integration infrastructure.
