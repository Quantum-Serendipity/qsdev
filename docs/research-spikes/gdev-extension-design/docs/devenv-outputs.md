# devenv.sh Outputs System
- **Source**: https://devenv.sh/outputs/
- **Retrieved**: 2026-05-12

## Overview

The outputs feature, introduced in version 1.1, enables definition of Nix derivations through devenv's module system. Exposes packages for consumption by external tools.

## Language Integration

Each language provides an `import` function utilizing ecosystem-specific packaging tools:

```nix
{ config, ... }: {
  languages.rust.enable = true;
  languages.python.enable = true;

  outputs = {
    rust-app = config.languages.rust.import ./rust-app {};
    python-app = config.languages.python.import ./python-app {};
  };
}
```

- **Rust**: Employs `crate2nix`
- **Python**: Utilizes `uv2nix`

## Building Outputs

```
$ devenv build          # all outputs
$ devenv build outputs.rust-app  # specific output
```

## Custom Module Options

```nix
{ pkgs, lib, config, ... }: {
  options = {
    myapp.package = pkgs.lib.mkOption {
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

Available types: `config.lib.types.outputOf` (explicit type) and `config.lib.types.output` (simplified).
