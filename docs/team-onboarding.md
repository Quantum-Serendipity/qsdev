# Team Onboarding Guide

This guide walks team leads through choosing profiles, customizing security policies, configuring Claude Code skills and hooks, and rolling out qsdev to an engineering team.

## Choosing a Profile

Two profile types work together:

- **Project-type profiles** (`--profile`) bundle languages, services, Claude Code permissions, skills, and hooks for a project archetype.
- **Infrastructure profiles** (`--infra-profile`) encode organization-wide choices: registry proxy, Nix cache, build cache, vulnerability scanner, and update tool.

### Project-Type Profiles

| Profile | Best For |
|---------|----------|
| `go-web` | Go HTTP services with PostgreSQL and Redis |
| `ts-fullstack` | TypeScript/React applications with PostgreSQL and Redis |
| `ts-backend` | TypeScript API services with PostgreSQL and Redis |
| `python-data` | Data science, ML, and analytics projects |
| `python-web` | Python web applications with PostgreSQL and Redis |
| `rust-cli` | Command-line tools and systems utilities |
| `rust-web` | Rust web services with PostgreSQL and Redis |
| `java-web` | Java/Gradle web applications with PostgreSQL and Redis |
| `elixir-web` | Elixir/Phoenix applications with PostgreSQL and Redis |
| `dotnet-web` | .NET web applications with PostgreSQL and Redis |

Start with a profile and override individual settings:

```bash
# Use the Go web profile but swap Redis for MongoDB
qsdev init --profile go-web --service postgres,mongodb --yes
```

Flags explicitly set on the command line always take precedence over profile defaults.

### Infrastructure Profiles

| Profile | When to Use |
|---------|-------------|
| `consulting-default` | Multi-client consulting shops; Nexus proxy, OSV + Socket scanning, Renovate with 3-day age gate |
| `startup-github` | GitHub-native; GitHub Packages, OSV + Socket scanning, Dependabot |
| `enterprise` | Regulated environments; Artifactory, Snyk + Socket scanning, Renovate with 7-day age gate, Cosign signing |

```bash
qsdev init --profile go-web --infra-profile enterprise --yes
```

### Compliance Levels

Each security tier maps to a compliance level that controls age-gating thresholds, required hooks, and SBOM policy:

| Level | Age Gate | Required Hooks | SBOM Policy |
|-------|---------|----------------|-------------|
| `baseline` | 72 hours | ripsecrets, gitleaks | Off |
| `enhanced` | 168 hours (1 week) | ripsecrets, gitleaks, semgrep | On release |
| `strict` | 336 hours (2 weeks) | ripsecrets, gitleaks, semgrep, license-compliance | Every build |

### Container Runtime

qsdev auto-detects whether your project uses Docker or Podman and generates runtime-specific configs:

```bash
qsdev container detect     # Show detected runtime and capabilities
qsdev container migrate    # Analyze compose files for Docker-to-Podman compatibility
```

When Podman is detected, qsdev generates rootless-aware configurations, blocks Docker socket mounts, and adds Podman-specific deny rules. On NixOS, it generates a Podman rootless setup guide at `docs/nixos-podman-rootless.md`.

## Configuring Security Policies

### Permission Presets

Package installs are hook-gated (the user is asked for confirmation), not blocked outright. Choose a permission preset based on your team's risk tolerance:

| Preset | Allow | Deny | Ask | Notes |
|--------|-------|------|-----|-------|
| `minimal` | `Read(*)`, basic build/test commands | All base deny rules + ecosystem-specific | `nix flake update` | Read-only by default; every write requires approval |
| `standard` | `Read(*)`, `Edit(*)`, `Write(*)`, `Bash(git *)`, build/test/lint, Nix dev commands | All base deny rules + ecosystem-specific | `nix flake update`, `pip install -r`, `pip install -e .` | Recommended for most teams |
| `permissive` | Everything in standard + `Bash(make *)`, `Bash(docker *)` | All base deny rules + ecosystem-specific | Same as standard | For teams with Docker/Make workflows |
| `supply-chain-only` | Minimal | All base + ecosystem deny rules | (none) | Supply chain defense only; no dev tooling permissions |
| `custom` | Only `ExtraAllowPatterns` from config | All base + ecosystem + `ExtraDenyPatterns` | (none) | Full manual control |

