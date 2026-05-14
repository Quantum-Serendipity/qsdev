# Rejected Features Reconsideration & Consulting-Specific Operations Research

## Research Questions

1. Should any of gdev's 13 rejected features be reconsidered for a "one stop shop" goal?
2. What consulting-specific operational tools are missing from gdev?
3. Should gdev integrate with or replace runtime version managers?

---

## Part 1: Rejected Feature Reconsideration

The implementation plan rejected 13 features using a three-test framework: (1) Is there a purpose-built tool? (2) Is it file generation or runtime behavior? (3) Does it compound with existing features? Each rejection is re-evaluated below against the "one stop shop" consulting platform goal.

### Feature 1: Standalone Task Runner

**Original rejection**: devenv 2.0 native tasks handle this.

**Updated assessment**: **KEEP REJECTED. Rejection strengthened.**

devenv 2.0 (released March 2026) significantly exceeded the capabilities assumed at rejection time. The task system now includes:
- Full DAG-based parallel execution with `before`/`after` dependency ordering
- Caching via `status` commands and `execIfModified` with content hash tracking
- JSON-based data passing between tasks (`$DEVENV_TASK_INPUT`, `$DEVENV_TASKS_OUTPUTS`)
- Process integration (all `processes` auto-exposed as `devenv:processes:<name>` tasks)
- Lifecycle hooks (`devenv:enterShell`, `devenv:enterTest`)
- Namespace support with `namespace:task` convention
- CLI input overrides (`--input`, `--input-json`)
- Shell messages via `$DEVENV_TASK_OUTPUT_FILE`
- Language-specific execution via `package` attribute

This is not a toy task runner -- it has feature parity with Taskfile and exceeds just on every dimension except YAML syntax simplicity. The only tool that arguably offers more is mise (which combines version management + env vars + tasks), but devenv already handles all three of those concerns for gdev.

**What gdev should do**: Generate ecosystem-specific devenv task definitions (e.g., `go:build`, `npm:test`, `rust:lint`) as part of the devenv addon. Consider `gdev run <task>` as a thin wrapper around `devenv tasks run`.

**Competing tools**: mise tasks, just, Taskfile, make. None add value over devenv tasks in a devenv-based environment.

---

### Feature 2: Container Management

**Original rejection**: Docker/Podman exist. devenv 2.0 has built-in process manager.

**Updated assessment**: **PARTIAL RECONSIDERATION. Add devcontainer generation only.**

The rejection of container lifecycle management (start/stop/build) remains correct -- Docker Compose and Podman handle this. However, there is a specific gap: **devcontainer.json generation**.

devenv.sh already has native devcontainer support (`devcontainer.enable = true` in devenv.nix generates `.devcontainer.json` automatically). gdev should:
1. Enable `devcontainer.enable = true` in generated devenv.nix when the project targets remote/cloud development
2. This is pure file generation (passes test 2) and compounds with devenv's existing capability (passes test 3)

This is not "container management" -- it is configuration file generation, which is gdev's core competency. The distinction matters because some consulting clients require Codespaces or devcontainer-based workflows, and gdev should generate the config seamlessly.

**Minimal integration**: Add `devcontainer.enable = true` to generated devenv.nix. One line. Zero maintenance.

**Competing tools**: devcontainers spec, GitHub Codespaces, Gitpod. All consume `.devcontainer.json`, which devenv already generates.

---

### Feature 3: CI Execution

**Original rejection**: Local CI execution is complex; nektos/act (56k+ stars) exists.

**Updated assessment**: **KEEP REJECTED.**

act's limitations confirm the rejection was correct:
- Only supports Linux runners (no macOS/Windows)
- Default images lack many tools GitHub Actions provides
- Matrix jobs with port conflicts fail (shared network namespace)
- Service containers and systemd-dependent actions don't work
- Secrets injection requires careful manual configuration

Local CI execution is a "90% solution" problem where the last 10% causes debugging headaches that exceed the value. gdev should not own this complexity.

