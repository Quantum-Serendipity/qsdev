# Nix-Based Continuous Integration

- **Source**: https://compilersaysno.com/posts/nix-based-continuous-integration/
- **Retrieved**: 2026-03-20

## Performance Data

The article contains minimal quantitative performance data.

### Specific Metrics
- **10 seconds saved** by eliminating Docker from the GitHub Actions CI pipeline when installing Nix directly onto the base image (vs. running in Docker)

### Qualitative Observations
- Developers with powerful local machines can often run builds faster than shared CI servers, particularly during peak hours when "jobs queued up"
- Caching benefits occur when running `preflight.sh` locally, as "download and compile caches will be hit from the previous compilation"
- Some projects have test suites "too expensive" to run locally, though noted as uncommon for mid-size projects

### Deferred Analysis
The author defers detailed performance analysis to a future post: "In the next step, we will look at caching, trading a bit of complexity to speed up our builds."

**Summary**: Focuses primarily on architectural benefits of Nix CI rather than comprehensive performance benchmarking. Only one concrete number (10 seconds saved).
