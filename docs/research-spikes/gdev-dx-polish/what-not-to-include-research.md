# What NOT to Include Research

## Research Question

What features would bloat gdev or duplicate existing capabilities? Where is the line between "developer platform" and "too much"?

## The Bloat Trap

Feature creep in developer tools follows a predictable pattern:
1. Tool solves a core problem well (environment setup)
2. Users request adjacent features ("since you manage my env, can you also manage my tasks/deploys/secrets/monitoring?")
3. Each feature seems individually reasonable
4. The tool becomes a poorly-maintained alternative to purpose-built tools
5. Developers stop trusting it and revert to manual workflows

**The antidote**: Every proposed feature must answer "does this reduce friction that ISN'T already addressed by a tool in the stack?"

gdev's stack already includes: devenv (env management), Nix (package management), direnv/devenv hook (env switching), Renovate (dependency updates), pre-commit/prek (hooks), git-cliff (changelog), commitlint (commit messages), OSV Scanner (vulnerability scanning), Semgrep (SAST), Gitleaks (secrets), SecretSpec (secret management), starship (prompt), and Claude Code (AI assistance).

## Features to Explicitly Reject

### 1. Built-in Task Runner

**Why it's proposed**: "Developers need to run build/test/lint commands."

**Why reject**: devenv 2.0+ has a full-featured task system with parallel execution, dependency ordering, lifecycle hooks, and caching. Adding a second task runner (just, Taskfile, mise) creates tool overlap and confusion about which is "the one." See task-runner-research.md.

**What to do instead**: Generate devenv task definitions for common operations in the devenv addon.

### 2. Docker/Container Management

**Why it's proposed**: "gdev knows about Docker as an ecosystem, why not manage containers?"

**Why reject**: devenv 2.0 has a built-in process manager with restart policies, readiness probes, and dependency ordering. Docker Compose, Podman, and devenv's own container support are purpose-built. gdev generating Dockerfiles is in scope (ecosystem module); gdev running/managing containers is not.

**What to do instead**: Generate Docker-related devenv.nix configuration (container registry, Hadolint, etc.) but do not manage container lifecycle.

### 3. CI/CD Pipeline Execution

**Why it's proposed**: "gdev generates CI workflows, why not run them locally?"

**Why reject**: `act` (local GitHub Actions runner) exists and is well-maintained. nektos/act has 56K+ GitHub stars. Local CI execution is a deep, complex problem (secret injection, service containers, matrix strategies). gdev should generate `.github/workflows/`, not execute them.

**What to do instead**: Document `act` as the recommended local CI tool. Optionally include it in devenv packages.

### 4. Deployment Automation

**Why it's proposed**: "gdev handles the dev environment, why not deployment?"

**Why reject**: Deployment is orthogonal to development environment setup. It requires infrastructure knowledge (Kubernetes, cloud provider, DNS, secrets) that varies enormously between projects. Tools like Terraform/OpenTofu (which gdev supports as an ecosystem), Pulumi, ArgoCD, and Flux are purpose-built.

**What to do instead**: Nothing. Deployment is out of scope.

### 5. Project Scaffolding / Code Generation

**Why it's proposed**: "qsdev init detects project type -- why not also scaffold a new project?"

**Why reject**: `create-react-app`, `cargo init`, `go mod init`, `dotnet new`, `mix phx.new` -- every ecosystem has its own scaffolding tool with ecosystem-specific best practices. gdev adding project scaffolding means maintaining 27+ project templates that drift from upstream best practices.

**What to do instead**: qsdev init works on EXISTING projects. For new projects, use the ecosystem's scaffolding tool first, then `qsdev init`.

### 6. IDE/Editor Configuration Beyond Claude Code

**Why it's proposed**: "gdev generates Claude Code settings, why not VS Code settings, Neovim config, JetBrains settings?"

