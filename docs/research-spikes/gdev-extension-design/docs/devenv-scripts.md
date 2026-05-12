# devenv.sh Scripts
- **Source**: https://devenv.sh/scripts/
- **Retrieved**: 2026-05-12

## Overview
The Scripts section documents how to define and manage shell scripts within devenv projects. Scripts are exposed when entering the environment and can leverage packages defined in the configuration.

## Basic Example
A simple script definition includes executable code and required packages:

```nix
{ pkgs, ... }:
{
  packages = [ pkgs.curl pkgs.jq ];
  scripts.silly-example.exec = ''
    curl "https://httpbin.org/get?$1" | jq '.args'
  '';
}
```

When executed in the devenv shell, this script processes JSON from an HTTP endpoint.

## Aliases & Arguments
Scripts can forward command-line arguments using the `"$@"` pattern:

```nix
scripts.foo.exec = ''
  npx @foo/cli "$@";
'';
```

## Runtime Packages
Dependencies can be scoped to individual scripts without polluting the global environment:

```nix
scripts.analyze-json = {
  exec = ''
    curl "https://httpbin.org/get?$1" | jq '.args'
  '';
  packages = [ pkgs.curl pkgs.jq ];
  description = "Fetch and analyze JSON";
};
```

## Pinning Package Paths
Scripts can directly reference package locations:

```nix
scripts.silly-example.exec = ''
  ${pkgs.curl}/bin/curl "https://httpbin.org/get?$1" | ${pkgs.jq}/bin/jq '.args'
'';
```

## Language-Specific Scripts
Scripts support multiple languages through the `package` and `binary` attributes:

**Python Example:**
```nix
scripts.python-hello = {
  exec = ''
    print("Hello, world!")
  '';
  package = config.languages.python.package;
  description = "hello world in Python";
};
```

**Nu Shell Example:**
```nix
scripts.nushell-greet = {
  exec = ''
    def greet [name] {
      ["hello" $name]
    }
    greet "world"
  '';
  package = pkgs.nushell;
  binary = "nu";
  description = "Greet in Nu Shell";
};
```

**External File:**
```nix
scripts.file-example = {
  exec = ./file-script.sh;
  description = "Script loaded from external file";
};
```

## Shell Entry Helper
The documentation includes an advanced example displaying available scripts upon shell entry using `enterShell` and helper utilities to format script descriptions.
