# Changelog

All notable changes to qsdev are recorded in this file. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- The MCP server's opt-in `qsdev_nix_run` tool is now held to what a Bash call
  may do, not only to the Bash deny rules: a call whose Bash equivalent (the
  `nix run` line, the program it runs, a `-c` script or stdin) self-protection
  refuses (a write or delete of a protected file, a human-only `qsdev`
  command, an evasion pattern), or that a Bash ask rule would ask about (a
  package install through a shell), is refused before nix runs. The
  selfprotect hook also judges `mcp__<server>__qsdev_nix_run` calls as those
  Bash command lines.

- Self-protection judges the PowerShell tool's human-only commands in the
  words PowerShell passes a program: `qsdev 'teardown','--force'`,
  `& qsdev teardown,--force`, `& qsdev @('teardown','--force')`, a splat,
  `Start-Process qsdev -ArgumentList 'teardown','--force'` and
  `[Diagnostics.Process]::Start('qsdev','teardown --force')` are now denied
  (SP-014) as their Bash spellings are; before, the array syntax hid the
  subcommand. A PowerShell line that sets or clears `CLAUDECODE` or a Claude
  Code settings variable is denied (SP-008).

- `qsdev devenv doctor` now runs `qsdev --version` on the `qsdev` the
  project's Claude Code hooks find on PATH and fails `--check` when it does
  not satisfy the project's `qsdev_version`, naming its path and version.
  Before, it only looked the binary up, so a stale `qsdev` earlier on PATH,
  which the fail-closed selfprotect hook would run, passed. A missing `qsdev`
  now gets a PATH hint naming the running binary instead of "no nix package
  is known for qsdev".

- An org defaults file (`$QSDEV_ORG_CONFIG` or the pinned overlay) that fails
  to load, for example because it tries to loosen the built-in security
  floor, now stops every command that generates or changes the project:
  `init`, `init --update`, `update`, `enable`, `disable`, `repair`,
  `claude *` and `devenv *` exit non-zero naming the file and
  `qsdev defaults validate`, and `qsdev check` fails `config_catalog` (high).
  Before, they skipped the whole file (including its tightening entries),
  generated from the built-in defaults and exited 0, recording the skip only
  in the session log. Read-only invocations (`status`, `doctor`,
  `--dry-run`) still run on the built-in defaults and print a warning.

- `qsdev devenv doctor` enforces host version floors: devenv >= 2.1 (the
  `require_version` the generated `devenv.yaml` declares) and nix >= 2.4 (the
  first release with `nix profile` and flakes) are required, and python3 >=
  3.9 is required by projects with Python hooks. A host with devenv 1.x, nix
  below 2.4, or a required tool whose version cannot be read now fails
  `doctor --check`, which lists each problem as "Install X: <fix>", "Upgrade
  X to >= Y: <fix>" or "Could not determine X version (need >= Y)" instead of
  "All required tools are present.". The full report marks such a required
  tool as failed rather than a warning, and the JSON report gains each tool's
  `min_version`. No generated files change.
- `qsdev devenv setup`, the init/join auto-setup and the bootstrap devenv step
  verify what they installed. Setup re-runs the checks and fails (exit
  non-zero) when a selected tool is still missing from PATH or below its
  floor, naming the PATH fix or the version found; auto-setup prints
  "Prerequisites installed." only after that verification passes. The
  bootstrap step upgrades a devenv below 2.1 instead of reporting it
  "already installed", and fails when the version on PATH is still too old
  afterwards. An installed nix below 2.4 is not reinstalled: the Nix
  installer cannot upgrade it, so setup and doctor point at the in-place
  upgrade instead. The init/join prerequisite gate applies the same floors,
  so an outdated devenv or nix reaches the verified auto-setup instead of
  being reported "OK". Setup also fails when it leaves a required tool
  missing or outdated (an installed nix below 2.4, or a tool deselected at
  the prompt), listing each with its upgrade hint.
- `qsdev devenv doctor` (and the MCP `doctor` tool) require the programs the
  project's Claude Code hooks look up on PATH, including the interpreter of
  each hook script and programs inside the generated fail-closed wrappers. A
  python3 inside the project (an activated `.venv`, devenv's
  `.devenv/state/venv`) is never run; its version is read from the
  environment's `pyvenv.cfg`. Doctor and `qsdev check` resolve hook programs
  on the PATH of the shell they run in, which they assume is the PATH Claude
  Code starts hooks with: run them from the activated devenv shell (in CI,
  inside `devenv shell`). A hook program spelled as another Python version
  (`python`, `python3.11`) or, on Windows, in another case or with a
  PATHEXT extension keeps the python3 floor. When the Claude Code settings
  cannot be read, every doctor output warns that the hook programs were not
  checked.
