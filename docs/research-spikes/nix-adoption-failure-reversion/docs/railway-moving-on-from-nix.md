<!-- Source: https://blog.railway.com/p/introducing-railpack -->
<!-- Retrieved: 2026-03-20 -->

# Why We're Moving on From Nix (Railway)

**Author:** Jake Runzer
**Date:** March 4, 2025
**Organization:** Railway

## Scale and Context
Railway built over 14 million applications using Nixpacks in approximately three years. While the tool worked adequately for roughly 80% of users, this left approximately 200,000 Railway users encountering limitations regularly. The company determined a major builder overhaul was necessary to scale from one million to 100 million users.

## Core Problems with Nix Implementation

### Version Management Issues
The fundamental challenge stemmed from Nix's commit-based package versioning architecture. Rather than supporting granular version selection, only the latest major version of each package remained accessible, with versions tethered to specific commits in the nixpkgs repository. Maintaining patch versions required hardcoding commit hashes -- an unmaintainable approach for contributors unfamiliar with Nix internals. For languages like Node and Python, Railway could only support the most recent major version.

### Stability Concerns
When updating commit hashes to access newer package versions, all other package versions simultaneously updated, creating unexpected build failures. As the team noted, "we feel bad when users can't access the latest packages, but feel worse when previously functional builds suddenly fail."

### Image Size and Caching Limitations
Using Nix-based dependency installation generated massive Docker images (often exceeding 500MB for Python applications) due to single-layer /nix/store structures containing both build and runtime dependencies. The approach provided minimal control over layer caching, and Railway's deployment ID environment variables caused post-injection layers to never cache.

## The Railpack Solution
Launched in beta in March 2025, Railpack was rewritten in Go and abandons Nix entirely. Key improvements include:

- **Granular versioning:** Full major.minor.patch version support
- **Image reduction:** 38% smaller Node images and 77% smaller Python images versus Nixpacks
- **Improved caching:** Direct BuildKit integration enabling sharable layer caches across environments
- **Version locking:** Dependencies captured at successful build completion prevent future breakage from default version changes

## Technical Architecture
Railpack employs three-stage processing:
1. **Analyze:** Examine code for dependencies and build requirements
2. **Plan:** Generate JSON-serializable build plans with explicit step dependencies
3. **Generate:** Construct BuildKit LLB graphs enabling parallel execution and precise layer control

The tool currently supports Node, Python, Go, PHP, and static HTML deployments, with built-in optimization for frameworks like Vite, Astro, Create React App, and Angular.
