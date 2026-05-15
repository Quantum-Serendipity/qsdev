# Consulting Engineer Daily-Driver Tooling Audit

## Executive Summary

Software consulting engineers at firms like Thoughtworks, Slalom, Accenture, Cognizant, and EPAM require a broad toolset beyond language runtimes and security scanners. This audit catalogs the tools, services, and CLIs that engineers need installed and configured on day one of a new engagement, organized by category. For each tool, we assess commonality (essential/common/niche), Nix availability, configuration requirements beyond installation, and devenv.sh native support.

The biggest gaps in gdev's current coverage are **cloud provider CLIs with credential management** (used on virtually every engagement), **Kubernetes/container orchestration tools** (standard for cloud-native work), **developer productivity CLI tools** (the "modern coreutils" that senior engineers expect), and **git platform CLIs** (gh/glab for PR workflows). These represent the highest-value expansion targets.

---

## 1. Cloud Provider CLIs & Credential Management

### Core CLIs

| Tool | Commonality | Nix-Packaged | Config Required | devenv.sh Native |
|------|-------------|--------------|-----------------|------------------|
| `aws-cli` v2 | Essential | Yes (`awscli2`) | Profiles, SSO, region | No |
| `gcloud` (Google Cloud SDK) | Essential | Yes (`google-cloud-sdk`) | Project, auth, region | No |
| `az` (Azure CLI) | Essential | Yes (`azure-cli`) | Subscription, tenant | No |
| `aws-vault` | Common | Yes (`aws-vault`) | Keyring backend, profiles | No |
| `aws-sso-cli` | Common | Yes (in nixpkgs) | SSO start URLs, org profiles | No |
| `saml2aws` | Common | Yes (`saml2aws`) | IdP URL, MFA config | No |
| `aws-sso-util` | Common | Likely (Python tool) | SSO config | No |
| `aws-nuke` | Niche | Yes | Account filters, safety config | No |

### Credential Management Patterns

Consulting firms face a unique challenge: engineers rotate between client engagements, each with different cloud accounts, SSO providers, and access patterns. The standard patterns are:

1. **AWS SSO/Identity Center** (most common in 2025-2026): `aws sso login --profile CLIENT_NAME` or via `aws-sso-cli` for multi-org management. Engineers maintain multiple SSO start URLs in `~/.aws/config`, one per client organization.

2. **SAML-based SSO**: Firms using Okta, Azure AD, or OneLogin as IdP use `saml2aws` to authenticate and get temporary credentials. This is the legacy pattern but still very common at enterprise clients.

3. **aws-vault**: Wraps credential management with system keyring storage. Supports both IAM users (legacy) and SSO profiles. The consultant-specific pattern is `aws-vault exec CLIENT_A -- terraform plan` to avoid credential leakage between client contexts.

4. **GCP Application Default Credentials**: `gcloud auth application-default login` for local development, with project switching via `gcloud config configurations activate CLIENT_NAME`.

5. **Azure CLI**: `az login` with tenant selection, often combined with `az account set --subscription CLIENT_SUB_ID`.

### gdev Integration Opportunity

**High value.** A `qsdev cloud` addon or profiles within the devenv addon could:
- Install all three major cloud CLIs plus credential helpers
- Template `~/.aws/config` with SSO profiles per engagement
- Configure `gcloud` named configurations per client
- Set up `az` subscription aliases
- Integrate `aws-vault` or `aws-sso-cli` as the default credential wrapper
- Set engagement-specific environment variables (`AWS_PROFILE`, `GOOGLE_CLOUD_PROJECT`, `AZURE_SUBSCRIPTION_ID`) via direnv/devenv enterShell

---

## 2. Developer Productivity CLI Tools

### Modern Coreutils Replacements

