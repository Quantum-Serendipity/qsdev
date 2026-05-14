# Why I Avoid Using Nix to Build Docker Images — McCurdy

- **Source**: https://www.mccurdyc.dev/posts/2024/09/why-i-avoid-using-nix-to-build-docker-images/
- **Retrieved**: 2026-03-20

## Key Reasons Against Nix for Docker Images

### High Learning Curve vs. Low Benefit
Dockerfiles are "an amazing interface" with "extremely low" adoption cost. Nix has an "extremely high learning curve," making cost-benefit unfavorable unless there's a compelling need.

### Organizational Adoption Requirements
Even team-level Nix adoption isn't sufficient — "the entire organization" needs to be committed since "teams change."

### Parallelization Issues
Running `nix develop --command` in parallel fails due to Nix store locking, producing SQLite database busy errors. This makes CI pipelines slower and prevents concurrent task execution.

### Single Source of Truth Problem
Maintaining both Dockerfile and Nix configurations duplicates dependency management.

## Performance Data

One concrete metric: **"Running `nix develop` in CI takes too long (>2min)"** — this motivated baking dependencies into a final image instead.

## Practical Approach
Adds optional Nix flakes to projects for those who want them, while providing Dockerfiles as the primary development interface.

## Relevance
This is a counter-argument source — demonstrates real-world friction where Nix adds CI overhead rather than reducing it, especially for teams not fully committed to Nix.
