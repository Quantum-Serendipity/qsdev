# A Look at NixOS/Nixpkgs Evaluation Times Over the Years

- **Source**: https://discourse.nixos.org/t/a-look-at-nixos-nixpkgs-evaluation-times-over-the-years/65114
- **Retrieved**: 2026-03-20

## Key Performance Metrics

### NixOS Evaluation Times (NixOS 15.09 to 25.05)
- 2015: ~0.4 seconds
- 2025: ~3 seconds
- **7.5x slowdown** for minimal configurations over 10 years

### Trend
"NixOS is getting slower faster than Nix is getting faster." While the Nix evaluator itself has improved, these gains are overshadowed by NixOS/Nixpkgs complexity increases.

## Identified Bottlenecks

- **Documentation generation:** "quite heavy hitter" involving extensive source filtering operations
- **pkgs/by-name implementation:** ~100-150ms overhead, requires traversing entire directory tree
- **Attrset merging:** inefficient `binaryMerge` operations due to lacking native Nix builtins

## Relevance to CI

Evaluation time is a component of every Nix CI build. For large flakes or NixOS configurations, evaluation overhead adds seconds-to-minutes before any actual building begins. This is an inherent overhead that conventional CI approaches (Docker, apt-get) do not have.