| Tool | Replaces | Commonality | Nix-Packaged | Config Required | devenv.sh Native |
|------|----------|-------------|--------------|-----------------|------------------|
| `ripgrep` (rg) | grep | Essential | Yes | Minimal (.ripgreprc) | No |
| `fd` | find | Essential | Yes | Minimal | No |
| `bat` | cat | Common | Yes | Theme config | No |
| `eza` | ls | Common | Yes | Alias setup | No |
| `fzf` | - | Essential | Yes | Shell integration | No |
| `jq` | - | Essential | Yes | None | No |
| `yq` | - | Common | Yes | None | No |
| `delta` | diff | Common | Yes | Git config | No |
| `sd` | sed | Niche | Yes | None | No |
| `zoxide` | cd | Common | Yes | Shell integration | No |
| `tldr` | man | Common | Yes | None | No |
| `ncdu` | du | Common | Yes | None | No |
| `htop`/`btop` | top | Common | Yes | None | No |

### Data Processing & Transformation

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `jq` | JSON processing | Essential | Yes |
| `yq` | YAML processing | Common | Yes |
| `xsv` | CSV processing | Niche | Yes |
| `miller` (mlr) | Multi-format data | Niche | Yes |
| `dasel` | Multi-format selector | Niche | Yes |

### HTTP & Network Tools

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `curl` | HTTP client | Essential | Yes |
| `httpie` (http/https) | Human-friendly HTTP | Common | Yes |
| `wget` | File download | Common | Yes |
| `dig` | DNS queries | Common | Yes (bind.dnsutils) |
| `mtr` | Network diagnostics | Niche | Yes |
| `nmap` | Port scanning | Niche | Yes |

### Terminal Multiplexing & Session Management

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `tmux` | Terminal multiplexer | Common | Yes |
| `zellij` | Modern multiplexer | Niche | Yes |
| `screen` | Legacy multiplexer | Legacy | Yes |

### Environment & Configuration

| Tool | Purpose | Commonality | Nix-Packaged | devenv.sh Native |
|------|---------|-------------|--------------|------------------|
| `direnv` | Per-directory env vars | Essential | Yes | Yes (built-in) |
| `mise` (formerly rtx) | Polyglot version manager | Common | Yes | Overlaps with devenv |
| `envsubst` | Template variable substitution | Common | Yes (gettext) | No |
| `age` | Modern file encryption | Common | Yes | No |
| `sops` | Secrets-in-files encryption | Common | Yes | No |

### Notes on mise vs devenv.sh

mise (ThoughtWorks Radar Adopt) and devenv.sh solve overlapping problems: both manage tool versions and environment variables. devenv.sh is more powerful (Nix-based, services, reproducibility) but mise has broader adoption outside the Nix ecosystem. For gdev, devenv.sh is the foundation, making mise redundant for version management. However, clients who already use `.mise.toml` or `.tool-versions` may want compatibility.

### gdev Integration Opportunity

**High value.** These are the "muscle memory" tools that senior engineers expect to exist. A `qsdev devtools` profile or extension to the devenv addon could:
- Install a curated "modern coreutils" bundle (ripgrep, fd, bat, eza, fzf, jq, yq, delta, zoxide, tldr)
- Configure shell integrations (fzf keybindings, zoxide init, bat as MANPAGER)
- Set git config for delta as pager
- Make this opt-in per engagement but on by default for new setups

---

## 3. Database & Data Tools

### Interactive CLI Clients

| Tool | Database | Commonality | Nix-Packaged | devenv.sh Native |
|------|----------|-------------|--------------|------------------|
| `pgcli` | PostgreSQL | Common | Yes | No (service only) |
| `mycli` | MySQL | Common | Yes | No (service only) |
| `litecli` | SQLite | Niche | Yes | No |
| `redis-cli` | Redis | Common | Yes (redis pkg) | No (service only) |
| `mongosh` | MongoDB | Common | Yes | No (service only) |
| `psql` | PostgreSQL | Essential | Yes (postgresql) | Via service |
| `mysql` | MySQL | Essential | Yes (mysql) | Via service |

All dbcli tools (pgcli, mycli, litecli) provide auto-completion, syntax highlighting, and multi-line query support. They are strict quality-of-life improvements over the bare `psql`/`mysql` clients.

### Migration Tools

