# Changelog

All notable changes to qsdev are recorded in this file. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

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
