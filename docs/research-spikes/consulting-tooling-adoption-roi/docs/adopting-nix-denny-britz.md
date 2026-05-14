# Adopting Nix — Denny Britz

- **Source**: https://dennybritz.com/posts/adopting-nix/
- **Retrieved**: 2026-03-20

## CI/CD Performance Data

### Docker Image Building
Most concrete performance claim: "When the Docker build cache is busted, it may take 10-15 minutes" with traditional Dockerfiles. In contrast, "Generating a Docker image after you've already run `nix build` (e.g. for testing) only takes a few seconds."

This represents a significant speedup for iterative builds, though the comparison isn't entirely apples-to-apples since the Nix approach assumes the binary is already built.

### Caching Benefits

1. **Internal Nix caching:** "Images are fast to build because nix re-uses its cached work from `/nix/store`"
2. **Cachix integration:** "It can be built once in Github CI and uploaded to the cache. Any project that depends on it will fetch it from there," eliminating redundant rebuilds across projects.

### Tradeoffs and Downsides
- No specific numbers showing Nix made builds slower
- Integration tools like `crane` and `pip2nix` are "still early," potentially introducing build issues requiring workarounds
- Platform-specific code adds complexity