The `standard` and `permissive` presets also set `defaultMode: "default"` and `disableBypassPermissionsMode: "disable"` to prevent developers from bypassing the permission model.

### Deny Rule Categories

All presets share the same deny rules, organized into 15 categories:

1. **JS Package Managers** -- npm, npx, yarn, pnpm, bun install commands
2. **Python** -- pip, pip3, pipx, uv install commands
3. **Rust** -- cargo add, cargo install
4. **Go** -- go get, go install
5. **Ruby** -- gem install, bundle install/add/update
6. **PHP** -- composer require/install/update
7. **Nix** -- nix-env imperative installs, nix profile, cachix use
8. **System** -- apt, brew, pacman, dnf, yum, apk, snap
9. **Pipe-to-Shell** -- curl/wget piped to bash/sh
10. **Shell Wrapping** -- bash/sh/zsh -c wrapping of install commands
11. **env/command Prefix** -- env/command prefix bypasses
12. **sudo Prefix** -- sudo-prefixed install commands
13. **Subprocess Escape** -- python/node/ruby/perl subprocess calls
14. **eval/xargs** -- eval and xargs indirect execution
15. **Destructive Ops** -- git push --force, git reset --hard, rm -rf, .env/.secrets reading

Each ecosystem module also contributes its own deny rules (e.g., JavaScript modules block `npx create-*` patterns).

### Customizing Deny Rules

Add extra deny or allow patterns in `.qsdev.yaml`:

```yaml
claude:
  permissions: standard
  extra_deny:
    - "Bash(terraform apply *)"
    - "Bash(kubectl delete *)"
  extra_allow:
    - "Bash(terraform plan *)"
```

## Configuring Skills

### Built-in Skills

Six built-in skills are available for Claude Code workflows:

| Skill | Description | Language-Specific |
|-------|-------------|-------------------|
| `deploy` | Deploy to staging/production via CI pipeline | No |
| `review-pr` | Structured pull request review with checklist | No |
| `security-review` | Security-focused code review with OWASP checks | No |
| `generate-tests` | Generate test suites for existing code | No |
| `refactor` | Refactor code for clarity, performance, and maintainability | No |
| `db-migration` | Create safe, reversible database schema migrations | Go, Python, JavaScript |

### qsdev Operations Skills

In addition, 11 qsdev operations skills are auto-generated during `qsdev init`, providing Claude Code with structured commands for managing the environment (adding dependencies, running checks, updating configs, etc.).

### Managing Skills

Install skills at init time or add them later:

```bash
# At init time
qsdev init --claude-skills deploy,security-review --yes

# Add to an existing project
qsdev claude add-skill generate-tests

# List all available skills and their install status
qsdev claude list-skills
```

## Configuring Hooks

Hook presets control Claude Code runtime behavior:

| Hook | Effect |
|------|--------|
| `safety-block` | Installs `package-guard.py` as a PreToolUse hook; intercepts package install commands in real-time |
| `credential-scan` | Scans Write/Edit operations for credentials before they reach disk |
| `destructive-prevention` | Blocks destructive Bash commands (rm -rf, git push --force, etc.) |
| `file-boundary` | Prevents Write/Edit/Read operations outside the project tree |
| `tool-gates` | Enforces per-tool approval policies on all tool invocations |
| `soc2-audit` | Logs session start/end, tool invocations, and checkpoints for SOC 2 compliance (4-event audit trail with monthly rotation) |
| `auto-format` | Runs formatters after file writes |
| `pre-commit` | Runs pre-commit checks before git operations |
| `audit-log` | Logs all tool invocations for compliance auditing (simpler alternative to soc2-audit) |

All hooks run inside the sandbox when available (see `qsdev sandbox status`).

```bash
# At init time
qsdev init --claude-hooks safety-block,pre-commit --yes

# Add to an existing project
qsdev claude add-hook audit-log
```

