# Why I stopped using NixOS and went back to Arch Linux
- **Source URL**: https://dev.to/sleeyax/why-i-stopped-using-nixos-and-went-back-to-arch-4070
- **Retrieved**: 2026-03-20
- **Type**: Blog post (DEV Community)

## Author
Sleeyax

## Duration of NixOS Use
~1 year (installed May 17, 2024)

## Use Case
Daily driver on laptop; also runs Arch Linux on desktop PC

## Reasons for Switching Away

1. **System Instability:** Frequent breakdowns requiring configuration fixes before updates succeed. Components randomly fail after reboot (audio, Bluetooth, Electron apps). Author notes: "I broke Arch only once in 5 years whereas NixOS already breaks before updating."

2. **Cryptic Error Messages:** Stack traces are verbose but unhelpful, with actual errors buried at the bottom. Real issue example: logseq package removed without alternatives suggested.

3. **Massive Update Sizes:** NixOS keeps multiple package generations alongside old versions instead of replacing in-place. Every glibc update triggers rebuilds of dependent packages, causing disk bloat.

4. **Compilation Overhead:** Regular maintenance updates take 4-5+ hours on slower hardware. Binary caches frequently miss packages due to system differences, forcing local compilation.

5. **Poor Documentation:** Arch Wiki provides better answers than NixOS Wiki. Documentation is vague, outdated, and assumes deep Nix familiarity. Practical examples scarce.

## Switched To
Arch Linux

## Things Author Appreciated About NixOS
- Reproducible system configuration
- Ability to revert to previous generations
- Declarative configuration approach
- System immutability concept

## Author's Caveat
Acknowledges NixOS benefits exist but aren't worth the daily friction for desktop use; suggests enterprise/specific use cases remain viable.
