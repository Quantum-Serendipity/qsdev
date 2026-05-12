<!-- Source: https://devenv.sh/binary-caching/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv Binary Caching Documentation

## Overview

devenv integrates with Cachix to provide seamless binary caching for Nix packages. The system allows developers to avoid rebuilding packages from source by utilizing pre-built binaries from configured caches.

## Default Caching Behavior

The official Nix binary cache (cache.nixos.org) supplies pre-compiled binaries for most packages. When packages are modified or sourced from non-upstream locations, Nix compiles them locally instead of downloading binaries.

devenv.cachix.org is automatically included as a default pull cache, mirroring the official NixOS cache for the devenv-nixpkgs/rolling nixpkgs input.

## Configuration Requirements

**Authentication Setup:**
Users must configure a `CACHIX_AUTH_TOKEN` environment variable using either a personal authentication token or a per-cache token from Cachix settings. No separate Cachix client installation is necessary.

**Initial Setup:**
1. Create an account on Cachix.org
2. Establish an organization and cache
3. Set the authentication token

## Cache Configuration in devenv.nix

**Pulling Binaries:**
```
{ cachix.pull = [ "mycache" ]; }
```

**Pushing Binaries:**
```
{ cachix.push = "mycache"; }
```

## Conditional Pushing Strategy

Rather than enabling binary uploads for all users, teams typically restrict pushing to CI environments or explicitly enable it via devenv.local.nix:

```
echo '{ cachix.push = "mycache"; }' > devenv.local.nix
```

This approach prevents unnecessary cache uploads from individual developer machines.

## Disabling Cachix Integration

The integration can be disabled entirely while maintaining external cache access:

```
{ cachix.enable = false; }
```

Nix will continue using externally configured caches, including the official NixOS cache.
