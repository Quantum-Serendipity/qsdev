# devenv Binary Caching Configuration
- **Source**: https://devenv.sh/binary-caching/
- **Retrieved**: 2026-05-12

## Core Integration

devenv provides "seamless integration with binary caches hosted by Cachix" without requiring separate client installation. The system handles caching automatically once configured.

## Cache Configuration

**Pull Configuration:**
Users specify caches to download binaries from via cachix.pull = [ "mycache" ]; in devenv.nix. By default, "devenv.cachix.org is added to the list of pull caches" automatically, which mirrors the official NixOS cache.

**Push Configuration:**
Binary pushing is configured with cachix.push = "mycache"; in devenv.nix. The documentation recommends conditional pushing through devenv.local.nix for CI environments to prevent every user from pushing to caches.

## Authentication

Push access requires setting CACHIX_AUTH_TOKEN=XXX as an environment variable, using either personal auth tokens or per-cache tokens created in cache settings.

## Trust Model & Security Details

The documentation does NOT discuss:
- nix.conf modifications
- trusted-users requirements
- Specific trust models or signature verification
- Permission escalation details

## Disabling Integration

Users can completely disable Cachix integration via cachix.enable = false; while maintaining access to externally configured caches like the official NixOS cache.
