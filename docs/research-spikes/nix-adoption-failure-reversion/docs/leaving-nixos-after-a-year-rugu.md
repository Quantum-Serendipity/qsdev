<!-- Source: https://www.rugu.dev/en/blog/leaving-nixos/ -->
<!-- Retrieved: 2026-03-20 -->

# Why I'm Leaving NixOS After a Year

**Author:** Ugur Erdem Seyfi
**Date:** August 3, 2025
**Duration of NixOS Use:** Approximately one year
**Previous article:** "Switching from Arch to NixOS" published roughly one year earlier

## Key Pain Points & Frustrations

### Installation & Configuration Challenges
The author describes a three-option dilemma when new programs don't work as intended: debugging the NixOS module, creating manual systemd units, or using containerization. He notes that "NixOS hates pre-compiled programs," forcing users to learn specialized tools like nix-ld and buildFhsEnv.

### Abstraction Layer Problems
A central complaint centers on leaky abstractions. The author states: "When things go wrong, you now have an additional layer to worry about." He argues NixOS presents itself as declarative while remaining procedurally dependent on FHS-designed programs underneath.

### Time Cost
The author emphasizes that "Tasks that would take very little time on a traditional FHS-based distro can take significantly longer on NixOS." Despite NixOS's reproducibility promises, he found "no real benefits in practice" compared to traditional dotfiles.

## What He Liked About NixOS
- The underlying philosophy of reproducibility and declarative configuration
- System functionality when properly configured
- Theoretical benefits for multi-device synchronization and reproducibility

## The ROI Argument
His initial post predicted improved returns on investment with increased usage. Instead, the opposite occurred. He concludes the costs of learning and maintaining NixOS outweighed practical benefits for his use case.

## Resolution
The author switched back to Arch Linux, acknowledging NixOS remains valuable for users who heavily depend on cross-device configuration synchronization and require system-level reproducibility.
