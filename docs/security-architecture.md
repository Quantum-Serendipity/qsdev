# Security Architecture

## Threat Model

### Adversary Goals

1. **Dependency confusion / typosquatting** — Trick the developer or AI agent into installing a malicious package with a name similar to a legitimate one.
2. **Supply chain compromise** — Inject malicious code into a legitimate package via account takeover, build system compromise, or CI action hijacking.
3. **Agent exploitation** — Use Claude Code's tool-calling capability to run arbitrary package installs, pipe-to-shell commands, or destructive operations.
4. **Credential exfiltration** — Read secrets from the developer's environment, `.env` files, or cloud provider credentials.
5. **Lockfile manipulation** — Modify lockfiles to redirect package resolution to attacker-controlled registries.

### Trust Boundaries

- **Developer workstation** — The devenv shell is the primary trust boundary. Clean mode strips credentials; pre-commit hooks scan for secrets.
- **Claude Code sandbox** — Ask rules, deny rules, and PreToolUse hooks constrain what the AI agent can execute. The permission model prevents bypassing these controls.
- **CI pipeline** — harden-runner constrains network egress; vulnerability scanners block known-bad dependencies.
- **Package registry** — Registry proxies (Nexus, Artifactory, GitHub Packages) provide a controlled ingestion point. pnpm workspace configuration enforces `strictDepBuilds` and `trustPolicy: no-downgrade`.

## Defense Layers

qsdev implements 14 layers of defense across supply chain, environment, and agent security.

| # | Layer | Catches |
|---|-------|---------|
| 1 | Age-gating | Zero-day package takeovers |
| 2 | Install script blocking | Arbitrary code at install time |
| 3 | Lockfile enforcement | Silent dependency redirection |
| 4 | Vulnerability scanning | Known CVEs in the dependency tree |
| 5 | PreToolUse hooks (package-guard) | AI agent adding unvetted packages |
| 6 | Nix hardening | Impure builds, environment leaks |
| 7 | SAST (Semgrep) | Dangerous code patterns in source |
| 8 | Secrets scanning (ripsecrets + gitleaks) | Credentials committed to source |
| 9 | Container security | Vulnerable base images, misconfigurations |
| 10 | License compliance | Non-permissive transitive dependencies |
| 11 | Cloud credential isolation | Cross-project credential leakage |
| 12 | Policy engine | Policy violations and unsafe tool invocations |
| 13 | Package and MCP risk scoring | Unvetted packages and untrusted MCP servers |
| 14 | Agent self-protection | Tampering with guardrails, config, and audit trail |

### Layer 1: Age-Gating

New package versions are blocked for a configurable period after publication. This provides a window for the community to discover and report compromised releases before they enter your project.

| Infrastructure Profile | Minimum Release Age | Update Tool |
|------------------------|--------------------:|-------------|
| `consulting-default` | 3 days (4320 min) | Renovate |
| `startup-github` | None (0 days) | Dependabot |
| `enterprise` | 7 days | Renovate |

For pnpm workspaces, age-gating is additionally enforced at install time via `minimumReleaseAge: 4320` in `pnpm-workspace.yaml`.

### Layer 2: Install Script Blocking

Per-ecosystem configuration files disable install-time script execution — the single most exploited attack vector in package supply chains.

| Ecosystem | Config File | Key Setting |
|-----------|------------|-------------|
| JavaScript (npm) | `.npmrc` | `ignore-scripts=true` |
| JavaScript (yarn Berry) | `.yarnrc.yml` | `enableScripts: false` |
| JavaScript (yarn Classic) | `.yarnrc` | `ignore-scripts true` |
| JavaScript (pnpm) | `.npmrc` + `pnpm-workspace.yaml` | `ignore-scripts=true`, `strictDepBuilds` |
| Python | `pip.conf` | `--no-deps` enforcement |
| Rust | `.cargo/config.toml` | Registry pinning |
| Ruby | `devenv.nix` env | `BUNDLE_DISABLE_EXEC_LOAD=true` |
| PHP | `composer.json` config | Script restrictions |
| .NET | `nuget.config` | Source pinning |

pnpm workspaces additionally enforce `blockExoticSubdeps` to prevent subdependencies from pulling in unexpected transitive packages.

