# How Go Mitigates Supply Chain Attacks

- **Source**: https://go.dev/blog/supply-chain
- **Retrieved**: 2026-05-12

## Overview

Go's tooling and design help mitigate supply chain attack risks at various stages of the software development lifecycle.

## Key Mitigation Strategies

### 1. All Builds are "Locked"

- **No automatic version updates**: Unlike most package managers, Go modules don't have separate constraints and lock files. The `go.mod` file fully determines every dependency version.
- **Build failure on incomplete go.mod**: Since Go 1.16, build commands fail if `go.mod` is incomplete.
- **Explicit dependency changes**: Only `go get` and `go mod tidy` modify `go.mod`, and these aren't expected to run automatically in CI.
- **Minimal version selection**: Transitive dependencies are resolved to versions specified in the dependency's `go.mod`, not latest versions.

### 2. Version Contents Never Change (Immutability)

- **go.sum file**: Contains cryptographic hashes of each dependency.
- **Checksum Database (sumdb)**: A global append-only cryptographically-verifiable list of go.sum entries. Every module globally uses the same dependency contents.

### 3. VCS is the Source of Truth

- **No package repository accounts**: The import path embeds VCS information. `go mod download` fetches directly from the VCS.
- **Go Module Mirror as proxy only**: Authors don't upload versions; the proxy uses the same `go mod download` logic.
- **Sandboxed VCS execution**: The proxy runs VCS tools in a robust sandbox.

### 4. Building Code Doesn't Execute It

**Explicit security design goal**: "It is an explicit security design goal of the Go toolchain that neither fetching nor building code will let that code execute, even if it is untrusted and malicious."

- **No post-install hooks**: Unlike many ecosystems with first-class support for post-install scripts, Go lacks this feature.
- **Init functions are different**: While any package can define `init` functions (which do execute at runtime), modules that don't contribute code to a specific build have no security impact.

### 5. Cultural Emphasis on Minimal Dependencies

- **"A little copying is better than a little dependency"**: A Go proverb that shapes ecosystem practices.
- **Rich standard library**: HTTP stack, TLS, JSON encoding, etc. provided by stdlib.
- **Small dependency trees**: Complex applications can be built with just a handful of dependencies.

## Comparison to Other Ecosystems

- Go's `go.mod` is a unified constraints+lock file (vs separate files)
- Go fetches from VCS directly (vs uploading to centralized repositories)
- Go explicitly rejects post-install script execution
- Go uses minimal version selection (vs "latest version" approaches)
- The proxy's sandboxed VCS execution is unique to Go