**Why reject**: IDE configuration is deeply personal and highly variable. VS Code alone has thousands of settings. gdev cannot know whether a developer uses VS Code, Neovim, Zed, Helix, or Emacs. Claude Code is special because gdev's security model requires specific Claude Code configuration (deny rules, hooks, permissions).

**What to do instead**: Generate Claude Code config only. For VS Code, generate `.vscode/extensions.json` (recommended extensions) at most -- and only if the user opts in.

### 7. Full OTEL Infrastructure (Collector + Storage + Dashboards)

**Why it's proposed**: "qsdev enables OTEL for Claude Code, why not ship the whole monitoring stack?"

**Why reject**: Running Prometheus + Loki + Grafana is infrastructure operations, not development environment configuration. It requires Docker or Kubernetes, persistent storage, and ongoing maintenance. claude-code-otel exists for teams that want the full stack.

**What to do instead**: Generate OTEL environment variables pointing at the firm's collector. The collector is infrastructure, not gdev's concern.

### 8. Package Manager Installation/Management

**Why it's proposed**: "gdev detects npm/pnpm/yarn -- why not install the right one?"

**Why reject**: devenv.nix already declares which packages are available, including package managers. `devenv shell` provides them. gdev should not independently manage tool installation -- it generates the devenv.nix that declares what's needed, and Nix provides it.

**What to do instead**: This is already working correctly. gdev generates devenv.nix with the right packages; devenv/Nix provides them.

### 9. Git Server/Hosting Integration (GitHub API, GitLab API)

**Why it's proposed**: "gdev could create repos, set branch protection, configure webhooks."

**Why reject**: API integration is a maintenance nightmare (auth tokens, API versioning, rate limits, error handling). Terraform has GitHub and GitLab providers that handle this declaratively. gdev generates files; it does not call external APIs.

**What to do instead**: Generate CI workflow files and templates. Repository configuration is Terraform/Pulumi territory.

### 10. Dependency Vulnerability Database / Advisory System

**Why it's proposed**: "gdev could maintain a curated vulnerability database for faster scanning."

**Why reject**: OSV.dev, GitHub Advisory Database, and NVD already exist and are maintained by organizations with dedicated security teams. gdev should consume these databases (via OSV Scanner), not compete with them.

**What to do instead**: Already handled -- OSV Scanner is integrated in Phase 5.

## The Decision Framework

For any proposed gdev feature, apply these three tests:

### Test 1: Is There a Purpose-Built Tool?

If a well-maintained, widely-adopted tool already solves this problem, gdev should integrate with it (generate config, include in devenv packages) rather than reimplement it. gdev's value is curation and configuration, not reimplementation.

### Test 2: Is It File Generation or Runtime Behavior?

gdev's core competency is generating configuration files. Features that require runtime behavior (process management, API calls, continuous monitoring) are outside its natural scope. The exception is diagnostic commands (doctor, status) that read state without modifying it.

### Test 3: Does It Compound with Existing Features?

Good features multiply the value of what gdev already does. Branch naming enforcement compounds with commitlint and git-cliff. PR templates compound with CI workflow generation. A task runner does NOT compound -- it replaces devenv's native capability.

## The "Won't Have" List

Maintaining an explicit "won't have" list protects product focus:

- **Won't have**: Task runner, container management, CI execution, deployment, code scaffolding, IDE configuration (except Claude Code), OTEL infrastructure, package manager installation, Git server API integration, vulnerability database
- **Will have**: File generation, diagnostic commands (doctor/status/info), thin wrappers around ecosystem-native commands (outdated), coordinated updates of gdev-managed artifacts, security configuration

## Depth Checklist

- [x] Underlying mechanism explained -- feature creep patterns, decision framework, "won't have" list
- [x] Key tradeoffs -- feature richness vs maintenance burden, user requests vs product focus
- [x] Compared to alternatives -- each rejected feature named its purpose-built alternative
- [x] Failure modes -- bloat trap progression, trust erosion, maintenance spiral
- [x] Concrete examples -- 10 specific features with rejection rationale
- [x] Standalone-readable -- yes