**What gdev could do**: Include `act` in devenv packages as an optional tool. Document it. Do not manage it.

---

### Feature 4: Deployment

**Original rejection**: Out of scope. Orthogonal to dev environment.

**Updated assessment**: **KEEP REJECTED.** No change in reasoning. Deployment is infrastructure-specific (Terraform/Pulumi/ArgoCD territory) and varies enormously between clients. gdev already supports Terraform/OpenTofu as an ecosystem module.

---

### Feature 5: Code Scaffolding

**Original rejection**: Every ecosystem has its own scaffolding tool. Maintaining 27+ templates would drift.

**Updated assessment**: **RECONSIDER. Add Copier template support for consulting-firm project templates.**

The original rejection correctly identifies that gdev should not replace `cargo init`, `npm create`, or `dotnet new`. However, it misses a specific consulting use case: **firm-wide project templates**.

Copier (not Cookiecutter) is the right tool because it uniquely supports **template updates** -- when the firm's project template evolves (new security policies, updated CI workflows, new linting rules), Copier can merge those updates into existing projects via `copier update`. This is precisely the lifecycle management a consulting firm needs.

The integration would be:
1. gdev includes Copier in devenv packages
2. Firm maintains Copier templates in a Git repo (project structure, CI workflows, security config, gdev config)
3. `gdev init --from <template-url>` wraps `copier copy` for new projects
4. `gdev update --template` wraps `copier update` for existing projects
5. gdev's own generated files (.gdev.yaml, devenv.nix, CLAUDE.md) become part of the Copier template

This passes all three tests:
- Purpose-built tool exists (Copier) -- gdev wraps, doesn't reimplement
- File generation only -- Copier generates files, gdev orchestrates
- Compounds with existing features -- templates include gdev config, security policies, CI workflows

**Minimal integration**: Include Copier in devenv packages. Add `gdev init --from` flag. Medium effort, high value for consulting standardization.

**Competing tools**: Cookiecutter (no update support), Yeoman (Node.js, complex), structkit (newer, unproven).

---

### Feature 6: IDE Config Beyond Claude Code

**Original rejection**: Deeply personal, highly variable. Claude Code is special because of security model.

**Updated assessment**: **PARTIAL RECONSIDERATION. Add .editorconfig and .vscode/extensions.json generation.**

The rejection of comprehensive IDE configuration remains correct -- gdev cannot and should not manage Neovim plugins, JetBrains settings, or VS Code themes. However, two specific files are universal, low-maintenance, and high-value:

1. **`.editorconfig`** -- Defines indent style, tab width, end-of-line, encoding, trim trailing whitespace. Supported by virtually every editor (VS Code, JetBrains, Vim, Emacs, Sublime, Helix, Zed) without plugins in most cases. This is a consensus standard, not personal preference.

2. **`.vscode/extensions.json`** -- Lists recommended extensions. When a developer opens the project in VS Code, they see "This workspace has extension recommendations." This is opt-in (recommendations, not requirements) and low-maintenance.

Both are file generation (test 2) and compound with gdev's ecosystem detection (test 3). gdev already knows the project's language ecosystems; generating appropriate editorconfig rules and extension recommendations is trivial.

**Minimal integration**: Generate `.editorconfig` with ecosystem-appropriate rules. Generate `.vscode/extensions.json` with relevant extensions (ESLint for JS/TS, rust-analyzer for Rust, etc.). Both opt-in via `.gdev.yaml` config.

**What NOT to do**: Do not generate `.vscode/settings.json` (too opinionated), `.idea/` (JetBrains-specific), or any editor-specific keybindings/themes.

---

### Feature 7: OTEL Infrastructure

**Original rejection**: Running Prometheus + Loki + Grafana is infrastructure ops. Just env vars.

**Updated assessment**: **PARTIAL RECONSIDERATION. Provide devenv service template, not default infrastructure.**

