# devenv.yaml Reference
- **Source**: https://devenv.sh/reference/yaml-options/
- **Retrieved**: 2026-05-12

## Configuration Options

**clean.enabled**
- Type: boolean
- Default: false
- Added in: 1.0
- Description: "Clean the environment when entering the shell."

**clean.keep**
- Type: list of string
- Default: []
- Added in: 1.0
- Description: Environment variables retained during cleanup

**imports**
- Type: list of string
- Default: []
- Description: "A list of relative paths, absolute paths, or references to inputs to import `devenv.nix` and `devenv.yaml` files."

**impure**
- Type: boolean
- Default: false
- Added in: 1.0
- Description: "Relax the hermeticity of the environment."

**inputs**
- Type: attribute set of input
- Default: inputs.nixpkgs.url: github:cachix/devenv-nixpkgs/rolling
- Description: Map of Nix inputs

**inputs.<name>.flake**
- Type: boolean
- Default: true
- Description: Whether input contains flake.nix or devenv.nix

**inputs.<name>.follows**
- Type: string
- Description: "Another input to 'inherit' from by name."

**inputs.<name>.inputs.<name>.follows**
- Type: string
- Description: Override nested inputs by name

**inputs.<name>.overlays**
- Type: list of string
- Default: []
- Description: Overlays to include from the input

**inputs.<name>.url**
- Type: string
- Description: URI specification of the input

**nixpkgs.android_sdk.accept_license**
- Type: boolean
- Default: false
- Description: Accept Android SDK license; settable via environment variable

**nixpkgs.allow_broken**
- Type: boolean
- Default: false
- Added in: 1.7
- Description: "Allow packages marked as broken."

**nixpkgs.allow_non_source**
- Type: boolean
- Default: true (nixpkgs default)
- Description: "Allow packages not built from source."

**nixpkgs.allow_unsupported_system**
- Type: boolean
- Default: false
- Added in: 2.0.5
- Description: "Allow packages that are not supported on the current system."

**nixpkgs.allow_unfree**
- Type: boolean
- Default: false
- Added in: 1.7
- Description: "Allow unfree packages."

**nixpkgs.allowlisted_licenses**
- Type: list of string
- Default: []
- Description: License names to allow using nixpkgs license attribute names

**nixpkgs.blocklisted_licenses**
- Type: list of string
- Default: []
- Description: License names to block using nixpkgs license attribute names

**nixpkgs.cuda_capabilities**
- Type: list of string
- Default: []
- Added in: 1.7
- Description: "Select CUDA capabilities for nixpkgs."

**nixpkgs.cuda_support**
- Type: boolean
- Default: false
- Added in: 1.7
- Description: "Enable CUDA support for nixpkgs."

**nixpkgs.rocm_support**
- Type: boolean
- Default: false
- Added in: 2.0.7
- Description: "Enable ROCm support for nixpkgs."

**nixpkgs.permitted_insecure_packages**
- Type: list of string
- Default: []
- Added in: 1.7
- Description: "A list of insecure permitted packages."

**nixpkgs.permitted_unfree_packages**
- Type: list of string
- Default: []
- Added in: 1.9
- Description: "A list of unfree packages to allow by name."

**nixpkgs.per_platform.<system>**
- Type: attribute set of nixpkgs config
- Added in: 1.7
- Description: Per-platform nixpkgs configuration with same options as nixpkgs

**profile**
- Type: string
- Added in: 1.11
- Description: "Default profile to activate. Can be overridden by `--profile` CLI flag."

**reload**
- Type: boolean
- Default: true
- Added in: 2.0
- Description: "Enable auto-reload of the shell when files change."

**secretspec.enable**
- Type: boolean
- Default: false
- Added in: 1.8
- Description: "Enable secretspec integration."

**secretspec.profile**
- Type: string
- Added in: 1.8
- Description: "Secretspec profile name to use."

**secretspec.provider**
- Type: string
- Added in: 1.8
- Description: "Secretspec provider to use."

**strict_ports**
- Type: boolean
- Default: false
- Description: "Error if a port is already in use instead of auto-allocating the next available port."

**require_version**
- Type: boolean | string
- Default: not set
- Added in: 2.1
- Description: Enforce specific devenv CLI version; accepts constraint operators (>=, <=, >, <, =) or bare version for exact match