| Tool | Ecosystem | Commonality | Nix-Packaged |
|------|-----------|-------------|--------------|
| `dbmate` | Language-agnostic (Go) | Common | Yes |
| `flyway` | JVM-based, multi-DB | Common | Yes |
| `liquibase` | JVM-based, multi-DB | Common | Yes |
| `atlas` | Schema-as-code (Go) | Growing | Yes |
| `golang-migrate` | Go | Common | Yes |
| `sqitch` | Perl-based, multi-DB | Niche | Yes |

Note: Most language ecosystems have their own migration tools (Prisma/Drizzle/Knex for JS/TS, Alembic for Python, Entity Framework for .NET, ActiveRecord for Ruby). These are better handled by per-ecosystem detection than global installation.

### Data Exploration

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `dbeaver` | Universal DB GUI | Common | Yes |
| `usql` | Universal SQL CLI | Niche | Yes |
| `sqlite3` | SQLite CLI | Common | Yes |

### gdev Integration Opportunity

**Medium value.** When gdev detects a devenv.sh service (e.g., PostgreSQL), it could auto-install the matching enhanced CLI client (pgcli). Migration tools are per-project, better left to ecosystem modules. The language-agnostic tools (dbmate, atlas) could be offered as optional packages.

---

## 4. Git Platform Integration

### Platform CLIs

| Tool | Platform | Commonality | Nix-Packaged | Config Required |
|------|----------|-------------|--------------|-----------------|
| `gh` | GitHub | Essential | Yes | `gh auth login` |
| `glab` | GitLab | Common | Yes | `glab auth login` |
| Bitbucket CLI | Bitbucket | Niche | No (unofficial) | Token config |

`gh` is essential for any GitHub-hosted project. It handles PR creation, issue management, CI status checking, release management, and repository browsing. In the consulting context, engineers commonly need `gh` configured against the **client's** GitHub org (not the consulting firm's).

`glab` serves the same role for GitLab and is standard when clients use self-hosted GitLab instances.

### Git Enhancement Tools

| Tool | Purpose | Commonality | Nix-Packaged | Config Required |
|------|---------|-------------|--------------|-----------------|
| `delta` | Better diffs (pager) | Common | Yes | `.gitconfig` pager setup |
| `lazygit` | Terminal UI for git | Common | Yes | None |
| `git-lfs` | Large file storage | Common | Yes | `git lfs install` |
| `git-crypt` | Transparent encryption | Common | Yes | GPG/age key setup |
| `git-cliff` | Changelog generation | Niche | Yes | `cliff.toml` |
| `git-absorb` | Fixup commit helper | Niche | Yes | None |
| `pre-commit` | Hook framework | Common | Yes | `.pre-commit-config.yaml` |
| `commitizen` | Conventional commits | Niche | Yes | Config file |

### Secrets in Git

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `git-crypt` | Transparent file encryption | Common | Yes |
| `sops` | Value-level encryption (YAML/JSON) | Common | Yes |
| `age` | Modern encryption (used with sops) | Common | Yes |

SOPS + age is the modern standard for secrets-in-git. It encrypts only values (not keys) in structured files, keeping diffs readable. git-crypt encrypts entire files transparently but makes diffs useless. For consulting work where repos often contain client-specific secrets (API keys, connection strings), both patterns are common.

### gdev Integration Opportunity

**High value.** gdev already configures git hooks and has a GitHub MCP server. Adding:
- Auto-install `gh` and/or `glab` based on engagement git remote
- Configure `delta` as git pager
- Install `lazygit` as optional productivity tool
- Install `git-lfs` and run `git lfs install` when `.gitattributes` contains LFS patterns
- Bundle `sops` + `age` for secrets management

---

## 5. Container & Orchestration Tools

### Core Kubernetes Tools

| Tool | Purpose | Commonality | Nix-Packaged | Config Required |
|------|---------|-------------|--------------|-----------------|
| `kubectl` | K8s CLI | Essential | Yes | `~/.kube/config` |
| `kustomize` | K8s manifest overlays | Common | Yes | None |
| `helm` | K8s package manager | Essential | Yes | Repo setup |
| `k9s` | Terminal K8s UI | Common | Yes | None |
| `kubectx`/`kubens` | Context/namespace switching | Common | Yes | None |
| `stern` | Multi-pod log tailing | Common | Yes | None |
| `krew` | kubectl plugin manager | Niche | Yes | None |

