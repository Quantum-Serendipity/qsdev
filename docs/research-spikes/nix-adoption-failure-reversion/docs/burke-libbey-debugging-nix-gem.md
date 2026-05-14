<!-- Source: https://notes.burke.libbey.me/debugging-nix-gem/ -->
<!-- Retrieved: 2026-03-20 -->

# Debugging a Nix Build Failure - Burke Libbey

## Context
Burke Libbey documented troubleshooting a Nix compilation issue while moving Shopify development environments to Nix. After contributing improvements to Bundix for fetching gems from private servers, he encountered a build failure with the gctrack gem (version 0.1.0).

## The Error
Compilation failed with missing C99 declarations:
> "implicit declaration of function 'clock_gettime' is invalid in C99"

The compiler couldn't find `CLOCK_MONOTONIC`.

## Investigation Method

1. **Located the source**: Found a related GitHub issue on nixpkgs with identical problem but no solution
2. **Created minimal reproduction**: Built simple test case using `bundlerEnv` with Ruby 2.5
3. **Followed the call chain**: Traced from `bundlerEnv` through nixpkgs source to `bundled-common`, then `buildRubyGem`, to `/pkgs/development/ruby-modules/gem/default.nix`
4. **Used Nix tooling**: Employed `nix show-derivation` to inspect the actual build configuration

## Root Cause
Mismatch between impure system dependencies and available build inputs. The derivation had:
> "impure dependencies like /bin/sh and /usr/lib/libSystem.B.dylib present without corresponding headers"

The build lacked explicit header files despite having access to the runtime library through impure system inputs — a macOS-specific Nix limitation.

## Relevance
This illustrates the kind of deep, platform-specific debugging that Shopify engineers had to do during the first Nix adoption attempt. Building Ruby gems with native extensions on macOS via Nix was a significant source of friction — especially relevant for a Ruby-on-Rails shop like Shopify.
