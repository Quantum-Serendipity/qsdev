# Three Years of Nix and NixOS: The Good, the Bad, and the Ugly
- **Source URL**: https://pierrezemb.fr/posts/nixos-good-bad-ugly/
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Pierre Zemb

## Duration
Three years

## Use Case
Personal Linux workstation; distributed systems development and testing; infrastructure management

## Key Complaints & Pain Points

### Daily Friction
- "On a normal system, if you want to add a shell alias, you edit `.bashrc` and you're done. In NixOS, there are no quick edits."
- Requires system rebuilds for minor configuration changes

### Learning Curve
- "Learning the Nix ecosystem is a big commitment" with steep isolation from standard Linux knowledge
- Requires mastering the Nix language, derivations, and Flakes
- "needing an AI to help with basic packaging shows how hard the language is to learn"

### Ecosystem Incompatibility
- Binary incompatibility due to non-standard filesystem hierarchy
- "you can't just download a pre-compiled binary and expect it to work"
- Requires `patchelf` modifications for pre-built binaries
- Build tools with hardcoded paths (e.g., Gradle Protobuf plugin) require workarounds
- Some cryptography libraries require `buildFHSUserEnv` fallback

### Language Barrier
- Functional programming paradigm differs significantly from mainstream languages
- "Simple things can be hard to figure out"

## Final Decision
**Stayed with NixOS.** Zemb emphasizes reproducibility as "a superpower" outweighing frustrations. He explicitly states he "wouldn't switch away from NixOS" despite acknowledging the challenges.

## Alternative Entry Recommendation
Recommends trying Nix package manager first on existing systems rather than full OS migration.