- The user (org) defaults file can no longer lower the built-in security
  floor. Its `security_hooks` and `unset_vars` now add to the built-in
  lists instead of replacing them, and a file that keeps a stripped
  credential in `keep_vars`, weakens a built-in compliance level (drops a
  required hook, shortens the age gate, turns off script blocking, the
  Claude audit log or license scanning, loosens the Claude permission
  preset, or renumbers it), maps a tier to a lower or weaker compliance
  level, or tiers an always-on hook out of a security level now fails to
  load (`qsdev defaults validate` names each
  field; previously it reported such a file as valid). `qsdev defaults
  validate` now checks the overlay it reports (`$QSDEV_ORG_CONFIG` or the
  home overlay) even before it is pinned, instead of silently validating
  the pinned overlay. Catalog layers now
  apply in the order built-in, user, project, so the committed project
  policy, which may only add, can no longer be erased by a user file: where
  both set the same deny set, the result is the user list plus the
  project's additions; the project file is still judged against the
  built-in catalog, so a user file never makes it fail to load. A compliance level's required hook that is listed
  only in `hook_tiers` (which never enables a hook) is rejected.

- `qsdev check` now fails (high severity, `devenv_security_floor`) when the
  hand-edited `devenv.nix`, together with `devenv.local.nix`, no longer
  enables a security git hook (the always-on hooks, the custom hooks, and
  the compliance level's required hooks) or no longer strips a credential
  variable that the generated `devenv.nix` does. Before, a `devenv.nix`
  with `ripsecrets.enable = false;` or a deleted `unsetEnvVars` entry
  passed, because the generated-file checks skip `devenv.nix`. It also
  fails when a security hook's settings (`entry`, `excludes`, ...) differ
  from the generated file, or when a module cannot be verified statically
  (`imports`, a computed `unsetEnvVars`). Disabling a formatter or linter
  hook is still allowed.

- Cloud selector variables are no longer stripped from the devenv shell
  (U11-WS4). `AWS_DEFAULT_REGION`, `GCLOUD_PROJECT`, `CLOUDSDK_CORE_PROJECT`,
  `AZURE_TENANT_ID` and `AZURE_SUBSCRIPTION_ID` pick an account context but
  grant no access, so they left the credential list and the generated
  `unsetEnvVars`; a region set by the `aws_default_region` extra used to be
  unset again right after devenv exported it. The selected AWS, GCP and Azure
  modules now add their selectors to devenv.yaml `clean.keep`, so these values
  pass through from your shell: `AWS_PROFILE`, `AWS_REGION`,
  `AWS_DEFAULT_REGION`, `CLOUDSDK_ACTIVE_CONFIG_NAME`,
  `CLOUDSDK_CORE_PROJECT`, `GOOGLE_CLOUD_PROJECT`, `ARM_SUBSCRIPTION_ID` and
  `ARM_TENANT_ID`. The `aws_default_region` extra now sets `AWS_REGION` as
  well as `AWS_DEFAULT_REGION`, and the wizard asks for the `aws_profile`
  extra. Run `qsdev init --update` to regenerate devenv.nix and devenv.yaml.
- `qsdev sandbox exec` keeps the project's control plane read-only for the
  hook categories with a writable project (formatter, generator,
  test-runner): git's hooks, config, info and modules, `.claude` (except the
  hook log directory `.claude/logs`), the project data and state
  directories, `.envrc`, `devenv.nix`/`devenv.yaml` and their `.local`
  variants, `devenv.lock`, the devenv and direnv caches (`.devenv`, except
  devenv's state directory `.devenv/state`, which holds `GOPATH` and the
  Python venv and is created beforehand in a devenv project, and `.direnv`),
  `.mcp.json`, `.qsdev.yaml` and `.qsdev.local.yaml`, and the
  npm, pnpm, Yarn, Bun and pre-commit configuration. A hook that writes one
  that exists gets "Read-only file system", and the directories holding them
  (such as `.git`) cannot be renamed away. A missing one, or a symlinked one
  such as a pre-commit configuration linked into the Nix store, cannot be
  made read-only: a hook that creates or replaces one makes the run exit 2,
  naming it, and qsdev moves what the hook left to
  `<name>.qsdev-quarantined-<time>` (nothing is deleted) and puts a replaced
  symlink back. Git staging and commits still work. The semble analytics
  hook now logs to `.claude/logs/semble-searches.jsonl` (was
  `.qsdev/analytics/`), next to the audit log, and both logging hooks
  refuse to append through a symlink planted in the log directory.
