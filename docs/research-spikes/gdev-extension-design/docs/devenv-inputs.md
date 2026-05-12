# devenv.sh Inputs System
- **Source**: https://devenv.sh/inputs/
- **Retrieved**: 2026-05-12

## Overview

The inputs system in devenv functions as dependency management for developer environments. As stated in the documentation, "Inputs allow you to refer to Nix code outside of your project while preserving reproducibility."

## Default Configuration

If no `devenv.yaml` file exists, the system defaults to:

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  git-hooks:
    url: github:cachix/git-hooks.nix
```

## Accessing Inputs in devenv.nix

Dependencies declared as inputs become available as function arguments. For example, with a `devenv.yaml` containing:

```yaml
inputs:
  nixpkgs-stable:
    url: github:NixOS/nixpkgs/nixos-23.11
```

You can access these in `devenv.nix`:

```nix
{ inputs, pkgs, ... }:

let
  pkgs-stable = import inputs.nixpkgs-stable { system = pkgs.stdenv.system; };
in {
  packages = [ pkgs-stable.git ];

  enterShell = ''
    git --version
  ''
}
```

## Special Built-in Inputs

The system automatically provides three special inputs to `devenv.nix`:

- **pkgs**: Contains all available packages for your system from nixpkgs
- **lib**: A collection of Nix data structure manipulation functions
- **config**: The final resolved configuration, allowing references to other defined options

Example usage:

```nix
{ pkgs, lib, config, ... }:

{
  env.GREET = "hello";

  enterShell = ''
    echo ${config.env.GREET}
  '';
}
```

## Supported URI Formats

The `inputs.<name>.url` field supports multiple URI schemes:

### GitHub
- `github:NixOS/nixpkgs/master`
- `github:NixOS/nixpkgs?rev=238b18d7b2c8239f676358634bfb32693d3706f3`
- `github:org/repo?dir=subdir`
- `github:org/repo?ref=v1.0.0`

### GitLab
- `gitlab:owner/repo/branch`
- `gitlab:owner/repo/commit`
- `gitlab:owner/repo?host=git.example.org`

### Git Repositories
- `git+ssh://[email protected]/NixOS/nix?ref=v1.2.3`
- `git+https://git.somehost.tld/user/path?ref=branch&rev=fdc8ef970de2b4634e1b3dca296e1ed918459a9e`
- `git+file:///some/absolute/path/to/repo`

### Mercurial
- `hg+https://...`
- `hg+ssh://...`
- `hg+file://...`

### Sourcehut
- `sourcehut:~misterio/nix-colors/21c1a380a6915d890d408e9f22203436a35bb2de?host=hg.sr.ht`

### Tarballs
- `tarball+https://example.com/foobar.tar.gz`

### Local Files
- `path:/path/to/repo`
- `file+https://`
- `file:///some/absolute/file.tar.gz`

**Note**: Path inputs copy entire directories to the Nix store without respecting `.gitignore`. The documentation recommends using `git+file` instead to avoid unnecessary copying of large development directories.

## Following Inputs

Inputs can reference other inputs through the `follows` mechanism. This serves two purposes:

1. Inherit inputs from other `devenv.yaml` files or external flake projects
2. Reduce redundant downloads by overriding nested inputs

Nested inputs use `/` as a separator for referencing.

### Example: Inherit from Base Project

```yaml
inputs:
  base-project:
    url: github:owner/repo
  nixpkgs:
    follows: base-project/nixpkgs
```

### Example: Override Nested Input

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

## CLI Input Management

Add inputs without manual file editing:

```bash
$ devenv inputs add nixpkgs-stable github:NixOS/nixpkgs/nixos-23.11
```

Create new inputs that follow existing ones:

```bash
$ devenv inputs add my-input github:org/repo --follows nixpkgs
```

## Lock File Management

When executing devenv commands, the system resolves inputs like `github:NixOS/nixpkgs/nixpkgs-unstable` into specific commit revisions and records them in `devenv.lock`. This mechanism ensures reproducibility across environments.

To update inputs to newer commits, execute `devenv update` or configure specific revisions/branches at the input level through the `devenv.yaml` reference documentation.
