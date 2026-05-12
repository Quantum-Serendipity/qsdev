# devenv.sh Inputs Configuration
- **Source**: https://devenv.sh/inputs/
- **Retrieved**: 2026-05-12

## Definition and Purpose

Inputs serve as a dependency management system for developer environments. They allow you to "refer to Nix code outside of your project while preserving reproducibility."

The default configuration includes `nixpkgs` and `git-hooks` inputs, but projects can extend this by declaring additional inputs in `devenv.yaml`.

## How Inputs Are Pinned

**devenv.lock file**: When running commands, devenv converts flexible references like `github:NixOS/nixpkgs/nixpkgs-unstable` into specific commit revisions and records them in `devenv.lock`. This mechanism "ensures that your environment is reproducible."

**Granularity of pinning**: The lock file captures full commit hashes. For example: `github:NixOS/nixpkgs?rev=238b18d7b2c8239f676358634bfb32693d3706f3` explicitly specifies a revision.

## Individual Package Pinning

The documentation does not explicitly address whether individual packages within nixpkgs can be version-pinned separately. However, the architecture allows importing alternative nixpkgs instances (like stable vs. rolling versions) through separate inputs, providing indirect package version control.

## Supported Input Sources

- GitHub, GitLab, Git repositories
- Mercurial, Sourcehut
- Tarballs and local files

## Security Implications

The documentation does not discuss security considerations for input management, authentication mechanisms, or supply-chain risks.