The rejection of shipping OTEL infrastructure as a default remains correct. However, devenv.sh natively supports the OpenTelemetry Collector as a service (`services.opentelemetry-collector.enable = true`), and the devenv service catalog also includes Prometheus. This means:

1. A local OTEL Collector + Prometheus stack can be declared in devenv.nix as services
2. No Docker required -- these run as native devenv processes
3. gdev could provide a **service template** (a devenv.nix snippet) that teams opt into

The key distinction from the rejected "OTEL infrastructure" is:
- **Rejected**: gdev ships and manages a Prometheus + Grafana stack
- **Reconsidered**: gdev provides a devenv.nix snippet that enables `services.opentelemetry-collector` and `services.prometheus` for teams that want local observability

This is file generation (test 2) and compounds with the existing OTEL env var generation (test 3).

**Minimal integration**: Add an "observability" service template to gdev's devenv addon that can be enabled via `.gdev.yaml`. The template enables the OTEL Collector service with a sensible default config. Teams opt in; it is not default.

**Competing tools**: claude-code-otel (Docker Compose stack), Docker Desktop's built-in OTEL.

---

### Feature 8: Package Manager Installation

**Original rejection**: devenv.nix declares packages; Nix provides them.

**Updated assessment**: **KEEP REJECTED.** No change. devenv + Nix fully handles this. gdev generates the devenv.nix that declares the right packages.

---

### Feature 9: Git Server API

**Original rejection**: API integration is maintenance nightmare. gdev generates files, not API calls.

**Updated assessment**: **KEEP REJECTED.** No change. Terraform GitHub/GitLab providers handle declarative repo configuration. The `gh` CLI (GitHub) and `glab` CLI (GitLab) handle interactive operations. gdev should not call APIs.

However, note that gdev should include `gh` and `glab` in devenv packages (Category G gap from coverage matrix). This is "install the tool" not "manage the tool."

---

### Feature 10: Vulnerability Database

**Original rejection**: OSV.dev, GitHub Advisory Database, NVD exist. OSV Scanner already integrated.

**Updated assessment**: **KEEP REJECTED.** No change. Maintaining a vulnerability database requires a dedicated security team. OSV Scanner consumes these databases; gdev configures OSV Scanner.

---

### Feature 11: Merge Queue Automation

**Original rejection**: Out of scope.

**Updated assessment**: **KEEP REJECTED.** Merge queues are a Git platform feature (GitHub Merge Queue, GitLab Merge Trains). Configuration belongs in repository settings or Terraform, not gdev.

---

### Feature 12: Release Automation

**Original rejection**: Out of scope. (Note: plan.md lists 10 rejected features but subsequent analysis added merge queue, release automation, and Nix flake management as additional rejections, totaling 13.)

**Updated assessment**: **KEEP REJECTED.**

git-cliff (already in gdev Phase 16) handles changelog generation and version bump calculation. commitlint (already in gdev) enforces conventional commits. The gap between this and full release automation (semantic-release, changesets) is:

1. **Package publishing** -- CI/CD concern, not dev environment
2. **Release PR creation** -- GitHub Actions concern (release-please)
3. **Multi-package coordination** -- Monorepo tool concern (changesets, Lerna)

Adding semantic-release would introduce a Node.js runtime dependency and a heavy plugin ecosystem. changesets is better (intentional releases vs automated parsing) but is JS-ecosystem-specific. Neither fits gdev's "file generation, not runtime behavior" principle.

**What gdev could do**: Generate a GitHub Actions workflow that uses git-cliff + release-please for automated releases. This is file generation.

---

### Feature 13: Nix Flake Management

**Original rejection**: devenv-only. gdev targets devenv.

**Updated assessment**: **KEEP REJECTED.** gdev generates devenv.nix, not flake.nix. devenv abstracts away flake complexity. If a team needs raw flakes, they are beyond gdev's target audience.

---

### Reconsideration Summary