**qsdev's own installs.** Packages qsdev installs globally itself (Claude Code in the bootstrap and `qsdev devenv setup`, MCP servers via `qsdev mcp install`) run outside the project `.npmrc` and the package guard, so each command carries the same controls: an exact pinned release from the catalog, an age gate (`npm --before`, `uv --exclude-newer`) and, for npm, `--ignore-scripts`. The one exception is the Claude Code bootstrap, whose pinned package needs its `postinstall` to place its native binary; that script runs, but only for the exact, age-gated release the catalog pins. devenv and direnv come from `nix profile install` of nixpkgs pinned to one commit (never the mutable `nixpkgs` registry entry) with `accept-flake-config` forced off, so the install cannot pick up a flake's extra substituters or trusted keys. Every bootstrap install re-detects the binary afterwards and fails if it is still not on `PATH`. See [Machine Bootstrap](configuration-reference.md#machine-bootstrap).

### Layer 3: Lockfile Enforcement

- **Pre-commit hooks** — The `lock-file-audit` custom hook flags changes to `devenv.lock`, `flake.lock`, `package-lock.json`, and `pnpm-lock.yaml` with a warning to verify the diff during code review.
- **CI** — Security scan workflows verify lockfile integrity as part of the build.
- **pnpm workspace** — `trustPolicy: no-downgrade` prevents lockfile changes that regress dependency versions.
- **CLAUDE.md rules** — Generated project documentation instructs Claude Code to never modify lockfiles without explicit approval.

### Layer 4: Vulnerability Scanning

| Scanner | Profiles | Integration |
|---------|----------|-------------|
| OSV-Scanner | `consulting-default`, `startup-github` | CI workflow + PreToolUse hook |
| Snyk | `enterprise` | CI workflow step (Snyk CLI image pinned by digest) |
| Socket.dev | All profiles | MCP server for behavioral analysis |

Generated CI steps are pinned to immutable references: actions to full commit SHAs and container images to `sha256` digests. The Snyk step runs the `snyk/snyk` image directly, pinned by digest, rather than through `snyk/actions`. That action runs the image by its mutable `node` tag, so pinning the action's SHA would not pin the code that runs with `SNYK_TOKEN`. qsdev's own CI checks that every pinned SHA and digest still resolves upstream, and fails once an image pin is more than 90 days behind its tag.

The package-guard hook (Layer 5) queries OSV.dev in real time when the AI agent requests a package install — blocking packages with known vulnerabilities before they enter the dependency tree.

### Layer 5: PreToolUse Hooks (package-guard)

The `package-guard` hook runs as a Claude Code PreToolUse interceptor on every `Bash` tool invocation:

1. **Pattern matching** — Detects install commands across all supported package managers.
2. **OSV.dev vulnerability check** — Queries the OSV API for known vulnerabilities in the requested package.
3. **Age-gate enforcement** — Rejects packages published less than the configured minimum release age. The age is that of the exact version the manager would install: for NuGet it comes from nuget.org's registration API, which also tells the guard which versions are unlisted (never picked for a latest or floating version; a pinned unlisted version is aged by its catalog `created` time, since nuget.org stamps unlisted versions as published in 1900). `--prerelease` is checked as the newest pre-release it selects.
4. **Allow or block** — Permits the install (with approval) if the package passes both checks; blocks it otherwise with an explanation.

Package install commands live in the `ask` list (not `deny`), meaning the hook gets a chance to validate them before the user sees a prompt. Only bypass vectors that cannot be safely validated remain in `deny`. For .NET that means adding a package reference is ask-gated and guard-checked, while commands that download and run a NuGet package or install outside the project's references (`dotnet tool install/update/exec/run`, `dnx`, `dotnet dnx`, `dotnet new install`/`-i`, `dotnet package update`, the `nuget` CLI) are denied by the .NET ecosystem module.

### Layer 6: Nix Hardening

The generated `devenv.yaml` enforces:

- **`impure: false`** — Prevents the build from accessing anything outside the Nix store.
- **`allow_unfree: false`** — Blocks unfree packages unless explicitly listed.
- **`allow_broken: false`** — Blocks broken packages.
- **`clean.enabled: true`** — Strips the shell environment on entry, keeping only a minimal allowlist (TERM, HOME, USER, SSH_AUTH_SOCK, etc.).

The generated `devenv.nix` additionally:

- **Unsets 38 credential-bearing variables** — AWS, GCP, Azure, GitHub, GitLab, Docker, database, secrets management, and generic API keys.
- **Sets `DEVENV_SECURITY_HARDENED=true`** — A sentinel flag verified by `devenv test`.
- **Installs security pre-commit hooks** — ripsecrets, check-added-large-files, no-commit-to-branch, check-merge-conflict, shellcheck, statix.
- **Installs custom hooks** — lock-file-audit and nix-secrets-check (detects hardcoded credentials in `.nix` files).

### Layer 7: SAST (Semgrep)

Semgrep is an AlwaysOn tool that provides static analysis during development:

- Detects dangerous code patterns (command injection, path traversal, unsafe deserialization).
- The generated `qsdev-security-scan` devenv task runs `semgrep` with the registry rule packs of the detected ecosystems (`p/golang`, `p/python`, `p/owasp-top-ten` and so on), plus the project's own rules in `.semgrep/` when that directory exists. It runs with `--metrics=off --error`, so a finding fails the task.
- The generated `.semgrepignore` excludes build output, caches, fixtures, vendored dependencies and virtual environments from the scan.
- The posture SAST layer counts as enabled only when semgrep is enabled and the `qsdev-security-scan` task in `devenv.nix` runs it. If the task does not run semgrep, the layer is partial.

**OpenGrep** (opt-in via `qsdev enable opengrep`) adds 96 taint-focused rules targeting injection flaws, deserialization, and authentication bypasses across 7 frameworks: Next.js, FastAPI, Gin, NestJS, SvelteKit, Prisma, and Drizzle. OpenGrep is not in nixpkgs, so enabling it writes a pinned derivation to `.opengrep/nix/default.nix`, which the devenv.nix package list imports. The derivation fetches the prebuilt release binary for the host platform (x86_64/aarch64 Linux and macOS) and checks it against the release asset's SHA-256 digest. The rule library goes to `.opengrep/rules/core/`, and the generated `security-scan` devenv task runs `opengrep scan --config .opengrep/rules/core --error`, so a finding fails the task. OpenGrep has no project config file, so none is generated. `qsdev disable opengrep` removes the derivation, the rule library and the task step.

### Layer 8: Secrets Scanning (ripsecrets + gitleaks)

Two complementary scanners ensure credentials never reach the repository:

| Tool | Stage | Scope |
|------|-------|-------|
| ripsecrets | Pre-commit hook | Fast, low-false-positive scan on staged files |
| gitleaks | AlwaysOn tool + CI | Full-repo scan including git history |

Both are configured automatically during `qsdev init`. No manual setup required.

### Layer 9: Container Security

When a Dockerfile or Containerfile is detected, qsdev generates runtime-aware security configs for both Docker and Podman:

- **Hadolint configuration** — Linting rules for Dockerfile best practices (no `latest` tags, no root user, etc.) and a DL3026 trusted-registry allowlist. The default allowlist is `docker.io`, `gcr.io` and `ghcr.io`. Hadolint matches registries only, not namespaces, so trusting `docker.io` admits every Docker Hub account, including typosquats. Narrow the list to the registries your organization actually uses in the `qsdev init` wizard's "Trusted container registries" field (comma-separated, stored as the `trusted_registries` extra).
- **Syft SBOM generation + Grype vulnerability scanning** — CI workflow steps that produce a software bill of materials and scan built images for OS and library vulnerabilities. (Trivy was removed after the March 2026 supply chain compromise.)
- **Image signing policy** — `.cosign/policy.yaml` is a Sigstore [policy-controller](https://docs.sigstore.dev/policy-controller/overview/) `ClusterImagePolicy` derived from the project's git origin remote. For a `github.com/<owner>/<repo>` remote it applies only to images under `ghcr.io/<owner>/<repo>` (and paths below it) and admits them only when they are keyless-signed by a GitHub Actions workflow in that repository (exact issuer `https://token.actions.githubusercontent.com`, anchored case-insensitive subject regexp). It does not cover third-party base images or other registries; the policy-controller's `no-match-policy` decides those, so add a policy for each source you trust. If you publish elsewhere, edit `spec.images`. When there is no origin remote, or the remote is not on `github.com` (GitLab, GitHub Enterprise Server, self-hosted), qsdev cannot know the registry path or signing identity, so it writes a template in which every line is commented out and every value is a `TODO` placeholder: applying it as-is creates nothing. The file uses the `manual-merge` strategy, so once you edit it, an update writes the regenerated version beside it (`.cosign/policy.yaml.new`) instead of overwriting it; `qsdev disable container-security` removes that sidecar along with the policy.
- **Base image pinning (not enforced)** — qsdev does not generate Dockerfiles and does not check that base images are pinned. Hadolint rejects `latest` and untagged images (DL3006/DL3007) but accepts any mutable tag such as `node:20`. Pin each `FROM` to a digest (`image:tag@sha256:...`) yourself, and let Dependabot or Renovate update the digests.
- **Runtime-aware deny rules** — In Podman mode, Docker socket mount commands are blocked to prevent accidental privilege escalation.

### Layer 10: License Compliance

License compliance is opt-in (`qsdev enable license-compliance`). It uses [ScanCode Toolkit](https://github.com/aboutcode-org/scancode-toolkit), installed from nixpkgs (`python3Packages.scancode-toolkit`):

- `.scancode.yml` is a ScanCode license policy (the `license_policies` format that `scancode --license-policy` reads). Each entry names a ScanCode license key, its SPDX identifier, a label and a `compliance_alert`:
  - **Approved** (no alert): MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, 0BSD, Unlicense, CC0-1.0.
  - **Restricted** (`warning`): LGPL-2.0/2.1/3.0, MPL-1.1, MPL-2.0, EPL-1.0, EPL-2.0, CDDL-1.0, CDDL-1.1 and Artistic-2.0. The LGPL versions are listed in both their `-only` and `-or-later` forms.
  - **Prohibited** (`error`): GPL-1.0/2.0/3.0 and AGPL-1.0/3.0 in both their `-only` and `-or-later` forms, plus SSPL-1.0 and BUSL-1.1. ScanCode maps the deprecated SPDX identifiers (`GPL-2.0`, `GPL-2.0+`, `AGPL-3.0` and so on) onto the same license keys, so they are covered too.
- The generated `qsdev-security-scan` devenv task runs `scancode --license --license-policy .scancode.yml` over the project and pipes the JSON result through a `jq` check. The task fails when any scanned file has a license whose `compliance_alert` is `error`, and lists restricted licenses for manual review without failing. ScanCode itself only annotates files with the matching policy entries and never fails a scan, which is why the task needs the `jq` check. The check also fails when ScanCode reports an error in its scan headers, such as a policy file it rejects for a duplicate `license_key`; ScanCode then applies no policy but still exits 0.
- The scan covers the dependency directories (`node_modules/`, `vendor/`, `third_party/`, virtual environments), because the licenses of your dependencies are stored there. It skips build output, caches, framework output, `.devenv/`, test fixtures and the policy files themselves. Scanning a large `node_modules/` tree takes a while. Dependencies stored outside the project, such as the Go module cache or the Cargo registry, are not scanned unless they are vendored.
- ScanCode matches every license key in a detected expression, so a dual-licensed file such as `MIT OR GPL-2.0-or-later` is reported as prohibited even though the MIT option is allowed.
- `.license-exceptions.yml` is a record of approved exceptions for reviewers. The scan does not read it.
- The posture license-compliance layer counts as enabled only when the tool is enabled and the `qsdev-security-scan` task in `devenv.nix` runs the policy scan. If the task does not run it, the layer is partial.

### Layer 11: Cloud Credential Isolation

When AWS, GCP, or Azure project files are detected, qsdev generates a 3-layer credential isolation configuration:

| Layer | Mechanism | Effect |
|-------|-----------|--------|
| Environment separation | Per-project credential variables in devenv.nix | Prevents ambient credential access across projects |
| Credential file masking | Read-deny rules for credential file paths | Blocks agent access to stored credentials |
| Agent deny rules | Claude Code deny rules for auth CLI commands | Prevents credential refresh or modification |

Detection triggers:

| Provider | Indicators |
|----------|-----------|
| AWS | `cdk.json`, `samconfig.toml`, SAM templates, Terraform `aws` provider, `serverless.yml` |
| GCP | `cloudbuild.yaml`, `firebase.json`, `app.yaml`, Terraform `google`/`google-beta` provider |
| Azure | `azure-pipelines.yml`, `.bicep` files, `azure.yaml`, Terraform `azurerm` provider |

Cloud CLIs remain available for read-only operations. Authentication and credential modification commands are denied.

`qsdev devenv doctor` (under **Cloud Credential Isolation**) and `qsdev check` verify all three layers for each cloud provider in `.qsdev.yaml`'s `languages`. The check is static and runs no cloud CLI:

| Layer | Verified from | `qsdev check` when missing |
|-------|---------------|----------------------------|
| Environment separation | `AWS_PROFILE`, `CLOUDSDK_ACTIVE_CONFIG_NAME` or `ARM_SUBSCRIPTION_ID` declared in `devenv.nix` or `devenv.local.nix` (as `env.NAME` or inside `env = { ... }`) | warning, low severity |
| Credential file masking | `Read(...)` rules in `permissions.deny`, or `sandbox.filesystem.denyRead`, in `.claude/settings.json` | failure, high severity |
| Agent deny rules | The provider's credential-command `Bash(...)` rules in `permissions.deny` | failure, high severity |

An empty value, a `<description>` template left from the generated guidance, or a value containing a placeholder marker such as `PLACEHOLDER`, `CHANGEME` or `YOUR_` counts as unset. A variable is only a warning because its value is account-specific and often lives in the untracked `devenv.local.nix`, which a CI checkout does not have. The masking and deny layers are generated by qsdev, so a missing path or rule means `.claude/settings.json` was edited or predates the current rule set: run `qsdev init --update` to restore it. Doctor reports a provider as `isolated`, `degraded` (only the environment layer is missing) or `misconfigured` (a generated layer is missing). The masking and deny layers guard the Claude Code agent, so a project set up without Claude Code (`qsdev init --devenv-only`) that has no `.claude/settings.json` is judged on environment separation only.

Doctor also runs the health checks each configured ecosystem module contributes, under **Ecosystem Checks**. For the cloud modules these are an environment-variable check (`AWS_PROFILE`, `CLOUDSDK_ACTIVE_CONFIG_NAME`, `ARM_SUBSCRIPTION_ID`) and a login check (`aws sts get-caller-identity`, `gcloud auth print-access-token`, `az account show`). They are static too:

- An environment check passes (`[OK]`) when `devenv.nix`/`devenv.local.nix` declare a real value for the variable, so it also passes when doctor runs outside the devenv shell. When neither file declares it, doctor's own environment is checked instead. A declaration is judged first because the devenv shell exports it over the surrounding shell's value. The same placeholder rules apply: an empty, `<description>` or `PLACEHOLDER` value is reported as a warning, never as healthy.
- A login check never runs the cloud CLI. Running it would contact the provider, and `gcloud auth print-access-token` prints a live token to stdout. Doctor only looks up the CLI on `PATH`. It shows `-` with the command for you to run yourself, or a warning when the CLI is not on `PATH` (the devenv shell provides it).

These checks are advisory. They appear in `--json` output under `module_checks` and do not change the `--check` exit code.

### Layer 12: Policy Engine

YAML-based security policies define fine-grained rules evaluated at tool invocation time. Each rule specifies conditions, an action, a severity level, and a bypass tier.

**Condition types** (10): `tool_match`, `path_glob`, `regex_match`, `command_match`, `file_existence`, `file_type`, `denied_path_check`, `semantic`, `all`, `any`, `not`.

**Actions** (4): `block` (exit 2), `warn` (exit 0 + finding), `audit` (exit 0 + monitored finding), `prompt` (interactive approval, falls back to block without a terminal).

**Bypass tiers** (3):
- `enforce_always` — Cannot be bypassed. Used for self-protection rules.
- `session` — Can be bypassed with `qsdev session allow <rule-id>` (interactive confirmation required; applies machine-wide until `qsdev session clear`).
- `command` — Can be bypassed per-invocation.

Policy evaluation runs in under 50 microseconds per rule. Output is available in human-readable, JSON, and SARIF 2.1.0 formats.

```bash
qsdev policy check              # Human-readable posture summary
qsdev policy check --sarif      # SARIF 2.1.0 output for CI integration
qsdev policy list               # List all active rules
qsdev policy show <rule-id>     # Inspect a specific rule
```

### Layer 13: Package and MCP Risk Scoring

**Package risk scoring** evaluates packages across 28 probes in 6 weighted categories:

| Category | Weight | What it measures |
|----------|-------:|-----------------|
| Vulnerability | 0.35 | CVE counts by severity, EPSS score, KEV listing, reachability |
| Behavioral | 0.20 | Typosquatting, install scripts, network at install, obfuscation |
| Publication | 0.15 | Package age, release frequency, changelog presence |
| Maintainer | 0.12 | Maintainer count, publisher switching, 2FA status |
| Provenance | 0.10 | SLSA level, Sigstore signature, npm provenance, checksum |
| Popularity | 0.08 | Download count, dependent count |

Probes produce a weighted score mapped to a letter grade (A through F). Grade ceilings prevent strong popularity or provenance from masking critical vulnerability findings. Unavailable signals are excluded from the denominator rather than penalized.

**MCP trust scoring** evaluates MCP servers across 9 probes in 3 categories:

| Category | Weight | Probes |
|----------|-------:|--------|
| Content origin | 0.45 | Source verification, npm registry checks, content signing |
| Installation and update | 0.30 | Update mechanism safety, pinned version, offline capability |
| Vulnerability and attestation | 0.25 | Known vulnerability databases, user attestation |

Trust scores feed a 3-tier model (high / medium / low trust) and drive confused deputy mitigation via cross-tool deny rule projection.

**MCP configuration in doctor.** `qsdev devenv doctor` lists each server in the project's `.mcp.json` under **MCP Servers** and validates its entry statically. It never starts a server, because `.mcp.json` is repository content and may name any command (a server launched through `npx` or `uvx` would also download a package). Doctor checks that each stdio command is on `PATH`, that each remote URL is an absolute `https://` URL (plain `http://` only for localhost), and that the environment variables a server needs are set: those its catalog definition requires and those its command, arguments, URL, `env` or `headers` reference as `${VAR}` without a `:-default`. A server is `ok`, `degraded` (only an unset variable) or `misconfigured` (a missing command or a bad URL). The section appears in `--json` output under `mcp_servers` and does not change the `--check` exit code. To check that a server actually answers, run `qsdev mcp status`, which starts only servers that match a trusted definition.

**Pinned MCP launches.** Claude Code starts `.mcp.json` servers itself, outside the package guard, the Bash deny rules and lockfile pinning. qsdev therefore pins every catalog server that a package launcher (`npx`, `uvx`, ...) fetches at session start to an exact release, and generation refuses any launcher spec, catalog or configured, that does not name one. `qsdev mcp install` installs exactly the pinned release under the release-age cutoff, and while the project records that install, `.mcp.json` runs the installed executable, so nothing is fetched at session start. See [`.mcp.json`](configuration-reference.md#mcpjson).

### Layer 14: Agent Self-Protection

Self-protection runs as the first PreToolUse hook, before all other hooks. It blocks the AI agent from tampering with its own guardrails, security configuration, or audit trail.

**18 Tier 1 enforce-always rules** across 5 categories:

| Category | Rule IDs | What it protects |
|----------|----------|------------------|
| Config protection | SP-001–SP-008 | Config file writes/reads/deletes, symlinks, path traversal, /proc reads, copy/redirect, env var manipulation |
| MCP integrity | MCP-001, MCP-002, MCP-005 | Tool description injection, cross-tool protected path access, server config tampering |
| Binary integrity | INT-001 | Modification of security binaries in `.qsdev/bin/` |
| Bypass prevention | SP-011–SP-014 | Bypass variable exports, bypass commands, audit trail writes, CLI security control commands |
| Process protection | SP-009–SP-010 | Process management targeting qsdev/claude, hook script modification |

All 18 rules use deny-override combining: if any rule denies, the tool call is blocked. All rules are evaluated on every call; multiple denials are collected and reported.

**Evasion detection** runs alongside the rule set and catches:
- Base64-to-shell pipelines (`base64 -d | bash`)
- Printf hex-escape-to-shell pipelines
- Eval with variable expansion
- Hard links to protected paths
- `/dev/fd/` and `/proc/self/fd/` file descriptor tricks
- `/proc/self/root/` and `/proc/PID/root/` traversal

**Git code-execution check (GIT-001)** parses every git invocation, including one inside `sh -c` or `eval`, and blocks the forms that run a program no permission rule or hook sees, or that skip the repository's hooks: per-invocation config (`-c`, `--config-env`, `GIT_CONFIG_*` variables), `--exec-path=`, `GIT_EXTERNAL_DIFF`, `git config`, `--no-verify` and its abbreviations, a clustered `-n` on commit or am, and `--output`. The permission deny rules cover the plain spellings; this check covers what a prefix glob cannot express, and leaves commit messages that mention these options alone.

**Path canonicalization** resolves all file paths through `canon.Canonicalize` before rule evaluation, preventing relative-path and symlink-based evasion.

## Project Security Floor

The committed `.qsdev.yaml` declares a security floor (`security.level`,
raised by `client.security_level`) and an optional client MCP policy
(`client.blocked_mcp_servers` / `allowed_mcp_servers`). `qsdev init` resolves
it, together with the developer's `.qsdev.local.yaml`, through a single
resolver in create, join and update, and generation never goes below it:

- the compliance level and the hooks it requires are raised to the floor,
  and a security switch it mandates cannot be disabled locally;
- `.qsdev.local.yaml` may only add to or tighten the committed
  configuration: a looser permission level (catalog `strictness` ranks
  `minimal` > `standard` > `permissive`; unranked presets are not
  comparable and fail closed), a `tools.disabled` entry, `tools.config`, a
  changed `claude_code.enabled`, package manager or service option, and any
  floor violation are ignored and reported. Its additions reach generation
  only, never the committed `.qsdev.yaml`;
- a forbidden MCP server is dropped from `.mcp.json` whatever requested it,
  including entries already present in a committed or hand-edited file.

The resolver has no organization-defaults layer, so a project without a
`security` block is not silently raised to built-in defaults. See the
[configuration reference](configuration-reference.md#security-floor-client-policy-and-local-overrides).

## Hook Execution Isolation

The self-protection layer (Layer 14) runs as the first PreToolUse hook. It evaluates before package-guard, credential-scan, and all other hooks. Guardrail-tampering attempts are blocked before any other hook logic executes.

Hooks run inside a sandboxed environment that restricts filesystem access, network, and syscalls. The sandbox degrades gracefully based on available kernel features:

| Tier | Isolation | Requires |
|------|-----------|----------|
| Full | Bubblewrap + Landlock + seccomp-BPF + cgroups v2 | Linux 5.13+, bwrap |
| BwrapWithoutLandlock | Bubblewrap + seccomp-BPF | Linux, bwrap |
| BwrapWithoutSeccomp | Bubblewrap namespaces only | Linux, bwrap |
| SystemdRun | systemd-run resource limits | systemd |
| Unsandboxed | No isolation (macOS, minimal Linux) | — |

Run `qsdev sandbox status` to see the active tier on your system. Five hook category profiles (linter, formatter, network-linter, generator, test-runner) control which resources each hook type can access.

## Security Spectrum Positioning

Development security exists on a spectrum from zero configuration to full lockdown. Each increment of security adds corresponding friction. qsdev is deliberately positioned at the optimal inflection point — the highest protection achievable before productivity costs become structural.

### The Eight-Tier Framework

| Tier | Name | Example Controls | Ongoing DX Cost |
|------|------|-----------------|----------------|
| 0 | No Security | Trust everything; no lockfiles; credentials in source | None |
| 1 | Basic Hygiene | Lockfiles committed; .gitignore; SSH keys | Negligible |
| 2 | Dependency Awareness | Vulnerability scanning; Dependabot/Renovate; SBOM | Minutes/week |
| 3 | Active Defense | Age-gating; install-script blocking; secrets scanning; SAST | 3–10s/commit |
| 4 | Environment Hardening | Nix hermetics; credential scrubbing; build sandboxing | None after setup |
| 5 | Agent-Aware Security | PreToolUse hooks; deny rules; MCP gating; self-protection | None after setup |
| 6 | Process Isolation | VM per project; ephemeral environments; network partitioning | 10–20% permanent |
| 7 | Full Lockdown | Air-gapped; HSMs; mandatory multi-person approval | 20–40% permanent |

`qsdev init` delivers **Tiers 2–5** in under two minutes.

### Why Tiers 4–5 (Not Higher)

Three constraints converge at qsdev's position:

**1. Threat model alignment.** The realistic threat surface for development teams — supply chain attacks (454K malicious packages/year), credential theft (28.6M secrets leaked in 2025), AI agent exploitation (73% vulnerable to prompt injection) — is fully addressed by Tiers 3–5. Tier 6–7 defenses protect against nation-state EM surveillance, physical infiltration, and classified-data handling — threats outside the model for commercial software teams.

**2. DX cost cliff.** Tiers 0–5 have manageable or zero ongoing costs (qsdev eliminates setup cost through generation). At Tier 6, costs become *structural* — VM boundaries impose 5–15 minute cold starts, 3–10x slower incremental builds, and eliminate GPU passthrough. These costs cannot be removed by better tooling because they are inherent to the isolation model.

**3. Diminishing marginal returns.** Each tier from 0→5 provides substantial, measurable security improvement (92% malware catch from age-gating; complete elimination of install-script attacks; fail-closed agent policy). Tier 5→6 adds negligible protection against realistic threats while imposing catastrophic productivity loss — equivalent to losing 1–4 developers on a 10-person team.

### The Configuration Cost Innovation

Traditional Tier 4–5 setup takes 2–5 days of a security engineer's time: researching per-ecosystem best practices, writing Nix configurations, crafting deny rules, implementing hooks, testing interactions. Most teams never attempt it — not because they disagree with the security value, but because the configuration cost is prohibitive.

qsdev eliminates the configuration barrier by generating correct, ecosystem-specific security configurations from a single command. The ongoing cost after generation is 3–10 seconds per commit (pre-commit hooks) — indistinguishable from a project without security hardening.

### Quantified Effectiveness

| Defense | Metric | Source |
|---------|--------|--------|
| Age-gating (24–72h) | 92% of PyPI malware caught within 24h | PyPI security reports |
| Install-script blocking | Eliminates #1 exploited npm attack vector | npm security advisories |
| Secrets scanning (ripsecrets) | 0.32s full-repo scan (95x faster than trufflehog) | Benchmark on Sentry repo |
| MCP datamarking + trust scoring | Attack success rate reduced from ~60% to <2% | MCP security research |
| Nix content-addressing | Every artifact verified by SHA-256 hash | Nix store guarantees |
| Policy evaluation | <50 microseconds per rule | Internal benchmarks |

### Comparison to Alternatives

| Tool/Approach | Tier Coverage | Gap vs. qsdev |
|---------------|---------------|---------------|
| npm audit / Snyk | 2–3 (partial) | No environment hardening, no agent security, no multi-ecosystem |
| Socket.dev | 3 (behavioral only) | No age-gating, no isolation, no agent controls |
| Dev Containers | 4 (isolation only) | No supply chain hardening, no agent awareness |
| Raw Nix | 4 (reproducibility only) | No security configuration, no ecosystem modules |
| Manual Claude Code hooks | 5 (partial) | No supply chain integration, no self-protection harness |

qsdev is the only tool spanning Tiers 2–5 across 30 ecosystems with integrated AI agent security.

## Permission Model

qsdev generates Claude Code permissions in `.claude/settings.json` using a two-tier model: **ask** rules (hook-gated, user-prompted) and **deny** rules (unconditionally blocked).

### Ask Rules (~60 rules)

Package install commands are placed in the `ask` list. When Claude Code attempts one, the PreToolUse package-guard hook validates the request (OSV check + age-gate) before the user sees an approval prompt. This allows legitimate installs while blocking dangerous ones.

Ask rules cover:

| Category | Examples |
|----------|----------|
| JS Package Managers | `npm install`, `yarn add`, `pnpm add`, `bun add` |
| Python | `pip install`, `uv add`, `pipx install` |
| Rust | `cargo add`, `cargo install` |
| Go | `go get`, `go install` |
| Ruby | `gem install`, `bundle add` |
| PHP | `composer require` |
| .NET | `dotnet add package`, `dotnet add <PROJECT> package`, `dotnet package add` |
| System | `nix profile install`, `apt install`, `brew install` |

### Deny Rules (~90 rules)

Commands that represent bypass vectors — ways to circumvent the hook-gating — remain unconditionally denied. These cannot be validated safely, so they are blocked outright.

| Category | Examples | Count |
|----------|----------|------:|
| Pipe-to-Shell | `curl \| bash`, `wget \| sh` | ~8 |
| Shell Wrapping | `bash -c *npm install*`, `sh -c *pip install*` | ~14 |
| Subprocess Escape | `python -c *subprocess*`, `node -e *child_process*` | ~9 |
| eval/xargs | `eval *npm install*`, `xargs cargo install` | ~7 |
| env/command Prefix | `env npm install`, `command pip install` | ~10 |
| sudo Prefix | `sudo npm install`, `sudo apt install` | ~8 |
| Remote Package Execution | `npx <package>`, `pnpm dlx`, `yarn dlx`, `bunx`, `npm exec` | ~8 |
| Destructive Ops | `git push --force`, `rm -rf /`, `Read(./.env)` | ~6 |
| Nix Bypass | `nix-env -i`, `cachix use` | ~8 |
| Uncategorized | Per-ecosystem edge cases | ~14 |

### Permission Presets

| Preset | Philosophy |
|--------|-----------|
| **minimal** | Read-only by default. Only `Read(*)` and basic build/test commands are allowed. Every write or edit requires approval. |
| **standard** | Productive development. `Read`, `Edit`, `Write`, `git`, build/test/lint, and Nix dev commands are allowed. Package installs are hook-gated (ask). Bypass vectors are denied. |
| **permissive** | Standard plus `make` and `docker` commands. For teams using Makefiles or Docker-based workflows. |
| **supply-chain-only** | Minimal permissions focused exclusively on supply chain defense. Deny rules and package-guard hook without broader development tooling permissions. |
| **custom** | Only explicitly configured allow/deny patterns. Full manual control. |

Select a preset during `qsdev init` or set it in `.qsdev.yaml`:

```yaml
claude:
  permission_preset: standard
```

### AlwaysOn Tools

The following tools are installed and available to Claude Code without per-invocation approval:

| Tool | Purpose |
|------|---------|
| semgrep | SAST scanning |
| gitleaks | Secrets detection |
| semble | Semantic code search |
| version-sentinel | Dependency version tracking |
| context7 MCP | Documentation context |
| github MCP | GitHub API access |
| socket MCP | Dependency behavioral analysis |
| agent-postmortem | Session analysis and learning |
| package-guard | PreToolUse hook for install validation |

## CI Security Integration

### Generated Workflows

Infrastructure profiles (tier `standard` and above) generate
`.github/workflows/security-scan.yml` when the profile configures a
vulnerability scanner or CI runner protection. It has two jobs:

- **`security-scan`**
  - **harden-runner** (profiles with `ci_protection: harden-runner`) — audits
    network egress from the runner.
  - **Validate lock files** — fails when a tracked manifest has no committed
    lock file, for every manifest/lock-file pair in the ecosystem catalog.
  - **OSV-Scanner / Snyk / Grype** (profile-dependent) — scans dependencies
    for known vulnerabilities.
- **`ecosystem-ci`** (when the project's languages contribute CI commands) —
  runs every selected ecosystem module's `CICommands` in the project's devenv
  shell (`cachix/install-nix-action`, then `devenv shell`), so CI uses the
  toolchains `devenv.nix` pins. Steps are grouped by phase: all **install**
  steps first (lock-file enforcing installs such as `npm ci --ignore-scripts`,
  `pnpm install --frozen-lockfile`, `cargo build --locked`,
  `dotnet restore --locked-mode`, `uv sync --locked`,
  `terraform init -lockfile=readonly`), then **test** steps, then **scan**
  steps (audits such as `cargo audit`, `govulncheck`, `pip-audit`,
  `npm audit --audit-level=moderate`, Grype on the built container image).
  The `.npmrc` `audit-level` setting only sets `npm audit`'s exit code — `npm ci`
  and `npm install` never fail on audit results — so the `npm audit` step is
  what makes moderate-or-higher advisories fail CI for npm projects.
  The modules add these audit tools (`cargo-audit`, `pip-audit`,
  `bundler-audit`, `syft`, `grype`, `govulncheck`) to the
  `devenv.nix` packages, so they are on the shell's PATH locally and in CI.
  A drifted or missing lock entry therefore fails CI before anything builds.
  Other lock-enforcing installs include `stack build --lock-file=error-on-write`,
  `swift package resolve --force-resolved-versions`, `helm dependency build`
  (refused when a chart declares dependencies without a `Chart.lock`), sbt's
  `dependencyLockCheck` against `build.sbt.lock`, and `renv::restore()` followed
  by a check that fails unless `renv::status()` reports the project in sync.
  Every command fails on its own, whatever shell options run it: pipelines set
  `pipefail` (`helm template | kubeconform`), loops fail when any item fails
  (`bash -n` on each `*.sh` file, `luarocks install --only-deps` on each
  rockspec), and scanners that only report are made to fail (PSScriptAnalyzer
  error findings and parse errors; sbt-dependency-check at CVSS 7 and above).
  Tools that are not nixpkgs packages are provisioned by the job itself: the sbt security
  plugins through `sbt --addPluginSbtFile`, and a pinned PSScriptAnalyzer from
  PSGallery. Commands that only apply to one package manager or project shape
  are emitted only for it: the sbt tasks for sbt builds (not Mill), the renv
  steps for renv projects (`renv.lock`), the LuaRocks install for rockspec
  projects, and the flake checks for flake projects (`flake.nix`, recorded by
  detection as the Nix module's `flake=true` extra).
  Commands come from each module configured with the
  language's package manager and extras from `.qsdev.yaml`; a command
  containing a GitHub Actions expression (`${{`) is refused at generation
  time, since GitHub would evaluate it before the shell ran the step.

### Generated-File Drift in CI

`qsdev check` fails a CI run when a machine-owned generated file, such as
`.claude/hooks/package-guard.py`, a rule, a skill or a generated workflow, was
edited or deleted. The local generation state (`.devinit/`) is gitignored, so
CI verifies the files against the committed `.qsdev-generated.sha256` manifest
of their SHA-256 digests, which every command that regenerates files rewrites.
A project with `.qsdev.yaml` but no usable manifest also fails, rather than
skipping the check. The manifest is only as trustworthy as review of the
changes to it: a pull request that edits a hook and updates its digest passes
this check, and shows both changes in the diff. See
[`.qsdev-generated.sha256`](configuration-reference.md#qsdev-generatedsha256).

### Branch Name Hygiene

The always-on `branch-naming` tool installs a pre-push hook that checks the
current branch against `git.branch_pattern` in `.qsdev.yaml`. Its default
accepts any portable ASCII branch name but rejects shell metacharacters, a
leading `-` and non-ASCII characters, the branch names that turn into command
injection where CI interpolates `${{ github.head_ref }}` unquoted into a
script. It is a local hook, so `git push --no-verify` (or a branch created on
the forge) skips it: workflows must still pass branch names through
environment variables rather than inline expressions. See
[Git settings](configuration-reference.md#git-settings).

### Custom Conformance Policy

A committed `.qsdev-policy.yaml` adds project-specific requirements to the
built-in baseline and enhanced conformance levels. `qsdev status` gates its
exit code on them at `--audit-level high` and stricter, and `qsdev check`
reports each as a high-severity check. The policy fails closed: a requirement
on dependency vulnerability counts passes only after a conclusive fresh scan
(`--scan`), so zero counts from a skipped or failed scan never read as clean,
and a malformed policy file fails instead of being ignored. See
[`.qsdev-policy.yaml`](configuration-reference.md#qsdev-policyyaml).

### Generated Update Configuration

- **Renovate** (`consulting-default`, `enterprise`) — `renovate.json` with `minimumReleaseAge`, `automergeType: "pr"` for patches (enterprise), and lockfile maintenance.
- **Dependabot** (`startup-github`) — `.github/dependabot.yml` with configured update schedules.

### Registry Proxy and Binary Caches

An explicitly selected infrastructure profile routes package installs through
the organization's pull-through registry proxy and adds its Nix binary cache.
The built-in profiles carry no endpoints: qsdev refuses to generate until the
real ones are configured, and rejects documentation placeholders
(`example.com` hosts, the `myorg` Cachix cache, an all-zero public key), so
selecting a profile never silently leaves installs on the public registries.
Registry and cache credentials stay in the environment and are never written
into generated files. See
[Infrastructure settings](configuration-reference.md#infrastructure-settings).

### SBOM Generation

- **Syft** (all profiles) — Generates software bill of materials.
- **Cosign** (`enterprise` only) — Signs the SBOM for supply chain attestation.

## Self-Update Verification

`qsdev update` (and the hidden `self-update` alias) installs a new binary only
after authenticating the release:

1. `checksums.txt` must carry a Sigstore bundle (`checksums.txt.sigstore.json`)
   produced by the release workflow. qsdev verifies it **in-process** with
   [sigstore-go](https://github.com/sigstore/sigstore-go); no external `cosign`
   is run, so nothing on `PATH` (a devenv profile, a direnv-added directory, a
   user shim) can vouch for a release.
2. The only trust anchor is the Sigstore public-good trusted root (Fulcio CA,
   Rekor and CT log keys, timestamp authority) embedded in the binary at
   build time. It is not refreshed from the network or read from
   `~/.sigstore` at update time. Maintainers refresh it with
   `go generate ./internal/selfupdate`, which fetches it through an
   authenticated TUF update.
3. The signing certificate must match the **exact** identity of this
   release's workflow run:
   `https://github.com/<owner>/<repo>/.github/workflows/release.yml@refs/tags/<tag>`,
   issued by `https://token.actions.githubusercontent.com`. A signature from
   another workflow, another tag or another repository is rejected.
4. The bundle must include a Rekor inclusion proof, an embedded SCT and a
   trusted timestamp. The archive's SHA-256 is then checked against the
   authenticated `checksums.txt`.

Any verification failure aborts the update and leaves the current binary in
place. A release that publishes **no** bundle is refused by default; `--no-strict`
(on both `qsdev update` and `qsdev self-update`) installs such an unsigned
release for dev or self-built releases. `--no-strict` never overrides a bundle
that is present but fails verification.

If Sigstore rotates its keys after a qsdev release was built, that binary may
be unable to verify newer releases; install the newer release with the
[install script](../README.md#quick-start) or a package manager instead.

## Security Validation

The generated `devenv.nix` includes an `enterTest` script that verifies security controls:

```bash
devenv test
```

This validates:

1. Pre-commit hooks are installed in `.git/hooks/`
2. Credential variables (`AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, `VAULT_TOKEN`, `DATABASE_PASSWORD`) are not present in the environment
3. `ripsecrets` finds no secrets in tracked files
4. The `DEVENV_SECURITY_HARDENED` sentinel flag is set

Run `devenv test` in CI to continuously verify that security controls have not been disabled.

For a full security posture assessment including all 14 layers:

```bash
qsdev status
```

This outputs a score (0–100), letter grade, and per-layer breakdown showing which controls are active, degraded, or missing.

The score weighs defense coverage (40%), configuration health (30%) and
dependency health (30%). Dependency health is measured only by a vulnerability
scan (`qsdev status --scan`). Without one it is unknown, not clean, so:

- the dependency sub-score is reported as `unscanned` (`null` in JSON, with
  `dependencies.status: "unscanned"`) and the score is computed from defense
  and configuration alone, instead of counting the dependencies as a clean 100;
- the `no-critical-vulns` baseline check and the `no-high-vulns` enhanced check
  report `unknown`, so baseline and enhanced conformance (and the conformance
  badge) read `UNKNOWN` instead of `PASS`. An unknown result is never a pass,
  but it is not a failure either: the exit code without `--scan` is unchanged,
  and `qsdev status` warns that the vulnerability part of the audit level
  cannot fire.

A scan that fails is different: the dependency score is deducted for each
failed ecosystem and the vulnerability checks fail, so the exit gate fails
closed. A project with no detected dependency ecosystem has nothing to scan
and passes these checks.

## Known Limitations

### Hook Bypass Vectors

- **Aliases and functions** — Shell aliases (`alias npm='npm'`) or functions that wrap install commands are not caught by pattern-based rules.
- **Encoded commands** — Base64-to-shell and printf-hex-to-shell patterns are detected by the evasion layer (Layer 14). Novel encoding schemes not covered by the current pattern set remain a potential bypass.
- **Indirect execution via scripts** — Running a script file that internally calls install commands bypasses the PreToolUse hook.
- **New package managers** — Rules must be updated when new package managers emerge.

### Environment Hardening

- **Clean mode is advisory** — A determined developer can re-export stripped variables. The `devenv test` validation catches this retroactively but not in real-time.
- **Host Nix configuration** — `devenv.yaml` settings only apply within the devenv shell. The host system's `nix.conf` may allow impure builds.

### CI Limitations

- **Action pinning** — Generated workflows pin to major version tags (e.g., `@v4`), not commit SHAs. Consider pinning to SHAs for maximum supply chain security.
- **Self-hosted runners** — harden-runner's network egress controls are most effective on GitHub-hosted runners. Self-hosted runners may require additional network controls.

### Scope

- **Runtime dependencies** — The system hardens the development environment and CI pipeline. It does not scan or constrain runtime container images or deployed artifacts beyond build-time scanning.
- **Claude Code updates** — The bootstrap installs a pinned Claude Code release, but Claude Code's own auto-updater can later move that install to newer releases outside the pin and the age gate. Set `DISABLE_AUTOUPDATER=1` in the environment where the pin must hold.
- **Secret management** — Credential stripping prevents accidental exposure in the dev shell but does not replace a proper secret management system (Vault, AWS Secrets Manager, etc.).

### Policy Engine Limitations

- **Semantic conditions** — The `semantic` condition type is defined in the schema but not yet implemented. Rules using it always evaluate to false.
- **Prompt action** — Interactive prompting falls back to block when stdin is not a terminal (e.g., in CI). Full interactive approval is deferred to a future release.

## Further Reading

- [Configuration Reference](configuration-reference.md) — Every generated file, its purpose, and merge strategy
- [Team Onboarding](team-onboarding.md) — Infrastructure profiles, team policies, rollout playbook
- [Migration Guide](migration-guide.md) — Adding qsdev to existing projects with pre-existing configs