### Local Kubernetes Development

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `minikube` | Local K8s cluster | Common | Yes |
| `kind` | K8s in Docker | Common | Yes |
| `k3d` | Lightweight K8s (k3s in Docker) | Common | Yes |
| `skaffold` | Build/push/deploy automation | Common | Yes |
| `tilt` | Live-reload K8s dev | Common | Yes |
| `devspace` | K8s dev tool | Niche | Yes |

### Container Tools

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `docker` / `docker-compose` | Container runtime | Essential | Yes |
| `podman` / `podman-compose` | Rootless containers | Common | Yes |
| `buildah` | OCI image building | Niche | Yes |
| `skopeo` | Container image operations | Niche | Yes |
| `dive` | Image layer analysis | Common | Yes |
| `trivy` | Container vulnerability scan | Common | Yes |
| `hadolint` | Dockerfile linter | Common | Yes |

### gdev Integration Opportunity

**High value.** gdev already covers Docker and Helm as ecosystem modules. The gap is K8s operational tools:
- Install `kubectl` + `k9s` + `kubectx`/`kubens` as a K8s bundle
- Auto-detect `kubeconfig` paths for multi-cluster access
- Install `stern` for multi-pod log viewing
- Offer local cluster tools (kind/k3d) as optional for engagement setup
- `helm` is already an ecosystem module; ensure it also covers K8s deployment use cases

---

## 6. Infrastructure as Code

### Core IaC CLIs

| Tool | Purpose | Commonality | Nix-Packaged | Config Required | devenv.sh Native |
|------|---------|-------------|--------------|-----------------|------------------|
| `terraform` | IaC provisioning | Essential | Yes (BSL license) | Provider plugins, backend | Yes (language) |
| `opentofu` | OSS Terraform fork | Growing | Yes | Same as Terraform | Yes (language) |
| `pulumi` | IaC in general-purpose langs | Common | Yes | Backend, state config | No |
| `cdktf` | Terraform CDK | Niche | Via npm | None beyond Terraform | No |
| `terragrunt` | Terraform wrapper | Common | Yes | `terragrunt.hcl` | No |
| `tflint` | Terraform linter | Common | Yes | `.tflint.hcl` | No |
| `tfsec` / `trivy` | Terraform security scan | Common | Yes | None | No |
| `infracost` | Cost estimation | Common | Yes | API key | No |

### Notes

gdev already covers Terraform and OpenTofu as language ecosystem modules (Tier 1). The gap is the surrounding tooling: `terragrunt` for multi-environment management, `tflint` for linting, `infracost` for cost estimation. These are commonly needed together on consulting engagements where infrastructure cost accountability matters.

### gdev Integration Opportunity

**Medium value.** The Terraform/OpenTofu ecosystem modules should include optional companion tools (terragrunt, tflint, infracost) as part of the ecosystem configuration.

---

## 7. Documentation & Diagramming

### Diagram-as-Code Tools

| Tool | Format | Commonality | Nix-Packaged |
|------|--------|-------------|--------------|
| Mermaid CLI (`mmdc`) | Mermaid.js diagrams | Common | Yes (`mermaid-cli`) |
| `d2` | D2 language diagrams | Growing | Yes |
| `plantuml` | PlantUML diagrams | Common | Yes |
| `graphviz` (dot) | DOT graph language | Common | Yes |
| `kroki` | Unified diagram API | Niche | Self-hosted |

Mermaid is the dominant diagram-as-code format in 2025-2026, natively supported by GitHub Markdown, GitLab, MkDocs Material, and most documentation platforms. D2 is gaining traction for its cleaner syntax and better layout engine. PlantUML remains common in enterprise Java shops.

### Documentation Generators

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `mkdocs` + Material | Python-based doc site | Common | Yes |
| `mdbook` | Rust-based book builder | Common | Yes |
| `docusaurus` | React-based doc site | Common | Via npm |
| `hugo` | Static site generator | Common | Yes |
| `asciidoctor` | AsciiDoc processor | Niche | Yes |

