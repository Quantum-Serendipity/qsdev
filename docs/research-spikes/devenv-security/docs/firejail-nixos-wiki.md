# Firejail on NixOS
- **Source**: https://wiki.nixos.org/wiki/Firejail
- **Retrieved**: 2026-05-12

## Overview

Firejail is "an easy to use SUID sandbox program that reduces the risk of security breaches by restricting the running environment of untrusted applications using Linux namespaces, seccomp-bpf and Linux capabilities."

## Installation

Enable Firejail system-wide by adding this to your NixOS configuration:

```nix
programs.firejail.enable = true;
```

## Basic Usage

Launch applications in a sandboxed environment directly:

```bash
firejail bash
```

## How wrappedBinaries Works

The `wrappedBinaries` configuration mechanism replaces standard application executables with Firejail-wrapped versions:

- **Overwrites the usual program path** so users run the sandboxed version automatically
- **Requires three key properties**: the executable path, an associated Firejail profile, and optional additional arguments
- **Persists across sessions** without requiring manual command invocation

Example wrapping Librewolf and Signal Desktop:

```nix
programs.firejail = {
  enable = true;
  wrappedBinaries = {
    librewolf = {
      executable = "${pkgs.librewolf}/bin/librewolf";
      profile = "${pkgs.firejail}/etc/firejail/librewolf.profile";
      extraArgs = [ "--ignore=private-dev" ];
    };
  };
};
```

## Nix Store Path Integration

Firejail configurations reference Nix store paths directly using `${pkgs.packageName}` syntax, allowing dynamic resolution of package locations. Profiles are sourced from `${pkgs.firejail}/etc/firejail/`, ensuring consistency with installed versions.

## Advanced Features

**Network isolation with Tor**: Configure virtual network bridges routing traffic through local Tor services.

**Desktop integration**: Symlink application icons into user directories or create system-wide icon packages.

## Key Limitation for devenv Use Case

Firejail uses SUID -- it requires system-level NixOS configuration (`programs.firejail.enable = true`), not per-project configuration. This makes it unsuitable for a portable devenv boilerplate that should work without system-level changes.
