<!-- Source: https://carlosbecker.com/posts/bye-nix/ -->
<!-- Retrieved: 2026-03-20 -->

# Moving on from Nix

**Author:** Carlos Alexandro Becker
**Date:** June 17, 2025
**Duration of Nix Use:** Over 2 years (dotfiles management on macOS)

## Why He Left Nix

### 1. Total System Takeover
Nix requires "nixifying everything," preventing use of popular plugin managers like Lazy, Mason, and tpm since nix owns read-only folders. Desktop applications had poor integration -- Spotlight ignored the Nix Apps symlinks -- forcing continued reliance on Homebrew anyway.

### 2. Perpetually Outdated Packages
Even nix-unstable lags significantly behind upstream releases. Packaging custom versions requires copying .nix files, updating versions, and potentially cascading dependency updates. As Becker notes, this becomes "a lot of work" with unreliable results.

### 3. Excruciatingly Slow Applies
Configuration changes took several minutes to apply, making even simple edits feel glacially slow compared to the value proposition offered.

### 4. Reproducibility Wasn't Necessary
For personal machines, reproducibility provided minimal practical benefit. Becker changed laptops only four times in a decade and still encountered setup issues post-nix despite the theoretical guarantees. He prioritizes backups over hermetic builds.

### 5. Massive Disk Overhead
The nix store consumed over 60GB despite aggressive garbage collection -- versus 6GB for equivalent Homebrew packages on the same Mac.

### 6. Contributing Remains Difficult
The nixpkgs repository is unwieldy (4.7GB cloned), making contributions feel burdensome due to git operations alone.

### 7. Complexity Without Commensurate Benefit
The system violated KISS principles for his use case.

## Current Solution
Becker replaced nix with a lightweight shell script for dotfile management, reinstated Homebrew for macOS packages, and acknowledges this approach sacrifices reproducibility but prioritizes simplicity and efficiency -- "good enough" for personal computing needs.

## Nuanced Take
He recognizes nix remains valuable where genuine reproducibility needs exist, particularly in NixOS contexts, but determined his laptop doesn't require that level of control.