- Module deny rules now block the spellings they used to miss (U11-WS1),
  including after global options and behind an `env VAR=x` prefix. Newly
  denied for the agent:
  - Containers (docker, podman): `image pull` and `compose pull` alongside
    `pull` (a global option before the subcommand, such as `--context` or
    `-H`, is covered; `docker exec web git pull` stays allowed); `--cap-add`, `--security-opt seccomp=unconfined` /
    `apparmor=unconfined` / `systempaths=unconfined` / `label=disable` (also
    in the legacy `:` spelling), `--userns=host`, a host-root mount written
    source-last, quoted or not (`--mount type=bind,target=/h,source=/`), and
    a quoted host-root volume (`-v "/:/host"`).
  - Terraform/OpenTofu: `test`, `refresh`, `taint`, `untaint`, `console`,
    `login`, `logout`, `state replace-provider` and `workspace delete`
    (`workspace new` and `workspace select` stay allowed).
  - Bazel and bazelisk: `run` of any `@repo//...` target wherever the label
    appears (after flags or `--`), `test` or `coverage` of an `@repo//...`
    target, and every `--lockfile_mode` other than `error` (`update`,
    `refresh`, `off`, in any case, quoted, escaped or `$'...'`, or
    space-separated) on any command. A Starlark flag naming an external label
    before the run target (`bazel run --@rules_python//...=3.12 //app:main`)
    and an `@` in run arguments after `--` are over-blocked.
  - Ansible: `ansible` and `ansible-playbook` given a vault password
    (`--vault-password-file`, `--vault-pass-file`, `--ask-vault-pass`, `-J`,
    `--vault-id`); ad-hoc `ansible` runs given module arguments (`-a`,
    `--args`, which includes the default `command` module); ad-hoc runs of
    `debug`, `shell`, `command`, `raw`, `script` and `expect` (also
    `win_shell`, `win_command`, `-mshell` and any collection-qualified name
    such as `ansible.legacy.shell`), matched as the module option's value so
    `--limit webshell` stays allowed; and `ansible-pull`. Playbook runs against
    a local or dev inventory and argument-free ad-hoc runs such as
    `ansible all -m ping` stay allowed.
  - PowerShell: the installer cmdlets and PSResourceGet's `isres`/`udres`
    aliases as whole words anywhere in a PowerShell tool call (after `;`,
    `|`, `&&` or a newline, in a scriptblock, module-qualified), so
    `Update-ModuleManifest` and `Update-ScriptFileInfo` stay allowed; and
    `pwsh`/`powershell` behind an `env` prefix in Bash. In the Bash tool a
    command that mentions both `pwsh` (or `powershell`) and a cmdlet name,
    such as `git commit -m 'powershell: document Install-Module'`, is
    over-blocked. The redundant lowercase `PowerShell(...)` copies are
    dropped, since that tool matches rules case-insensitively.

  When you need one of these operations, run it yourself in a terminal. Deny
  rules from an earlier `qsdev init` (such as `Bash(bazel run @*)`) stay valid
  until `qsdev init --update` replaces them.
- `qsdev mcp serve` no longer mounts `qsdev_nix_run` by default, in any
  deployment mode (U21-02). Opt in with `--allow-nix-run`,
  `QSDEV_MCP_ALLOW_NIX_RUN=1` or `mcp_serve.allow_nix_run: true` in the user
  defaults file (`~/.config/qsdev/defaults.yaml` under the account's home,
  or an overlay pinned with `qsdev defaults pin`).
  Standalone mode now also needs `--gateway-allow-nix-run`, as gateway mode
  did, and gateway mode now needs the base opt-in as well. A project defaults
  file (`.qsdev/defaults.yaml`) that sets the new `mcp_serve` section is
  rejected. The server logs at startup whether each gated tool is mounted and
  which opt-in mounted it.

