<!-- Source: https://www.dgt.is/blog/2025-01-10-nix-death-by-a-thousand-cuts/ -->
<!-- Retrieved: 2026-03-20 -->

# Nix - Death by a Thousand Cuts

**Author:** Jono (jono.codes / dgt.is)
**Date:** January 10, 2025

## Author Background
Highly experienced software engineer and DevOps professional with decades of industry experience. Has used Linux daily since the 1990s, attended NixCon US, engaged with the community, donated to the NixOS Foundation, and spent approximately 2 years using NixOS as their primary desktop OS.

## Core Thesis
"In its current state (2025), I don't generally recommend desktop use of Nix(OS), even for seasoned Linux users." Despite loving Nix conceptually, they found it created more problems than solutions through incremental friction rather than catastrophic failures.

## Specific Pain Points & Criticisms

### 1. Excessive Complexity & Multiple Approaches
The fundamental problem: "Using a Nix system often feels like using a programming language. There are many ways to do the same thing." Users must navigate numerous competing patterns (home-manager, nix-darwin, Flakes vs legacy configs, overlays, FHS environments) without clear guidance on which approach to use.

### 2. Resource & Bandwidth Consumption
- Daily updates routinely download 500MB (comparable to entire distro installations)
- Heavy disk write pressure problematic for SD cards and devices with limited write cycles
- Build processes can grind machines with 16 cores/64GB RAM to unusable states
- GitHub API throttling requires credential management for simple package updates

### 3. Package Quality & Currency Issues
- "Nix boasts that is has more packages then other distros" but quantity doesn't reflect quality
- Unstable branch packages lag years behind upstream sources (author found examples of packages *years* behind)
- Inconsistent package integration: some have extensive Nix-specific configuration options while others provide only cursory support
- Requires manual workarounds for programs like Duplicati and rclone

### 4. Language & Learning Curve
The Nix language itself presents barriers: "I don't appreciate the logic and structure, and having to learn a brand new language before you can even turn the machine or send an email rubs me the wrong way."

Lacks adequate tooling -- no comprehensive language server with autocomplete for available options, forcing users to search fragmented online discussions.

### 5. Documentation Fragmentation
Official documentation assumes substantial Nix knowledge. Unlike Arch Wiki's clear command-to-result approach, NixOS docs read as "Here are some things that worked for me" with outdated blog post references and inconsistent guidance.

### 6. Build Failure Diagnostics
Error messages are notoriously cryptic. Debugging requires tracing through discussions, and advanced debugging tools like breakpoints are poorly documented.

### 7. Desktop Integration Issues
XDG/.desktop file associations unreliable -- sometimes requiring logout, reboot, or Wayland/X11 switching. Desktop manager limitations prevent running both GNOME and KDE simultaneously without modifying configs. Display manager functionality compromised.

### 8. Development Environment Friction
"...there are times that the environment breaks and I cant get any work done." Nix shells add layers of abstraction that complicate IDE integration. Python development with conda/pip/pdm requires three-level nesting (Nix shell -> conda shell -> conda environment). IDE launches (PyCharm) require terminal activation rather than direct GUI launching.

### 9. Configuration Cruft & Maintenance Burden
Author's configs filled with TODO/NOTE comments documenting abandoned attempts. Cryptic incantations (like AppImage binfmt registrations with hex masks) lack clarity. Messy symlink workarounds for partial Nix adoption create "guilt" over impurity flags.

### 10. Legacy & Hidden Quirks
Package search misrepresents availability -- found packages marked available for aarch64-darwin that actually aren't installable. These quirks documented through community forums rather than official channels.

## Concrete Examples of Failures

### ZFS Encryption Setup
Directed to unmaintained personal GitHub repo for LUKS+ZFS configuration. Account disappeared months later, leaving user with magical incantations they don't understand. Setup remains stable but fragile.

### Firefox Extensions
Multiple attempted configurations commented out in configs. Never achieved reliable declarative management.

### Conda Integration
Official docs claim simplicity; reality involved wrestling with fish shell integration. Found 4-year-old Conda derivation. Required filing bug report (resolved after 5 months). Workflow now: Nix shell -> Conda shell -> Conda environment activation.

### npm link & Monorepo Development
Nix rejected symlink-based npm linking, forcing migration to modern npm workspaces.

### Syncthing State Management
User services through home-manager lack state management support. Syncthing constantly asks to re-add shares despite config declarations.

### Flatpak Integration
System rebuilds hang for 10 minutes during package installation with no feedback. Official config option causes extended build times. Manual Flatpak operations conflict with Nix's immutability paradigm but aren't truly prevented.

### Display Manager Choice
Switching between GNOME and KDE required maintaining separate configs with differing lines -- rebuilds take seconds but the principle violates display manager functionality.

## What Worked Well

### Declarative Configuration
Successfully swapped entire system configurations between two laptops for a week-long conference trip. "This was so easy - I could not have dreamed up something like this before Nix."

### Simple Services
"Need to run a database? A web server with SSL? A sound service? These mostly can be done with a single line or two." Sweet spot: machines running under 10 services (mail servers, NAS, Nextcloud). Reproducibility exceeds Docker's non-deterministic builds.

### Ephemeral Shells & nix run
Genuine superpowers for temporary environments and trying new programs without system installation.

### Syncthing Configuration
"syncthing options in Nix are first class" with elegant declarative setup exceeding non-Nix systems.

### Home-manager Dotfiles
User configuration management superior to symlink-based approaches for groups, shell, git, SSH configs.

## Community Assessment
Positive: Enthusiastic, helpful technical community; steady PR contributions from users getting started.

Negative: Community-dependent distribution where users must be contributors. 2024 political turmoil drove away core contributors. GitHub fragmentation means no central package discussion space (unlike AUR's unified pages). Server costs remain significant burden on NixOS Foundation.

## Decision & Path Forward

**Current Status:** Scaling back Nix adoption but not abandoning entirely.

**Future Plans:**
- Keep NixOS on home servers (where simplicity shines)
- Migrate active workstations to: Nix package manager + home-manager only, or potentially return to traditional Linux systems temporarily
- Described self as experiencing "Nix Purgatory" -- unable to return to pre-Nix chaos but unable to escape incremental pain

**Quote capturing sentiment:** "NixOS is shit. The problem is, all other OS are even worse."

## Constructive Suggestions (Implied)
1. Standardized workflows for common use cases (desktop, development)
2. Language server with IDE autocomplete for available options
3. Centralized package discussion (separate from GitHub issues)
4. Granular package pinning simpler than checksum-based references
5. Better error messages and debugging tooling
6. Community mirroring/CDN solutions for bandwidth management
7. Documentation that addresses variability in supported packages
8. Display manager that actually allows simultaneous desktop environments

## Notable Quotes
"I spent so much time cobbling together hacky configs that when I step back and look at the house of cards I have built, it is apparent Nix has created more problems then solutions for me."

"Can you really ever make it to Nix Nirvana? ...It's true -- you may always be seeing the amazing benefits of Nix, but you will also constantly be struggling with or mucking with configs."
