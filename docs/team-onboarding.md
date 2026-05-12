# Team Onboarding Guide

This guide walks team leads through choosing profiles, customizing security policies, configuring Claude Code skills and hooks, and rolling out gdev-secure-devenv-bootstrap to an engineering team.

## Choosing a Profile

Two profile types work together:

- **Project-type profiles** (`--profile`) bundle languages, services, Claude Code permissions, skills, and hooks for a project archetype.
- **Infrastructure profiles** (`--infra-profile`) encode organization-wide choices: registry proxy, Nix cache, build cache, vulnerability scanner, and update tool.

### Project-Type Profiles

| Profile | Best For |
|---------|----------|
| `go-web` | Go HTTP services with PostgreSQL and Redis |
| `ts-fullstack` | TypeScript/React applications with PostgreSQL and Redis |
| `python-data` | Data science, ML, and analytics projects |
| `rust-cli` | Command-line tools and systems utilities |

Start with a profile and override individual settings:

```bash
# Use the Go web profile but swap Redis for MongoDB
gdev init --profile go-web --service postgres,mongodb --yes
```

Flags explicitly set on the command line always take precedence over profile defaults.

### Infrastructure Profiles

| Profile | When to Use |
|---------|-------------|
| `consulting-default` | Multi-client consulting shops; Nexus proxy, OSV/Socket scanning, Renovate with 3-day age gate |
| `startup-github` | GitHub-centric teams; GitHub Packages, Dependabot, Turborepo caching |
| `enterprise` | Regulated environments; Artifactory, Snyk scanning, 7-day age gate, Cosign SBOM signing |

```bash
gdev init --profile go-web --infra-profile enterprise --yes
```

### Registering Custom Profiles

Use the Go API to register organization-specific profiles in your gdev plugin:

```go
package main

import (
    "fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devinit"
)

func init() {
    devinit.Configure(
        devinit.WithProfiles(map[string]devinit.Profile{
            "myorg-backend": {
                Description: "MyOrg backend service: Go 1.24, PostgreSQL, standard permissions",
                Languages: []devinit.LanguageSpec{
                    {Name: "go", Version: "1.24"},
                },
                Services:        []string{"postgres"},
                Direnv:          true,
                ClaudeCode:      true,
                PermissionLevel: "standard",
                Skills:          []string{"deploy", "security-review", "review-pr"},
                Hooks:           []string{"safety-block", "pre-commit", "auto-format"},
            },
        }),
    )
}
```

## Configuring Security Policies

### Permission Presets

Choose a Claude Code permission level based on your team's risk tolerance:

| Preset | Allow | Deny | Ask | Notes |
|--------|-------|------|-----|-------|
| `minimal` | `Read(*)`, basic build/test commands | All base deny rules + ecosystem-specific | `nix flake update` | Read-only by default; every write requires approval |
| `standard` | `Read(*)`, `Edit(*)`, `Write(*)`, `Bash(git *)`, build/test/lint, Nix dev commands | All base deny rules + ecosystem-specific | `nix flake update`, `pip install -r`, `pip install -e .` | Recommended for most teams |
| `permissive` | Everything in standard + `Bash(make *)`, `Bash(docker *)` | All base deny rules + ecosystem-specific | Same as standard | For teams with Docker/Make workflows |
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

### Customizing Deny Rules via Go API

```go
claudecode.Configure(
    claudecode.WithConfig(claudecode.Config{
        DefaultPermissions: "standard",
        ExtraDenyPatterns: []string{
            `Bash(terraform apply *)`,
            `Bash(kubectl delete *)`,
        },
        ExtraAllowPatterns: []string{
            `Bash(terraform plan *)`,
        },
    }),
)
```

## Configuring Skills

Six built-in skills are available:

| Skill | Description | Language-Specific |
|-------|-------------|-------------------|
| `deploy` | Deploy to staging/production via CI pipeline | No |
| `review-pr` | Structured pull request review with checklist | No |
| `security-review` | Security-focused code review with OWASP checks | No |
| `generate-tests` | Generate comprehensive test suites for existing code | No |
| `refactor` | Refactor code for clarity, performance, and maintainability | No |
| `db-migration` | Create safe, reversible database schema migrations | Go, Python, JavaScript |

