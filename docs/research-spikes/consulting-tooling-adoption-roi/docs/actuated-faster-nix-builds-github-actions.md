# Faster Nix Builds with GitHub Actions and Actuated

- **Source**: https://actuated.com/blog/faster-nix-builds
- **Retrieved**: 2026-03-20

## x86_64 Build Times

| Runner | Build Time |
|--------|-----------|
| GitHub hosted runner | ~4 min 20 sec |
| Actuated runner (4 CPU, 8GB RAM) | ~2 min 15 sec |

**Improvement:** Approximately 50% faster on more powerful hardware.

## aarch64 Build Times (QEMU Emulation vs. Native)

| Platform | Build Time |
|----------|-----------|
| GitHub Runner with QEMU | ~55 min |
| Actuated Runner with QEMU | ~19 min 40 sec |
| Native Raspberry Pi 4 (8GB) | ~11 min 47 sec |
| Raspberry Pi 4 with NVMe over USB-C | 10 min 49 sec |
| Ampere Altra on Equinix Metal | 3 min 29 sec |
| AMD Epyc on Equinix Metal (x86_64) | 1 min 57 sec |

## Key Finding

"Whatever the Arm hardware you pick, it'll likely be faster than QEMU, even when QEMU is run on the fastest bare-metal available, the slowest Arm hardware will beat it by minutes."

## Relevance to CI/CD Benchmarking

This article compares hardware runners, not Nix vs. conventional CI. The improvements shown are from faster hardware, not from Nix-specific advantages. The data is useful for understanding CI infrastructure costs but does NOT support claims about Nix reducing build times relative to Docker/conventional approaches.
