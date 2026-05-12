# Content-Addressed Derivations in Nix
- **Source**: https://wiki.nixos.org/wiki/Ca-derivations
- **Retrieved**: 2026-05-12

## Overview
Content-addressed (CA) derivations represent an experimental feature extending Nix's package model. This enhancement enables "early cutoff" optimization and modifies Nix's trust architecture.

## How They Work
CA derivations allow the system to "stop a rebuild if it can be proved that the end-result will be the same as something already known." This mechanism promises significant reductions in computational and storage overhead for both individual systems and build infrastructure like Hydra.

## Current Status
**Experimental and Unstable**: The feature remains available only in unstable Nix versions and requires explicit activation. As of the last wiki update (September 18, 2025), CA derivations have not reached stable status.

## Enabling CA Derivations

**On NixOS**, add to configuration.nix:
```nix
nix.settings.experimental-features = ["ca-derivations"]
```

**Non-NixOS systems** should ensure `/etc/nix/nix.conf` contains:
```
experimental-features = ca-derivations
```

## Implementation
Individual derivations require opt-in by setting `__contentAddressed = true` in `mkDerivation` calls, or globally via `config.contentAddressedByDefault = true`.

## Security and Trust Model
The documentation notes CA derivations "change the Trust model of Nix, allowing for example several users to share the same store without trusting each other." This represents a fundamental shift in how systems manage authenticated package access.

## Verification
Verify proper implementation using `nix path-info --sigs {outPath}`, which should return output containing `ca:fixed:r:…` notation.
