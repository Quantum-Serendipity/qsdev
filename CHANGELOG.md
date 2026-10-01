# Changelog

All notable changes to qsdev are recorded in this file. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

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

### Security

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
  regeneration, and the saved answers are rewritten to `true` (U18-01).
- The self-protection hook now blocks agent writes, deletes, moves and copies
  out of the repository for the inputs the settings generator trusts: the
  state directory (`.devinit/`, holding the answers and state manifest), the
  devenv answers mirror (`.devenv/.qsdev-answers.yaml`), `.envrc`, and the org
  overlay (`~/.config/qsdev/` and the file named by `QSDEV_ORG_CONFIG`),
  including `$HOME`/`${HOME}` and `$QSDEV_ORG_CONFIG` spellings and deleting
  or replacing a directory above the overlay (such as `rm -rf ~/.config`).
  An agent command that sets `QSDEV_ORG_CONFIG` (`QSDEV_ORG_CONFIG=... qsdev
  claude update`, `export QSDEV_ORG_CONFIG=...`) is denied too, since it
  would point a regeneration at an unprotected file. The overlay is protected
  at the location the hook process sees; a variable set outside the agent's
  commands (for example in the environment Claude Code was started with)
  moves it. Reading them is still allowed. An agent command that touches them, for
  example `git add .envrc` or `cp .envrc /tmp/`, is now denied: make the
  change yourself, or run the qsdev command that owns the file
  (`qsdev init --update`, `qsdev claude update`).
- A stray or planted qsdev marker in a parent directory (for example a shared
  or temporary directory) can no longer capture root resolution, session logs
  or configuration for a git repository created beneath it (U01-01, XS-N5).
  Directories that are not inside a git repository are not yet covered.
