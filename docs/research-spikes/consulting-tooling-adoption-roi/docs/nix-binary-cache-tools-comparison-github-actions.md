# A Performance Comparison of 3 Nix Binary Cache Tools in GitHub Actions

- **Source**: https://zenn.dev/trifolium/articles/1a2eeca4775e56?locale=en
- **Retrieved**: 2026-03-20

## Overview

Detailed benchmark comparing three Nix binary cache tools in GitHub Actions:
- cachix-action (uses Cachix cloud storage)
- cache-nix-action (uses GitHub Actions Cache)
- magic-nix-cache-action (uses GitHub Actions Cache)

## Environment 1: Official Binary Cache Available (marp project)

### Job Time (Cache Utilization — warm cache)
| Tool | Time | vs. No Cache |
|------|------|-------------|
| No caching | 36.50s | baseline |
| cachix-action | 42.00s | 115% (slower) |
| cache-nix-action | 31.75s | 87% (best) |
| magic-nix-cache-action | 54.00s | 148% (slower) |

### Build Time Only (Cache Utilization)
| Tool | Time | vs. No Cache |
|------|------|-------------|
| No caching | 24.00s | baseline |
| cachix-action | 25.25s | 105% |
| cache-nix-action | 9.25s | 39% (best) |
| magic-nix-cache-action | 30.75s | 128% |

### Cache Generation Overhead
- magic-nix-cache-action generation: 428.50s (1174% overhead vs. no-cache baseline)

## Environment 2: Without Official Binary Cache (zenn project)

### Job Time (Cache Utilization — warm cache)
| Tool | Time | vs. No Cache |
|------|------|-------------|
| No caching | 66.25s | baseline |
| cachix-action | 40.50s | 61% (39% reduction) |
| cache-nix-action | 29.75s | 45% (55% reduction, best) |
| magic-nix-cache-action | 66.50s | 100% (no improvement) |

### Build Time Only (Cache Utilization)
| Tool | Time | vs. No Cache |
|------|------|-------------|
| No caching | 51.50s | baseline |
| cachix-action | 21.25s | 41% (59% reduction) |
| cache-nix-action | 9.00s | 17% (83% reduction, best) |
| magic-nix-cache-action | 38.50s | 75% (25% reduction) |

### Cache Generation Overhead
| Tool | Time | vs. No Cache |
|------|------|-------------|
| cachix-action | 99.00s | 149% |
| cache-nix-action | 108.25s | 163% |
| magic-nix-cache-action | 377.00s | 569% |

## Cache Storage
| Tool | Storage Backend | Files | Size |
|------|----------------|-------|------|
| cachix-action | Cachix | 287 | 69 MB |
| cache-nix-action | GitHub Actions | 2 | 1130 MB |
| magic-nix-cache-action | GitHub Actions | 2202 | 1177 MB |

## Key Findings

1. **cache-nix-action** was the best performer across both test scenarios
2. **cachix-action** showed good improvement when official binary cache was unavailable (39-59% build time reduction)
3. **magic-nix-cache-action** showed little to no improvement and massive overhead during cache generation
4. Results vary significantly depending on project configuration
5. The effect of caching was limited in some environments — overhead can exceed benefits

## Important Context

These benchmarks compare **Nix cache tools against each other and against uncached Nix builds** — they do NOT compare Nix builds against non-Nix (Docker, apt-get, etc.) builds. The improvements shown are within the Nix ecosystem only.