### Architecture Decision Records

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `adr-tools` | ADR management CLI | Common | Yes |
| `log4brains` | ADR + static site | Niche | Via npm |

gdev Phase 14 already includes a `write-adr` skill for Claude Code. The gap is the underlying tooling: `adr-tools` for the ADR CLI, and Mermaid CLI for diagram generation.

### gdev Integration Opportunity

**Low-medium value.** These are project-specific rather than universally needed. Consider:
- Installing `mermaid-cli` and `d2` as optional diagram tools
- Installing `adr-tools` alongside the write-adr skill
- Making documentation generators per-project (detect `mkdocs.yml` or `book.toml`)

---

## 8. API Development Tools

### CLI HTTP Clients

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `curl` | Universal HTTP client | Essential | Yes |
| `httpie` (http/https) | Human-friendly HTTP | Common | Yes |
| `xh` | Rust-based httpie alternative | Niche | Yes |

### API-Specific Tools

| Tool | Protocol | Commonality | Nix-Packaged |
|------|----------|-------------|--------------|
| `grpcurl` | gRPC | Common | Yes |
| `evans` | gRPC (interactive) | Niche | Yes |
| `graphqurl` | GraphQL | Niche | Via npm |
| `bruno` | Multi-protocol (git-native) | Growing | Yes |

### API Specification Tools

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `openapi-generator` | Generate client/server code | Common | Yes |
| `swagger-cli` | Validate OpenAPI specs | Common | Via npm |
| `buf` | Protobuf/gRPC toolchain | Common | Yes |
| `protoc` | Protocol buffer compiler | Common | Yes |
| `spectral` | OpenAPI/AsyncAPI linter | Common | Via npm |

### Notes

Bruno is emerging as the git-native alternative to Postman. It stores API collections as plain-text files in the repo, making them version-controllable. This aligns well with consulting workflows where API definitions should live alongside code.

### gdev Integration Opportunity

**Medium value.** `httpie` and `grpcurl` are broadly useful. `buf`/`protoc` belong to gRPC ecosystem detection. OpenAPI tools are project-specific. Consider:
- Adding `httpie` and `jq` (already in productivity tools) to the default bundle
- Installing `grpcurl` and `buf` when protobuf files are detected
- Making Bruno available as an optional API tool

---

## 9. Communication & Collaboration CLIs

### Issue Tracking CLIs

| Tool | Platform | Commonality | Nix-Packaged | Config Required |
|------|----------|-------------|--------------|-----------------|
| `jira-cli` (ankitpokhrel) | Jira Cloud/Server | Common | Yes | Server URL, API token |
| `linear-cli` | Linear | Niche | Via npm/JSR | API key |
| Atlassian CLI (ACLI) | Jira/Confluence | Niche | No (commercial) | License + config |

### Communication CLIs

| Tool | Platform | Commonality | Nix-Packaged | Config Required |
|------|----------|-------------|--------------|-----------------|
| Slack CLI | Slack | Niche | Partial | Workspace auth |
| `slackcli` (open-source) | Slack | Niche | No | API token |

### Notes

The reality is that most consulting engineers interact with Jira, Linear, and Slack through web UIs or desktop apps rather than CLIs. The CLI tools exist but are used primarily by power users and for automation/scripting. MCP servers for these platforms (already partially planned in gdev) provide more value for the Claude Code workflow than standalone CLIs.

### gdev Integration Opportunity

**Low value for CLIs, medium value for MCP servers.** The MCP approach (Jira MCP, Linear MCP, Slack MCP) is more aligned with gdev's Claude Code integration than installing standalone CLIs. Consider:
- Adding Jira MCP and Linear MCP to the MCP server catalog
- Making `jira-cli` available as an optional tool for power users
- Skip Slack CLI (the desktop app and MCP are sufficient)

---

## 10. Time Tracking & Engagement Management

### Time Tracking Platforms

| Platform | CLI Available | Commonality | Notes |
|----------|---------------|-------------|-------|
| Harvest | REST API only | Common | No official CLI; third-party wrappers exist |
| Toggl Track | `toggl` CLI | Common | Unofficial CLI tools available |
| Clockify | REST API only | Common | No official CLI |
| Everhour | REST API only | Common | Embeds in Jira/GitHub |