| # | Feature | Original Verdict | New Verdict | Action |
|---|---------|-----------------|-------------|--------|
| 1 | Task runner | Rejected | **Keep rejected** (strengthened) | Generate devenv task definitions |
| 2 | Container management | Rejected | **Partial reconsider** | Add devcontainer.enable to devenv.nix |
| 3 | CI execution | Rejected | **Keep rejected** | Include act as optional devenv package |
| 4 | Deployment | Rejected | **Keep rejected** | No change |
| 5 | Code scaffolding | Rejected | **Reconsider** | Integrate Copier for firm project templates |
| 6 | IDE config beyond Claude Code | Rejected | **Partial reconsider** | Generate .editorconfig + .vscode/extensions.json |
| 7 | OTEL infrastructure | Rejected | **Partial reconsider** | Provide opt-in devenv service template |
| 8 | Package manager installation | Rejected | **Keep rejected** | No change |
| 9 | Git server API | Rejected | **Keep rejected** | Include gh/glab in devenv packages |
| 10 | Vulnerability database | Rejected | **Keep rejected** | No change |
| 11 | Merge queue automation | Rejected | **Keep rejected** | No change |
| 12 | Release automation | Rejected | **Keep rejected** | Generate CI workflow for git-cliff releases |
| 13 | Nix flake management | Rejected | **Keep rejected** | No change |

**Net result**: 8 confirmed rejections, 1 full reconsideration (scaffolding via Copier), 3 partial reconsiderations (devcontainer generation, .editorconfig/.vscode, OTEL service template), 1 strengthened rejection (task runner).

---

## Part 2: Consulting-Specific Operational Tools

### 2.1 Time Tracking CLIs

**Landscape assessment**:
- **Toggl**: 5+ CLI tools on GitHub (Python, Rust, Go). Most mature is AuHau/toggl-cli (Python). watercooler-labs/toggl-cli (Rust) is newer and uses the v9 API.
- **Clockify**: lucassabreu/clockify-cli (Go) is comprehensive. mentarch/clockify-cli offers keychain storage, rich reports, cross-platform support.
- **Harvest**: zenhob/hcl is the only notable CLI. Ruby-based, appears less actively maintained.

**Quality assessment**: All are community-maintained, single-developer projects. None approach the quality of official vendor CLIs (like `gh` for GitHub). Reliability for a firm-wide standard is questionable.

**gdev integration assessment**: This is an **organizational concern**, not a gdev concern. The firm chooses a time tracking vendor; individual engineers install the CLI. gdev could:
1. Include a time tracking CLI in devenv packages (once the firm picks a vendor)
2. Configure the API key via SecretSpec integration
3. Surface time tracking reminders in shell entry messages

**Recommendation**: Do not build time tracking features into gdev. If the firm standardizes on Clockify (free tier, best CLI ecosystem), include `clockify-cli` in devenv packages and configure the API key via SecretSpec. This is "install the tool" level, not "manage the tool."

### 2.2 Client Environment Isolation

**Current patterns in industry**:
1. **Directory-based isolation**: `~/work/client-a/`, `~/work/client-b/` with per-directory git conditional includes
2. **AWS profile-per-client**: Named profiles in `~/.aws/config` with SSO or IAM role per client
3. **Separate SSH keys**: SSH config `Host` blocks mapping to different identity files
4. **VPN per client**: Manual VPN switching (often via separate apps)
5. **Credential tools**: aws-vault (keychain-stored AWS creds + MFA + AssumeRole), Granted (multi-account console access via browser containers)

**The gap**: These patterns are all manually configured today. No existing tool bundles cloud creds + git identity + SSH key + VPN + time tracking into a single switchable profile.

**gdev integration assessment**: This is a **strong gdev opportunity**. A "client profile" in `.gdev.local.yaml` that bundles:

