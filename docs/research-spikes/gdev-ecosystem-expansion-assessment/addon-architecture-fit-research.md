# Addon Architecture Fit Assessment

## Research Question

For all identified ecosystem expansion opportunities, do they fit within gdev's existing 3-addon architecture (devenv, claudecode, devinit), or do they warrant new addons?

## Answer: All Expansions Fit Within Existing 3-Addon Architecture

No fourth addon is warranted. Every expansion category maps cleanly to the existing addon split when you apply gdev's design principle: **devenv generates environment files, claudecode generates AI config, devinit orchestrates and detects.**

---

## Expansion-to-Addon Mapping

### devenv Addon Expansions

The devenv addon gets the most new content because most expansions are "add packages and services to devenv.nix." This follows the existing ecosystem module pattern — detection heuristics + devenv.nix fragment + optional config files.

| Expansion | What Goes in devenv | Pattern |
|-----------|-------------------|---------|
| **Cloud CLIs** | `awscli2`, `google-cloud-sdk`, `azure-cli`, `aws-vault`, `saml2aws` in devenv.nix packages | New `cloud` module category |
| **K8s tools** | `kubectl`, `kubectx`, `k9s`, `stern`, `kustomize`, `helm-ls` in devenv.nix packages | New `kubernetes` module category |
| **Shell/workstation tools** | Modern coreutils (ripgrep, fd, bat, fzf, jq, yq, delta, eza, zoxide) managed via `qsdev setup` personal shell mode, NOT per-project devenv.nix | New `qsdev setup --shell` mode in Phase 10 |
| **Dev services** | Kafka, MinIO, Mailpit, Keycloak, NATS service sub-templates | Same pattern as existing 6 services |
| **API tools** | `httpie`, `grpcurl`, `buf`, `bruno`, `hurl`, `k6` in packages | Detection-triggered (`.proto`, `openapi.yaml`) |
| **DB migration tools** | `flyway`, `liquibase`, `prisma-engines`, `diesel-cli`, `sqlx-cli`, `goose`, `atlas`, `dbmate` | Detection-triggered (config files) |
| **Git tools** | `gh`, `glab`, `delta`, `lazygit`, `git-lfs`, `git-crypt`, `sops`, `age` in packages | Detection-triggered (`.github/`, `.gitattributes`) |
| **Documentation tools** | `mkdocs`, `mdbook`, `mermaid-filter`, `d2`, `plantuml` in packages | Detection-triggered (`mkdocs.yml`, `book.toml`) |
| **LSP servers** | Per-language LSP servers alongside language runtimes | Extend existing ecosystem modules |
| **Observability sidecar** | Docker-based `grafana/otel-lgtm` via `qsdev enable observability` | New service category (Docker sidecar) |
| **Devcontainer** | `devcontainer.enable = true` in devenv.nix | One-line toggle |

**New module categories needed**: `cloud`, `kubernetes` (following same interface as language ecosystem modules)

**New service sub-templates**: Kafka (Tier 1), MinIO, Mailpit, Keycloak, NATS (Tier 2)

### claudecode Addon Expansions

The claudecode addon gets expanded MCP server configuration and CLAUDE.md content for new tool categories.

| Expansion | What Goes in claudecode | Pattern |
|-----------|------------------------|---------|
| **MCP servers** | MySQL, SQLite MCP (auto-detected); Terraform, Sentry MCP (detect-and-offer); Atlassian, Linear, Slack, Datadog, Grafana, GitLab, AWS, Azure MCP (optional) | Extend existing .mcp.json generation |
| **DB migration docs** | CLAUDE.md sections documenting project's migration tool and workflow | Extend existing section-marker system |
| **API tool docs** | CLAUDE.md sections for OpenAPI specs, gRPC definitions, testing patterns | Extend existing section-marker system |
| **New skills** | Skills for cloud operations (AWS/GCP/Azure), K8s debugging, migration management | Extend existing skill library |

**Key constraint**: MCP 40-tool ceiling means never >6 servers simultaneously active. Auto-detection + three-tier security model governs which servers are configured.

### devinit Addon Expansions

The devinit addon gets new detection heuristics, client profile system, Copier integration, and file generation capabilities.

| Expansion | What Goes in devinit | Pattern |
|-----------|---------------------|---------|
| **Cloud detection** | Terraform provider detection, CI config parsing, deployment manifest detection | Extend existing detection engine |
| **K8s detection** | `k8s/`, `helm/`, `kustomization.yaml`, `skaffold.yaml` detection | Extend existing detection engine |
| **Client profiles** | sops+age encrypted profiles in `~/.qsdev/clients/`, selectable during `qsdev init`. Non-secret values (aws_profile name, git email, registry URLs) baked into project config. Secret values generate SecretSpec entries resolved at devenv runtime via provider (keyring/1Password/env). Two-layer: sops+age at rest, SecretSpec at runtime. | Extend init flow + profile selection + SecretSpec generation |
| **Copier integration** | `qsdev init --from <template>`, `qsdev update --template`, template registry | Extend existing init flow |
| **.editorconfig** | Always generate on `qsdev init` | New file type in atomic write pipeline |
| **.vscode/extensions.json** | Generate on `qsdev enable vscode` | New file type in atomic write pipeline |
| **Wizard expansion** | New form groups for cloud providers, K8s, services, API tools | Extend existing huh wizard |

---

## Why No Fourth Addon?

The strongest candidate for a fourth addon was **client profiles**, but the correct design is init-time profile selection rather than runtime switching. Client profiles live in `~/.qsdev/clients/` and are selected during `qsdev init` to pre-populate cloud config, git identity, registry endpoints, and compliance level into the project's devenv. This is init-time orchestration — squarely devinit's job.

