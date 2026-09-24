# Configuration Reference

Ecosystem-specific files only appear when that language is selected.

| File | Purpose | Merge Strategy |
|------|---------|----------------|
| `devenv.nix` | Deterministic environment (languages, services, packages) | `manual-merge` |
| `devenv.yaml` | Environment inputs with security hardening | `overwrite` |
| `.envrc` | Automatic shell activation via direnv | `overwrite` |
| `.pre-commit-config.yaml` | Linting, formatting, lockfile enforcement | `overwrite` |
| `.claude/settings.json` | AI agent permissions (~90 deny + ~60 ask rules) | `three-way-merge` |
| `.claude/hooks/package-guard.py` | Real-time package install interception | `overwrite` |
| `.claude/skills/*.md` | 6 built-in + 11 qsdev ops skills | `library-managed` |
| `.claude/rules/*.md` | Language-specific convention rules | `library-managed` |
| `.claude/agents/semble-search.md` | Semantic code search agent | `library-managed` |
| `.claude/qsdev-reference.md` | CLI and workflow reference for Claude | `library-managed` |
| `.mcp.json` | MCP servers (4 default + tool/detection activated) | `three-way-merge` |
| `CLAUDE.md` | Project context for AI agents | `section-marker` |
| `.qsdev.yaml` | Declarative project configuration | `skip` |
| `.semgrepignore` | Paths Semgrep skips | `overwrite` |
| `.gitleaks.toml` | Gitleaks secret detection config | `overwrite` |
| `.gitignore` | Updated entries for qsdev-managed paths | `section-marker` |
| `.npmrc` / `pip.conf` / ... | Per-ecosystem security configs (only created if absent; an existing project config is never replaced) | `skip` |
| `pnpm-workspace.yaml` | pnpm security config with age-gating (pnpm projects; only created if absent) | `skip` |
| `.github/labeler.yml` | PR auto-labeling rules (only created if absent) | `skip` |
| `.github/workflows/labeler.yml` | Labeler GitHub Actions workflow (only created if absent) | `skip` |
| `.github/pull_request_template.md` | PR template with security checklist (only created if absent) | `skip` |
| `renovate.json` or `.github/dependabot.yml` | Automated dependency updates with age-gating | `overwrite` |
| `.github/workflows/security-scan.yml` | CI vulnerability scanning workflow | `overwrite` |
| `docs/security-overview.md` | Human-readable security posture documentation | `overwrite` |
| `.syft.yaml` | Syft SBOM scanner configuration | `overwrite` |
| `.grype.yaml` | Grype vulnerability scanner configuration | `overwrite` |
| `.hadolint.yaml` | Dockerfile linting rules (when container detected; only created if absent) | `skip` |
| `.cosign/policy.yaml` | Sigstore policy-controller image signing policy (when container-security enabled). Enforcing only for a `github.com` origin remote; otherwise a commented-out template. See [Layer 9](security-architecture.md#layer-9-container-security) | `manual-merge` |
| `.scancode.yml` | ScanCode license policy (when license-compliance enabled) | `overwrite` |
| `.license-exceptions.yml` | Record of approved license exceptions (when license-compliance enabled; only created if absent) | `skip` |
| `docker-compose.gateway.yaml` | MCP Gateway container for detected AI frameworks without native hook enforcement (written by `qsdev update`; skipped with `--skip-container`, which also leaves an existing file untouched; removed once no framework needs it) | `three-way-merge` |

## Merge Strategies

Each generated file has an assigned merge strategy that controls how it is handled when it already exists. Every command that writes generated files (`qsdev init`, `qsdev update`, `qsdev enable`, `qsdev disable` and the `qsdev claude` regeneration commands) applies the same strategy:

| Strategy | Behavior |
|----------|----------|
| `overwrite` | File is always regenerated. User modifications are lost on update. |
| `three-way-merge` | Three-way merge using stored base, current on-disk content, and new generated content. User changes are preserved when they do not conflict with template changes. YAML files merge key by key: a value you changed keeps your version (also when the template changed it too), keys you added or deleted stay that way, and lists are merged as a whole value. A YAML file that holds several documents, or whose merge would leave an alias without its anchor, is reported as an update failure and left unchanged. |
| `section-marker` | Tool-managed sections (delimited by markers) are replaced; user content outside markers is preserved. |
| `manual-merge` | A `.new` sidecar file is written alongside the existing file. The user is shown a diff and must merge manually. |
| `library-managed` | File is always updated to the latest version. Intended for files from a versioned skill/rule library where the tool owns the content. |
| `skip` | Skip-if-exists. The file is created only when it is absent. An existing file that qsdev did not generate, or that you edited after qsdev generated it, belongs to you: it is never overwritten, not even with `--force`, and it is not recorded as qsdev output. qsdev only refreshes its own unmodified output. To switch to the generated version, delete your file and re-run the command. |

When a file has been **deleted** by the user, it is not recreated unless `--force` is used. When a file is **unmodified** (matching stored hash), it is always safe to regenerate regardless of strategy.

A tracked file that the generators no longer produce is cleaned up by `qsdev update`: removed when unmodified, left in place and untracked when you edited or deleted it. This includes a file retired from a still-enabled tool, as long as that tool's files were regenerated in the same update. Files of an enabled tool that the update did not regenerate (for example, because the project uses `--devenv-only`) are left to `qsdev disable`.

`qsdev disable <tool>` removes only files qsdev generated and recorded. A tool file that qsdev did not generate, such as your own PR template that `qsdev enable` kept, is left in place even with `--force`.

---

## Project Configuration

### `.qsdev.yaml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `skip` |
| **Purpose** | Declarative project configuration shared across the team |

This is the source-of-truth file that teammates use to reproduce the same environment. When committed to version control, running `qsdev init --mode join` reads this file to produce an identical setup without re-running the wizard.

**Project root.** qsdev commands (`check`, `status`, `enable`, `update`, the `devenv` and `claudecode` commands, logs and bug reports) can be run from any subdirectory: they walk up from the working directory to the nearest directory holding `.qsdev.yaml` (a regular file), the `.devinit/` state directory, or a `.qsdev/` project data directory — except in your home directory, where `~/.qsdev/` is the per-user data directory (logs, cache, docs), not a project. Outside any project the working directory is used. `qsdev init` is the exception: it always initializes the directory it is run in.

Structure:

```yaml
# qsdev project configuration — generated by qsdev init.
version: 2
qsdev_version: ">= 0.5.0"   # semver constraint; init writes a lower bound on the generating release
tier: standard               # supply-chain-only | standard | full; always written by init
profile: go-web              # project-type profile (--profile)
infra_profile: enterprise    # infrastructure profile (--infra-profile)
languages:
  - name: go
    version: "1.24"
  - name: javascript
    package_manager: pnpm
services:
  - name: postgresql
    version: "16"
  - name: redis
security:
  level: enhanced
claude_code:
  enabled: true
  permission_level: standard
  skills: [deploy, review-pr, security-review-owasp]
  mcp_servers: [context7, github, socket, semble]
hooks:
  file_boundary:
    extra_read_paths: [/opt/android-sdk]   # read-only; see Hook settings
  tool_gates:
    denied: [WebFetch, "mcp__github__*"]   # see Hook settings
infrastructure:
  registry_proxy: https://repo.corp.internal/artifactory   # "none" opts out
  nix_cache: corp                                           # URL, or a Cachix cache name; "none" opts out
  nix_cache_public_key: "corp.cachix.org-1:<base64 key>"
  build_cache: sccache
  build_cache_url: ""                                       # e.g. a self-hosted Turborepo remote cache
git:
  branch_pattern: '^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9._-]+$'   # optional; see Git settings
mcp:
  disabled_tools: [qsdev_nix_run]   # MCP tools the qsdev MCP server refuses to run
```

`version` is the schema version (currently `2`). Each profile key is
validated against its own registry by `qsdev check`:

- `profile` is the project-type profile the project was created from
  (`qsdev init --profile`): `go-web`, `ts-fullstack`, `ts-backend`,
  `python-data`, `python-web`, `rust-cli`, `rust-web`, `java-web`,
  `elixir-web`, `dotnet-web`, or one registered by an embedding tool.
- `infra_profile` is the infrastructure profile (`qsdev init
  --infra-profile`): `consulting-default`, `startup-github` or
  `enterprise`. It selects the generated CI workflow, Renovate/Dependabot
  configuration and security documentation, and applies the profile's
  registry proxy, Nix binary cache and build cache (see
  [Infrastructure settings](#infrastructure-settings)). When absent, only
  `consulting-default`'s CI, Renovate and security-documentation files are
  generated and none of its components is applied: only the
  `infrastructure:` settings you set yourself are (`registry_proxy`, and
  `nix_cache` with `nix_cache_public_key`, checked like a profile's).

`qsdev init` records both keys, and `qsdev init --mode join` restores both
from the committed file, so a joining teammate generates the same CI,
Renovate/Dependabot and security files as the project's creator (an explicit
`--infra-profile` on the join command overrides the committed value).

### Language settings

Each `languages:` entry carries a `version`, a `package_manager` and a list
of `extras` (`key=value`, or a bare `key` meaning `true`) that the language's
ecosystem module reads. Detection fills them in from the project's files;
on the customize path of the `qsdev init` wizard, each selected language that
has settings gets its own screen, pre-filled from those values (else the
module's default), and every answer is stored under the key the module reads:

| Language | Wizard settings (stored as) |
|---|---|
| Go | Go version (`version`) |
| JavaScript/TypeScript | Node.js version (`version`), package manager (`package_manager`), TypeScript (`extras: typescript`) |
| Python | Python version (`version`), package manager (`package_manager`: `pip`, `uv`, `poetry`) |
| Rust | channel (`extras: channel=stable\|beta\|nightly`) |
| Java/Kotlin | build tool (`package_manager`: `maven`, `gradle`, `both`), JDK (`version`), Kotlin (`extras: kotlin`) |
| Scala | build tool (`extras: build_tool=sbt\|mill`), JDK (`extras: jdk_version`) |
| C#/.NET | SDK major version (`version`) |
| PHP, Ruby | version (`version`) |
| C/C++ | build system (`extras: build_system=cmake\|meson\|make`), package manager (`package_manager`: `conan`, `vcpkg`, `none`) |
| Terraform/OpenTofu | tool (`extras: variant=terraform\|opentofu`), Terraform version (`version`) |
| Haskell | build tool (`extras: build_tool=cabal\|stack`) |
| Clojure | build tool (`extras: build_tool=tools-deps\|leiningen`) |
| Dart/Flutter | Flutter SDK (`extras: flutter`) |
| Containers | trusted hadolint registries (`extras: trusted_registries`, comma-separated) |

An answer is recorded only when it differs from what the entry already
resolves to, so accepting a pre-filled value leaves the entry unchanged, and
a version left empty falls back to the detected version. For C/C++ an
explicit `package_manager` takes precedence over the `package_manager` extra
that older configurations and detection record; likewise for Java, where
`package_manager` (also set by `--java-build-tool` and project-type profiles)
takes precedence over the `build_tool` extra.

### Git settings

`git.branch_pattern` is the POSIX extended regular expression the always-on
`branch-naming` tool's pre-push hook (in `devenv.nix`) checks the current
branch name against with `grep -E`. It is optional; when it is absent the
hook enforces a broad default,

```
^[A-Za-z0-9][A-Za-z0-9._/@+-]*$
```

which accepts any existing convention (`feat/login`, `audit/deep-review`,
`users/jane/JIRA-12_fix`, `dependabot/npm_and_yarn/@types/node-20.1.0`) and
rejects only names that are hazardous where branch names are interpolated
unquoted, such as CI scripts and shell prompts: shell metacharacters, a leading
`-`, and non-ASCII characters (homoglyphs, bidirectional overrides). Set a
stricter pattern to enforce a team convention. A detached `HEAD` and the
`main`, `master` and `develop` branches are always accepted.

The committed value is authoritative: `qsdev init`, `--mode join` and
`--update` read it from `.qsdev.yaml`, so after changing it run
`qsdev init --update` to regenerate the hook (`.qsdev.local.yaml` cannot
override it). The pattern must compile as a POSIX ERE and be printable ASCII
without a single quote, because it is embedded in a shell single-quoted string
in `devenv.nix`; `qsdev check` reports an invalid pattern and generation
refuses to render one.

### MCP tool deny list

`mcp.disabled_tools` lists tools of qsdev's own MCP server
(`qsdev mcp serve`) that its guardrail refuses to run, for every caller. It
names MCP tools such as `qsdev_nix_run`, `qsdev_security_scan`,
`qsdev_credential_vend` or `analyze_session`, which are not the same thing as
qsdev catalog tools. The `agent-postmortem` and `version-sentinel` servers in
`.mcp.json` are the same server restricted to one tool module
(`qsdev mcp serve --module <name>`), so the list governs their tools
(`analyze_session`, `list_failure_patterns`,
`generate_verification_checklist`, `check_versions`, `detect_drift`,
`manifest_coverage`, `version_history`) too. `tools.disabled` lists catalog tools (`gitleaks`, `semgrep`, ...);
it controls what `qsdev init` generates and never blocks an MCP tool.

```yaml
mcp:
  disabled_tools:
    - qsdev_nix_run          # no process execution through MCP
    - qsdev_credential_vend
```

- `qsdev check` fails (`config_validation`, high severity) on a name the
  server cannot mount: its generic tools, the tools of every tool module
  (security, devenv, status, agent-postmortem, version-sentinel), and every
  framework adapter's tools. The same check fails on an MCP
  tool name placed in `tools.disabled`.
- The server reads the list once at startup, so restart it after a change.
  It still starts when an entry names no tool it provides, but logs a warning
  for each one, because such an entry leaves the tool you probably meant
  runnable.
- `qsdev_policy_check` reports the list as `mcp_enforced_deny`, and reports
  a tool in it as denied with rule `mcp.disabled_tools`. It warns when the
  file on disk differs from the list the running server enforces.
- The key is set only in `.qsdev.yaml`: `.qsdev.local.yaml` cannot add to
  or remove from it. Re-creating a project (`qsdev init --force`) keeps it.

### MCP credential vending

`qsdev_credential_vend` exchanges the host's ambient cloud identity for
short-lived credentials (AWS STS, GCP IAM Credentials, Azure Managed
Identity). Its output is exempt from the MCP server's secret redaction, so
it is opt-in. The server does not mount it unless `security.credential_vend`
is enabled, and it vends only the identities the allow-lists name:

```yaml
security:
  credential_vend:
    enabled: true
    aws:
      role_arns:                     # AssumeRole targets, matched exactly
        - arn:aws:iam::123456789012:role/ci/deploy
      allow_session_token: false     # the default; see below
    gcp:
      service_accounts:              # email or numeric unique ID
        - ci@my-project.iam.gserviceaccount.com
    azure:
      scopes:                        # the default management scope must be listed to be used
        - https://storage.azure.com/.default
      identities:                    # user-assigned managed-identity client IDs
        - 00000000-0000-0000-0000-000000000001
```

- Without the block, or with `enabled: false`, the tool is not offered to
  any client. A call to it on a server that does mount it is still checked
  against the block and refused (`status: denied`, naming the setting that
  would allow it) before any ambient credential is loaded.
- An AWS request without `role_arn` calls `GetSessionToken`. Those
  credentials carry every permission of the ambient IAM user, so it is
  refused unless `aws.allow_session_token: true`.
- A GCP `service_account`, an Azure `scope` and an Azure `identity` must
  each be in their list. A request naming no Azure identity uses the
  system-assigned one.
- `qsdev check` fails on an entry that cannot match a request (a role ARN
  with a wildcard, a malformed service account, scope or client ID).
- The block is set only in `.qsdev.yaml`: `.qsdev.local.yaml` cannot set it.
  The self-protection hook (GD-001) blocks an agent edit that enables it,
  allows `GetSessionToken`, or adds an allow-list entry. The server reads it
  at startup, so restart it after a change.

`qsdev_nix_run` runs its target with the server's environment minus every
variable `qsdev_env_info` withholds (tokens, keys, passwords, and the
`AWS_*`, `GCP_*`, `GOOGLE_*`, `AZURE_*`, `GH_*` and `GITHUB_*` namespaces), so the child cannot
print the server's credentials. In gateway mode (`--deploy-mode=gateway`)
the tool is not mounted, because it would run Nix packages on the gateway
host for every framework the gateway serves. Pass `--gateway-allow-nix-run`
or set `QSDEV_GATEWAY_ALLOW_NIX_RUN=true` to mount it.

### Infrastructure settings

The built-in infrastructure profiles choose technologies, never endpoints:
your registry proxy, Nix cache and key are organization-specific, so they
live under `infrastructure:` (or come from `qsdev init --registry-proxy`,
`--nix-cache` and `--nix-cache-public-key`, which are recorded there). An
explicitly selected `infra_profile` is applied in full, and generation stops
with an error naming each missing setting instead of silently leaving
installs on the public registries:

| Profile component | Required setting | Applied as |
|---|---|---|
| Registry proxy (`consulting-default`: Nexus, `enterprise`: Artifactory) | `registry_proxy` (or a per-ecosystem `registry_proxy_overrides` entry), for each proxied ecosystem the project uses | Each package manager's config (`.npmrc`, `pip.conf`, `GOPROXY`, `.cargo/config.toml`, `nuget.config`, Maven/Gradle), using the vendor's group/virtual repository paths (`/repository/npm-group/`, `/api/npm/npm-virtual/`, ...); `registry_proxy_paths` entries win |
| Nix binary cache (Cachix in all three) | `nix_cache` and `nix_cache_public_key` | `cachix.pull` in `devenv.nix` for a Cachix cache, and the `trusted-substituters`/`trusted-public-keys` of `docs/nix-conf-hardening.md` |
| Build cache (`sccache`, or Turborepo for `startup-github`) | none; `build_cache_url` optional (Turborepo only) | `infrastructure.build_cache`: sccache as Rust's `rustc-wrapper` (with the package); `TURBO_API` from `build_cache_url` |

`startup-github`'s GitHub Packages hosts your organization's own packages
and needs a token even to read, so it is not a pull-through proxy: nothing
is routed through it unless you set `registry_proxy_overrides`.

Endpoints must be `https` URLs (plain `http` only to `localhost`) without
embedded credentials, and documentation placeholders are rejected: hosts
under `example.com`/`.net`/`.org` or the `.example`, `.invalid` and `.test`
domains, the `myorg` Cachix cache, and an all-zero public key. Set
`registry_proxy: none` or `nix_cache: none` to keep a profile's CI and
update tooling without that component. Credentials (`NEXUS_TOKEN`,
`ARTIFACTORY_TOKEN`, `CACHIX_AUTH_TOKEN`, the sccache S3 credentials,
`SNYK_TOKEN`) are read from the environment and never written by qsdev;
`docs/security-overview.md` lists the ones the profile expects and describes
the proxy and caches actually applied.

Schema version 1 used a single `profile` key, which `qsdev init` filled with
the infrastructure profile. Version 1 files still load: an infrastructure
profile name under `profile` is read as `infra_profile`, and any other value
stays the project-type `profile`. The file itself is rewritten at version 2
by the next `qsdev init --update` (or any command that records changes in
`.qsdev.yaml`), or explicitly with `qsdev config migrate --write`. A version
2 file cannot be read by releases that only support version 1, so update
qsdev across the team before committing the migrated file.

`tier` records the security tier the project was generated at. `qsdev init`
always writes it: the `--tier` value, else `supply-chain-only` when that
permission level was chosen, else the catalog's `default_tier` (`standard`).
`qsdev init --yes` without `--tier` is therefore identical to
`qsdev init --yes --tier standard`: the same `.qsdev.yaml` and the same
generated files. `claude_code.permission_level` is written only when a
permission preset was chosen explicitly (`--claude-permissions` or the
wizard); otherwise the tier's `default_permission_preset` applies.
`qsdev status`, `qsdev init --mode join`, `qsdev init --update` and posture
scoring read it directly. Only a legacy file without a `tier` key has its tier
inferred: a `supply-chain-only` permission level means that tier; the default
MCP servers (`context7`, `github`, `socket`, and `semble`) never imply `full`,
only a server outside that set does; anything else is `standard`. The next
`qsdev init --update` writes the inferred tier back (a `tier` already in
`.qsdev.yaml` wins over the saved answers).

Older releases treated those default MCP servers as `full`, so a legacy
project initialized without `--tier` may carry full-tier-only files (the
`qsdev-*` operation skills, consulting agents and `.claude/qsdev-reference.md`).
It is now read as `standard`, and `qsdev init --update` removes those files
if they are unmodified. To keep them, add `tier: full` to `.qsdev.yaml`
before updating.

### Hook settings

The `hooks:` block configures the generated Claude Code hooks. It is team
policy: only the committed `.qsdev.yaml` sets it (`.qsdev.local.yaml`
rejects the key), and init, join and update refresh it from there.

```yaml
hooks:
  file_boundary:
    extra_read_paths:
      - /opt/android-sdk
      - ~/.local/share/my-sdk
  tool_gates:
    allowed: [Read, Grep, Glob, Edit, Write, Bash, "mcp__context7__*"]
    denied: [WebFetch, WebSearch, "mcp__github__delete_*"]
```

- **`file_boundary.extra_read_paths`** widens what the `file-boundary` hook
  lets the agent *read*. The hook confines Write, Edit, MultiEdit,
  NotebookEdit, Read, Grep and Glob to the project directory. Read, Grep
  and Glob may also reach the dependency sources an LSP go-to-definition
  lands in: the Go module cache (`GOMODCACHE`, else `$GOPATH/pkg/mod`),
  `GOROOT`, the Cargo registry and git checkouts, rustup toolchains,
  `~/.m2/repository`, `~/.gradle/caches`, `~/.claude/plugins` and
  `/nix/store`. `extra_read_paths` adds directories to that read-only list,
  for example an SDK installed outside those caches. Write and Edit are
  still denied there. Each entry must be an absolute path or start with
  `~/`. The filesystem root, the home directory or a directory containing
  it, a path that is, contains or lies inside a credential store (`~/.ssh`,
  `~/.aws`, `~/.config` with its `gcloud` and `helm` stores, `/etc` with
  `/etc/shadow`, and the rest of the sandbox deny list), `..` segments,
  commas and control characters are rejected by `qsdev check` and stop
  init with an error. The hook itself also ignores an entry that resolves,
  through symlinks, to the root, the home directory or an ancestor of it.
  The paths reach the hook through the
  `FILE_BOUNDARY_EXTRA_READ_PATHS` variable in the `env` block of
  `.claude/settings.json`. `FILE_BOUNDARY_STRICT_MODE=true` revokes every
  out-of-project allowance, including these paths.
- **`tool_gates.allowed`** and **`tool_gates.denied`** are the policy the
  `tool-gates` hook enforces on every tool call. Entries are Claude Code
  tool names (`Bash`, `WebFetch`, `mcp__github__delete_repo`), matched
  case-sensitively, where `*` matches any run of characters
  (`mcp__github__*` is every tool of the `github` MCP server). A tool
  matching `denied` is always blocked. When `allowed` is non-empty, a tool
  matching none of its entries is blocked too; an empty `allowed` allows
  every tool not denied. Entries may contain only letters, digits, `_`, `-`
  and `*`; anything else (commas, spaces, control or invisible characters)
  is rejected by `qsdev check` and stops init with an error. The lists
  reach the hook through the `TOOL_GATES_ALLOWED` and `TOOL_GATES_DENIED`
  variables in the `env` block of `.claude/settings.json`, and are
  generated only while the `tool-gates` hook is enabled. With neither list
  set the hook has **no policy**: it runs on every call but allows every
  tool. `qsdev claude hooks list` then shows it as `yes (no policy)`, the
  init preview as `tool-gates (no policy)`, and `qsdev check` reports a
  `claude_hook_no_policy` warning.

`qsdev check` also fails (`claude_hook_env_changed`, high severity) when a
variable qsdev generates from this block is missing from, or holds another
value in, the `env` block of `.claude/settings.json`, since the hooks read
their policy only from there. It compares against the committed
`.qsdev.yaml`, so a policy change that has been committed but not yet
generated fails the same way. `qsdev init --update` writes it.

#### Security floor, client policy and local overrides

`qsdev init` (create over an existing `.qsdev.yaml`, `--mode join`, and
`--update`) resolves `.qsdev.yaml` and the developer's gitignored
`.qsdev.local.yaml` through one resolver, so what the committed file declares
reaches the generated files:

```yaml
security:
  level: strict               # the project's security floor
client:
  name: acme
  security_level: strict      # a client can raise, never lower, the floor
  blocked_mcp_servers: ["*"]  # "*" blocks every server not allowed below
  allowed_mcp_servers: [context7]
```

- **Security floor.** The stricter of `security.level` and
  `client.security_level` is the floor. Generation uses at least that
  compliance level, and the hooks it requires are enabled (`enhanced`:
  pre-commit; `strict`: pre-commit, audit-log and auto-format). A
  `security.*` switch the floor mandates (`age_gating`, `script_blocking`,
  `lock_enforcement`, `vuln_scanning`) cannot be turned off locally.
- **Client compliance level.** `client.security_level` also enables the
  catalog tools its required pre-commit hooks come from (for example
  `gitleaks` and `semgrep`) unless the project records its own decision
  about them in `tools.enabled`/`tools.disabled`, and supplies
  `claude_code.permission_level` when none is set (`minimal` for `strict`,
  `standard` otherwise).
- **Client MCP policy.** A server named in `blocked_mcp_servers` is never
  written to `.mcp.json`, whichever source asked for it
  (`claude_code.mcp_servers`, the default servers, an enabled tool, or the
  semble agent tool). With `"*"`, only the servers in `allowed_mcp_servers`
  are written. Under a policy, a Claude Code project at the `standard` tier
  or above always gets a `.mcp.json` (possibly with no servers), and init,
  join and update also remove forbidden servers already in an existing
  `.mcp.json`.
- **Local overrides.** `.qsdev.local.yaml` (gitignored) adds to the
  committed configuration for your own checkout. It may only add or
  tighten, never remove or loosen:

  | Key | Local effect |
  |-----|--------------|
  | `extra_packages` | Added to the generated `devenv.nix` packages |
  | `languages` | A new language is added; a committed one takes the local `version`. Its committed `package_manager` cannot change |
  | `services` | A new service is added; a committed one takes the local `version` and gains new `options`, but a committed option cannot change |
  | `tools.enabled` | Enables the tools, including ones `.qsdev.yaml` disables |
  | `claude_code.skills`, `claude_code.mcp_servers` | Added (servers still subject to the client MCP policy) |
  | `claude_code.permission_level` | Applied only when stricter than the committed level (or, when none is committed, the tier's preset): `minimal` is stricter than `standard`, which is stricter than `permissive`. `custom` and `supply-chain-only` are not comparable, so a local override can neither switch to them nor away from them |
  | `security.level`, `security.*` | Can raise the floor, never lower it |
  | `tools.disabled`, `tools.config`, `claude_code.enabled` | Ignored: only `.qsdev.yaml` sets them |
  | `hooks` | Not accepted: the local file fails to parse, since only `.qsdev.yaml` sets hook policy |

  Every ignored or raised setting is reported as a warning (for example
  `security.level: baseline raised to strict`,
  `claude_code.permission_level: permissive raised to standard` or
  `tools.disabled: gitleaks ignored (...)`). Create (over an existing
  `.qsdev.yaml`), join and update generate with the local additions, but
  never write them to `.qsdev.yaml` or the saved answers, so a local override
  never reaches the team through them; the files generated for your checkout
  (for example `devenv.nix` with your extra packages) do contain them. A
  project being created for the first time has no committed configuration
  yet, so the local file takes effect from the next join or update. An
  invalid local value (such as a package that is not a Nix attribute path)
  stops init with an error. Re-creating a project (`qsdev init --force`)
  records the security level it generated with, so a local raise in effect
  at that moment becomes the committed level.

There is no organization-defaults layer: a project without a `security`
block is generated with the settings it was created with, not raised to
qsdev's built-in defaults. An unreadable `.qsdev.yaml` or `.qsdev.local.yaml`
stops init and update with an error rather than generating without its policy.
Re-creating a project (`qsdev init --force`) keeps the committed `client`
block, `security` switches, `hooks` block, `git` settings and `tools.config`,
and never
records a lower `security.level` than the committed one.

---

## Devenv Files

### `devenv.yaml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `overwrite` |
| **Purpose** | Top-level devenv.sh configuration with security hardening |

Key contents:

- `require_version: ">=2.1"` -- Minimum devenv version
- `inputs.nixpkgs.url: "github:NixOS/nixpkgs/nixpkgs-unstable"` -- Nixpkgs input
- `inputs.git-hooks.url: "github:cachix/git-hooks.nix"` -- Pre-commit hook framework (when hooks are needed)
- `impure: false` -- Prevents impure builds
- `allow_unfree: false` -- Blocks unfree packages
- `allow_broken: false` -- Blocks broken packages
- `permitted_unfree_packages: []` -- Empty allowlist
- `permitted_insecure_packages: []` -- Empty allowlist
- `clean.enabled: true` -- Strips environment on shell entry
- `clean.keep: [TERM, HOME, USER, ...]` -- Minimal variable allowlist

### `devenv.nix`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `manual-merge` |
| **Purpose** | Nix expression defining packages, services, hooks, and shell scripts |

This is the most complex generated file and the one most likely to be customized by users, hence the `manual-merge` strategy. Key sections:

- **Packages** -- Base packages (`git`, `jq`, `curl`, `coreutils`) plus language-specific toolchains and extra packages
- **Services** -- Database, cache, messaging, and infrastructure services (PostgreSQL, Redis, MySQL, MongoDB, Elasticsearch, RabbitMQ, Kafka, MinIO, Mailpit, Keycloak, NATS)
- **Pre-commit hooks** -- Language-independent hooks (`ripsecrets`, `check-added-large-files`, `no-commit-to-branch`, `check-merge-conflicts`, `shellcheck`, and `statix` from `enhanced` up), custom hooks (`lock-file-audit`, `nix-secrets-check`) and each selected language's formatters, linters and scanners, tiered by the effective security level (see [Pre-commit hook tiers](#pre-commit-hook-tiers))
- **Tool sections** -- Sections contributed by enabled tools, such as the always-on `branch-naming` pre-push hook (see [Git settings](#git-settings)) and the opt-in `commit-ticket` prepare-commit-msg hook
- **Environment variables** -- `DEVENV_SECURITY_HARDENED=true` sentinel, user-defined variables, credential variable unsetting
- **enterShell** -- Security posture banner displayed on shell entry
- **enterTest** -- Validation script for `devenv test`

On update, if the file has been modified, a `devenv.nix.new` sidecar is created and the user is shown a diff.

#### Pre-commit hook tiers

The catalog's `hook_tier_order` and `hook_tiers` assign pre-commit hooks to
one tier per security level (`baseline`, `enhanced`, `strict`). `devenv.nix`
gets the hooks of the effective security level's tier (the stricter of
`security.level` and `client.security_level`) and of every tier below it:

| Tier | Hooks | Runs at |
|------|-------|---------|
| `baseline` | Security hooks (`ripsecrets`, `gitleaks`, `semgrep`, `opengrep`, `nix-secrets-check`, `lock-file-audit`, `shellcheck`, `govulncheck`, `bandit`, `tfsec`) and repository hygiene (`check-added-large-files`, `no-commit-to-branch`, `check-merge-conflicts`) | Every level |
| `enhanced` | Language formatters and linters (`gofmt`, `govet`, `staticcheck`, `ruff`, `mypy`, `eslint`, `prettier`, `rustfmt`, `clippy`, `statix`, ...) | `enhanced`, `strict` |
| `strict` | None (strict adds audit logging and auto-format, not pre-commit hooks) | `strict` |

Security hooks are never tiered out: only non-security hooks sit above
`baseline`. A hook no tier lists (for example a tool's `commit-ticket` or
`branch-naming` hook) runs at every level. A project without a security level
gets every hook. An org or project catalog overlay can move hooks between
tiers; `hook_tier_order` must match `security_levels`, a hook may appear in
only one tier, and a compliance level's `required_pre_commit_hooks` may not be
tiered above that level.

### `.envrc`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `overwrite` |
| **Purpose** | direnv activation script |

Contents:

```bash
eval "$(devenv direnvrc)"
use devenv
```

After generation, run `direnv allow` to activate.

### `.pre-commit-config.yaml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `overwrite` |
| **Purpose** | Pre-commit hook configuration for linting, formatting, and lockfile enforcement |

Hooks include:

- **ripsecrets** -- Scans staged files for secrets
- **check-added-large-files** -- Blocks files over size threshold
- **no-commit-to-branch** -- Prevents direct commits to main/master
- **check-merge-conflict** -- Detects unresolved merge markers
- **shellcheck** -- Shell script linting
- **statix** -- Nix linting
- **lock-file-audit** -- Flags lockfile changes that need manual verification
- **nix-secrets-check** -- Detects hardcoded credentials in `.nix` files

---

## Claude Code Files

### `.claude/settings.json`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `three-way-merge` |
| **Purpose** | Claude Code permission rules, deny rules, ask rules, and hook configuration |

Top-level structure:

```json
{
  "env": {
    "FILE_BOUNDARY_EXTRA_READ_PATHS": "/opt/android-sdk"
  },
  "permissions": {
    "defaultMode": "default",
    "disableBypassPermissionsMode": "disable",
    "allow": ["Read(*)", "Edit(*)", "Write(*)", "Bash(git status *)", "..."],
    "deny": ["Bash(npx *)", "Bash(nix-env -i *)", "...~90 rules"],
    "ask": ["Bash(npm install *)", "Bash(pip install *)", "...~60 rules"]
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py",
            "timeout": 30,
            "statusMessage": "Checking package install safety..."
          }
        ]
      }
    ]
  }
}
```

The permission model uses approximately **90 deny rules** and **60 ask rules**:

- **Deny rules** block dangerous operations outright: npx, nix-env imperative installs, system package managers, pipe-to-shell, shell wrapping, env/command prefix bypass, sudo-prefixed installs, subprocess escapes, eval/xargs, and destructive operations.
- **Ask rules** gate package install operations (npm, pip, cargo, go, gem, composer, dotnet) through the PreToolUse hook, which performs age-gating and vulnerability checks before allowing the install.
- **MCP tool deny rules** deny, as whole tools, the file tools of MCP servers in the fallback (lowest) trust tier: every tool of the reference `filesystem` server and `mcp__github__create_or_update_file`. A permission rule cannot scope an MCP tool by its path argument, so a path rule such as `Read(./.env)` cannot be carried over to `mcp__filesystem__read_file`; the tool is denied outright instead. Servers are scored from their generated `.mcp.json` definition and the known-server database, and a server qsdev does not configure scores into the fallback tier. Manual overrides in `~/.qsdev/trust.yaml` apply only to the enforce hook, not to the committed `settings.json`. Tools of higher-tier servers stay available and are path-checked by the confused-deputy PreToolUse hook. See [MCP trust scoring](security-architecture.md#layer-13-package-and-mcp-risk-scoring).

The three-way merge during updates preserves any custom allow/deny rules you have added while incorporating new rules from template upgrades.

`env` holds variables qsdev generates to configure its hooks
(`FILE_BOUNDARY_EXTRA_READ_PATHS`, from `hooks.file_boundary.extra_read_paths`;
`TOOL_GATES_ALLOWED` and `TOOL_GATES_DENIED`, from `hooks.tool_gates`)
alongside any you add. A regeneration sets the generated variables to the
committed policy's values, removes one the policy no longer produces unless
you changed it, and keeps your own variables.

#### Permission Presets

| Preset | Philosophy |
|--------|-----------|
| **minimal** | Read-only by default. Only `Read(*)` and basic build/test commands are allowed. Every write or edit requires approval. |
| **standard** | Productive development. `Read`, `Edit`, `Write`, `git`, build/test/lint, and Nix dev commands are allowed. Package installs are ask-gated. Bypass mode is disabled. |
| **permissive** | Standard plus `make` and `docker` commands. For teams that use Makefiles or Docker-based workflows. |
| **supply-chain-only** | Minimal permissions focused on supply chain defense. Deny rules and package-guard hook without broader development tooling permissions. |
| **custom** | Only explicitly configured allow/deny patterns. Full manual control for advanced use cases. |

### `.claude/hooks/package-guard.py`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `overwrite` |
| **Purpose** | PreToolUse hook that intercepts package install commands at runtime |

This Python script is the runtime complement to the static ask rules. It receives the tool name and input from Claude Code's PreToolUse event and validates commands matching install patterns against age-gating and vulnerability data before allowing them to proceed.

### `.claude/hooks/audit-log.sh`

| | |
|---|---|
| **Generated by** | `qsdev init` (when `audit-log` hook is enabled) |
| **Merge strategy** | `overwrite` |
| **Purpose** | PostToolUse hook that logs all tool invocations for audit trail |

### `.claude/skills/*.md`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `library-managed` |
| **Purpose** | Skill files defining reusable Claude Code capabilities |

#### Built-in skills (6)

| Skill | Description |
|-------|-------------|
| `deploy.md` | Deploy to staging/production via CI pipeline |
| `review-pr.md` | Structured pull request review with checklist |
| `security-review-owasp.md` | Security-focused code review with OWASP checks |
| `generate-tests.md` | Generate test suites for existing code |
| `refactor.md` | Refactor code for clarity, performance, and maintainability |
| `db-migration.md` | Create safe, reversible database schema migrations |

#### qsdev ops skills (11)

| Skill | Description |
|-------|-------------|
| `qsdev-init` | Initialize qsdev project configuration |
| `qsdev-onboard` | Onboard existing project to qsdev |
| `qsdev-setup` | Install missing prerequisites |
| `qsdev-enable` | Enable a tool in the project |
| `qsdev-disable` | Disable a tool from the project |
| `qsdev-update` | Update qsdev-managed configuration |
| `qsdev-doctor` | Run qsdev health diagnostics |
| `qsdev-status` | Show qsdev configuration status |
| `qsdev-tools` | List available qsdev tools |
| `qsdev-detect` | Detect project ecosystems |
| `qsdev-add-dep` | Add a dependency or package safely |

As library-managed files, all skills are updated to the latest version during `qsdev update`.

### `.claude/rules/*.md`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `library-managed` |
| **Purpose** | Convention rule files based on detected languages |

Available rule files (generated per-language):

| Rule File | Applies To |
|-----------|-----------|
| `go-conventions.md` | Go projects |
| `typescript-conventions.md` | JavaScript/TypeScript projects |
| `python-conventions.md` | Python projects |
| `rust-conventions.md` | Rust projects |
| `java-conventions.md` | Java/Kotlin projects |
| `dotnet-conventions.md` | .NET projects |
| `docker-conventions.md` | Docker projects |
| `terraform-conventions.md` | Terraform projects |
| `security-rules.md` | All projects (security policies) |

Updated automatically on `qsdev update`.

### `.claude/agents/semble-search.md`

| | |
|---|---|
| **Generated by** | `qsdev init` (when semble MCP server is enabled) |
| **Merge strategy** | `library-managed` |
| **Purpose** | Delegated agent for semantic code search |

Provides a specialized agent that uses semble for meaning-based code search. Invoked by Claude Code when natural-language code discovery is more effective than text search.

### `.claude/qsdev-reference.md`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `library-managed` |
| **Purpose** | CLI and workflow reference loaded via @-import from CLAUDE.md |

Contains the full qsdev command reference, security policies, and common workflow instructions. Keeps `CLAUDE.md` lean while providing Claude Code with complete operational knowledge.

### `CLAUDE.md`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `section-marker` |
| **Purpose** | Project documentation and instructions for Claude Code |

Contains project name, detected languages, build commands, security policies, and ecosystem-specific instructions. Tool-managed sections are delimited by markers; content you add outside these sections is preserved during updates.

### `.mcp.json`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `three-way-merge` |
| **Purpose** | MCP server configuration for Claude Code |

Three servers are configured by default in `.mcp.json` (context7, github, socket), with additional servers activated based on tool enablement and project detection. Servers passed with `--mcp` are added to these, not a replacement for them:

| Server | Command | Purpose | Activation |
|--------|---------|---------|------------|
| `context7` | `npx @upstash/context7-mcp@4.1.1` | Library documentation lookup | Default |
| `github` | `github-mcp-server stdio` (Nix package) | GitHub API access | Default |
| `socket` | HTTP `https://mcp.socket.dev/` | Behavioral dependency analysis | Default |
| `semble` | `uvx --from semble[mcp]==0.6.0 semble` | Semantic code search | Opt-in (`--agent-semble` or `qsdev enable semble`) |
| `agent-postmortem` | `qsdev mcp serve --module agent-postmortem` | Session analysis and failure patterns | Enabled when tool active |
| `version-sentinel` | `qsdev mcp serve --module version-sentinel` | Dependency version monitoring | Enabled when tool active |
| `local-docs-devdocs` | `npx @madhan-g-p/devdocs-mcp-server@1.0.1` | Local DevDocs API references | On when detected |
| `local-docs-zim` | `openzim-mcp` (installed by `qsdev mcp install`) | Offline Stack Exchange via ZIM | Opt-in |
| `mcp-nixos` | `uvx --from mcp-nixos==3.1.0 mcp-nixos` | NixOS packages and options | Opt-in |
| `postgres` | `uvx --from postgres-mcp==0.3.0 postgres-mcp --access-mode=restricted` | Read-only database querying ([Postgres MCP Pro](https://github.com/crystaldba/postgres-mcp)); `DATABASE_URI` is set from `DATABASE_URL` | Opt-in (`qsdev enable postgres-mcp`) |

**Pinned launch specs.** Every server that `npx`, `uvx` or another package launcher would fetch at session start names an exact release (npm `name@1.2.3`, PyPI `name==1.2.3`, container `image@sha256:...`). These launches happen outside the package guard, the Bash deny rules and lockfile pinning, so an unpinned spec would run whatever release the registry serves that day. Generation refuses an unpinned launcher from any source, including servers in the claudecode addon's `mcp_servers` configuration, with an error naming the server; pin the release to fix it. Shell wrappers (`sh -c ...`) cannot be inspected here, and `qsdev mcp grade` fails them closed.

**Installed binaries.** Servers with an install method (context7, semble, local-docs-devdocs, local-docs-zim, mcp-nixos, postgres) can be installed with `qsdev mcp install <name>`, which installs exactly the pinned release (subject to the same release-age cutoff as project dependencies) and records it in `.claude/.qsdev-claude-state.yaml`. While that record matches the pinned release, `qsdev init --update` writes the installed executable (for example `context7-mcp`) into `.mcp.json` instead of the launcher, so no package is fetched at session start. `qsdev mcp update` moves an installed server to the release the current qsdev pins; `qsdev mcp remove` drops the record, and the next regeneration returns to the pinned launcher. The installed binary must be on the `PATH` Claude Code starts with. The install record lives in the committed project state, so the committed `.mcp.json` runs the binary for every teammate: after cloning or `qsdev init --mode join`, run `qsdev mcp update --all` to install the same pinned releases on your machine (`qsdev mcp health` reports a server whose binary is missing).

Structure:

```json
{
  "mcpServers": {
    "context7": {
      "command": "npx",
      "args": ["@upstash/context7-mcp@4.1.1"]
    },
    "github": {
      "command": "github-mcp-server",
      "args": ["stdio"],
      "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"}
    },
    "socket": {
      "type": "http",
      "url": "https://mcp.socket.dev/"
    },
    "semble": {
      "command": "uvx",
      "args": ["--from", "semble[mcp]==0.6.0", "semble"]
    }
  }
}
```

Three-way merge preserves custom server entries you add while updating built-in server configurations, except servers the committed `client.blocked_mcp_servers` policy forbids, which are removed (see [Security floor, client policy and local overrides](#security-floor-client-policy-and-local-overrides)).

---

## Machine Bootstrap

### Claude Code install (`bootstrap_tools.claude-code`)

When `claude` is not on `PATH`, the bootstrap step "Install Claude Code" and `qsdev devenv setup` install the exact release pinned in the catalog's `bootstrap_tools` section, never the registry's latest:

```yaml
bootstrap_tools:
    claude-code:
        install_method: npm-global        # the only supported method
        package_name: "@anthropic-ai/claude-code"
        version: "2.1.273"                # exact release; ranges and dist-tags are refused
        allow_install_scripts: true
```

Both run `npm install -g [--ignore-scripts] --before=<now - 3 days> @anthropic-ai/claude-code@<version>`. `--before` age-gates the install the way `min-release-age=3` gates project installs (a global install does not read the project `.npmrc`): npm resolves the package and its dependencies only among releases published before the cutoff, so a pinned release younger than 3 days fails to install until it has aged. A missing, unpinned or unsupported entry, or a `package_name` starting with `-`, stops the install with an error rather than falling back to an unpinned install.

`allow_install_scripts` is off by default, which adds `--ignore-scripts`. It is on for Claude Code because the package needs its one lifecycle script: its `postinstall` (`install.cjs`) hard-links the native binary from the matching platform package (`@anthropic-ai/claude-code-<os>-<arch>`, pinned to the same exact version and carrying no scripts of its own) over the `bin/claude.exe` placeholder. Without it, `claude` is a stub that only prints an error.

To move to a newer release before qsdev ships a new pin, set `version` in `~/.config/qsdev/defaults.yaml` (`qsdev defaults edit`) to an exact release at least 3 days old; the entry is deep-merged, so only the fields you set change. An already installed `claude` is left as it is.

### devenv and direnv install (`bootstrap_tools.devenv`, `bootstrap_tools.direnv`)

When `devenv` or `direnv` is not on `PATH`, the bootstrap steps "Install devenv" and "Install direnv" (and, for devenv, `qsdev devenv setup`) install the attribute pinned in the catalog from nixpkgs pinned to one commit:

```yaml
bootstrap_tools:
    devenv:
        install_method: nix-profile
        flake: "github:NixOS/nixpkgs/d233902339c02a9c334e7e593de68855ad26c4cb"
        package_name: devenv            # attribute path within the flake
    direnv:
        install_method: nix-profile
        flake: "github:NixOS/nixpkgs/d233902339c02a9c334e7e593de68855ad26c4cb"
        package_name: direnv
```

Both run `nix profile install --option accept-flake-config false <flake>#<package_name>`. The flake must name a single commit: a `github:`/`gitlab:`/`sourcehut:` reference written `<owner>/<repo>/<40-hex rev>` or `<owner>/<repo>?rev=<40-hex rev>`, or a `git+` reference with a `rev=<40-hex rev>` query parameter. A registry name such as `nixpkgs` (which resolves through the mutable user and system flake registries), even with a `rev=` (the registry could map it to a `path:` or tarball flake that ignores the rev), a `path:`, tarball or file reference, a branch or a tag is refused, as is a flake starting with `-` or an attribute that is not a plain attribute path. `accept-flake-config` is forced off, so a `nixConfig` the flake declares (extra substituters, trusted public keys) is ignored even if your `nix.conf` would accept it. The shipped revision is the nixpkgs-unstable commit qsdev itself builds against; it provides devenv 2.1.2, which satisfies the `require_version: '>=2.1'` in generated `devenv.yaml` files.

To install from a newer nixpkgs, set `flake` in `~/.config/qsdev/defaults.yaml` to `github:NixOS/nixpkgs/<commit>`.

### Post-install check

After any bootstrap install command exits successfully, qsdev detects the tool's binary again. If it is still not resolvable on `PATH` (for example because `~/.nix-profile/bin` or npm's global `bin` directory is not on `PATH` in the current shell), the step fails with an error telling you to add that package manager's bin directory to `PATH` (or open a new shell) and re-run, rather than reporting success.

---

## Policy Engine Files

### `.qsdev/policy/*.yaml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `library-managed` |
| **Purpose** | YAML security policy definitions for the policy engine |

Policy files define rules evaluated as PreToolUse hooks. Each rule specifies a condition tree, an action (block/warn/audit/prompt), a severity level (critical/high/medium/low), and a bypass tier (enforce_always/session/command).

Policy files are decoded strictly: an unknown key is a load error, not a silently ignored field. The only `settings` key is `fail_mode` (`fail_closed`, the default, or `fail_open`). An `action` takes `type`, `message` and, for `prompt`, `default_on_timeout`. Keys that earlier versions accepted but never acted on (`settings.evaluation_timeout_ms`, `settings.log_format`, `settings.inherit_from`, `action.exit_code`, `action.stderr`, `action.timeout_seconds`) are rejected; remove them from existing policy files. A blocking action always exits with code 2.

```bash
qsdev policy list          # List all rules with severity and bypass tier
qsdev policy show <id>     # Show full rule definition
qsdev policy check         # Evaluate posture and show findings
```

### `.qsdev/policy.nix`

| | |
|---|---|
| **Generated by** | Not generated: optional, written by the project |
| **Approved by** | `qsdev sandbox approve` (recorded in `~/.qsdev/sandbox-policy-approvals.json`) |
| **Purpose** | Hook sandbox policy used by `qsdev sandbox exec` |

A Nix expression that evaluates to the sandbox policy: filesystem
`allowRead`/`allowWrite`/`deny`, network mode, resource limits, the five hook
category profiles (`hookCategories`) and per-hook `hookOverrides`. Without
the file, `sandbox exec` uses the built-in defaults. Use `--policy <path>` to
point `sandbox exec` or `sandbox approve` at another file.

The policy configures the sandbox that contains the repository's own hooks,
so it is treated as privileged configuration rather than trusted because it
is in the repository:

- **Approval.** `sandbox exec` evaluates the policy only when its exact
  content has been approved. The approval covers the policy file and every
  regular `*.nix` file beside it (the only files the policy can import), is
  recorded per policy path in `~/.qsdev/sandbox-policy-approvals.json`, and
  any later change to those files, including a `git pull`, voids it. An
  unapproved or changed policy blocks every sandboxed hook (exit 2) with a
  message naming the approve command; it never falls back to the defaults.
  `qsdev sandbox approve` evaluates the policy, shows its digest, the files
  covered and any mounts that will be skipped, and records the approval
  after you confirm. It requires an interactive terminal outside any AI
  agent session, and coding agents are blocked from running it by
  self-protection rule SP-014.
- **Restricted evaluation.** The policy is evaluated from a private copy of
  the approved files with `nix eval --option restrict-eval true`, the search
  path limited to that copy, `allowed-uris` empty, import-from-derivation
  and `builtins.exec` off (even if `nix.conf` or `NIX_CONFIG` enables
  them) and `NIX_PATH` removed. It cannot read other files
  (`builtins.readFile ~/.ssh/...` fails), fetch URLs, run commands or read
  the environment (`builtins.getEnv` returns `""`). Imports from
  subdirectories or outside the policy directory, including `<nixpkgs>`,
  fail. (Pure evaluation cannot be combined with `nix eval --file`.) The
  result is cached in `~/.qsdev/cache/sandbox-policy/`, keyed by the
  approved digest.
- **Mount allowlist.** Every mount the policy declares (`filesystem.allowRead`,
  `filesystem.allowWrite` and each `extraMounts` entry) must have both its
  source and target inside the project directory or under `/nix/store`,
  checked for the literal and symlink-resolved paths. Anything under `/run`
  or `/var/run` (the systemd user bus, ssh-agent and gpg-agent sockets,
  `docker.sock`), a source that is a socket, pipe or device, and the
  credential deny list are refused as well. A refused mount is skipped with a
  warning; the rest of the policy still applies.

```bash
qsdev sandbox approve                    # Review and approve .qsdev/policy.nix
qsdev sandbox approve --policy other.nix # Approve another policy file
```

### `~/.qsdev/session-state.json`

| | |
|---|---|
| **Generated by** | `qsdev session allow` |
| **Merge strategy** | Internal (per user, outside the project) |
| **Purpose** | Tracks active bypass grants, each bound to one project and one Claude Code session |

```bash
qsdev session allow <rule-ids> --session <claude-session-id>   # Grant bypasses (interactive confirmation)
    [--project <dir>]   # Project the grant applies to (default: the project containing the current directory)
    [--ttl <duration>]  # Grant lifetime (default 8h for session-tier rules, 1h for command tokens; max 24h)
qsdev session list      # Show active grants with project, session and expiry
qsdev session clear     # Remove all grants
```

Every grant records the canonical project root, the Claude Code session ID
(the `session_id` of the hook payload, which a blocked call's message names),
and an expiry. The enforce hook applies a
grant only to calls from that session in that project and before it expires,
so a bypass never carries over to another project, a later session, or a
colliding rule ID in another project's policy.

- A rule with `bypass_tier: session` is lifted for every call of the session
  until the grant expires.
- A rule with `bypass_tier: command` receives a one-shot token: the next call
  the rule matches runs and spends it; later calls are blocked again. A token
  is spent only when the call is otherwise allowed, and parallel calls redeem
  it under a lock, so only one of them runs. A call that cannot take the lock
  within 2 seconds is blocked and the token stays unspent.

Granting requires an interactive terminal outside any AI agent session (it is
refused when `CLAUDECODE` is set) and an explicit confirmation, and coding
agents are blocked from running `qsdev session allow` by self-protection rule
SP-014. Only rules with `bypass_tier: session` or `command` can be granted;
rules with `bypass_tier: enforce_always` (or unknown rule IDs) are rejected.

The file is version 2 (`{"version": 2, "grants": [...]}`, mode 0600).
Unscoped overrides written by earlier releases (`sessionBypassOverrides`) are
ignored, reported by `qsdev session list`, and dropped on the next write.

### `.qsdev-policy.yaml`

| | |
|---|---|
| **Generated by** | You (optional; qsdev never writes it) |
| **Read by** | `qsdev status`, `qsdev check` |
| **Purpose** | Project-specific conformance requirements, enforced alongside the built-in baseline |

```yaml
conformance:
  custom:
    name: strict
    requirements:
      - name: no-critical-vulns
        check: dependencies.totals.critical == 0
      - name: sast-enabled
        check: tools.semgrep.enabled == true
      - name: nix-hardened
        check: defense.nix-hardening.status == enabled
      - name: score-floor
        check: score.total >= 80
```

Each `check` is one comparison (`==`, `<=` or `>=`) against the posture report:

| Expression | Operators | Value |
|---|---|---|
| `defense.<layer>.status` | `==` | `enabled`, `disabled`, `partial`, `not-applicable` |
| `dependencies.totals.<severity>` | `==`, `<=`, `>=` | integer; severity is `critical`, `high`, `moderate`, `low`, `info` or `unknown` |
| `config.score` | `==`, `<=`, `>=` | number |
| `score.total` | `==`, `<=`, `>=` | number |
| `tools.<name>.enabled` | `==` | `true`, `false` |

`score.total` weighs dependency health only after a scan: without `--scan`
the dependency sub-score is unknown (`null` in the JSON report) and the total is
computed from defense and configuration alone.

`qsdev status` shows the result as the **Custom** conformance level (`--verbose`
lists each requirement with its actual value), and a failing requirement fails
the exit gate at `--audit-level high` (the default) and every stricter level.
`qsdev check` reports each requirement under **Custom Conformance**; a failing
one is high severity, so it fails the default `--audit-level medium`.

Every requirement fails closed:

- A `dependencies.totals.*` requirement passes only when a fresh vulnerability
  scan completed conclusively in the same run. Pass `--scan` to `status` or
  `check`; without it, or when an ecosystem's scan errors, or when any
  vulnerability's severity is unresolved, zero totals mean "unknown" and the
  requirement fails.
- An invalid expression fails its requirement with the parse error.
- The file is decoded strictly: an unknown key, a `custom` section with no
  requirements, a requirement without a name or check, or a duplicate name
  fails a single `policy-file` requirement instead of being skipped.

---

## Documentation Pipeline

Documentation sets are stored in `~/.qsdev/docs/` (user-level, not per-project).

```bash
qsdev docs download        # Download configured documentation sets
qsdev docs status          # Show installed sets and disk usage
qsdev docs verify          # Verify corpus integrity/signatures against the manifest
qsdev docs outdated        # Check for newer versions
qsdev docs update          # Update outdated sets
qsdev docs clean           # Remove downloaded sets
qsdev docs enable <set>    # Enable a documentation set
qsdev docs disable <set>   # Disable a documentation set
```

Two documentation formats are supported:
- **DevDocs** -- API references from devdocs.io for detected ecosystems, stored as JSON
- **ZIM** -- Stack Exchange archives from openzim, stored as ZIM files

DevDocs `db.json` is sanitized at download time (invisible/tag/control characters
stripped) before the external documentation MCP server indexes it, eliminating the
invisible-Unicode prompt-injection vector with no impact on legitimate content.

## Content Signing & Verification

Content signing protects the documentation corpus (and other artifacts) against
tampering using detached Minisign (Ed25519) signatures, verified against trusted
public keys. The implementation is pure-Go, so no `minisign` binary is required.

```bash
qsdev content keygen --out qsdev          # Generate a key pair (qsdev.pub / qsdev.key)
qsdev content sign <file> --key qsdev.key # Produce a detached <file>.minisig
qsdev content verify <file>               # Verify against trusted keys (--json, --require-trusted)
qsdev content keys                        # List trusted public keys
```

Trusted public keys are loaded from `~/.qsdev/keys/` (`*.pub`). `qsdev docs verify`
reuses this to check signed corpus files, falling back to the manifest SHA-256 for
unsigned sets. Servers serving content with a trusted, signature-verified binary can
reach the `Attested` compliance grade reported by `qsdev mcp grade`.

The lookup-docs skill routes documentation queries through 4 sources in priority order: local DevDocs, Stack Exchange ZIM, mcp-nixos, Context7 (web fallback).

---

## Security Tool Files

The configuration files of project tools (everything outside the `ai-agent`
category of `qsdev list`: `.semgrepignore`, `.gitleaks.toml`, `cliff.toml`,
`.commitlintrc.yml`, `secretspec.toml`, `.starship.toml`, the container,
license and GitHub workflow files below) belong to the project, not to Claude
Code. `qsdev init`, `qsdev init --update`, `--mode join` and `qsdev repair`
generate them whether or not Claude Code is configured, including with
`--devenv-only` or `--claude-only`. Which tools get files follows each
tool's default policy:

| Policy | Files generated |
|---|---|
| `always-on` | At the `standard` and `full` tiers, unless force-disabled (`qsdev disable --force`) |
| `on-when-detected` | At the `standard` and `full` tiers while the tool is enabled (detected or enabled explicitly) |
| `opt-in` | At every tier once enabled, exactly as `qsdev enable <tool>` writes them |

Agent tools (skills, sub-agents, MCP servers) are generated with the rest of
the Claude Code configuration and only when Claude Code is configured.

### `.semgrepignore`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `overwrite` |
| **Purpose** | Paths the Semgrep scan skips |

Lists the default scan exclusions (build output, caches, coverage, framework output, `.devenv/`, test fixtures, vendored dependencies and virtual environments) in the gitignore syntax Semgrep reads from the scan root. A `.semgrepignore` replaces Semgrep's built-in ignore list, so qsdev writes the full set. It also excludes `.semgrep/`: the rule files there are passed with `--config`, and scanning them as code would let a rule match its own pattern.

```gitignore
# Semgrep ignore file — auto-generated by qsdev (gitignore syntax).
build/
dist/
node_modules/
vendor/
.devenv/
*.egg-info/
# Semgrep rules, not scan targets
.semgrep/
```

Semgrep has no project config file that selects registry rule packs, so qsdev generates none. The `qsdev-security-scan` devenv task passes the rule packs of the detected ecosystems (for example `p/golang` and `p/owasp-top-ten`) as `--config` flags, adds `--config .semgrep` when the project has a `.semgrep/` directory, and runs with `--metrics=off --error`, so a finding fails the task:

```bash
semgrep --config p/golang --config p/owasp-top-ten $(if [ -d .semgrep ]; then echo --config .semgrep; fi) --metrics=off --error .
```

Put your project's own Semgrep rules (rule YAML files with `id`, `pattern` and `message`) in `.semgrep/`. qsdev never writes or deletes files there.

### `.gitleaks.toml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `overwrite` |
| **Purpose** | Gitleaks secret detection configuration |

Keeps Gitleaks' built-in detection rules (`[extend] useDefault = true`) and adds
allowlisted paths that are safe to exclude from secret scanning. Allowlist paths
are regular expressions anchored to a path-segment boundary:

```toml
# Gitleaks configuration — auto-generated by qsdev
title = "gitleaks config"

# Keep Gitleaks' built-in detection rules; this file only adds an allowlist.
[extend]
  useDefault = true

[allowlist]
  description = "Allowlisted paths and patterns"
  regexes = []
  paths = [
    '''(^|/)\.devenv/''',
    '''(^|/)testdata/''',
    '''(^|/)node_modules/''',
    '''(^|/)vendor/''',
  ]
```

### `.scancode.yml`

| | |
|---|---|
| **Generated by** | `qsdev enable license-compliance` |
| **Merge strategy** | `overwrite` |
| **Purpose** | ScanCode license policy applied by the security-scan task |

A license policy in the format `scancode --license-policy` reads. ScanCode has no project config file, so the `qsdev-security-scan` devenv task passes this file on the command line. Each entry names a [ScanCode license key](https://scancode-licensedb.aboutcode.org/), the matching SPDX identifier, a label and a `compliance_alert`. An empty alert marks an approved license, `warning` a restricted one that needs review, and `error` a prohibited one. The prohibited set covers the `-only` and `-or-later` forms of GPL and AGPL. ScanCode maps deprecated SPDX identifiers such as `GPL-2.0+` onto the same keys.

```yaml
license_policies:
  - license_key: mit
    spdx_license_key: MIT # pragma: allowlist secret
    label: Approved License
    compliance_alert: ''
  - license_key: mpl-2.0
    spdx_license_key: MPL-2.0 # pragma: allowlist secret
    label: Restricted License
    compliance_alert: 'warning'
  - license_key: agpl-3.0-plus
    spdx_license_key: AGPL-3.0-or-later # pragma: allowlist secret
    label: Prohibited License
    compliance_alert: 'error'
```

The task runs ScanCode over the project and checks its JSON output with `jq`. It fails if any file has a prohibited license or ScanCode reports an error in its scan headers (for example a policy with a duplicate `license_key`, which ScanCode otherwise ignores while exiting 0), and prints restricted licenses without failing:

```bash
scancode --quiet --license --license-policy .scancode.yml --ignore '.git' --ignore '.scancode.yml' ... --ignore 'testdata' --json - . | jq -r '...'
```

The `--ignore` patterns skip build output, caches, framework output, `.devenv/`, test fixtures and the two policy files. Dependency directories such as `node_modules/` and `vendor/` are scanned, because the licenses being checked are stored there.

### `.license-exceptions.yml`

| | |
|---|---|
| **Generated by** | `qsdev enable license-compliance` |
| **Merge strategy** | `skip` |
| **Purpose** | Record of approved license exceptions |

Created once with an empty `exceptions:` list and an example entry (package, license, justification, approver, date), and never overwritten. It is a record for reviewers. The license scan does not read it.

---

## Per-Ecosystem Security Configs

These files are generated by ecosystem modules based on the selected languages.

Files with the `skip` strategy are conventional package-manager or tool configs that projects usually maintain themselves. qsdev creates them only when they are absent. If one already exists, qsdev leaves it untouched and does **not** apply its hardening settings to it. Copy the settings listed below into your existing file yourself.

### JavaScript/TypeScript

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `.npmrc` | `skip` | `ignore-scripts=true`, `min-release-age=3` (needs npm >= 11.10.0, which the generated `devenv.nix` provides as `languages.javascript.npm.package` for every Node.js major; `qsdev check` fails when the `npm` on `PATH` is older), registry configuration, audit settings (only created if absent). `audit-level=moderate` sets only `npm audit`'s exit code; installs never fail on audit results, so the `ecosystem-ci` job runs `npm audit` to enforce it |
| `.yarnrc.yml` | `skip` | `enableScripts: false`, registry configuration (Yarn Berry; only created if absent) |
| `.yarnrc` | `skip` | `ignore-scripts true`, registry configuration (Yarn Classic v1; only created if absent) |
| `pnpm-workspace.yaml` | `skip` | pnpm security config with age-gating (when pnpm is detected; only created if absent) |
| `bunfig.toml` | `skip` | `install.minimumReleaseAge` in seconds (when bun is detected; only created if absent) |
| `.nvmrc` | `overwrite` | Pinned Node.js version |

**Package manager.** A pin in `package.json` decides the package manager: the Corepack `packageManager` field (`"pnpm@10.17.0"`), then `devEngines.packageManager`. The pin applies even before the project has a lockfile. Without a pin, the lockfile decides, in this order: `pnpm-lock.yaml`, `yarn.lock`, `bun.lock`/`bun.lockb`, `package-lock.json`/`npm-shrinkwrap.json`. If none of these exist, qsdev uses npm. A Yarn pin also tells Yarn Classic (`yarn@1.x`, hardened through `.yarnrc`) apart from Yarn Berry (`yarn@2+`, hardened through `.yarnrc.yml`).

**Subproject layouts.** When the repository root has no `package.json`, qsdev looks for one up to three directories deep, for example a Go or Python service with its UI in `frontend/` or `web/`. `node_modules/`, `vendor/` and hidden directories are skipped. qsdev then treats that directory as the JavaScript project:

- It is recorded as the `directory` extra, and devenv.nix sets `languages.javascript.directory = "${config.devenv.root}/<dir>"`.
- The hardening file from the table above is written into that directory, for example `frontend/.npmrc`.
- The eslint and prettier hooks run from that directory, on staged files under it only, so ESLint finds the subproject's `eslint.config.*` and Prettier its `.prettierignore`. The CI install command and the build, test and lint tasks also run there.
- `qsdev check` and `qsdev status` look for the lock file and the hardening file in that directory.
- `.qsdev.yaml` does not record the directory. `qsdev init`, `qsdev init --mode join` and `qsdev check` detect it again, so every teammate generates the same files.

devenv supports only one JavaScript project directory. If several subprojects exist, qsdev picks one in this order: a directory with a lockfile, then the shallowest, then the first by name. Detection lists any other subprojects in a warning. `package.json` files nested inside the chosen directory are treated as its workspace members. Directory names with spaces or shell metacharacters are skipped.

### Python

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `pip.conf` | `skip` | Registry configuration, hash-checking mode (only created if absent) |

### Rust

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `.cargo/config.toml` | `skip` | Registry pinning, build settings (only created if absent) |

### Java/Kotlin

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| Maven `settings.xml` or Gradle config | `skip` | Repository pinning (only created if absent) |

### C#/.NET

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `nuget.config` | `skip` | Source pinning (only created if absent) |
| `Directory.Build.props` | `skip` | Build properties (only created if absent) |

### Ruby

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `.gemrc` | `skip` | Source pinning (only created if absent) |

Bundler hardening (`BUNDLE_FROZEN`, `BUNDLE_DISABLE_EXEC_LOAD`) is exported as environment variables from `devenv.nix` rather than written to the user-owned `.bundle/config`.

### PHP

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| Composer config | `overwrite` | Script restrictions, repository pinning |

### Docker

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `.hadolint.yaml` | `skip` | Dockerfile linting rules (only created if absent) |

### Ansible

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `ansible-lint` config | `overwrite` | Linting and security rules |

### C/C++

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| `.clang-tidy` | `overwrite` | Static analysis configuration |
| `.clang-format` | `overwrite` | Code formatting rules |

### Scala

| File | Merge Strategy | Purpose |
|------|---------------|---------|
| sbt or Mill config | `overwrite` | Repository pinning |

### AWS

When AWS project files are detected (CDK, SAM, Terraform `aws` provider):

| Control | Mechanism |
|---------|-----------|
| Deny rules | Blocks `aws configure`, `aws sts assume-role`, credential cat commands |
| Read-deny rules | Blocks Read access to `~/.aws/credentials`, `~/.aws/config`, `~/.aws/sso/cache/` |
| Environment | Sets `AWS_PROFILE` / `AWS_DEFAULT_REGION` in devenv.nix only when the `aws_profile` / `aws_default_region` extras are configured; otherwise they are inherited from your shell |

### GCP

When GCP project files are detected (Cloud Build, Firebase, Terraform `google` provider):

| Control | Mechanism |
|---------|-----------|
| Deny rules | Blocks `gcloud auth print-access-token`, `gcloud config set`, credential cat commands |
| Read-deny rules | Blocks Read access to `~/.config/gcloud/application_default_credentials.json`, `~/.config/gcloud/credentials.db` |
| Environment | Documents `CLOUDSDK_ACTIVE_CONFIG_NAME`, `CLOUDSDK_CORE_PROJECT` and `GOOGLE_CLOUD_PROJECT` as comments in devenv.nix; set real values in `devenv.local.nix` or with `qsdev init --env` |
| GKE auth | When Helm or container files are also detected, installs `google-cloud-sdk` with the `gke-gcloud-auth-plugin` component |

### Azure

When Azure project files are detected (Pipelines, Bicep, Terraform `azurerm` provider):

| Control | Mechanism |
|---------|-----------|
| Deny rules | Blocks `az account get-access-token`, `az login --service-principal`, credential cat commands |
| Read-deny rules | Blocks Read access to `~/.azure/accessTokens.json`, `~/.azure/msal_token_cache.json` |
| Environment | Documents `ARM_SUBSCRIPTION_ID` and `ARM_TENANT_ID` as comments in devenv.nix; set real values in `devenv.local.nix` or with `qsdev init --env` |

---

## GitHub Workflow Files

### `.github/labeler.yml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `skip` |
| **Purpose** | PR auto-labeling rules based on changed file paths |

Labels PRs automatically with categories like `documentation`, `infrastructure`, `security`, `dependencies`, and per-ecosystem labels based on file glob patterns.

### `.github/workflows/labeler.yml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `skip` |
| **Purpose** | GitHub Actions workflow that runs the labeler on PRs |

### `.github/pull_request_template.md`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `skip` |
| **Purpose** | Standardized PR template with security checklist |

Includes sections for summary, type of change, security checklist (when security tools are enabled), testing, and per-ecosystem items.

If the repository already has its own PR template or labeler configuration, `qsdev init`, `qsdev enable pr-templates` and `qsdev enable pr-labels` keep it and report that they did, and `qsdev disable` never deletes it. A copy qsdev created is refreshed while you leave it unchanged. Once you edit it, qsdev stops updating it, and `qsdev disable` deletes it only with `--force`.

### `.github/workflows/security-scan.yml`

| | |
|---|---|
| **Generated by** | `qsdev init` (infrastructure profile) |
| **Merge strategy** | `overwrite` |
| **Purpose** | CI workflow for lock-file enforcement, ecosystem CI commands and vulnerability scanning |

Generated at tier `standard` and above when the infrastructure profile
configures a vulnerability scanner or CI protection. The `security-scan` job
runs harden-runner (profile-dependent), lock file validation and
OSV-Scanner/Snyk/Grype (profile-dependent). The `ecosystem-ci` job, present
when the project's `languages` contribute CI commands, installs Nix and devenv
and runs each language module's CI commands in the devenv shell, install phase
first (frozen/locked installs), then test, then scan (audits). Which commands
run follows each language's `package_manager` and `extras`, for example
`javascript` with `package_manager: pnpm` runs `pnpm install --frozen-lockfile`
(with `npm`, `npm ci --ignore-scripts` then the scan step
`npm audit --audit-level=moderate`),
`haskell` with `build_tool=stack` runs `stack build --lock-file=error-on-write`,
and `scala` with `build_tool=mill` gets no sbt steps. Some commands need a
setting detection records: `r` runs its renv steps only with
`package_manager: renv`, `lua` its LuaRocks install only with
`package_manager: luarocks`, and `nix` its flake checks only with the `flake`
extra (set when the project has a `flake.nix`). A setting the language entry
does not record is taken from detection when the workflow is generated, so
`qsdev init --update` adds these steps to projects created before detection
recorded it.
See [Security Architecture](security-architecture.md#generated-workflows).

Every step is pinned immutably: actions to a full commit SHA, and the Snyk step to a container image digest (`uses: docker://snyk/snyk@sha256:...`, running `snyk test --all-projects` with `SNYK_TOKEN`). The Snyk step does not use `snyk/actions`, because that action runs the `snyk/snyk:node` image by its mutable tag, so a SHA pin on the action would not pin the code that receives the token. The image comes from the `node` variant, which carries Node.js but not the other ecosystems' build tools; Snyk's `--all-projects` needs those tools to resolve some manifests (for example `go.mod`). The language-specific variants are not used because their entrypoint installs the scanned project's dependencies (for example `pip install -r requirements.txt` or `mvn install`) before scanning, which would run the repository's install hooks with `SNYK_TOKEN` in the environment. qsdev bumps these pins in new releases; `qsdev init --update` picks them up.

---

## Infrastructure Profile Files

Generated when an infrastructure profile is active.

### `renovate.json`

| | |
|---|---|
| **Generated by** | Profiles using Renovate (`consulting-default`, `enterprise`) |
| **Merge strategy** | `overwrite` |
| **Purpose** | Renovate bot configuration with age-gating |

Key settings include `minimumReleaseAge` (3 or 7 days), `automergeType` for patch updates, and lockfile maintenance schedules.

### `.github/dependabot.yml`

| | |
|---|---|
| **Generated by** | Profiles using Dependabot (`startup-github`) |
| **Merge strategy** | `overwrite` |
| **Purpose** | Dependabot update configuration |

### `docs/security-overview.md`

| | |
|---|---|
| **Generated by** | `qsdev init` (all infrastructure profiles) |
| **Merge strategy** | `overwrite` |
| **Purpose** | Human-readable security posture documentation |

Describes the defense layers, profile configuration, and supply chain security measures active in the project.

---

## Gitignore

### `.gitignore`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | `section-marker` |
| **Purpose** | Adds qsdev-managed entries without disturbing existing content |

Entries added under the `# qsdev local configuration` marker:

```gitignore
# qsdev local configuration
.devinit/
.qsdev/
.qsdev.local.yaml
.direnv/
.devenv/
.claude/logs/
.claude/hook-audit.log*
```

The `.envrc`, `.qsdev.yaml` and `.qsdev-generated.sha256` files are intentionally not gitignored -- they should be committed for team reproducibility and CI enforcement. The `.devinit/` state directory is local to each checkout.

---

## State Files

These files track the state of generated configuration for the update workflow. The `.devinit/` files are local to each checkout (gitignored); the generated-file manifest is committed.

### `.qsdev-generated.sha256`

| | |
|---|---|
| **Generated by** | `qsdev init`, `qsdev init --update`, `qsdev init --mode join`, `qsdev enable`/`disable`, `qsdev repair`, `qsdev check --auto-fix` |
| **Merge strategy** | Internal (rewritten whenever the init state is saved) |
| **Purpose** | Committed record of the machine-owned generated files, for CI |

Lists the SHA-256 digest of every machine-owned generated file (strategies `overwrite`, `append`, `skip` and `library-managed`: hooks, rules, skills, `package-guard.py`, workflows and the like), one `<sha256>  <path>` line per file in `sha256sum` format, sorted by path. Human-edited files (`section-marker`, `three-way-merge`, `manual-merge`, `merge`: `CLAUDE.md`, `.claude/settings.json`, `devenv.nix`, ...) are left out because local edits to them are expected.

It exists because the generation state below is gitignored, so a CI checkout has none. `qsdev check` verifies each listed file against it: an edited file fails at medium severity, a deleted one at high. When `.qsdev.yaml` is present but the manifest is missing, malformed or lists no files, the `generated_manifest` check fails at high severity; run `qsdev init --update` (or `qsdev check --auto-fix`, which rebuilds it from the local state) and commit the file. Because it follows the `sha256sum` format, `sha256sum --check --strict .qsdev-generated.sha256` also works without qsdev.

Commit it together with the generated files it describes. A change to it in a pull request is a change to what CI accepts as generated, so review it like one.

### `.devinit/.qsdev-init-answers.yaml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | Internal (not updatable by user) |
| **Purpose** | Saved wizard answers for non-interactive updates |

Contains selected languages, services, permission level, skills, hooks, MCP servers, and infrastructure profile. Used by `qsdev update` to regenerate files without re-running the wizard.

### `.devinit/.qsdev-init-state.yaml`

| | |
|---|---|
| **Generated by** | `qsdev init` |
| **Merge strategy** | Internal (not updatable by user) |
| **Purpose** | File generation state tracking |

Tracks every generated file with its path, content hash, and base content (for three-way merge). Used to detect user modifications during `qsdev update`.

### `.devenv/.qsdev-state.yaml`

Per-addon state for `qsdev devenv` operations.

### `.claude/.qsdev-claude-state.yaml`

Per-addon state for `qsdev claude` operations. Also stores template and skill library version identifiers.

---

## Version Control Recommendations

**Commit these files** (required for team reproducibility):

- `.qsdev.yaml`
- `.qsdev-generated.sha256` (lets `qsdev check` verify generated files in CI)
- `.claude/.qsdev-claude-state.yaml`
- `.envrc`
- All generated configuration files

**Do not commit** (added to `.gitignore` automatically):

- `.devinit/` (local generation state and saved answers)
- `.qsdev/` (logs, backups and other local data)
- `.devenv/` (Nix build artifacts)
- `.direnv/` and `.qsdev.local.yaml` (per-developer overrides)

Teammates run `qsdev init --mode join` to produce identical environments from the committed `.qsdev.yaml`.
