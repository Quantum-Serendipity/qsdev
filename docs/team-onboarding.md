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
| `consulting-default` | Multi-client consulting shops; enhanced security (semgrep, gitleaks, secretspec) |
| `startup-github` | GitHub-native; baseline security, minimal overhead |
| `enterprise` | Regulated environments; strict security, audit logging, SBOM |

```bash
qsdev init --profile go-web --infra-profile enterprise --yes
```

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
| `generate-tests` | Generate comprehensive test suites for existing code | No |
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

Four hook presets control Claude Code runtime behavior:

| Hook | Effect |
|------|--------|
| `safety-block` | Installs `package-guard.py` as a PreToolUse hook; intercepts package install commands in real-time |
| `auto-format` | Runs formatters after file writes |
| `pre-commit` | Runs pre-commit checks before git operations |
| `audit-log` | Logs all tool invocations for compliance auditing |

```bash
# At init time
qsdev init --claude-hooks safety-block,pre-commit --yes

# Add to an existing project
qsdev claude add-hook audit-log
```

## Configuring MCP Servers

Four AlwaysOn MCP servers are configured by default:

| Server | Purpose |
|--------|---------|
| `context7` | Library documentation lookup |
| `github` | GitHub API integration |
| `socket` | Package security analysis |
| `semble` | Semantic code search |

These are included automatically during `qsdev init`. No additional flags are needed.

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

When new template versions or skill library updates are available:

```bash
qsdev update --dry-run   # preview changes
qsdev update             # apply
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
| `qsdev update` | Update configs + devenv inputs |
| `qsdev enable <tool>` | Enable a security/AI tool |
| `qsdev disable <tool>` | Disable a tool |
| `qsdev list` | Show all available tools |
| `qsdev devenv doctor` | Diagnose environment issues |
| `qsdev devenv setup` | Install prerequisites (Nix, devenv, direnv) |
| `qsdev claude add-skill <name>` | Add a Claude Code skill |
| `qsdev claude add-hook <name>` | Enable a hook preset |
| `qsdev claude list-skills` | List available skills |
