# Using devenv with Nix Flakes
- **Source**: https://devenv.sh/guides/using-with-flakes/
- **Retrieved**: 2026-05-12

## Overview

Devenv integrates with Nix Flakes through the `devenv.lib.mkShell` function, allowing developers to "specify dependencies as inputs" and "pin those dependencies in a lock file" while defining structured project outputs.

## Choosing Between Approaches

The documentation recommends the dedicated devenv CLI for most projects because it offers "simplicity," "performance," and is "purpose-built for development environments with integrated tooling." However, Flakes integration suits scenarios where you "maintain an existing flake-based project ecosystem" or need downstream flake consumption.

### Feature Comparison

The devenv CLI provides advantages including external flake inputs, shared remote configs, built-in container support, protection from garbage collection, faster lazy evaluation, evaluation caching, and secretspec.dev integration. Nix Flakes support pure evaluation and cross-project references but lack some developer-specific features.

## Setting Up a Flake Project

Initialize a new project using: `nix flake init --template github:cachix/devenv`

This creates a `flake.nix` file and `.envrc` for optional direnv integration.

## Minimal flake.nix Configuration

```nix
{
  inputs = {
    nixpkgs.url = "github:cachix/devenv-nixpkgs/rolling";
    devenv.url = "github:cachix/devenv";
  };

  nixConfig = {
    extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=";
    extra-substituters = "https://devenv.cachix.org";
  };

  outputs = { self, nixpkgs, devenv, ... } @ inputs:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = devenv.lib.mkShell {
        inherit inputs pkgs;
        modules = [
          ({ pkgs, config, ... }: {
            packages = [ pkgs.hello ];
            enterShell = '' hello '';
            processes.run.exec = "hello";
          })
        ];
      };
    };
}
```

## Entering the Shell

Use `nix develop --no-pure-eval` to evaluate inputs, create `flake.lock`, and open the devenv shell.

The `--no-pure-eval` flag is required because "Flakes use pure evaluation by default, which prevents devenv from figuring out the environment its running in." An alternative involves overriding `devenv.root` to an absolute path, though this sacrifices portability.

## Running Processes and Tests

Within the shell, launch processes and services using `devenv up`:

```
$ devenv up
17:34:37 system | run.1 started (pid=1046939)
17:34:37 run.1  | Hello, world!
```

Execute tests with `devenv test`. However, "running tests with flakes doesn't support starting processes" -- use the devenv CLI instead.

## Direnv Integration

1. Install nix-direnv
2. Download the template `.envrc` from the devenv repository
3. Run `direnv allow`

The template sets `DEVENV_IN_DIRENV_SHELL=true`, enabling `devenv up` to skip re-evaluation and use cached environments, providing faster startup.

## Multiple Development Shells

Define multiple shells in a single `flake.nix`:

```nix
{
  devShells.${system} = {
    projectA = devenv.lib.mkShell { /* config */ };
    projectB = devenv.lib.mkShell { /* config */ };
  };
}
```

Access each shell via `nix develop --no-pure-eval .#projectA` or `nix develop --no-pure-eval .#projectB`.

## External Flakes

Reference external flake configurations without modifying your project repository:

```
nix develop --no-pure-eval file:/path/to/central/flake#projectA
```

External flakes support `github:` and `git:` references but lack lock file certainty -- local flakes provide superior version control.