### Reality Check

Time tracking in consulting firms is almost exclusively done through web UIs, browser extensions, or integrations with project management tools. There is no standard CLI tooling in this space. The closest thing to developer-friendly time tracking is Everhour's direct embedding in Jira/GitHub issues.

### gdev Integration Opportunity

**Very low value.** Not worth building. Time tracking is a web/app workflow, not a CLI workflow. If anything, a Claude Code skill that can log time via API would be more useful than a CLI tool.

---

## 11. IDE/Editor Setup Patterns

### Current Industry Practice

Consulting firms handle IDE standardization through several patterns:

1. **VS Code Extension Packs**: The most common approach. Firms create extension pack `.vsix` bundles or shared `extensions.json` files listing required extensions per engagement type. Standard extensions include:
   - ESLint + Prettier (code quality baseline)
   - GitLens (git history)
   - SonarLint (security shift-left)
   - Language-specific extensions
   - Dev Containers (reproducible environments)

2. **Shared `.vscode/settings.json`**: Checked into repos for per-project formatting rules, linter configs, and editor behavior.

3. **Dev Containers**: Docker-based reproducible environments that bundle all tools and extensions. ThoughtWorks Radar lists Dev Containers in the Trial ring for 2025. This overlaps significantly with devenv.sh's purpose.

4. **EditorConfig**: `.editorconfig` files for cross-editor formatting basics (indent style, line endings).

5. **JetBrains Shared Settings**: `.idea/` directory with code style and inspection configs. Less common in consulting (VS Code dominates).

### gdev's Position

