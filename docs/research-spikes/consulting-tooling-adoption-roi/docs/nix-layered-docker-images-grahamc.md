# Optimising Docker Layers for Better Caching with Nix — Graham Christensen

- **Source**: https://grahamc.com/blog/nix-and-layered-docker-images/
- **Retrieved**: 2026-03-20

## Performance Claims

The only quantitative performance claim: "The automatic splitting and prioritization has improved image push and fetch times by an order of magnitude." (10x improvement in push/fetch times — no baseline data provided.)

## Concrete Example

PHP and MySQL images shared **20 layers** automatically through Nix's optimization approach. No comparison to non-Nix layer sharing provided.

## What Is NOT Quantified

- Build time measurements (before/after)
- Image size comparisons
- Layer count differences in practical examples
- Memory usage impacts
- Network bandwidth savings with specific numbers

## Key Concept

Nix can automatically create multi-layered Docker images where layer sharing between unrelated images (e.g., PHP and MySQL) is automatic. In conventional Docker workflows, you would not bother trying to share layers between unrelated images. With Nix, this sharing happens without manual optimization.

## Relevance

This describes a mechanism (automatic Docker layer optimization) that could contribute to CI speedups for container-based deployments, but provides no measured CI build time data.