## Configuring MCP Servers

MCP servers are configured by default or activated based on project detection:

| Server | Purpose | Activation |
|--------|---------|------------|
| `context7` | Library documentation lookup | Default |
| `github` | GitHub API integration | Default |
| `socket` | Package security analysis | Default |
| `semble` | Semantic code search | Default |
| `agent-postmortem` | Session analysis and failure patterns | Default |
| `version-sentinel` | Dependency version monitoring | Default |
| `local-docs-devdocs` | Offline DevDocs API references | On when detected |
| `local-docs-zim` | Offline Stack Exchange via ZIM | Opt-in |
| `man-pages` | Local man page documentation | Opt-in |
| `mcp-nixos` | NixOS packages and options | Opt-in |

These are included automatically during `qsdev init`. No additional flags are needed.

Use `qsdev mcp grade` to check compliance levels and `qsdev mcp health` to verify connectivity:

```bash
qsdev mcp grade                # Show compliance grades for all servers
qsdev mcp grade context7       # Grade a specific server
qsdev mcp install <name>       # Install a server from the registry
qsdev mcp health               # Health check all configured servers
```

## Managing Security Policies

qsdev generates YAML security policies in `.qsdev/policy/`. These define fine-grained rules for what the AI agent can and cannot do, beyond the static deny/ask rules in `.claude/settings.json`.

### Inspecting Policies

```bash
qsdev policy list                      # List all rules
qsdev policy show <rule-id>            # Show rule details
qsdev policy check                     # Evaluate posture
qsdev policy check --sarif             # SARIF output for CI dashboards
qsdev policy check --audit-level high  # Fail only on high+ severity
```

### Session Bypass

Some rules support session-level bypass for temporary exceptions:

```bash
qsdev session allow RULE-001 RULE-002   # Bypass specific rules
qsdev session list                       # Show active bypasses
qsdev session clear                      # Remove all bypasses
```

Rules with `bypass_tier: enforce_always` (all 18 self-protection rules) cannot be bypassed. Rules with `bypass_tier: session` require per-session approval. Rules with `bypass_tier: command` can be bypassed per-invocation.

## Cloud Ecosystem Coverage

When AWS, GCP, or Azure project files are detected (CDK, SAM, Terraform providers, CLI config files, etc.), qsdev generates cloud-specific security configuration with 3 layers of credential isolation:

1. **Environment separation** — Cloud credential variables are unset in the devenv shell, preventing ambient credential access across projects.
2. **Credential file masking** — Read-deny rules block agent access to `~/.aws/credentials`, `~/.config/gcloud/`, and `~/.azure/`.
3. **Agent deny rules** — Authentication and credential modification commands (`aws configure`, `gcloud auth login`, `az login`) are denied.

Cloud CLIs remain available for read-only operations like listing resources or describing infrastructure.

`qsdev devenv doctor` includes a CloudProviders section verifying CLI availability and isolation status for each detected provider.

## Available Services

Project profiles include services by default (usually PostgreSQL and Redis). Add services individually:

```bash
qsdev devenv add-service kafka
qsdev devenv add-service minio
```

All 12 services bind to localhost with configurable ports:

| Service | Notes |
|---------|-------|
| PostgreSQL | |
| Redis | |
| MySQL | |
| MongoDB | |
| Elasticsearch | |
| RabbitMQ | |
| Kafka | KRaft mode by default, ZooKeeper fallback available |
| MinIO | S3-compatible API, exports AWS_ENDPOINT_URL |
| Mailpit | SMTP capture with web UI |
| Keycloak | Identity provider with admin console, exports OIDC env vars |
| NATS | Pub/sub and request/reply with optional JetStream persistence |

## Local Documentation Pipeline

For teams working in restricted network environments or wanting faster documentation lookups, qsdev supports a local documentation corpus:

```bash
qsdev docs download      # Download DevDocs + ZIM archives
qsdev docs status        # Show installed documentation sets
qsdev docs enable go     # Enable a documentation set
```