gdev explicitly rejected "IDE config beyond Claude Code" (Rejected Feature #6). This is reasonable: IDE preferences are deeply personal, and VS Code's extension marketplace handles distribution well. However, there are non-controversial IDE-adjacent things gdev could provide:
- `.editorconfig` templates
- `.vscode/extensions.json` with engagement-appropriate extension recommendations
- `.vscode/settings.json` with non-opinionated quality settings

### gdev Integration Opportunity

**Low value for direct IDE configuration.** The explicit rejection stands. However:
- EditorConfig generation could fit in `qsdev init`
- Extension recommendation files are low-controversy
- Dev Containers and devenv.sh serve the same purpose; gdev should position devenv.sh as the superior alternative

---

## 12. Secrets Management (Cross-Cutting Concern)

### Tools Landscape

| Tool | Purpose | Commonality | Nix-Packaged |
|------|---------|-------------|--------------|
| `sops` | Encrypted secrets in files | Common | Yes |
| `age` | Modern encryption (sops backend) | Common | Yes |
| `git-crypt` | Transparent git file encryption | Common | Yes |
| 1Password CLI (`op`) | Password manager CLI | Common | Yes |
| Bitwarden CLI (`bw`) | Password manager CLI | Common | Yes |
| `vault` (HashiCorp) | Secrets engine | Common | Yes |
| `infisical` | Open-source secrets management | Growing | Yes |
| `doppler` | Secrets management SaaS | Niche | Yes |

### Consulting Patterns

Consulting engineers typically need:
1. **Personal credential management**: 1Password or Bitwarden for client passwords, API tokens, SSH keys
2. **Repo-level secrets**: sops+age or git-crypt for encrypted config files
3. **Runtime secrets**: HashiCorp Vault or cloud-specific secrets managers (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault)
4. **SSH key management**: ssh-agent, 1Password SSH agent, or system keyring

### gdev Integration Opportunity

**Medium-high value.** Secrets management is a universal need that gdev's security-focused positioning should address:
- Install `sops` + `age` as default secrets tooling
- Offer `git-crypt` as alternative
- Integrate with 1Password CLI (`op`) for credential injection
- Configure SSH agent forwarding patterns

---

## Summary: Priority-Ranked Expansion Candidates

### Tier 1: Essential (Install by default)

| Category | Tools | Rationale |
|----------|-------|-----------|
| Cloud CLIs | `aws-cli`, `gcloud`, `az` | Used on virtually every engagement |
| Productivity CLIs | `ripgrep`, `fd`, `bat`, `fzf`, `jq`, `yq`, `delta`, `eza`, `zoxide`, `tldr` | "Modern coreutils" expected by senior engineers |
| Git platform | `gh` | PR workflows are universal |
| K8s basics | `kubectl`, `k9s`, `kubectx`/`kubens` | Standard for cloud-native work |
| HTTP tools | `httpie` | Human-friendly API interaction |
| Secrets | `sops`, `age` | Secrets-in-git is standard practice |

### Tier 2: Common (Install when detected or opted-in)

| Category | Tools | Trigger |
|----------|-------|---------|
| Cloud credentials | `aws-vault`, `aws-sso-cli`, `saml2aws` | AWS SSO/SAML detected |
| Git platform | `glab` | GitLab remote detected |
| Git tools | `lazygit`, `git-lfs`, `git-crypt` | User preference / `.gitattributes` |
| K8s development | `helm`, `kustomize`, `stern`, `kind`/`k3d` | K8s manifests detected |
| IaC companions | `terragrunt`, `tflint`, `infracost` | Terraform/OpenTofu detected |
| Container tools | `dive`, `trivy`, `hadolint` | Dockerfile detected |
| DB clients | `pgcli`, `mycli` | Matching devenv service active |
| API tools | `grpcurl`, `buf`, `protoc` | `.proto` files detected |
| Diagramming | `mermaid-cli`, `d2` | Diagram sources detected |
| Migration | `dbmate`, `atlas` | Opt-in |

### Tier 3: Niche (Available but not installed by default)

| Category | Tools | Notes |
|----------|-------|-------|
| Collaboration CLIs | `jira-cli`, `linear-cli` | MCP servers are better fit |
| Doc generators | `mkdocs`, `mdbook`, `hugo` | Project-specific |
| ADR tools | `adr-tools` | Alongside existing write-adr skill |
| Network tools | `nmap`, `mtr`, `wireshark-cli` | Debugging only |
| Time tracking | None recommended | Web/app workflow, not CLI |
| IDE config | EditorConfig templates only | Explicitly rejected scope |

### Not Recommended for gdev

| Category | Reason |
|----------|--------|
| Time tracking CLIs | No standard CLI exists; web/app workflow |
| Slack CLI | Desktop app + MCP server sufficient |
| Full IDE configuration | Explicitly rejected; deeply personal |
| mise/asdf | Redundant with devenv.sh native version management |
| Dev Containers | Competitive with devenv.sh (gdev's foundation) |

---

## devenv.sh Native Capabilities Summary

For reference, devenv.sh already provides these capabilities natively, which gdev should leverage rather than reinvent:

- **40+ services**: PostgreSQL, Redis, MySQL, MongoDB, Elasticsearch, RabbitMQ, Kafka, NATS, MinIO, Keycloak, Vault, Prometheus, Caddy, Nginx, Mailpit, Temporal, and more
- **50+ languages**: Including all gdev Tier 1-4 ecosystems
- **direnv integration**: Built-in environment variable management
- **Process management**: `devenv up` for service orchestration
- **100,000+ Nix packages**: Any CLI tool can be added via `packages = [ pkgs.ripgrep pkgs.fd ... ]`
- **Pre-commit hooks**: Native integration
- **Container support**: devcontainer generation

The key insight: almost every tool in this report is Nix-packaged and can be added to a devenv.sh configuration with a single line. gdev's value-add is not installation (Nix handles that) but **curation, detection, and configuration** -- knowing which tools to install for a given engagement context and configuring them correctly.

---

## Sources

All web searches documented in `docs/web-search-sources-index.md`. Key fetched sources:
- `docs/devenv-sh-services-languages-reference.md` -- Complete devenv.sh service and language catalog
- `docs/thoughtworks-technology-radar-vol34-tools.md` -- ThoughtWorks Radar Vol 34 tools
- `docs/13-cli-tools-developer-2025.md` -- Essential CLI tools list
- `docs/aws-tools-consultant-work.md` -- AWS credential management for consultants
