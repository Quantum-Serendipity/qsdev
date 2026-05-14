# Fast CI Builds with Nix — Quentin Dufour

- **Source**: https://quentin.dufour.io/blog/2024-08-10/fast-ci-build-with-nix/
- **Retrieved**: 2026-03-20

## Key Arguments

### The 10-Minute Rule
CI build cycles "should remain below 10 minutes to be useful." Builds exceeding this duration are "used much less often, missing the opportunity for feedback."

### Fresh Environments Problem
When using VM or container-based CI systems starting with fresh environments, developers "can wait more than 10 minutes before running a command that would actually check your code."

### Historical Comparison
- **Legacy Jenkins**: Fast builds through workspace reuse and artifact caching, but fragile
- **Modern containerized CI**: Fresh environments ensure reliability but dramatically increased build times ("build times skyrocketed")

### Nix as Solution
Proposes using shared nix-daemon with read-only volume mounting as a caching strategy.

### Caching Techniques Discussed
1. Direct dependency folder caching (target/ for Rust, node_modules/ for Node.js)
2. Dedicated tools like `sccache` for compilation artifact caching
3. Custom build images with pre-compiled dependencies
4. Nix-based caching with shared nix-daemon

### Limitations
"Fetching/updating the cache involves a non negligible amount of filesystem+network I/O" when using object stores like S3.

## Performance Data
**No before/after performance comparisons are provided for the Nix solution itself.** The article is architectural/conceptual rather than empirical.
