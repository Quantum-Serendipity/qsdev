# Nix CI Benchmarks — NixOS Discourse Discussion

- **Source**: https://discourse.nixos.org/t/nix-ci-benchmarks/71086
- **Retrieved**: 2026-03-20

## Key Discussion Points

### Performance Observations
The community found results "super weird with nix magic cache," suggesting unexpected behavior with one caching solution.

### Cost Comparison
According to the benchmark author's broad analysis:

- **Garnix and nixbuild.net**: Comparable costs, roughly equivalent pricing
- **Cachix**: Offered 14-day free trial; potentially usable within free 5GB tier
- **GitHub Actions**: Dramatically higher — approximately **40x more expensive** than nixbuild.net or garnix for closed-source repos

The author notes actual costs are approximations due to platform inconsistencies during testing.

### Replication Costs
For those considering rerunning benchmarks, the author suggests testing within free plan limits may be feasible for public repositories with approximately 10 commits.