1. **devinit is already the orchestration layer** — it coordinates what devenv and claudecode generate. Profile selection during init is orchestration.
2. **The command surface is small** — profile selection in the wizard, `qsdev init --profile <client>`, profile CRUD in `~/.qsdev/`. No separate addon needed.
3. **gdev's addon convention is one-per-concern** — devenv=environment, claudecode=AI, devinit=orchestration. Profile selection is orchestration.
4. **The config files are user-level** — they live in `~/.qsdev/`, not in project directories. devinit already manages user-level config.

Other candidates considered:
- **`ops` addon** for CI/CD, deployment, infrastructure — rejected features remain correctly rejected. No ops addon needed.
- **`cloud` addon** for cloud provider management — cloud CLIs are just devenv.nix packages. No runtime management. Not enough code for an addon.
- **`team` addon** for collaboration tools — team config is already in devinit (Phase 13). No separate addon.

---

## Implementation Impact on Existing Phases

Most expansions extend existing phases rather than creating new ones:

| Expansion | Extends Phase | How |
|-----------|--------------|-----|
| Cloud/K8s modules | **Phase 2** (ecosystem modules) | New module categories alongside language modules |
| Shell/workstation config | **Phase 10** (distribution/self-bootstrapping) | `qsdev setup` personal shell config mode — manages `~/.qsdev/shell/` dotfile fragments for coreutils, aliases, starship. Not per-project devenv. |
| Service templates | **Phase 3** (devenv core generation) | New service sub-templates |
| MCP servers | **Phase 4** (claudecode core) + **Phase 12** (extended integrations) | Expanded .mcp.json generation |
| API/DB/git/doc tools | **Phase 7** (ecosystem modules tiers 2-4) | New tool detection modules |
| Client profiles | **Phase 6** (wizard/orchestration) + **Phase 13** | Init-time profile selection from `~/.qsdev/clients/`, profile CRUD |
| Copier integration | **Phase 6** (wizard/orchestration) | New init flow path |
| .editorconfig/.vscode | **Phase 8** (migration/update/polish) | New generated file types |
| Observability sidecar | **Phase 12** (extended integrations) | New Docker sidecar lifecycle |
| Devcontainer toggle | **Phase 3** (devenv core generation) | One-line devenv.nix addition |
| Wizard expansion | **Phase 6** (wizard/orchestration) | New form groups |

**New phases potentially needed:**
- None for individual expansions. All fit within existing phases.
- However, the **volume** of new ecosystem modules (cloud, K8s, API, DB, git, docs) may warrant splitting Phase 7 or creating a Phase 7b for non-language ecosystem modules. Currently Phase 7 is "Tiers 2-4 language ecosystems" (19 modules). Adding cloud/K8s/API/DB/git/docs modules would add ~10-15 more detection+generation modules.

---

## Recommended Phase Amendments

1. **Phase 1**: Add modern coreutils to default packages list
2. **Phase 2**: Add `cloud` and `kubernetes` as module categories alongside language modules
3. **Phase 3**: Add Kafka service template (Tier 1); add MinIO/Mailpit/Keycloak/NATS as optional (Tier 2); add devcontainer.enable toggle
4. **Phase 4**: Expand MCP server generation to include MySQL/SQLite (auto), Terraform/Sentry (detect-and-offer), plus optional server catalog
5. **Phase 6**: Add Copier template path (`qsdev init --from <template>`); expand wizard with cloud/K8s form groups; add .editorconfig generation
6. **Phase 7**: Rename to "Ecosystem Modules — Tiers 2-4 & Non-Language Tools"; add API tool, DB migration, git tool, and documentation tool detection modules
7. **Phase 8**: Add .vscode/extensions.json generation to migration/update pipeline
8. **Phase 12**: Add observability sidecar (`qsdev enable observability`), expanded tool lifecycle for cloud/K8s tools
9. **Phase 13**: Add client profile system (`qsdev switch`, `~/.gdev/clients.yaml`)
10. **Phase 14**: Add cloud operations skills, K8s debugging skills, migration management skills

---

## Critical Dependency: devenv >= 2.0

The implementation plan implicitly depends on devenv 2.0 features. This must be an explicit minimum version requirement:

- **DAG-based task system** — `before`/`after` ordering, `status` caching, `execIfModified`, JSON data passing
- **Process integration** — `devenv:processes:*` auto-exposed as tasks
- **`devcontainer.enable`** — native devcontainer.json generation
- **Namespace support** — `namespace:task` convention for gdev-generated tasks

devenv 2.0 was released March 2026. All actively maintained installations should already be on 2.0+. `qsdev doctor` should check devenv version and warn if < 2.0.

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Phase 7 scope bloat (19 language + 15 tool modules) | Medium | Split into Phase 7a (languages) and Phase 7b (tools) |
| Client profiles become an identity management system | Medium | Strict scope: init-time pre-population only. No credential storage (sops+age encrypts at rest, SecretSpec resolves at runtime via existing providers). No SSO integration, no runtime switching. |
| MCP server count exceeds performance ceiling | Low | Hard limit of 6 simultaneous servers; tiered auto-configuration policy |
| devenv < 2.0 in the wild | Low | `qsdev doctor` version check + clear error. devenv 2.0 released March 2026 — 2+ months ago. |
| Copier dependency adds Python runtime requirement | Medium | Copier is optional; only needed when `--from` flag is used. Gate on Python availability. |