- Infrastructure endpoints in `.qsdev.yaml` are now validated whether or not
  an `infra_profile` is selected. A project whose `registry_proxy`,
  `registry_proxy_overrides` entry, `build_cache_url` or `nix_cache` uses plain
  `http` to a non-local host, embeds credentials (`user:pass@`) or names a
  documentation placeholder host, or whose `registry_proxy_paths` entry does
  not start with `/`, now fails `qsdev init` (including `--mode join` and
  `--update`) before writing anything, and `qsdev check` reports a
  `config_validation` failure. The error names the exact field (for example
  `infrastructure.registry_proxy_paths.npm`) and the fix: use `https`, supply
  credentials through the environment (the profile's `AuthEnvVar` variable),
  or write the path as an absolute path on the proxy host. Proxy paths are
  now joined onto `registry_proxy` as URL paths, so they can no longer change
  the proxy host.

- The compliance level now sets the release-age window (U09-01). Its catalog
  `age_gating_threshold_hours` (3 days for `baseline`, 7 for `enhanced`, 14 for
  `strict`) drives npm `min-release-age`, pnpm `minimumReleaseAge`, Yarn
  `npmMinimalAgeGate`, bun `minimumReleaseAge`, uv `UV_EXCLUDE_NEWER` and the
  CI `--exclude-newer`, and the package guard through the new
  `PACKAGE_GUARD_MIN_AGE_DAYS` entry in `.claude/settings.json` `env`. Each
  manager keeps its historical minimum (npm, pnpm and the package guard 3
  days; yarn, bun and uv 7 days), so no gate loosens, even under an
  organization catalog overlay with a shorter window, and `baseline` output
  is unchanged apart from that `env` entry.
  Renovate `minimumReleaseAge` and Dependabot `cooldown` use the larger of the
  infrastructure profile's delay (or an ecosystem override) and the level's
  window; `startup-github`, which had no cooldown, now gets the level's.
  `qsdev check` fails `security_config_javascript` when a JavaScript config's
  gate is below the window. The JavaScript configs (`.npmrc`,
  `pnpm-workspace.yaml`, `.yarnrc.yml`, `bunfig.toml`) are created only if
  absent: `qsdev init --update` rewrites one only while it is unchanged since
  qsdev generated it, so an existing `enhanced` or `strict` project with an
  edited file fails that check until you raise the value in the file. A
  `strict` project using uv gets `P14D`, which uv sees as a stale `uv.lock`:
  run `uv lock` inside the devenv shell after `qsdev init --update` and
  commit `uv.lock`, or the CI `uv sync --locked` fails.

- Project-root detection now stops at the git repository toplevel. A submodule
  or nested repository is its own project boundary: a `.qsdev.yaml`, `.devinit/`
  or `.qsdev/` in a directory above a nested `.git` no longer resolves from
  inside that repository, so its commands, logs and catalog overlays no longer
  act on the outer project. Projects whose markers sit at the repository
  toplevel, or in a subdirectory between the working directory and the
  toplevel (monorepos), resolve as before. If you relied on an outer project's
  configuration governing a nested repository, run `qsdev init` inside it.

- The `qsdev_mcp_list` MCP tool's `health` option now probes only the servers
  the project's `.mcp.json` configures that match a trusted definition, through
  the same gate as `qsdev mcp status`. A registry server the project does not
  configure reports `health.status` `not_configured`; a configured server that
  is not probed (a package launcher, qsdev's own server, an untrusted entry or a
  plain-http non-local URL) reports `not-probed`, the spelling `qsdev mcp
  status --json` already uses, replacing the tool's former `not_probed`.
- MCP health probes no longer follow HTTP redirects; an endpoint that answers
  with one is reported as failing, naming the refused target.
- `qsdev mcp status` and `qsdev mcp health` no longer start or dial any server
  by default. They check each `.mcp.json` entry statically, as `qsdev devenv
  doctor` does, and report whether a probe would start or dial it, or why not.
  `--json` output keeps the `servers` array, adds `"probed": false` and gives
  each server `status` `ok`, `degraded` or `misconfigured` with
  `probe_eligible` and `probe_skip_reason`. Pass `--probe` for the previous
  live check, whose JSON now adds `"probed": true`. `--probe-untrusted` now
  implies `--probe`. Without `--probe`, `mcp health` exits non-zero only when a
  server is misconfigured; a CI job that gated on liveness must add `--probe`.
- `qsdev outdated` no longer runs any ecosystem command by default. Those
  commands query package registries, and some (`mvn`, `gradle`, `bundle`,
  `mix`) download plugins or evaluate the project's build files. Without the
  new `--online` flag it prints the command each ecosystem would run and exits
  non-zero with an error naming `--online`, so a CI step cannot pass while
  checking nothing. `qsdev outdated --online` behaves as `qsdev outdated` did.
  A CI job that runs `qsdev outdated` must add `--online`.
- `devenv.nix`, `devenv.yaml`, the per-language security configs, development
  tasks and `secretspec.toml` are now generated from the same module settings
  the CI workflow already used: each language entry with the package manager
  and extras it leaves unset completed from detection (U12-12). Before, CI ran
  `uv sync --locked` for a Python project with a `uv.lock` while `devenv.nix`
  set up pip and no `uv`. This applies to `qsdev devenv init` with fresh
  detection as well as to recorded `.qsdev.yaml` entries, so a project whose
  recorded language entry lacks a detected package manager or extra (for
  example Python with `uv.lock`) will see a `devenv.nix` diff on the next
  `qsdev init --update`. `devenv.yaml` and `secretspec.toml` generation now
  also fail on a language no module implements, as `devenv.nix` generation
  already did.

### Fixed

- The generated `devenv.nix` for an Elixir project failed to evaluate: qsdev
  and devenv's Elixir module both defined the `mix-format` hook's package at
  the same priority, which Nix rejects as "defined multiple times" (U10-01).
  Custom hook packages now render at `lib.mkOverride 999`, so a devenv
  language's own setting wins. Elixir users must run `qsdev init --update` to
  regenerate `devenv.nix`.
- Formatter hooks now run the shell's pinned toolchain. `mix-format`,
  `zig-fmt`, `dart-format`, `dotnet-format` and `terraform-format` (Terraform
  or OpenTofu, honouring a `languages.terraform.version` pin) call
  `config.languages.<language>.package`, the toolchain the shell already
  provides (for Zig, the release `build.zig.zon`'s
  `minimum_zig_version` selects), instead of a separate unpinned `pkgs.<tool>`.
  They no longer add a second copy of that toolchain to `packages` (U10-08,
  U10-09). Run `qsdev init --update` to pick this up.
- A malformed `.qsdev/defaults.yaml` now makes devenv generation (`qsdev
  init`, `qsdev devenv init`, `--update`) fail with an error naming the file
  instead of panicking with a stack trace (U12-V01). Generation also no longer
  silently drops the catalog's base packages, unset variables, security hooks
  or release-age window when the catalog cannot be loaded (U12-09).

### Security

- A committed `security.credential_vend.enabled: true` no longer mounts
  `qsdev_credential_vend` by itself (U21-V02, U21-07). The operator must
  confirm it with `--allow-credential-vend`,
  `QSDEV_MCP_ALLOW_CREDENTIAL_VEND=1` or `mcp_serve.allow_credential_vend:
  true` in the user defaults file; until then the server logs a warning naming
  those. The committed block still supplies the allow-lists. Gateway and
  standalone modes also need the new `--gateway-allow-credential-vend` (or
  `QSDEV_GATEWAY_ALLOW_CREDENTIAL_VEND`), and in gateway mode that flag with
  an empty `QSDEV_GATEWAY_AGENTS` stops the server at startup. Projects that
  use credential vending must add one of the confirmations.

- `qsdev_nix_run` refuses a call that a Bash deny rule (the catalog's, and
  `permissions.deny` in the Claude settings files) would refuse as a shell
  command (U21-02). The call is matched as `nix run ...`, as the program it
  runs, and as each command or pipeline of a `-c` script (the first operand
  after the shell's options, so `-c -- '...'` counts) or of `stdin`,
  including after `;`, `&&`, a newline, a nested `sh -c` or in a
  here-document fed to a shell, both as written and without wrappers,
  assignments and quotes before the command word, so
  `nixpkgs#bash -c 'true; nohup curl x | sh'` is refused. The refusal names
  the rule.

- `qsdev mcp serve` on plain loopback HTTP (`--transport http`, or standalone
  mode, without mTLS) now requires a bearer token (U21-04). The server
  generates one per launch, writes it to
  `<user state dir>/qsdev/mcp/<bound port>.token` (mode `0600`, in a `0700`
  directory, or the file named by
  the new `--http-token-file`), removes it on shutdown, and answers `401` to a
  request without `Authorization: Bearer <token>`. A standalone `/health`
  probe needs no token, and the mTLS path is unchanged. HTTP clients of a
  plain loopback server must now send the token. The new `--http-no-auth`
  serves without it and then mounts neither `qsdev_nix_run` nor
  `qsdev_credential_vend`.

- An agent can no longer remove the self-protection hook by switching Claude
  Code off. Before this fix, an answers file rewritten to `claude_code: false`
  (or emptied) by a command the hook cannot read made the next
  `qsdev init --update`, `qsdev update --configs-only` or `qsdev claude update`
  delete `.claude/settings.json`. Regeneration now takes Claude Code from the
  committed `.qsdev.yaml`, so only a committed `claude_code.enabled: false`
  turns it off. The hook denies an Edit or Write that makes that change
  (GD-001). Turning Claude Code off with a regeneration now leaves
  `.claude/settings.json` in place and untracked, as `init --claude-code=false`
  already did. Only `qsdev teardown` deletes it. A tool can no longer own
  `.claude/settings.json`, or a directory that holds it, exclusively: the
  catalog rejects such a declaration from an org or project overlay, and
  `disable` never deletes the file (U18-01).
- A project without a committed `.qsdev.yaml` (one set up with only
  `qsdev claude init`) no longer honours an always-on opt-out recorded only in
  the answers file. `claude update`, `claude add-hook`/`add-skill`
  and `claude init` now reconcile against the committed config the same way
  `init --update` does, so a hand-edited `safety_block_opt_out: true` with
  `enabled_tools.attach-guard: false` no longer drops package-guard there.
  `qsdev disable <always-on tool> --force` now refuses in such a project,
  because the opt-out would not survive the next regeneration: run
  `qsdev init` first to create `.qsdev.yaml`.
- The self-protection hook's `find` check now derives its probe paths from the
  protected-path table, so a `find` that selects the answers file, the devenv
  answers mirror, `.envrc` or the org overlay by name pattern and runs a
  mutating action (`find . -name '*init-answers.yaml' -exec sed -i ... {} +`,
  `find ~ -name defaults.yaml -path '*qsdev*' -delete`) is denied, as the same
  shape already was for `.claude/settings.json`.
- `qsdev repair` reconciles the saved answers against the committed
  `.qsdev.yaml` before regenerating, as `update` and `enable`/`disable`
  already did. A `safety_block_opt_out` written only into the answers file no
  longer makes `repair --force` drop the package-guard hook; only the opt-out
  `qsdev disable attach-guard --force` commits does. A CI test now runs every
  command that writes `.claude/settings.json` (`init`, `init --update`,
  `init --mode join`, `claude init`/`update`/`add-hook`, `enable`, `disable`,
  `repair`) against hostile answers files and requires the self-protection
  hook every time and package-guard unless that opt-out is committed (B1, B2,
  B5).
- Generated Claude Code settings register the package-guard hook and write
  `.claude/hooks/package-guard.py` unless the safety-block opt-out
  (`qsdev disable attach-guard --force`) is recorded. A hand-edited
  `hooks.safety_block: false` no longer removes it: that answer now only
  mirrors the opt-out and is rewritten on the next regeneration (B1).
- Generated Claude Code settings always register the `qsdev selfprotect`
  hook. The `hooks.self_protection` answer is kept for schema compatibility
  but ignored: projects that saved `self_protection: false` gain the hook on
  the next `qsdev claude update`, `qsdev init --update` or other
  regeneration, and the saved answers are rewritten to `true` (U18-01). The
  `qsdev claude` subcommands also record `claude_code: true` in the answers
  they save, since the settings they generate always configure Claude Code.
- The self-protection hook now blocks agent writes, deletes, moves and copies
  out of the repository for the inputs the settings generator trusts: the
  state directory (`.devinit/`, holding the answers and state manifest), the
  devenv answers mirror (`.devenv/.qsdev-answers.yaml`), `.envrc`, and the org
  overlay (`~/.config/qsdev/` and the file named by `QSDEV_ORG_CONFIG`),
  including `$HOME`/`${HOME}` and `$QSDEV_ORG_CONFIG` spellings and deleting
  or replacing a directory above the overlay (such as `rm -rf ~/.config`).
  An agent command that sets `QSDEV_ORG_CONFIG` (`QSDEV_ORG_CONFIG=... qsdev
  claude update`, `export QSDEV_ORG_CONFIG=...`) is denied too, since it
  would point a regeneration at an unprotected file. qsdev now reads the
  default overlay below the home directory your operating system's user
  database records for your account, not below `$HOME` or `%USERPROFILE%`,
  so setting or clearing `HOME` for a qsdev run, however it is spelled and
  even inside a script the agent wrote, no longer moves the overlay. Accounts
  that come from a directory service (SSSD, LDAP or Active Directory, NIS,
  systemd-homed), which the static release builds cannot see in
  `/etc/passwd`, are resolved through `getent`, run only from a fixed system
  path with an empty environment. When the account still cannot be resolved
  (an arbitrary container uid, say), qsdev reads no home overlay at all
  rather than one below `$HOME`, and warns when it finds one there; point
  `QSDEV_ORG_CONFIG` at it to use it. If you keep an overlay below a `HOME`
  that differs from your account's home directory, move it there or point
  `QSDEV_ORG_CONFIG` at it. As defense in depth the hook also denies an agent
  command line that runs qsdev and sets or clears `HOME` or `USERPROFILE`:
  assignments, `export`/`declare`/`local` (including a name computed from a
  variable, as in `export $X=...`, and namerefs), `unset`, for and select
  loop variables, `read`, `mapfile`, `printf -v`, `getopts`, arithmetic,
  `${HOME:=...}`, `env -u HOME` or `env -i` in any option spelling, and
  `exec -c` (bash's `env -i`, which also drops `QSDEV_ORG_CONFIG`). It
  recognises qsdev when it is quoted or brace-expanded too (`$'qsdev'`,
  `{qsdev,}`). The hook does not see inside a script file the agent runs
  (`HOME=/tmp/x sh ./script.sh`); the account-anchored lookup is what covers
  that case. A file that sets the environment qsdev later runs in, as the
  generated environment and your shell load it (`devenv.yaml`,
  `devenv.local.yaml`, `devenv.nix`, `devenv.local.nix`, and the bash, zsh,
  ksh, fish and PowerShell startup files), may not be changed by the agent
  where it sets or removes `QSDEV_ORG_CONFIG` (SP-015). The agent may change
  these files only with Edit or Write, which check that; a shell command that
  writes one is denied (`echo ... >> ~/.zshrc`, `sed -i`, `cp`, `mv`, `rm` or
  `ln` onto one, `git restore devenv.nix`, an editor such as
  `vim ~/.bashrc`), while reading one stays allowed (`cat`, `grep`,
  `source` or `.`, `sed -n` whose script has no `w`, `W` or `e`, and the
  source of `ln -s`). `.env` and `.envrc.local` are not covered: the
  generated environment loads neither (its devenv.nix disables dotenv). SP-015
  is defense in depth and does not see a relocation spelled another way: a
  `.nix` file that `devenv.local.nix` or `devenv.local.yaml` imports, a
  fragment a startup file sources (`~/.bash_aliases`, `~/.bashrc.d/`, fish's
  `conf.d`, `~/.config/environment.d`, direnv's `direnvrc`), or a variable
  name built through an encoding; the overlay pin described below is
  what covers those. Setting `HOME` for other programs
  (`HOME=$(mktemp -d) go test ./...`) stays allowed. Deleting or moving a
  directory above the overlay is denied also through `~name`
  (`rm -rf ~alice/.config`) and when the hook cannot resolve the path
  but its literal end is such a directory (`rm -rf "$(cat f)/.config"`,
  `H=~; rm -rf $H/.config`, `rm -rf "$(echo ~)"/.conf*`), when it is
  relative to a `cd` the hook cannot resolve (`cd $HOME/.config && rm -rf
  qsdev`), and when it goes through `trash-put`, `gio trash`/`gio move` or
  `tar --remove-files`.
  What makes the overlay location hold is not the hook but a pin: qsdev reads
  the org overlay you approved with `qsdev defaults pin` (for the project, or
  with `--global` for every project without a pin of its own), and without a
  pin only the overlay in your account's home directory
  (`~/.config/qsdev/defaults.yaml`, located through the user database, not
  `HOME`). This holds for every run, yours included: whether a human runs
  qsdev cannot be told reliably (an agent can drop `CLAUDECODE` and fake a
  terminal with `script`), so a `QSDEV_ORG_CONFIG` that names another file,
  however it was set (a command line, a devenv import, a sourced shell
  fragment, a variable name spelled through an encoding), is ignored with a
  warning until you pin it. `qsdev defaults pin` is a command only a human
  may run: qsdev refuses it inside an agent session or without a terminal
  (a check an agent can spoof, so it is not what holds), and the
  self-protection hook (SP-014) denies it to the agent through wrappers
  (`env -u CLAUDECODE script -qec '...'`, `sh -c`, `eval`) and when a word
  of it is computed: a command word or subcommand word built from a
  variable, a command substitution, a brace expansion or a glob
  (`Q=qsdev; $Q defaults pin`, `qsdev defaults $P`,
  `qsdev defaults $(echo pin)`), or supplied by xargs
  (`echo pin | xargs qsdev defaults`), is treated as the sensitive word it
  may stand for. The same holds for every other human-only command. A
  command whose program and subcommand words are all computed (`$A $B`)
  is recognised only when the line assigns `qsdev` to a variable
  (`Q=qsdev; $Q $S`). The CLI named anywhere else (an argument, a quoted
  string, an assignment's value) with a human-only subcommand written out
  after it counts whichever program receives the text, since many run text
  as code (`trap "..." EXIT`, `git rebase -x "..."`, `x="..."; $x`,
  `GIT_EDITOR="..."`); text that only mentions such a command
  (`git commit -m "explain qsdev teardown"`, `grep -l "qsdev teardown"
  docs/*`) is denied too, as before, with a message saying it mentions the
  command, so reword it or run it yourself. The CLI named as data and
  followed by a glob or a variable is not a computed command, so
  `grep -rn qsdev internal/*.go` and `rg qsdev *.md` stay open, as do
  `--help` and `--version` (except `--version` before a computed word,
  which `self-update --version` may take as its value).
  Pins are kept in
  `~/.config/qsdev/org-overlay-pins.yaml`, which the hook protects with the
  overlay, so nothing the agent does in a checkout (`git clean -fdX`, a fresh
  clone) removes one; an overlay below the project or the temporary
  directory is never pinned. `qsdev check` (`org_overlay_pinned`, which names
  the overlay it reads and whether it is pinned) and `qsdev devenv doctor`
  report an overlay qsdev ignores. If you set `QSDEV_ORG_CONFIG`, run
  `qsdev defaults pin` once at your terminal; a CI job, which has no
  terminal, places its overlay at `~/.config/qsdev/defaults.yaml` instead.
  An account the user database has no entry for (an arbitrary container or
  CI uid such as `docker --user 1001`) has no home directory to keep a pin
  in, so there qsdev reads the overlay `QSDEV_ORG_CONFIG` names, as before,
  unless it lies below the project or the temporary directory, and
  `qsdev defaults pin` explains that no pin is needed. A lookup that fails
  for another reason (an unreachable directory service) reads no overlay.
  Execution contexts that do not inherit the environment the hook checks
  (`systemd-run --user`, a `tmux` session started earlier) are not modelled
  by the hook; the pin is what covers them.
  The overlay is protected at the location the hook process sees; a variable
  set outside the agent's commands (for example in the environment Claude
  Code was started with) moves it. Reading them is still allowed, as are
  `direnv allow`/`direnv deny` (with or without `.envrc`) and `printenv`. An
  agent command that touches them, for example `git add .envrc` or
  `cp .envrc /tmp/`, is now denied: make the change yourself, or run the
  qsdev command that owns the file (`qsdev init --update`,
  `qsdev claude update`). The hook cannot yet prove every reader read-only,
  so some reads of these files are denied too: `source .envrc` and
  `. ./.envrc` (run `direnv allow`, or source it in your own shell),
  `shellcheck .envrc`, `less`/`yq` on `.devinit/.qsdev-init-answers.yaml`
  (use `cat`, `grep` or `jq`, which stay allowed, or run them yourself) and
  `unset QSDEV_ORG_CONFIG`.
- A stray or planted qsdev marker in a parent directory (for example a shared
  or temporary directory) can no longer capture root resolution, session logs
  or configuration for a git repository created beneath it (U01-01, XS-N5).
  Directories that are not inside a git repository are not yet covered.
- The project defaults file (`.qsdev/defaults.yaml`) now comes from the
  project the command acts on (U01-WS1). `qsdev init` and `qsdev devenv init`
  take it from the directory they run in, so a non-git subdirectory of
  another project no longer inherits that project's hooks and deny rules;
  every other command keeps applying the enclosing project's file. `qsdev
  init` names the file it applies (`Project defaults: <path>`). A defaults
  file another local user could have written (owned by someone else, or in
  or under a world-writable directory, such as a `.qsdev/` left `0777` in a
  shared checkout) is now refused rather than applied: `qsdev init`,
  `qsdev defaults show` and every command that reads the defaults exit 1
  with `refusing project defaults <path>` and the fix (`chmod o-w`, or
  `chown` it to yourself). Group-writable files in a user-owned project
  (umask 002) are still accepted.
- A diagnostic no longer starts processes or goes to the network unasked
  (XD-01): default `qsdev mcp status` and `qsdev mcp health` used to start the
  trusted stdio servers and dial trusted remote endpoints, and the generated
  lookup-docs skill runs `qsdev mcp status --json` every time it loads.
