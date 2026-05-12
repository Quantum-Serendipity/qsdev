# devenv.yaml Complete Options Reference (May 2026)
- **Source**: https://devenv.sh/reference/yaml-options/
- **Retrieved**: 2026-05-12

## Clean Options
- **clean.enabled**: boolean | Default: false | "Clean the environment when entering the shell"
- **clean.keep**: list of string | Default: [] | "Environment variables to keep when cleaning"

## Core Behavior
- **impure**: boolean | Default: false | "Relax the hermeticity of the environment"
- **reload**: boolean | Default: true | Auto-reload when files change
- **strict_ports**: boolean | Default: false | Error if port in use vs auto-allocate

## Inputs Management
- **inputs**: attribute set of input | Default: nixpkgs rolling
- **inputs.<name>.url**: string | Input URI specification
- **inputs.<name>.flake**: boolean | Default: true | Whether input contains flake.nix
- **inputs.<name>.follows**: string | Inherit from another input
- **inputs.<name>.overlays**: list of string | Default: []

## Nixpkgs License & Package Control
- **nixpkgs.allow_unfree**: boolean | Default: false
- **nixpkgs.allow_broken**: boolean | Default: false
- **nixpkgs.allowlisted_licenses**: list of string | Default: []
- **nixpkgs.blocklisted_licenses**: list of string | Default: []
- **nixpkgs.permitted_unfree_packages**: list of string | Default: []
- **nixpkgs.permitted_insecure_packages**: list of string | Default: []

## Secretspec Integration
- **secretspec.enable**: boolean | Default: false
- **secretspec.profile**: string
- **secretspec.provider**: string

## Version Control
- **require_version**: boolean | string | Enforce specific CLI version