```yaml
clients:
  acme-corp:
    aws_profile: acme-prod
    git:
      user_name: "Colin Rushton"
      user_email: "colin@highspring.com"
      signing_key: "~/.ssh/acme-signing"
    ssh_key: "~/.ssh/acme-identity"
    time_tracking:
      provider: clockify
      workspace: "Acme Corp"
      project: "Platform Modernization"
    env_vars:
      CLIENT_ID: acme
      OTEL_SERVICE_NAME: acme-platform
```

`gdev switch acme-corp` would:
1. Set `AWS_PROFILE=acme-prod` (and optionally run `aws-vault exec`)
2. Set git user.name, user.email, signing key for the current shell
3. Set environment variables
4. Display the active client in the shell prompt (Starship integration)

This passes the three tests:
- No purpose-built tool exists for holistic client switching
- It is environment configuration (env vars, git config), not runtime behavior
- It compounds with existing features (SecretSpec for credentials, Starship for prompt, OTEL for billing attribution)

**Recommendation**: **Add client profiles as a new gdev feature.** This is the strongest consulting-specific differentiator identified. Estimated effort: medium (env var management + git config + Starship integration).

### 2.3 Engagement Lifecycle

**Assessment**: Templates for new client engagements (repo structure, CI setup, security policies, handoff documentation) are precisely the use case for Copier integration (Feature 5 reconsideration above). Evidence collection for compliance is already covered in Phase 15 (posture scoring + evidence reports).

**Recommendation**: Covered by Copier integration + existing Phase 15. No additional feature needed.

### 2.4 Multi-Tenant Environment Switching

**Assessment**: This is the same concern as "Client Environment Isolation" (2.2 above). The `gdev switch <client>` command is the answer.

**Additional consideration**: The shell prompt (Starship) should display the active client profile to prevent "wrong client" mistakes. This is a common consulting failure mode -- pushing to the wrong AWS account or committing with the wrong git identity.

**Recommendation**: Covered by client profiles (2.2).

### 2.5 Cost Tracking (AI Usage Per Client)

**Assessment**: Claude Code's native OTEL support already provides cost attribution via resource attributes (`project.name`, `client.id`). The observability research in `gdev-dx-polish` confirms this is profile-driven and already designed.

The broader question of cloud cost allocation (AWS/GCP spend per client) is an organizational finance concern, not a dev environment concern. Tools like AWS Cost Explorer tags, GCP billing labels, and Vantage/Infracost handle this.

**Recommendation**: Already covered by OTEL env var generation (Phase 14). Client profiles (2.2) would set the `OTEL_RESOURCE_ATTRIBUTES` to include `client.id=<name>` automatically.

### 2.6 Compliance and Audit Trails

**Assessment**: Phase 15 already designs posture scoring (defense coverage 40%, configuration health 30%, dependency health 30%) with evidence reports mapping to SOC2/HIPAA/ASVS frameworks. The evidence is machine-readable JSON with integrity verification.

The industry uses platforms like Vanta, Drata, and Secureframe for automated SOC2 evidence collection. gdev's evidence reports could feed into these platforms as one data source among many.

**What's missing**: gdev's evidence is point-in-time snapshots. Continuous monitoring (drift detection, real-time alerts) is the gap -- but this requires infrastructure (a server collecting reports), which gdev explicitly avoids.

**Recommendation**: Phase 15 covers the gdev-appropriate scope. For continuous monitoring, teams use Vanta/Drata/Secureframe. gdev's evidence JSON can be exported to these platforms.

### 2.7 Knowledge Management

**Assessment**: Engagement retrospectives, lessons learned databases, and technology radar are organizational knowledge management concerns. They live in Confluence/Notion/Obsidian, not in a CLI tool.

gdev's write-adr skill (Phase 14) handles Architecture Decision Records, which is the closest developer-tool-level knowledge management. Technology radar is a leadership/strategy artifact.

**Recommendation**: Out of scope. The write-adr skill is sufficient for dev-tool-level knowledge capture.

### 2.8 Consulting-Specific Tool Summary

