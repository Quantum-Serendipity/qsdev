# devenv.sh Nix Recipes
- **Source**: https://devenv.sh/recipes/nix/
- **Retrieved**: 2026-05-12

## 1. Getting a Recent Package from nixpkgs-unstable

Default uses `devenv-nixpkgs/rolling` (monthly updates). For bleeding edge:

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  nixpkgs-unstable:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
```

```nix
{ pkgs, inputs, ... }:
let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
in
{
  packages = [ pkgs-unstable.elmPackages.elm-test-rs ];
}
```

## 2. Contributing to nixpkgs

Point to fork for testing:
```yaml
inputs:
  nixpkgs:
    url: github:username/nixpkgs/branch
```

## 3. Adding a Directory to $PATH

```nix
enterShell = ''
  export PATH="$HOME/.mix/escripts:$PATH"
'';
```

## 4. Escaping Nix Curly Braces

Use `''${varname}` to escape in shell scripts within Nix strings.
