# devenv 1.1: Nested Nix Outputs Using the Module System
- **Source**: https://devenv.sh/blog/2024/09/11/devenv-11-nested-nix-outputs-using-the-module-system/
- **Retrieved**: 2026-05-12

## Overview

Announces devenv 1.1's support for Nix outputs, "designed to make outputs extensible, nested, and buildable as a whole by default."

## Nested Nix Outputs

```nix
{ pkgs, ... }: {
  outputs = {
    myproject.myapp = import ./myapp { inherit pkgs; };
    git = pkgs.git;
  };
}
```

`devenv build` constructs all outputs. Target specific outputs: `devenv build outputs.git`.

## Defining Outputs as Module Options

```nix
{ pkgs, lib, config, ... }: {
  options = {
    myapp.package = lib.mkOption {
      type = config.lib.types.outputOf lib.types.package;
      description = "The package for myapp";
      default = import ./myapp { inherit pkgs; };
    };
  };
  config = {
    outputs.git = pkgs.git;
  }
}
```

Uses `config.lib.types.outputOf` or `config.lib.types.output` for typed options.

## Output Composition

When importing another devenv.nix file, outputs merge together. Importing outputs from external applications as inputs remains pending.
