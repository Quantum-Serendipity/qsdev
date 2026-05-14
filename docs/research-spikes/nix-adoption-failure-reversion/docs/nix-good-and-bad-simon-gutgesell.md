<!-- Source: https://nomisiv.com/blog/nix-good-and-bad -->
<!-- Retrieved: 2026-03-20 -->

# Nix, the Good and the Bad

**Author:** Simon Gutgesell
**Date:** 2024

## The Good Aspects

### Declarativity
Configuration is explicitly declared in files rather than imperatively executed, making it trivial to answer "how did I configure that?" questions.

### NixOS Generations
Each rebuild creates a boot entry, allowing easy rollback via nixos-rebuild switch --rollback if updates cause problems.

### Nix Shell
Users can temporarily access programs without permanent installation using nix shell nixpkgs#<program>.

### Direnv + Flakes
Development environments automatically load with all dependencies when entering project folders.

## The Mixed Aspects

### Nixpkgs Repository
While source code transparency and easy contributions are beneficial, the repository faces overwhelming pull request backlogs (+5k), making reviews and merges extremely difficult.

### Documentation
Multiple resources exist (manuals, wiki, guides), but single-page formats load slowly and content is sometimes incomplete or inconsistent.

## The Bad Aspects

### Build Times
Evaluation can take 10+ minutes before actual building begins, significantly hindering development cycles.

### Error Messages
"Stack traces filling up half your terminal buffer" with cryptic references, often requiring trial-and-error debugging rather than clear guidance.

### Dynamically Linked Libraries
Nix demands an "all or nothing" approach. External precompiled binaries typically fail since libraries aren't in standard locations, requiring workarounds like nix-alien.

## Conclusion
The author characterizes Nix as "pretty bad, but it's the best that there is," noting few viable alternatives (Guix lacks features; Docker is reportedly worse). They continue using Nix despite frustrations.
