<!-- Source: https://devenv.sh/blog/2024/09/11/devenv-11-nested-nix-outputs-using-the-module-system/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv 1.1: Module System and Nested Outputs

## Core Functionality

devenv 1.1 introduces support for Nix outputs designed to be "extensible, nested, and buildable as a whole by default." This feature allows developers to expose Nix packages for installation and consumption by other tools.

## Nested Outputs Structure

The system supports organizing outputs in nested attribute sets. For example:

```nix
outputs = {
  myproject.myapp = import ./myapp { inherit pkgs; };
  git = pkgs.git;
};
```

All outputs can be built collectively using `devenv build`, or specific attributes can be targeted individually, such as `devenv build outputs.git`.

## Module Options Integration

The documentation describes how outputs can be defined as module options within devenv configurations. Options are declared using the `lib.mkOption` pattern, with custom output types recognized through `config.lib.types.outputOf lib.types.package` or the generic `config.lib.types.output` type.

When options are configured this way, the system automatically detects and builds all specified outputs without requiring explicit enumeration.

## Composition Model

Importing other `devenv.nix` files results in merged outputs, allowing modular composition where each logical unit can define both its environment and associated outputs together.

The documentation notes that importing outputs from external applications as inputs (rather than through composition) is a proposed feature awaiting community interest.