Downloaded documentation is served through MCP servers (local-docs-devdocs, local-docs-zim) and routed by the lookup-docs skill, which queries 5 sources in priority order: local DevDocs, Stack Exchange ZIM, man pages, mcp-nixos, Context7 (web fallback).

## Rolling Out to a Team

### Step 1: Choose and Test Profiles

Pick a project-type profile and infrastructure profile. Run on a sample project:

```bash
cd sample-project
qsdev init --profile go-web --infra-profile consulting-default --dry-run
```

Review the `--dry-run` output to verify the generated files match expectations.

### Step 2: Commit Generated Configuration

Run the init without `--dry-run` and commit all generated files:

```bash
qsdev init --profile go-web --infra-profile consulting-default --yes
git add -A
git commit -m "chore: add qsdev security-hardened devenv configuration"
```

### Step 3: Onboard Team Members

Each team member clones the repo and runs:

```bash
qsdev init --mode join
```

This detects the committed `.qsdev.yaml` and reproduces an identical environment locally, including Claude Code permissions, skills, hooks, and MCP servers. No manual configuration is needed -- every developer gets the same security posture.

### Step 4: Ongoing Updates

The update command runs three stages: binary self-update, config regeneration, and devenv input update. Run all stages or target specific ones:

```bash
qsdev update --check         # Check for available updates
qsdev update --dry-run       # Preview all changes
qsdev update                 # Run all three stages
qsdev update --self-only     # Update only the binary
qsdev update --configs-only  # Regenerate configs only
qsdev update --deps-only     # Update devenv inputs only
```

The update workflow respects user modifications via three-way merge. Files you have customized are merged intelligently rather than overwritten.

### Step 5: Enforce in CI

Add `qsdev check` to your CI pipeline to enforce configuration integrity and security hardening:

```bash
# In your CI workflow
qsdev check
```

`qsdev check` validates that security controls are present, deny rules are intact, and no configuration has drifted. It exits non-zero on violations and supports JSON, SARIF, and JUnit output formats for integration with CI dashboards.

### Step 6: Monitor Security Posture

Use `qsdev status` to see each project's security score and grade:

```bash
qsdev status
```

For multi-project visibility, `qsdev team-report` aggregates posture across repositories.

## Standardizing Across Repositories

For organizations with many repositories, define your standard configuration in a shared `.qsdev.yaml` template:

```yaml
profile: go-web
infra_profile: consulting-default
claude:
  permissions: standard
  skills:
    - deploy
    - security-review
    - review-pr
  hooks:
    - safety-block
    - pre-commit
    - auto-format
```

Each repository then runs:

```bash
qsdev init --profile myorg-api --yes
```

This ensures consistent security policies, tooling versions, and Claude Code permissions across the entire organization.

## Command Reference

| Command | Purpose |
|---------|---------|
| `qsdev init --profile X --yes` | Generate environment from a profile |
| `qsdev init --mode join` | Join an existing team environment |
| `qsdev status` | Security posture assessment (score + grade) |
| `qsdev check` | CI enforcement (config integrity, hardening) |
| `qsdev update` | Update binary + configs + devenv inputs (3-stage coordinated update) |
| `qsdev enable <tool>` | Enable a security/AI tool |
| `qsdev disable <tool>` | Disable a tool |
| `qsdev list` | Show all available tools |
| `qsdev devenv doctor` | Diagnose environment issues |
| `qsdev devenv setup` | Install prerequisites (Nix, devenv, direnv) |
| `qsdev claude add-skill <name>` | Add a Claude Code skill |
| `qsdev claude add-hook <name>` | Enable a hook preset |
| `qsdev claude list-skills` | List available skills |
| `qsdev mcp status` | MCP server health and connectivity |
| `qsdev mcp grade` | MCP server compliance grading |
| `qsdev mcp install <name>` | Install an MCP server |
| `qsdev docs download` | Download local documentation sets |
| `qsdev docs status` | Show installed documentation |
| `qsdev policy check` | Evaluate security policy posture |
| `qsdev policy list` | List security policy rules |
| `qsdev session allow <ids>` | Enable session bypass for rules |
| `qsdev session clear` | Remove session bypass overrides |
