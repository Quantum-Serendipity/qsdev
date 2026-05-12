<!-- Source: https://discourse.nixos.org/t/is-nix-vulnerable-to-supply-chain-attacks/72411 -->
<!-- Retrieved: 2026-05-12 -->

# Nix Supply Chain Attack Vulnerability Discussion

## Overview
The NixOS Discourse thread examines whether Nix is vulnerable to supply chain attacks, with participants debating trust models across different package management systems.

## Key Arguments

**Initial Concern:**
The questioner worried that someone with write access to nixpkgs could "inject malicious code to the build process" and distribute it via Hydra, comparing this unfavorably to Debian + Docker.

**Responses on Trust Models:**

tejing argues that trusting Nix isn't fundamentally different from other distributions. As stated: "In any distro, you always have to trust the distro's creators. They're the ones who package and distribute the core software."

Michael-C-Buckley reinforces this point, noting "The source of trust has to go somewhere. If it's not with the binaries, then it has to be with the source."

**Comparative Risk Analysis:**

TLATER challenges the premise that Docker/Debian offers superior protection, explaining that most Docker containers are "maintained by third parties" and that Debian explicitly separates developers from maintainers for review purposes. The assessment concludes: "The risk is probably roughly the same with all of them."

**Technical Mitigation:**

arianvp highlights Nix's unique advantage: users can "build everything from source" by removing binary cache substituters like `https://cache.nixos.org`, adding "That's the power of nix."

## Consensus
Supply chain vulnerability exists across all package managers, though Nix offers distinctive options for source-based verification.
