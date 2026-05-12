# devenv top-level.nix: unsetEnvVars Defaults and Security Options
- **Source**: https://raw.githubusercontent.com/cachix/devenv/main/src/modules/top-level.nix
- **Retrieved**: 2026-05-12

## Default unsetEnvVars List (26 variables)

HOST_PATH, NIX_BUILD_CORES, __structuredAttrs, buildInputs, buildPhase, builder, depsBuildBuild, depsBuildBuildPropagated, depsBuildTarget, depsBuildTargetPropagated, depsHostHost, depsHostHostPropagated, depsTargetTarget, depsTargetTargetPropagated, dontAddDisableDepTrack, doCheck, doInstallCheck, nativeBuildInputs, out, outputs, patches, phases, preferLocalBuild, propagatedBuildInputs, propagatedNativeBuildInputs, shell, shellHook, stdenv, strictDeps

## Security-Relevant Elements

**Hardening Options:**
The module includes a `hardeningDisable` option allowing selective disabling of hardening protections, currently used for Go toolchains.

**Assertions:**
One assertion validates overlay compatibility: "Using overlays requires devenv 1.4.2 or higher, while your current version is [version]."

**SDK Handling:**
On macOS, the default stdenv applies filtering to "remove the default apple-sdk" while allowing optional custom SDKs via configuration.

**Path Security:**
The runtime directory uses a hashed, abbreviated path component to ensure uniqueness, determinism, and compliance with Unix socket length constraints across operating systems.