Install skills at init time or add them later:

```bash
# At init time
gdev init --claude-skills deploy,security-review --yes

# Add to an existing project
gdev claude add-skill generate-tests
```

List all available skills and their install status:

```bash
gdev claude list-skills
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
gdev init --claude-hooks safety-block,pre-commit --yes

# Add to an existing project
gdev claude add-hook audit-log
```

## Configuring MCP Servers

Five built-in MCP servers are available:

| Server | Command | Required Environment |
|--------|---------|---------------------|
| `github` | `npx @anthropic-ai/mcp-github` | `GITHUB_TOKEN` |
| `filesystem` | `npx @anthropic-ai/mcp-filesystem` | (none) |
| `postgres` | `npx @anthropic-ai/mcp-postgres` | `DATABASE_URL` |
| `fetch` | `npx @anthropic-ai/mcp-fetch` | (none) |
| `socket` | `npx @anthropic-ai/mcp-socket` | `SOCKET_SECURITY_API_KEY` |

```bash
gdev init --mcp github,filesystem --yes
```

Custom MCP servers can be added via the Go API:

```go
claudecode.Configure(
    claudecode.WithConfig(claudecode.Config{
        MCPServers: []claudecode.MCPServerConfig{
            {
                Name:    "internal-docs",
                Command: "npx",
                Args:    []string{"@myorg/mcp-internal-docs"},
                Env:     map[string]string{"DOCS_API_KEY": "${DOCS_API_KEY}"},
            },
        },
    }),
)
```

## Rolling Out to a Team

### Step 1: Choose and Test Profiles

Pick a project-type profile and infrastructure profile. Run on a sample project:

```bash
cd sample-project
gdev init --profile go-web --infra-profile consulting-default --dry-run
```

Review the `--dry-run` output to verify the generated files match expectations.

### Step 2: Commit Generated Configuration

Run the init without `--dry-run` and commit all generated files:

```bash
gdev init --profile go-web --infra-profile consulting-default --yes
git add -A
git commit -m "chore: add gdev security-hardened devenv configuration"
```

### Step 3: Onboard Team Members

Each team member clones the repo and runs:

```bash
direnv allow   # if using direnv
devenv shell   # activates the environment
```

The generated `devenv.yaml` and `devenv.nix` are self-contained. Team members do not need to run `gdev init` again -- the committed files configure their environment automatically.

### Step 4: Ongoing Updates

When new template versions or skill library updates are available:

```bash
gdev init --update --dry-run   # preview changes
gdev init --update             # apply
```

The update workflow respects user modifications via file-specific merge strategies. See the [Configuration Reference](configuration-reference.md) for details on which files support three-way merge vs. overwrite.

### Step 5: Enforce in CI

The generated `.github/workflows/security-scan.yml` workflow runs vulnerability scanning and harden-runner in CI. The generated `devenv.nix` includes an `enterTest` script that validates security controls:

```bash
devenv test   # verifies hooks, credential stripping, secret scanning
```

Add `devenv test` to your CI pipeline to ensure security controls are not bypassed.

## Standardizing Across Repositories

For organizations with many repositories, create a shared gdev plugin that registers your custom profiles and infrastructure choices:

```go
package mygdev

import (
    "fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devinit"
    "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/profile"
)

func init() {
    // Register custom project-type profiles
    devinit.Configure(
        devinit.WithProfiles(map[string]devinit.Profile{
            "myorg-api":       myOrgAPIProfile,
            "myorg-frontend":  myOrgFrontendProfile,
            "myorg-data":      myOrgDataProfile,
        }),
    )
}
```

Each repository then runs:

```bash
gdev init --profile myorg-api --yes
```

This ensures consistent security policies, tooling versions, and Claude Code permissions across the entire organization.
