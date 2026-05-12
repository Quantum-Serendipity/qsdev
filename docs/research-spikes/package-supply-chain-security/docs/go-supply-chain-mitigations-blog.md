<!-- Source: https://go.dev/blog/supply-chain -->
<!-- Retrieved: 2026-05-12 -->

# How Go Mitigates Supply Chain Attacks - Full Content

## Overview
Modern software engineering relies on open-source dependencies, which creates supply chain attack risks. Go's tooling and design provide multiple layers of mitigation at various stages.

---

## 1. All Builds Are "Locked"

**Key principle:** No external changes can automatically affect a Go build.

### go.mod as Single Source of Truth
- Go modules use a single `go.mod` file that specifies exact dependency versions—not separate constraint lists and lock files
- Since Go 1.16, build commands (`go build`, `go test`, `go install`, `go run`) **fail if `go.mod` is incomplete**
- Only `go get` and `go mod tidy` commands modify `go.mod`
- These commands are not expected to run automatically in CI, ensuring dependency changes are deliberate and reviewable

### Minimal Version Selection
When a dependency is added with `go get`, its transitive dependencies use versions specified in the dependency's `go.mod` file—**not their latest versions**. This applies even to `go install example.com/cmd/devtoolx@latest`, where:
- The latest version of the tool is fetched
- But all dependencies are pinned by that tool's `go.mod` file

**Security benefit:** If a dependency is compromised, no one is affected until explicitly updating it, allowing time for ecosystem detection and code review.

---

## 2. Version Contents Never Change

**Key principle:** Module version contents must be immutable to prevent attackers from automatically compromising dependents by re-uploading a version.

### The go.sum File
- Contains cryptographic hashes of all dependencies contributing to the build
- Incomplete `go.sum` causes build errors
- Only `go get` and `go mod tidy` modify it, so changes accompany deliberate dependency updates
- Guarantees every build uses identical dependency contents

### The Checksum Database (sumdb)
A global append-only cryptographically-verifiable list of `go.sum` entries:

- When `go get` adds an entry to `go.sum`, it fetches from sumdb with cryptographic proof of sumdb integrity
- **Ensures global consistency:** Every module using the same version uses identical source code
- Makes it impossible for compromised dependencies or even Google infrastructure to target specific dependents with modified/backdoored code
- **Advantages:**
  - Requires no key management from module authors
  - Works seamlessly with Go's decentralized module system
  - Guarantees all users of (e.g.) `example.com/modulex@v1.9.2` use identical reviewed code

---

## 3. The VCS Is the Source of Truth

**Key difference from other ecosystems:** Go has no package repository upload step.

### Traditional Two-Account Risk
In most ecosystems:
- Code is developed in version control (VCS)
- Then uploaded to a package repository
- Creates two compromise points: VCS host and package repository
- Package repository accounts are used less frequently and more likely overlooked
- Malicious code can be hidden during upload (especially if source is modified during packaging)

### Go's Approach
- **No package repository accounts** exist for Go modules
- Import paths embed VCS information that `go mod download` uses to fetch directly from VCS
- Version tags define releases in the VCS

### The Go Module Mirror (Proxy)
A caching proxy—not a registry:
- Module authors don't register accounts or upload versions
- The proxy runs `go mod download` to fetch and cache versions using the same logic as the `go` tool
- The Checksum Database guarantees **only one source tree can exist** for a given module version
- Users see identical results whether fetching directly from VCS or through the proxy

**Availability benefit:** If a version disappears from VCS, the proxy can still serve cached copies, preventing "left-pad" style issues.

### VCS Sandbox Protection
- Running VCS tools on clients exposes significant attack surface
- The Go Module Mirror runs VCS tools in a robust sandbox
- **Default VCS support:** Only git and Mercurial enabled by default
- Users can still fetch code from off-by-default VCS systems via the proxy, but attackers can't reach that code in most installations
- Controlled by `GOVCS` setting

---

## 4. Building Code Doesn't Execute It

**Explicit security design goal:** Fetching or building code will not execute it, even if untrusted and malicious.

### Contrast with Other Ecosystems
- Many ecosystems support "post-install" hooks that run code during package fetch
- These have been exploited to compromise developer machines and "worm" through module authors
- Go explicitly rejects this pattern

### Go's Execution Model
- No post-install hooks
- Code only executes during testing or binary execution
- **Important caveat:** No security boundary within a build—any package contributing to the build can define `init` functions

### Dependency Isolation
- Modules that don't contribute code to a specific build have **no security impact on it**
- Example: Building `example.com/cmd/devtoolx` on macOS means Windows-only dependencies or dependencies of `example.com/cmd/othertool` cannot compromise your machine
- Mitigation: Only code paths used in your build can affect you

---

## 5. Cultural Mitigation: Minimal Dependency Trees

**The most important mitigation—a cultural principle:**

### The Go Proverb
> "A little copying is better than a little dependency"

### Ecosystem Characteristics
- Strong culture rejecting large dependency trees
- High-quality modules proudly wear the "zero dependencies" label
- Rich standard library provides common high-level building blocks:
  - HTTP stack
  - TLS library
  - JSON encoding
  - Additional modules in `golang.org/x/...`

### Risk Reduction
- Complex applications can be built with just a handful of dependencies
- No matter how good tooling is, it can't eliminate code reuse risk
- **Strongest mitigation:** Small dependency trees

---

## Summary of Protections

| Mechanism | Protection |
|-----------|-----------|
| **go.mod pinning** | Deterministic builds, no automatic updates |
| **go.sum + sumdb** | Immutable version contents, global consistency |
| **VCS as source of truth** | No upload compromise point, proxy is just cache |
| **Sandbox for VCS tools** | Limited VCS system exposure in default configurations |
| **No post-install hooks** | Code doesn't execute during fetch/build |
| **Minimal dependency culture** | Reduced attack surface through smaller trees |