| Tool Category | gdev Concern? | Recommendation | Effort |
|---------------|---------------|----------------|--------|
| Time tracking CLIs | Marginal | Include vendor CLI in devenv packages | Small |
| Client environment isolation | **Yes** | Client profiles with `gdev switch` | Medium |
| Engagement lifecycle | Covered | Copier templates + Phase 15 evidence | N/A |
| Multi-tenant switching | **Yes** | Same as client profiles | N/A |
| AI cost tracking | Covered | OTEL resource attributes per client | N/A |
| Compliance/audit | Mostly covered | Phase 15 + external platforms | Small |
| Knowledge management | No | write-adr skill sufficient | N/A |

**Key finding**: The single most impactful consulting-specific addition is **client profiles** (`gdev switch <client>`). Everything else is either already covered or out of scope.

---

## Part 3: Runtime Version Management Integration

### The Question

devenv.sh pins exact tool versions via Nix. Does this fully replace runtime version managers (mise, asdf, nvm, pyenv, rbenv), or do engineers need both?

### Analysis

**What devenv provides**:
- Exact version pinning via `devenv.lock` (content-addressed Nix store paths)
- Cross-platform (ARM macOS, x86-64 Linux) with native binaries
- Reproducible -- same compiler and dependencies across all machines and time periods
- Supports 50+ languages natively
- Automatic environment activation via shell hook (like direnv)
- Environment variables via `env` attribute
- Process management, services, tasks

**What mise provides that devenv doesn't**:
- Reads existing `.nvmrc`, `.python-version`, `.tool-versions` files (compatibility with non-Nix projects)
- 900+ installable tools via multiple backends (asdf plugins, cargo, npm, pipx, aqua)
- Simpler mental model (TOML config vs Nix language)
- Faster cold-start (no Nix evaluation overhead)
- `.mise.local.toml` for local overrides

**The redundancy question**: For gdev-managed projects, devenv **fully replaces** runtime version managers. The Nix store provides every tool at the exact version declared in devenv.nix. There is no need for nvm, pyenv, or mise to manage tool versions.

**The compatibility question**: For projects that a consulting engineer inherits from a client (which may use `.nvmrc`, `.tool-versions`, or `.python-version`), devenv can still work -- these files are human-readable and the versions can be mapped to Nix packages. But there is no automatic consumption of these files.

**The transition question**: Engineers coming from mise/asdf/nvm will have muscle memory for those tools. devenv's Nix-based approach has a steeper learning curve. However, gdev abstracts away the Nix complexity -- engineers run `gdev init` and get a working environment without writing Nix.

### Recommendation

**Do not integrate with or ship runtime version managers.** devenv is the version manager for gdev-managed projects. Adding mise or asdf would create confusion about which tool is authoritative for version management.

**Exception**: If a consulting engineer needs to work on a client project that is NOT gdev-managed and uses `.tool-versions` or `.nvmrc`, they install mise themselves for that project. This is outside gdev's scope -- gdev manages gdev-managed projects, not arbitrary client environments.

**What gdev could document**: A migration guide from mise/asdf/nvm to devenv, showing how `.tool-versions` entries map to `languages.*.version` in devenv.nix.

---

## Depth Checklist

- [x] Underlying mechanism explained -- devenv 2.0 task system, Copier template updates, client profile env switching, OTEL service configuration
- [x] Key tradeoffs -- file generation vs runtime behavior, "install the tool" vs "manage the tool", convenience vs maintenance burden
- [x] Compared to alternatives -- each feature compared to competing tools (mise, just, Taskfile, cookiecutter, semantic-release, etc.)
- [x] Failure modes -- template drift (Copier solves via updates), wrong-client commits (client profiles solve via prompt display), CLI quality (time tracking CLIs are community-maintained)
- [x] Concrete examples -- devenv task definitions, Copier template workflow, client profile YAML, devcontainer.enable toggle
- [x] Standalone-readable -- yes, includes all context needed for decisions without consulting original sources
