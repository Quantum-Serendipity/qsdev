# Nix 2.28 Experimental Features: Security-Related Overview
- **Source**: https://nix.dev/manual/nix/2.28/development/experimental-features
- **Retrieved**: 2026-05-12

## Key Security-Focused Features

### ca-derivations
Enables content-addressed derivations to "prevent rebuilds when changes to the derivation do not result in changes to the derivation's output." This feature prevents unnecessary recompilation and potential supply chain vulnerabilities through deterministic builds. Status: Experimental (tracking milestone 35).

### verified-fetches
Implements "verification of git commit signatures through the `fetchGit` built-in." This guards against compromised source code by cryptographically validating repository commits before fetching. Status: Experimental (tracking milestone 48).

### fetch-closure
Activates the `fetchClosure` built-in function, enabling controlled retrieval of pre-built store objects. Status: Experimental (tracking milestone 40).

### configurable-impure-env
Permits use of the `impure-env` setting, allowing administrators to control environment variable exposure during builds. Status: Experimental (tracking milestone 37).

### flakes
Enables reproducible Nix package management through locked dependencies. Status: Experimental (tracking milestone 27).

## Stabilization Criteria

Features advance to stable status when maintainers confirm the design is sensible, interactions are understood, and maintenance burden remains manageable — requiring substantial user feedback and demonstrated real-world adoption beforehand.
