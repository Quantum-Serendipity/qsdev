# Why I'm Leaving NixOS After a Year
- **Source URL**: https://www.rugu.dev/en/blog/leaving-nixos/
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Uğur Erdem Seyfi (handle: kugurerdem)

## Duration
Approximately one year

## What They Used It For
- Personal system configuration
- Server management with NixOS
- Experimentation with various setups and programs

## Specific Reasons for Leaving

1. **Troubleshooting Complexity:** When applications failed to work properly, the author faced multiple unsatisfying options: debugging NixOS modules (often requiring source code review), creating workarounds, or using containerization—negating NixOS's advantages.

2. **Pre-compiled Program Issues:** The system struggles with pre-compiled binaries. The author notes they were "forced to learn about other specific Nix tools (`nix-ld`, `buildFhsEnv`, creating your derivations, etc.)" just to handle common development scenarios.

3. **Leaky Abstractions:** Despite appearing declarative, NixOS is fundamentally procedural underneath. The author states: "when things go wrong, you now have an additional layer to worry about."

4. **Time Cost:** Configuration tasks requiring minutes on traditional Linux distributions consumed significantly more time on NixOS. The author emphasizes they didn't experience practical reproducibility benefits justifying this overhead.

5. **Impractical Philosophy-Reality Gap:** NixOS's declarative philosophy conflicts sharply with how underlying programs actually function.

## Switched To
Arch Linux

## Nuanced Takes
The author acknowledges NixOS provides genuine value for users managing multiple systems requiring strict reproducibility, but recognizes this doesn't match their personal workflow priorities.
