# Existing Corpus Review

## What We Already Know

### Fundamental Architecture Difference
Docker/devcontainers create isolated containers with complete userspace (process-level isolation via Linux namespaces). Nix adds specific dependencies to the host environment and runs natively — package-level isolation via content-addressed store paths, but no process isolation.

### Reproducibility
This is Nix's strongest differentiator. Nix provides hermetic reproducibility at both build and runtime via flake.lock pinning. Docker is reproducible at runtime but not at build time — the same Dockerfile can produce different images on different dates because `apt-get update` pulls latest packages. This point is consistent across all three source files.

### Performance
Nix has zero overhead (native execution, no container runtime, no filesystem layers). Docker has container overhead, and on macOS specifically, volume mount I/O penalties are significant. This matters for the consulting use case where developers spend most of their time in the dev environment.

### IDE Integration
Nix wins on transparency: tools appear on PATH via direnv, IDE works normally. Docker devcontainers require VS Code Remote Containers (mature) or JetBrains Gateway (adequate but less polished). The existing research notes a caveat: JetBrains IDE integration with Nix is itself the weakest point — the primary direnv plugin is unmaintained and SDK paths require manual reconfiguration after `flake.lock` updates.

### Composition
Nix composition is trivial — listing packages in a shell definition. Docker is limited to a single base image with multi-stage build complexity for layering. This advantage compounds in consulting where projects have diverse toolchain requirements.

### Learning Curve
Docker/devcontainers: moderate (Dockerfiles are widely known). Nix: steep (2-4 weeks to basic proficiency, months for deep expertise). The objections report calls this "valid" and notes that Shopify's adoption stalled with raw Nix and only succeeded after adopting devenv.sh. The Nix champion model mitigates this — most team members use pre-configured environments without needing to understand Nix.

### Credential and Client Isolation
Docker provides stronger default isolation (each container has its own environment, filesystem, network; credentials can be volume-mounted or injected per container). Nix devShell provides weaker isolation (shell environment on host, credentials accessible via host filesystem, relies on env var isolation via direnv). However, NixOS can be layered up to containers or VMs for stronger isolation when needed.

### Production Parity
Docker wins here — same containers from dev to prod. Nix uses a different model where `pkgs.dockerTools.buildImage` builds reproducible Docker images, but the dev environment itself is not containerized. The corpus frames this as complementary rather than competing: Nix replaces the mess of brew/apt/asdf/nvm/pyenv, not Docker itself.

### The "Complementary, Not Competing" Framing
All three source files converge on this message: Nix and Docker work well together. Nix builds reproducible Docker images. Nix handles dev environments; Docker handles production deployment. This framing is present in the objections report as the recommended presentation angle.

### Devcontainers Specifically
The corpus treats devcontainers as "Docker-based, but with IDE integration" — `.devcontainer.json` per project, credential forwarding via SSH agent or volume mounts, good developer experience but carrying Docker overhead. The system-isolation report calls Docker dev containers "the pragmatic choice" if the team already uses Docker and doesn't use NixOS.

### Comparison Table (Already Compiled)
The objections report (Section 9) contains a consolidated presentation-ready comparison table covering: overhead, reproducibility, composition, IDE integration, learning curve, team adoption, production parity, isolation strength, and ecosystem size.

## Gaps to Fill

### 1. Devcontainer-Specific Deep Dive
The existing research treats devcontainers as a minor sub-point of Docker. The corpus lacks detailed analysis of the devcontainer spec itself — features like lifecycle hooks, port forwarding, multi-container configurations, Features (reusable dev container components), and the open specification beyond VS Code.

### 2. Real-World Devcontainer Adoption Patterns
No data on how consulting firms or similar organizations actually use devcontainers at scale. The corpus has Nix adoption case studies (Shopify, Tweag, etc.) but no equivalent devcontainer adoption stories.

### 3. Devcontainer Ecosystem Maturity
No assessment of the devcontainer ecosystem — marketplace of pre-built Features, community templates, tooling beyond VS Code (GitHub Codespaces, JetBrains, DevPod, other implementations of the spec).

### 4. Multi-Client Isolation with Devcontainers
The corpus notes Docker provides container-level namespace isolation for multi-client scenarios, but doesn't explore how devcontainers specifically handle credential isolation, network segmentation, or switching between client contexts.

### 5. macOS Performance Data
The corpus mentions I/O overhead on macOS for Docker but provides no quantitative benchmarks. For a consulting firm pitch, concrete numbers (e.g., build time comparisons, file watch latency) would strengthen the argument.

### 6. Cost of Ownership Comparison
No analysis of total cost: infrastructure costs (Docker Desktop licensing vs. free Nix), maintenance burden, onboarding time with real numbers, or ongoing operational overhead of each approach.

### 7. Migration Path Analysis
No assessment of what it takes to migrate from devcontainers to Nix or vice versa. How do teams that already use devcontainers evaluate whether switching to Nix is worth it?

### 8. Devcontainer Reproducibility Improvements
The corpus claims Dockerfiles are non-deterministic, but doesn't address Docker's evolving reproducibility story — pinned base images by digest, BuildKit reproducible builds, or multi-stage builds with locked dependencies. The comparison may be stronger than "Dockerfiles can differ day to day" suggests for well-maintained setups.

### 9. GitHub Codespaces and Cloud Dev Environments
The corpus briefly mentions Gitpod and Codespaces in passing but doesn't analyze how cloud-hosted devcontainers change the calculus — particularly for consulting firms where client projects may have cloud workspace requirements.

### 10. Competitive Positioning Against Devbox/Devenv
The objections report notes a gap: "No comparison to Devbox, pixi, or other post-Docker developer environment tools." These tools bridge Nix and devcontainers and could be relevant to a competitive analysis.

### 11. When Devcontainers Actually Win
The existing research frames Nix as superior for dev environments with Docker as complementary for production. A competitive analysis needs an honest assessment of scenarios where devcontainers are genuinely the better choice — not just "if the team already uses Docker."

## Source Files Referenced

- **`research-spikes/nix-consulting-environments/system-isolation-research.md`** (Section 13) — Comparison table of NixOS isolation vs. Docker dev containers across 8 dimensions; verdict that Docker is "the pragmatic choice" for teams already using Docker; also compares to Vagrant and Qubes OS.

- **`research-spikes/nix-consulting-environments/docs/docker-vs-nix-isolation-comparison.md`** — Synthesized web research on Docker vs. Nix covering reproducibility, isolation model, performance, developer experience, credential isolation, and multi-client scenarios. Includes brief notes on devcontainers, Tailscale ACLs, and VMs as alternatives.

- **`synthesized-reports/working/objections-and-limitations.md`** (Section 2, "Why not just use Docker / devcontainers?") — The most complete treatment: honest framing of where Docker wins (learning curve, team familiarity, VS Code integration, production parity, ecosystem) and where Nix wins (zero overhead, composition, native IDE, true reproducibility). Includes a consolidated presentation-ready comparison table (Section 9).

- **`research-spikes/nix-consulting-environments/flakes-fundamentals-research.md`** (Section "vs. Docker-based Dev Environments") — Comparison table of Nix Flakes vs. Docker across 8 dimensions with verdict that Nix is superior for dev environments while Docker is better for production deployment.
