# Contributing to qsdev

Contributions are welcome. If you have ideas for new tools, plugins, ecosystem modules, configuration presets, or better defaults — I'd love to hear them.

## What's useful

- New or improved ecosystem modules (languages, frameworks, package managers)
- Security hardening configs for tools qsdev doesn't cover yet
- Better defaults and best practices for existing ecosystems
- Claude Code skills, hooks, deny rules, and MCP server configs
- devenv.sh service definitions and package sets
- Pre-commit hooks and linting configs
- Performance and ergonomic improvements to the CLI
- Documentation fixes and clarifications

## How to contribute

**Got an idea?** Open an issue. Describe what you'd like to see and why it's useful. Even rough ideas are fine — we can figure out the shape together.

**Want to submit code?** Fork the repo, make your changes on a branch, and open a PR. Keep PRs focused on one thing. Include a short description of what changed and why.

## Development setup

qsdev uses itself for development environment management.

**Prerequisites**: [qsdev](https://github.com/Quantum-Serendipity/qsdev) and
[devenv](https://devenv.sh) must be installed. On NixOS, the qsdev module
provides both.

```bash
git clone https://github.com/Quantum-Serendipity/qsdev.git
cd qsdev
direnv allow   # activates the devenv environment
go build ./...
go test ./...
```

The devenv environment provides Go tooling, pre-commit hooks, and security
scanning. Run `go vet ./...` and `golangci-lint run` before submitting.

Without direnv: `devenv shell` for manual activation.

`devenv shell` is the only development environment. `flake.nix` builds the
`qsdev` package and defines no `devShells`, so `nix develop` is not supported.
To add a development tool (for example `cosign`), run
`qsdev devenv add-package <name>`, which adds it to `devenv.nix`; do not add a
`devShells` output to `flake.nix`. `TestFlakeHasNoDevShells` enforces this.

### Bumping OpenGrep

The prebuilt OpenGrep CLI is packaged in `nix/opengrep/default.nix` (qsdev
embeds that file into projects that enable opengrep). To bump it, update
`version` and the per-platform `hash` values there, then run:

```bash
nix/opengrep/test-packaging.sh                # build, smoke-test, run the rule library
nix/opengrep/test-packaging.sh --hashes-only  # verify every platform's release hash
```

Both work from any directory and use the flake-pinned nixpkgs. CI runs the
same script in the `opengrep-nix` job on Linux and macOS (x86_64 and arm64),
plus `--hashes-only` once, so a wrong hash or a broken install path marks the
PR red. `nix/opengrep_ci_test.go` fails `go test` if that job loses a platform,
a step, or is made conditional or non-blocking.

## Dependency updates

Dependabot is this repository's only dependency bot
(`.github/dependabot.yml`). It covers Go modules, GitHub Actions, the Docker
base images and `flake.lock`, and every entry waits out a cooldown of at least
the catalog's baseline age gate (3 days; 14 days for Go major versions).

Dependabot cannot run the repository's generators, so
`.github/workflows/dependabot-fixup.yml` finishes each Dependabot branch with
one commit:

- **Go modules:** `go mod vendor`, which also removes vendored packages the new
  module versions dropped.
- **GitHub Actions:** `go generate ./internal/cigeneration/`, which syncs the
  action pin catalog with the workflows.
- **Nix:** copies `flake.lock`'s nixpkgs lock into `devenv.lock`, so both pin
  the same revision. `TestNixpkgsLocksAgree` fails if the two locks disagree.

### Dependabot fixup token

The fixup pushes with a fine-grained personal access token, because a push made
with the workflow's `GITHUB_TOKEN` would not trigger CI on the new commit. To
set it up:

1. Create a fine-grained PAT with **Repository access: only this repository**
   and **Repository permissions: Contents: Read and write**. Grant nothing
   else.
2. Store it as a **Dependabot** secret (Settings > Secrets and variables >
   Dependabot) named `DEPENDABOT_FIXUP_TOKEN`. Workflows that Dependabot
   triggers can read only Dependabot secrets, not Actions secrets.
3. Rotate it before it expires. Without it, the fixup job fails with an error
   that names the secret, rather than leaving the PR silently red.

The workflow's own `GITHUB_TOKEN` stays `contents: read`. The job runs only on
pushes by `dependabot[bot]` to `dependabot/**` branches, and executes no code
from the branch except `go mod vendor`. The pin generator it runs on GitHub
Actions branches is this repository's own code plus its vendored dependencies
at the versions on `main`, because those branches change workflow YAML and
never `go.mod` or `vendor/`.

**Rebase caveat:** Dependabot stops rebasing a branch once someone else has
committed to it, and the fixup commit counts. If a Dependabot PR falls behind
`main` or conflicts, comment `@dependabot recreate`. Dependabot rebuilds the
branch from scratch and the fixup runs again.

### CI tool pins

goreleaser, golangci-lint, gotestsum and govulncheck are not covered by any
Dependabot ecosystem. Their versions live only in `.github/tool-versions.env`.
Each job that uses one loads that file into `$GITHUB_ENV` and references
`${{ env.KEY }}`, and `TestToolVersionsSingleSource` fails if a workflow
hard-codes one of these versions.

Nix is pinned there too: `NIX_VERSION` and one `NIX_SHA256_<SYSTEM>` digest
per runner platform. CI jobs install it with `.github/scripts/install-nix.sh`,
which reads those pins, checks the official release tarball against its
digest before extracting it, and runs the bundled installer. The repository's
Actions allowlist admits no Nix action, and `TestWorkflowActionsAllowlisted`
fails if a workflow uses an action outside that allowlist.

`.github/workflows/tool-pins.yml` runs `scripts/bump-tool-pins.sh` every
Monday. The script moves each pin to the newest exact release that is at least
3 days old, never to an older one, and opens or updates a single PR from the
`tool-pins/bump` branch. To add a tool, add a `KEY=vX.Y.Z` line with a
`# source:` line above it (see the file's header). To bump a tool by hand, edit
the file, still respecting the 3-day age gate.

### Tool pins token

The tool pins PR is opened with a fine-grained personal access token, because a
PR opened with the workflow's `GITHUB_TOKEN` would not trigger CI. To set it
up:

1. Create a fine-grained PAT with **Repository access: only this repository**
   and **Repository permissions: Contents: Read and write** and **Pull
   requests: Read and write**. Grant nothing else. This is a separate token
   from `DEPENDABOT_FIXUP_TOKEN`, which does not need to open PRs.
2. Store it as an **Actions** secret (Settings > Secrets and variables >
   Actions) named `TOOL_PINS_TOKEN`.
3. Rotate it before it expires. Without it, the workflow fails with an error
   that names the secret.

The workflow runs only on its schedule or from the Actions tab
(`workflow_dispatch`), and its own `GITHUB_TOKEN` stays `contents: read`.
`TestToolPinsWorkflowScope` enforces this.

### Updated by hand

`devenv.lock`'s other inputs (devenv modules, git-hooks, flake-compat,
gitignore) are not managed by Dependabot. Update them by hand once a month:

1. Run `devenv update` in the devenv shell.
2. Check that each new revision is at least 3 days old (the catalog's baseline
   age gate). Hold back any input whose new revision is newer.
3. Make sure `nodes.nixpkgs.locked` in `devenv.lock` still matches `flake.lock`.
   `devenv update` also moves nixpkgs, so restore it from `flake.lock` if it
   changed. Dependabot owns the nixpkgs bump, and `TestNixpkgsLocksAgree`
   fails if the two locks disagree.
4. Run `go test ./nix/` and commit `devenv.lock`.

## Guidelines

- Keep changes focused. One PR, one concern.
- Follow existing code style and patterns.
- Add tests for new functionality.
- Don't introduce copyleft dependencies. Apache-2.0, MIT, BSD, and ISC are fine.
- Security-sensitive changes should note the threat model impact.

## License

By contributing, you agree that your contributions will be licensed under the [Apache-2.0 License](LICENSE).
