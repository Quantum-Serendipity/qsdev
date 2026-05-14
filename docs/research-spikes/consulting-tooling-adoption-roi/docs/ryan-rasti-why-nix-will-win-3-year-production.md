# Why Nix Will Win (and What's Stopping It): A 3-Year Production Story — Ryan Rasti

- **Source**: https://ryanrasti.com/blog/why-nix-will-win/
- **Retrieved**: 2026-03-20

## Team & Project Context
- **Team size**: 4 engineers
- **Tech stack**: Full-stack Elixir/React application
- **Duration**: 3-year production deployment

## Build Time Improvements
- **Early CI gains**: Reduced from 3 minutes to under 1 minute initially
- **Sustained performance**: As the project scaled, builds remained under 15 minutes despite complexity that would typically exceed 30 minutes
- **Attribution**: Aggressive caching mechanisms enabled these sustained gains

## Deployment Speed
- Emergency fix deployment: Able to deploy production fix from a developer's laptop with byte-for-byte identical binaries to what CI would generate

## Infrastructure Overhead
- CI runner maintenance: Self-hosted GitHub Actions runners required setup and ongoing maintenance to leverage Nix's caching benefits effectively

## Tradeoffs and Downsides
- Cross-platform Mac-to-Linux builds remained problematic
- Hermetic web app building required custom tooling forks
- Steep learning curve for new engineers
- Antiquated tooling lacking autocomplete and inline documentation

## Analysis

This is the single most-cited data point in the corpus for Nix CI improvements. Key limitations:
1. **One team, one project** — 4 engineers on one Elixir/React stack
2. **Self-hosted runners** — improvements partially attributable to self-hosted infrastructure, not just Nix
3. **"Typically exceed 30 minutes"** — the baseline is an estimate, not a measured pre-Nix build time
4. **Small scale** — 4 engineers does not represent consulting firm or enterprise scale
5. The 3 min -> under 1 min early improvement (~67% reduction) and 30+ min -> under 15 min sustained (~50% reduction) are consistent with but do not prove the "50-75%" claim
