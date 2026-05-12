<!-- Source: https://devenv.sh/inputs/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv Inputs - Complete Content

## Overview

Inputs function as dependency management for developer environments, allowing reference to external Nix code while maintaining reproducibility.

## Default Configuration

When `devenv.yaml` is omitted, these inputs are automatically included:

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  git-hooks:
    url: github:cachix/git-hooks.nix
```

## Accessing Inputs in devenv.nix

Inputs specified in `devenv.yaml` become arguments passed to the `devenv.nix` function. For example, a custom input like `nixpkgs-stable` can be accessed and instantiated:

```nix
{ inputs, pkgs, ... }:

let
  pkgs-stable = import inputs.nixpkgs-stable { system = pkgs.stdenv.system; };
in {
  packages = [ pkgs-stable.git ];
  
  enterShell = ''
    git --version
  '';
}
```

## Special Built-in Inputs

Three special inputs are automatically available:

- **pkgs**: The nixpkgs input containing all available packages for your system
- **lib**: A collection of functions for manipulating Nix data structures; searchable via noogle
- **config**: The final resolved environment configuration, supporting lazy evaluation for self-reference

The ellipsis pattern (`...`) allows safely omitting unused inputs.

## Supported URI Formats

devenv supports the same URI specification as Nix Flakes. Common formats include:

**GitHub:**
- `github:NixOS/nixpkgs/master`
- `github:NixOS/nixpkgs?rev=238b18d7b2c8239f676358634bfb32693d3706f3`
- `github:org/repo?dir=subdir`
- `github:org/repo?ref=v1.0.0`

**GitLab:**
- `gitlab:owner/repo/branch`
- `gitlab:owner/repo/commit`
- `gitlab:owner/repo?host=git.example.org`

**Git repositories:**
- `git+ssh://git@github.com/NixOS/nix?ref=v1.2.3`
- `git+https://git.somehost.tld/user/path?ref=branch&rev=fdc8ef970de2b4634e1b3dca296e1ed918459a9e`
- `git+file:///some/absolute/path/to/repo`

**Mercurial:**
- `hg+https://...`
- `hg+ssh://...`
- `hg+file://...`

**Sourcehut:**
- `sourcehut:~misterio/nix-colors/21c1a380a6915d890d408e9f22203436a35bb2de?host=hg.sr.ht`

**Tarballs:**
- `tarball+https://example.com/foobar.tar.gz`

**Local files:**
- `path:/path/to/repo`
- `file+https://`
- `file:///some/absolute/file.tar.gz`

Note: Path inputs don't respect `.gitignore` and copy entire directories to the Nix store, so `git+file` is recommended for large development directories.

## Following Inputs

Inputs can inherit from other inputs using the `follows` keyword. This supports two primary use cases:

1. Inheriting inputs from other `devenv.yaml` files or external Nix Flake projects
2. Reducing redundant downloads by overriding nested inputs with a single version

Nested inputs use `/` as a separator. Example of following a nested input:

```yaml
inputs:
  base-project:
    url: github:owner/repo
  nixpkgs:
    follows: base-project/nixpkgs
```

Example of overriding nested inputs to consolidate downloads:

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  git-hooks:
    url: github:cachix/git-hooks.nix
    inputs:
      nixpkgs:
        follows: nixpkgs
```

## CLI-Based Input Management

Inputs can be added programmatically without manual file editing:

```bash
$ devenv inputs add nixpkgs-stable github:NixOS/nixpkgs/nixos-23.11
```

To establish a following relationship:

```bash
$ devenv inputs add my-input github:org/repo --follows nixpkgs
```

## Locking and Updating

When any command executes, devenv resolves flexible input references (like `github:NixOS/nixpkgs/nixpkgs-unstable`) into specific commit revisions, writing results to `devenv.lock`. This ensures reproducibility across environments and time.

To update inputs to newer versions:

```bash
$ devenv update
```

Alternatively, revisions and branches can be pinned at the input level according to the devenv.yaml reference documentation.
