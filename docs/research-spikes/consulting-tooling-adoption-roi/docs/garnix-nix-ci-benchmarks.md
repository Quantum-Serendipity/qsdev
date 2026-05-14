# Nix CI Benchmarks — Garnix

- **Source**: https://garnix-io.github.io/benchmarks/
- **Retrieved**: 2026-03-20
- **Note**: Interactive dashboard failed to load numerical data (dashboard_data.json error). Qualitative findings only.

## CI Systems Compared

Five setups were tested:

1. GitHub Actions (serial and parallel, no caching)
2. GitHub Actions parallel with magic-nix-cache
3. GitHub Actions parallel with Cachix caching
4. GitHub Actions with nixbuild.net for remote building
5. Garnix (without incremental builds)

## Projects Benchmarked

Three repositories were evaluated:
- agda/agda
- crytic/echidna
- helix-editor/helix

Each used the last ten commits available at benchmark start time.

## Key Findings

- **Magic-nix-cache underperformed:** It provided "no apparent benefit" compared to parallel GitHub Actions without caching, contradicting project claims about speed improvements.
- **Cachix showed improvement:** This caching solution demonstrated substantial speedup advantages.
- **Garnix was fastest:** The document notes that "garnix performed best across all repos."
- **Echidna problematic:** Multiple CI systems experienced failures, timeouts, and disk space issues specifically with the echidna-redistributable package.

## Data Limitations

The benchmark acknowledges potential bias, noting it was created by garnix staff, making results "likely favorable to garnix." Users are encouraged to run their own tests for validation.

## Important Note

**No numerical timing data was extractable** — the interactive dashboard's data file failed to load. The qualitative conclusions above are from the page text, but specific build time numbers could not be captured.
