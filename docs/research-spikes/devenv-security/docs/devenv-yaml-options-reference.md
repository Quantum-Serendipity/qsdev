# devenv.yaml Options Reference
- **Source**: https://devenv.sh/reference/yaml-options/
- **Retrieved**: 2026-05-12

## Core Configuration

**inputs**
- Type: attribute set of input
- Default: `inputs.nixpkgs.url: github:cachix/devenv-nixpkgs/rolling`
- Description: "Map of Nix inputs"

**inputs.<name>.url**
- Type: string
- Description: "URI specification of the input"

**inputs.<name>.flake**
- Type: boolean
- Default: `true`
- Description: "Does the input contain `flake.nix` or `devenv.nix`"

**inputs.<name>.follows**
- Type: string
- Description: "Another input to inherit from by name"

**inputs.<name>.overlays**
- Type: list of string
- Default: `[]`
- Description: "A list of overlays to include from the input"

## Package Management & Security

**nixpkgs.allow_unfree**
- Type: boolean
- Default: `false`
- Description: "Allow unfree packages"

**nixpkgs.allow_broken**
- Type: boolean
- Default: `false`
- Description: "Allow packages marked as broken"

**nixpkgs.allow_unsupported_system**
- Type: boolean
- Default: `false`
- Description: "Allow packages that are not supported on the current system"

**nixpkgs.allowed_licenses**
- Type: list of string
- Default: `[]`
- Description: "List of license names to allow using nixpkgs attribute names"

**nixpkgs.blocklisted_licenses**
- Type: list of string
- Default: `[]`
- Description: "List of license names to block using nixpkgs attribute names"

**nixpkgs.permitted_unfree_packages**
- Type: list of string
- Default: `[]`
- Description: "List of unfree packages to allow by name"

**nixpkgs.permitted_insecure_packages**
- Type: list of string
- Default: `[]`
- Description: "List of insecure permitted packages"

## Environment Management

**clean.enabled**
- Type: boolean
- Default: `false`
- Description: "Clean the environment when entering the shell"

**clean.keep**
- Type: list of string
- Default: `[]`
- Description: "Environment variables to retain during cleaning"

**impure**
- Type: boolean
- Default: `false`
- Description: "Relax the hermeticity of the environment"

## Additional Security-Relevant

**require_version**
- Type: boolean | string
- Description: Enforce specific devenv CLI version

**secretspec.enable**
- Type: boolean
- Default: `false`
- Description: Enable secret specification support
