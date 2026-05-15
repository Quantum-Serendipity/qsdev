# Shell/Workstation, IDE Config & Tool Detection Module Design

Implementation unit designs for three expansion categories amending Phases 7, 8, and 10 of the qsdev plan. These units add personal workstation configuration, IDE config generation, and non-language ecosystem detection modules.

---

## Part A: Shell/Workstation Configuration (Phase 10 Amendment)

Phase 10 currently covers distribution and self-bootstrapping. These units add `qsdev setup --shell` as a personal workstation configuration mode that manages shell fragments in `~/.qsdev/shell/`. This is distinct from per-project devenv — these are personal tools installed system-wide via Nix profile.

---

### Unit 10.6: Shell Fragment Directory & Init System

**Description:** Implement the `~/.qsdev/shell/` fragment directory structure and the `init.sh` entry point that engineers source from their shell RC file.

**Context:** gdev's existing `qsdev setup` command (Phase 9) installs system prerequisites. `qsdev setup --shell` extends this to personal developer tooling. The key design constraint is non-destructive operation: gdev never modifies `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish` directly. Instead, it generates shell fragments in `~/.qsdev/shell/` and prints instructions for the engineer to add a single source line. The fragment architecture allows gdev to regenerate any fragment without risking user customizations. Shell detection uses `$SHELL` and `basename` to determine bash/zsh/fish, generating appropriately formatted fragments for the detected shell.

**Desired Outcome:** `qsdev setup --shell` creates `~/.qsdev/shell/` with an `init.sh` (or `init.fish`) that sources all fragments, and prints a one-liner the engineer pastes into their RC file. Re-running the command regenerates fragments idempotently without requiring the engineer to re-add the source line.

**Steps:**
1. Detect current shell from `$SHELL` environment variable. Support bash, zsh, fish. Fall back to bash if unrecognized.
2. Create `~/.qsdev/shell/` directory if it does not exist.
3. Generate `~/.qsdev/shell/init.sh` (bash/zsh) or `~/.qsdev/shell/init.fish` (fish):
   - Bash/zsh: sources all `*.sh` fragments in `~/.qsdev/shell/` via a glob loop.
   - Fish: sources all `*.fish` fragments in `~/.qsdev/shell/` via `for f in ~/.qsdev/shell/*.fish; source $f; end`.
4. Generate a `.shell-setup-state.json` in `~/.qsdev/` tracking: shell type, last setup timestamp, fragment versions, installed packages list.
5. Print instructions to the engineer:
   - Bash: `echo 'source ~/.qsdev/shell/init.sh' >> ~/.bashrc`
   - Zsh: `echo 'source ~/.qsdev/shell/init.sh' >> ~/.zshrc`
   - Fish: `echo 'source ~/.qsdev/shell/init.fish' >> ~/.config/fish/config.fish`
6. Check if the source line already exists in the RC file (read-only check). If present, skip the instruction and print "Shell integration already configured."
7. Use gdev's existing atomic write pipeline for all fragment writes (write to temp, rename).

**Acceptance Criteria:**
- [ ] Detects bash, zsh, and fish from `$SHELL`
- [ ] Creates `~/.qsdev/shell/` directory structure
- [ ] `init.sh` sources all `*.sh` fragments
- [ ] `init.fish` sources all `*.fish` fragments
- [ ] Never modifies RC files directly
- [ ] Prints correct source instruction per detected shell
- [ ] Detects existing source line and skips instruction
- [ ] State file tracks setup metadata for idempotency
- [ ] Re-running `qsdev setup --shell` is idempotent

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 2. Developer Productivity CLI Tools` — modern coreutils list, shell integration requirements
- `artifacts/os-prerequisite-detection-research.md § 3. Shell Integration` — shell detection, RC file locations, completion installation patterns
- `research-spikes/gdev-dx-polish/research.md § Shell & Environment Integration` — shell fragment architecture, non-destructive design

**Status:** Not Started

---

### Unit 10.7: Modern Coreutils Installation via Nix Profile

**Description:** Install the curated modern coreutils bundle system-wide via `nix profile install` so they are available outside of any project's devenv shell.

**Context:** Per-project devenv tools only exist inside `devenv shell`. Personal productivity tools (ripgrep, fd, bat, etc.) should be available everywhere — in the home directory, in repos without devenv, in system administration tasks. Nix profile installs packages to `~/.nix-profile/` which is on PATH for all shells. This is the correct layer for personal tools, distinct from devenv (per-project) and NixOS system packages (per-machine). The package list is curated from the consulting-daily-driver research: these are the tools senior engineers expect on any workstation.

**Desired Outcome:** After `qsdev setup --shell`, all curated coreutils are available in any terminal session without entering a devenv shell. Packages already installed are skipped.

**Steps:**
1. Define the curated package list (nixpkgs attribute names):
   - `ripgrep` — grep replacement
   - `fd` — find replacement
   - `bat` — cat replacement with syntax highlighting
   - `fzf` — fuzzy finder
   - `jq` — JSON processor
   - `yq-go` — YAML processor (Go version, single binary)
   - `delta` — git diff pager
   - `eza` — ls replacement
   - `zoxide` — smart cd
   - `starship` — cross-shell prompt
   - `sops` — secrets-in-files encryption
   - `age` — modern file encryption (sops backend)
2. Check which packages are already installed via `nix profile list` parsing.
3. Install missing packages via `nix profile install nixpkgs#<pkg>` for each.
4. Record installed packages in `.shell-setup-state.json` for future idempotency and uninstall support.
5. Handle errors gracefully: if a single package fails to install (e.g., nixpkgs channel mismatch), log the error, continue with remaining packages, and report failures at the end.
6. Support `--dry-run` flag to show what would be installed without installing.

**Acceptance Criteria:**
- [ ] All 12 packages install successfully via Nix profile
- [ ] Already-installed packages are skipped (idempotent)
- [ ] Failed installs do not block remaining packages
- [ ] `--dry-run` shows planned actions without executing
- [ ] Packages are available outside devenv shells (in any terminal)
- [ ] State file records which packages gdev installed (for future `qsdev setup --shell --remove`)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 2. Developer Productivity CLI Tools` — tool catalog with nixpkgs names, commonality assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 12. Secrets Management` — sops + age as standard secrets tooling
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § Summary: Tier 1` — essential tools list

**Status:** Not Started

---

### Unit 10.8: Shell Aliases & Coreutils Configuration Fragments

**Description:** Generate shell fragments that configure aliases, shell integrations, and tool-specific settings for the installed modern coreutils.

**Context:** Installing the tools is necessary but not sufficient. `bat` needs to be aliased to `cat`, `eza` to `ls`, `rg` to `grep` for muscle-memory compatibility. `fzf` needs shell keybindings (Ctrl-R for history, Ctrl-T for file picker). `zoxide` needs shell init to hook into `cd`. `delta` needs gitconfig integration. These are the configuration steps that transform individual binaries into a cohesive "modern shell" experience. All configuration is fragment-based — each concern gets its own file that `init.sh` sources.

**Desired Outcome:** After sourcing `init.sh`, the engineer has modern aliases, fzf keybindings, zoxide hooked into cd, delta as the git pager, and bat as the default pager.

**Steps:**
1. Generate `~/.qsdev/shell/aliases.sh` (bash/zsh) or `aliases.fish` (fish):
   - `alias ll='eza -la --git --icons'`
   - `alias la='eza -a'`
   - `alias lt='eza --tree --level=2'`
   - `alias cat='bat --paging=never'`
   - `alias grep='rg'`
   - `alias find='fd'`
   - `alias diff='delta'`
   - `export MANPAGER="sh -c 'col -bx | bat -l man -p'"` (bat as man pager)
   - `export PAGER='bat'`
2. Generate `~/.qsdev/shell/fzf.sh` (or `fzf.fish`):
   - `export FZF_DEFAULT_COMMAND='fd --type f --hidden --follow --exclude .git'`
   - `export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"`
   - `export FZF_ALT_C_COMMAND='fd --type d --hidden --follow --exclude .git'`
   - Source fzf shell integration: `eval "$(fzf --bash)"` / `eval "$(fzf --zsh)"` / `fzf --fish | source`
3. Generate `~/.qsdev/shell/zoxide.sh` (or `zoxide.fish`):
   - `eval "$(zoxide init bash)"` / `eval "$(zoxide init zsh)"` / `zoxide init fish | source`
4. Generate `~/.qsdev/shell/coreutils.sh` (or `coreutils.fish`):
   - `export BAT_THEME="OneHalfDark"` (sensible default, overridable)
   - `export RIPGREP_CONFIG_PATH="$HOME/.qsdev/shell/.ripgreprc"`
5. Generate `~/.qsdev/shell/.ripgreprc`:
   - `--smart-case`
   - `--hidden`
   - `--glob=!.git`
6. Generate `~/.qsdev/shell/git-delta.sh`:
   - Run `git config --global core.pager delta` (only if not already set to delta)
   - Run `git config --global interactive.diffFilter 'delta --color-only'`
   - Run `git config --global delta.navigate true`
   - Run `git config --global delta.side-by-side true`
   - Run `git config --global merge.conflictstyle zdiff3`
   - Guard each `git config` write with a check: only set if the current value is not already the target value (idempotent).
7. Only generate fragments for tools that are actually installed (check against state file from Unit 10.7). If `eza` is not installed, do not generate the `ll` alias.

**Acceptance Criteria:**
- [ ] Alias fragment provides modern replacements for ls, cat, grep, find, diff
- [ ] fzf keybindings work for Ctrl-R (history) and Ctrl-T (file picker) in bash/zsh/fish
- [ ] zoxide hooks into cd for all three shells
- [ ] delta is configured as git pager (git config, not just alias)
- [ ] bat is configured as MANPAGER
- [ ] Fragments are only generated for installed tools
- [ ] All fragments are idempotent on re-generation
- [ ] Fish-specific syntax used for fish fragments (no bash-isms)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 2. Developer Productivity CLI Tools` — tool configuration requirements
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 1.4 Git Productivity Tools` — delta configuration (3 lines of gitconfig), Home Manager `programs.git.delta` reference

**Status:** Not Started

---

### Unit 10.9: Starship Prompt Configuration

**Description:** Generate a Starship prompt configuration with gdev-aware custom modules that display project context (active devenv, detected ecosystems, cloud profile).

**Context:** Starship is a cross-shell prompt that reads a single `starship.toml` config file. It already supports modules for git, language versions, cloud profiles, and custom commands. A gdev-aware Starship config adds value by showing: whether the engineer is inside a devenv shell, which ecosystems are active, and the current cloud profile (AWS/GCP/Azure). This transforms the prompt from a generic display into a consulting-context-aware dashboard. Starship was included in the Nix profile install (Unit 10.7).

**Desired Outcome:** After `qsdev setup --shell`, the engineer's prompt shows git status, language versions, cloud context, and qsdev devenv status via Starship.

**Steps:**
1. Generate `~/.qsdev/shell/starship.toml`:
   - Base format: `$directory$git_branch$git_status$fill$all$line_break$character`
   - Enable built-in modules: `git_branch`, `git_status`, `nodejs`, `python`, `go`, `rust`, `java`, `dotnet`, `terraform`, `aws`, `gcloud`, `azure`, `docker_context`, `kubernetes`, `nix_shell`
   - Configure `nix_shell` module to show devenv status (devenv shells set `IN_NIX_SHELL`).
   - Add custom `gdev` module:
     ```toml
     [custom.gdev]
     command = "echo '⚙'"
     when = "test -f .qsdev.yaml"
     description = "gdev-managed project"
     style = "bold cyan"
     ```
   - Configure `directory` module with truncation for deep paths.
   - Set palette to a neutral professional theme (not garish defaults).
2. Generate `~/.qsdev/shell/starship.sh` (or `starship.fish`):
   - `export STARSHIP_CONFIG="$HOME/.qsdev/shell/starship.toml"`
   - `eval "$(starship init bash)"` / `eval "$(starship init zsh)"` / `starship init fish | source`
3. If the engineer already has a `~/.config/starship.toml`, do NOT overwrite it. Instead:
   - Generate `~/.qsdev/shell/starship.toml` as the gdev version.
   - Print a message: "Existing Starship config detected at ~/.config/starship.toml. gdev's config is at ~/.qsdev/shell/starship.toml. Set STARSHIP_CONFIG to use it, or merge the gdev modules into your existing config."
   - The `STARSHIP_CONFIG` export in the shell fragment will point to the gdev version, which takes precedence when the fragment is sourced (engineer can comment it out to keep their existing config).

**Acceptance Criteria:**
- [ ] `starship.toml` generated with git, language, cloud, and nix_shell modules
- [ ] Custom gdev module shows indicator in gdev-managed projects
- [ ] Starship init runs for detected shell (bash/zsh/fish)
- [ ] Existing `~/.config/starship.toml` is not overwritten
- [ ] STARSHIP_CONFIG env var points to gdev's config
- [ ] Prompt is professional and informative, not cluttered
- [ ] Re-running regenerates config idempotently

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 2. Environment & Configuration` — starship as cross-shell prompt
- `research-spikes/gdev-dx-polish/research.md § Shell & Environment Integration` — prompt integration, devenv shell indicators

**Status:** Not Started

---

## Part B: IDE Config Generation (Phase 8 Amendment)

Phase 8 currently covers migration, update, and polish. These units add `.editorconfig` generation to `qsdev init` and `.vscode/extensions.json` generation to `qsdev enable vscode`, both using the existing atomic write pipeline and hash tracking.

---

### Unit 8.8: EditorConfig Generation

**Description:** Generate a `.editorconfig` file as part of `qsdev init` output, with ecosystem-aware formatting rules derived from detected language modules.

**Context:** EditorConfig is universally supported (natively by JetBrains, Visual Studio; via near-universal extension in VS Code) and handles only mechanical formatting: indent style, line endings, charset, trailing whitespace. The git-docs-ide research confirmed this is Level 1 (Safe) on the IDE config risk spectrum — near-zero harm, universal benefit. It eliminates mixed-indentation commits and inconsistent line endings, which are especially costly on cross-platform consulting teams. The file is editor-agnostic, making it appropriate for `qsdev init` (not gated behind an editor-specific enable command). Rules are derived from the detected ecosystem modules: Go uses tabs, Python uses 4-space indentation, JS/TS uses 2-space by default (configurable).

**Desired Outcome:** Every `qsdev init` run produces a `.editorconfig` with project-appropriate formatting rules. The file is tracked by gdev's hash system and updated via `qsdev init --update`.

**Steps:**
1. Add `.editorconfig` to the file generation pipeline in the devenv addon (alongside devenv.nix, devenv.yaml, .envrc).
2. Always generate a base section:
   ```ini
   root = true

   [*]
   charset = utf-8
   end_of_line = lf
   insert_final_newline = true
   trim_trailing_whitespace = true
   indent_style = space
   indent_size = 2
   ```
3. Add ecosystem-specific overrides based on detected modules:
   - Go: `[*.go]` with `indent_style = tab`, `indent_size = 4`
   - Python: `[*.py]` with `indent_style = space`, `indent_size = 4`
   - Rust: `[*.rs]` with `indent_style = space`, `indent_size = 4`
   - Java/Kotlin: `[*.{java,kt,kts}]` with `indent_style = space`, `indent_size = 4`
   - C#: `[*.cs]` with `indent_style = space`, `indent_size = 4`
   - JS/TS: `[*.{js,jsx,ts,tsx}]` with `indent_style = space`, `indent_size = 2` (configurable via wizard)
   - YAML: `[*.{yml,yaml}]` with `indent_style = space`, `indent_size = 2`
   - Makefile: `[Makefile]` with `indent_style = tab`
   - Markdown: `[*.md]` with `trim_trailing_whitespace = false` (trailing spaces are significant in Markdown)
4. Track the generated `.editorconfig` via the existing GeneratedState hash system.
5. On `qsdev init --update`: if unmodified, regenerate. If modified, show diff (same strategy as other generated files).
6. On ecosystem changes (new module detected on update), add the corresponding section.

**Acceptance Criteria:**
- [ ] `.editorconfig` generated on every `qsdev init`
- [ ] Base section uses UTF-8, LF, final newline, trim whitespace, 2-space default
- [ ] Go section uses tabs
- [ ] Python/Rust/Java/C# sections use 4-space
- [ ] JS/TS section uses 2-space (configurable)
- [ ] Makefile section uses tabs
- [ ] Markdown section preserves trailing whitespace
- [ ] File tracked by hash system
- [ ] Update preserves user modifications (diff strategy)
- [ ] Only detected ecosystems generate language-specific sections

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 3.4 Analysis: What's Actually Harmful?` — EditorConfig harm analysis: "essentially none"
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 3.8 Reconsidered Recommendations` — "Always generate (.editorconfig) — non-controversial"
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 3.3 The Spectrum of IDE Configuration` — Level 1 (Safe), editor-agnostic

**Status:** Not Started

---

### Unit 8.9: VS Code Extensions Recommendation Generation

**Description:** Implement `qsdev enable vscode` to generate `.vscode/extensions.json` with ecosystem-detected extension recommendations.

**Context:** The git-docs-ide research confirmed that `.vscode/extensions.json` is Level 2 (Safe) on the IDE config risk spectrum: it creates recommendations, not requirements. VS Code shows a notification banner; the developer can install all, install selectively, or dismiss entirely. This is gated behind `qsdev enable vscode` (explicit opt-in) per the "never generate editor-specific config without asking" principle. The extension mapping is a maintained lookup table: each detected ecosystem maps to a set of VS Code extension IDs. The file follows gdev's existing tool lifecycle (`qsdev enable/disable`) from Phase 12.

**Desired Outcome:** `qsdev enable vscode` generates `.vscode/extensions.json` with recommendations matching the project's detected ecosystems. `qsdev disable vscode` removes it.

**Steps:**
1. Register `vscode` as a tool in the lifecycle system (Phase 12 interface).
2. Implement the ecosystem-to-extension mapping:
   - **Always**: `EditorConfig.EditorConfig`
   - **Go**: `golang.Go`
   - **Python**: `ms-python.python`, `charliermarsh.ruff`
   - **JS/TS**: `dbaeumer.vscode-eslint`, `esbenp.prettier-vscode`
   - **Rust**: `rust-lang.rust-analyzer`
   - **Java/Kotlin**: `vscjava.vscode-java-pack`
   - **C#/.NET**: `ms-dotnettools.csdevkit`
   - **Docker**: `ms-azuretools.vscode-docker`
   - **Terraform/OpenTofu**: `hashicorp.terraform`
   - **PHP**: `bmewburn.vscode-intelephense-client`
   - **Ruby**: `shopify.ruby-lsp`
   - **Helm**: `Tim-Koehler.helm-intellisense`
   - **YAML**: `redhat.vscode-yaml`
   - **Claude Code addon active**: `anthropics.claude-code`
   - **Claude Code addon NOT active**: `GitHub.copilot` (as fallback AI assistant)
3. Generate `.vscode/extensions.json`:
   ```json
   {
     "recommendations": [
       "EditorConfig.EditorConfig",
       ...detected extensions...
     ]
   }
   ```
4. Track via GeneratedState hash system.
5. `qsdev enable vscode` creates the file; `qsdev disable vscode` removes it and cleans up GeneratedState entry.
6. `qsdev init --update` regenerates if ecosystems changed (new extensions added for newly-detected ecosystems).
7. Use the three-way merge strategy from Unit 6.2 for updates: user-added recommendations are preserved, generated recommendations are updated.

**Acceptance Criteria:**
- [ ] `qsdev enable vscode` generates `.vscode/extensions.json`
- [ ] `qsdev disable vscode` removes `.vscode/extensions.json`
- [ ] EditorConfig extension always included
- [ ] Extension list matches detected ecosystems
- [ ] Claude Code extension included when claudecode addon active
- [ ] Copilot extension included as fallback when claudecode addon not active
- [ ] User-added recommendations survive `qsdev init --update`
- [ ] File tracked by hash system
- [ ] JSON is valid and properly formatted

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 3.4 Analysis: What's Actually Harmful?` — extensions.json harm analysis: "minimal"
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 3.9 Proposed qsdev enable vscode Flow` — detection, mapping, generation flow
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 3.5 The Extension Pack Pattern` — complementary approach for firm-wide standards

**Status:** Not Started

---

## Part C: Tool Detection Modules (Phase 7 Amendment)

### Phase 7 Split Recommendation

Current Phase 7 has 19 language ecosystem modules across 4 units. Adding 4 non-language tool detection modules would bring it to 23 modules with fundamentally different detection patterns (tool/config files vs language runtimes). Recommend splitting:

- **Phase 7a: Language Ecosystem Modules — Tiers 2-4** (existing Units 7.1-7.4, 19 modules)
- **Phase 7b: Non-Language Tool Detection Modules** (new Units 7b.1-7b.4, 4 modules)

Phase 7b depends on Phase 2 (module interface) and Phase 7a (pattern established). The non-language modules follow the same `EcosystemModule` interface but detect tools, config files, and project conventions rather than language runtimes. Each module has: detection heuristics, nixpkgs package additions for devenv.nix, and CLAUDE.md section generation where applicable.

---

### Unit 7b.1: Git Platform Detection Module

**Description:** Detect git hosting platform and repository features, install platform CLIs and git enhancement tools in devenv.nix, and configure git integration.

**Context:** The git-docs-ide research identified `gh` as the highest-impact single tool addition — most consulting projects are GitHub-hosted, and `gh` enables PR workflows, CI debugging, and code review from the terminal. Git-lfs is a binary requirement: repos with LFS will not function without it. The module detects hosting platform from repository markers and remotes, and repository features from `.gitattributes` content. Tools are added to the project's devenv.nix (per-project, not system-wide). `delta` and `lazygit` are offered as opt-in productivity enhancements.

**Desired Outcome:** Projects automatically get the correct platform CLI (gh or glab) and required git tools (git-lfs when LFS is used) in their devenv.nix. A CLAUDE.md section documents the platform workflow.

**Steps:**
1. Implement `GitPlatformModule` conforming to `EcosystemModule` interface.
2. Detection heuristics (in priority order):
   - `.github/` directory exists → GitHub platform
   - `.gitlab-ci.yml` exists → GitLab platform
   - `git remote -v` output contains `github.com` → GitHub platform
   - `git remote -v` output contains `gitlab.com` or known self-hosted GitLab patterns → GitLab platform
   - `.gitattributes` contains `filter=lfs` → LFS required
3. devenv.nix package additions:
   - GitHub detected: `pkgs.gh`
   - GitLab detected: `pkgs.glab`
   - LFS detected: `pkgs.git-lfs`
   - Always offer (opt-in via wizard/enable): `pkgs.delta`, `pkgs.lazygit`
4. devenv.nix enterShell hook additions:
   - LFS detected: `git lfs install --local` (per-repo, not global)
   - delta offered: git config fragment setting delta as pager (project-scoped via `git config --local`)
5. CLAUDE.md section generation (when GitHub detected):
   ```markdown
   ## Git Platform: GitHub
   - PR workflow: `gh pr create`, `gh pr review`, `gh pr merge`
   - CI status: `gh run list`, `gh run view`
   - Issues: `gh issue list`, `gh issue create`
   ```
   (Analogous section for GitLab with `glab mr` commands.)
6. CLAUDE.md section for LFS (when detected):
   ```markdown
   ## Git LFS
   - Large files tracked via Git LFS (see .gitattributes)
   - Run `git lfs pull` after clone to fetch LFS objects
   ```

**Acceptance Criteria:**
- [ ] Detects GitHub from `.github/` directory or remote URL
- [ ] Detects GitLab from `.gitlab-ci.yml` or remote URL
- [ ] Detects LFS requirement from `.gitattributes` filter=lfs
- [ ] Adds `gh` to devenv.nix when GitHub detected
- [ ] Adds `glab` to devenv.nix when GitLab detected
- [ ] Adds `git-lfs` to devenv.nix and runs `git lfs install` when LFS detected
- [ ] Offers `delta` and `lazygit` as opt-in packages
- [ ] Generates platform-specific CLAUDE.md section with workflow commands
- [ ] Module registers in ecosystem module registry
- [ ] Unit tests cover all detection heuristics

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 1.1 Platform CLIs` — gh and glab analysis, detection heuristics, auth requirements
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 1.2 Git Extensions` — git-lfs detection via .gitattributes filter=lfs
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 1.3 Git TUI Tools` — lazygit as opt-in recommendation
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 1.4 Git Productivity Tools` — delta as strongest candidate, 3 lines of gitconfig
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 4. Git Platform Integration` — priority ranking, consulting-specific patterns

**Status:** Not Started

---

### Unit 7b.2: Documentation Tools Detection Module

**Description:** Detect documentation generators, diagram-as-code tools, and ADR tooling from project config files, install matching tools in devenv.nix.

**Context:** Documentation tools are project-level dependencies with clear detection heuristics (config files). The git-docs-ide research confirmed these fit naturally in the devenv addon as part of ecosystem detection — the same pattern as language modules. Detection of `mkdocs.yml`, `book.toml`, `*.d2`, `*.puml`, and `docs/adr/` directories signals which tools to install. Unlike language runtimes, these tools have no security configs to generate — the value is purely ensuring the CLI is available in the devenv shell.

**Desired Outcome:** Projects with documentation infrastructure automatically get the correct generator/diagram tools in devenv.nix without manual package additions.

**Steps:**
1. Implement `DocumentationModule` conforming to `EcosystemModule` interface.
2. Detection heuristics:
   - `mkdocs.yml` exists → mkdocs detected. Parse for `theme: name: material` → mkdocs-material also needed.
   - `book.toml` exists → mdbook detected.
   - `*.d2` files exist (glob `**/*.d2`, max depth 3) → d2 detected.
   - `*.puml` or `*.plantuml` files exist (glob, max depth 3) → plantuml detected.
   - `docs/adr/` directory exists → adr-tools detected.
3. devenv.nix package additions:
   - mkdocs: `pkgs.mkdocs` (Python-based; use `pkgs.python3Packages.mkdocs`)
   - mkdocs-material: `pkgs.python3Packages.mkdocs-material`
   - mdbook: `pkgs.mdbook`
   - d2: `pkgs.d2`
   - plantuml: `pkgs.plantuml`, `pkgs.graphviz` (graphviz is a plantuml dependency for many diagram types)
   - adr-tools: `pkgs.adr-tools`
4. CLAUDE.md section generation for each detected tool:
   - mkdocs: `mkdocs serve` for local preview, `mkdocs build` for production
   - mdbook: `mdbook serve` for local preview, `mdbook build` for production
   - d2: `d2 input.d2 output.svg` compilation command
   - plantuml: `plantuml input.puml` compilation command
   - adr-tools: `adr new "Title"` for creating ADRs, link to ADR directory
5. Flag PlantUML's Java dependency in the detection output: "PlantUML detected — note: requires JVM runtime (added via plantuml package)."

**Acceptance Criteria:**
- [ ] Detects mkdocs from `mkdocs.yml`
- [ ] Detects mkdocs-material theme from YAML content
- [ ] Detects mdbook from `book.toml`
- [ ] Detects d2 from `*.d2` file glob
- [ ] Detects plantuml from `*.puml`/`*.plantuml` file glob
- [ ] Detects adr-tools from `docs/adr/` directory
- [ ] Adds correct nixpkgs packages to devenv.nix for each detection
- [ ] PlantUML includes graphviz as companion package
- [ ] CLAUDE.md sections generated with tool-specific commands
- [ ] Module registers in ecosystem module registry
- [ ] PlantUML Java dependency flagged in output

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 2.1 Diagramming Tools` — d2, plantuml, mermaid-cli analysis, nixpkgs availability
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 2.2 Documentation Generators` — mkdocs, mdbook, Hugo detection heuristics
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md § 2.3 ADR Tools` — adr-tools assessment, relationship to write-adr skill
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 7. Documentation & Diagramming` — tool commonality, consulting relevance

**Status:** Not Started

---

### Unit 7b.3: API Tools Detection Module

**Description:** Detect API development tooling from project markers (proto files, OpenAPI specs, Bruno collections), install matching CLI tools in devenv.nix, and generate CLAUDE.md sections documenting API workflows.

**Context:** The api-db-mcp research identified two tiers: auto-install tools (grpcurl + buf for gRPC projects) and offer-on-detection tools (openapi-generator, redocly for OpenAPI projects; bruno for Bruno collections). The detection heuristics are straightforward config/file markers. `httpie` is offered as a general-purpose HTTP tool for any project with API-related markers. These tools have no security configs to generate — the value is ensuring correct CLI availability and documenting API interaction patterns for Claude Code.

**Desired Outcome:** gRPC projects get `grpcurl` and `buf` automatically. OpenAPI projects get spec tooling. Bruno projects get the CLI. All get CLAUDE.md sections explaining the API workflow.

**Steps:**
1. Implement `APIToolsModule` conforming to `EcosystemModule` interface.
2. Detection heuristics:
   - `*.proto` files exist (glob `**/*.proto`, max depth 4) → gRPC/protobuf detected
   - `buf.yaml` or `buf.gen.yaml` exists → buf ecosystem detected (implies protobuf)
   - `openapi.yaml`, `openapi.json`, `swagger.yaml`, or `swagger.json` exists (check root and `api/` directory) → OpenAPI detected
   - `.redocly.yaml` exists → redocly configuration detected (implies OpenAPI)
   - `*.bru` files exist or `bruno.json` exists → Bruno API collections detected
   - Any of the above detected → offer `httpie` as general API tool
3. devenv.nix package additions:
   - Protobuf detected: `pkgs.grpcurl`, `pkgs.buf`
   - OpenAPI detected: `pkgs.openapi-generator-cli`, `pkgs.redocly-cli`
   - Bruno detected: `pkgs.bruno`
   - General API (any detection): offer `pkgs.httpie` (opt-in via wizard)
4. CLAUDE.md section generation:
   - gRPC/protobuf:
     ```markdown
     ## API: gRPC / Protocol Buffers
     - Lint protos: `buf lint`
     - Generate code: `buf generate`
     - Test endpoints: `grpcurl -plaintext localhost:50051 list`
     - Breaking change detection: `buf breaking --against .git#branch=main`
     ```
   - OpenAPI:
     ```markdown
     ## API: OpenAPI Specification
     - Validate spec: `redocly lint openapi.yaml`
     - Preview docs: `redocly preview-docs openapi.yaml`
     - Generate client: `openapi-generator-cli generate -i openapi.yaml -g typescript-axios -o generated/`
     ```
   - Bruno:
     ```markdown
     ## API: Bruno Collections
     - API collections stored in `.bru` files (version-controlled)
     - Run collection: `bruno run --env <environment>`
     - GUI: `bruno` (opens Bruno desktop app)
     ```

**Acceptance Criteria:**
- [ ] Detects gRPC from `.proto` files or `buf.yaml`
- [ ] Detects OpenAPI from `openapi.yaml`/`openapi.json`/`swagger.yaml`/`swagger.json`
- [ ] Detects Bruno from `.bru` files or `bruno.json`
- [ ] Adds `grpcurl` + `buf` for gRPC projects
- [ ] Adds `openapi-generator-cli` + `redocly-cli` for OpenAPI projects
- [ ] Adds `bruno` for Bruno projects
- [ ] Offers `httpie` as opt-in for any API project
- [ ] CLAUDE.md sections include tool-specific workflow commands
- [ ] Module registers in ecosystem module registry
- [ ] Detection globs have reasonable depth limits (no scanning node_modules)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 1.3 gRPC & Protobuf Tools` — grpcurl + buf as auto-install pair, detection via .proto and buf.yaml
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 1.4 OpenAPI/Swagger Tools` — openapi-generator-cli + redocly as recommended pair
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 1.1 HTTP Clients` — httpie as human-friendly HTTP client
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 1.6 API Tools Recommendation Summary` — tier ranking

**Status:** Not Started

---

### Unit 7b.4: Database Migration Detection Module

**Description:** Detect database migration tools from project config files, install CLI binaries with system dependencies in devenv.nix, and generate CLAUDE.md sections documenting the migration workflow.

**Context:** The api-db-mcp research established a key principle: gdev does NOT choose migration tools — that is a team decision. gdev's value is removing friction: detecting the chosen tool, ensuring its CLI and native dependencies are available in devenv.nix, and documenting the workflow in CLAUDE.md so Claude Code understands how to run migrations. Some tools need system packages installed (Flyway needs JVM, diesel-cli needs libpq), which is where devenv.nix adds genuine value. Other tools are installed by the project's package manager (Drizzle via npm, Alembic via pip) and only need CLAUDE.md documentation.

**Desired Outcome:** Migration tools are detected, their CLIs made available in devenv, and CLAUDE.md documents the migration workflow (directory, commands, naming conventions) for each project.

**Steps:**
1. Implement `DBMigrationModule` conforming to `EcosystemModule` interface.
2. Detection heuristics — tools requiring devenv.nix packages:
   - `flyway.conf` exists, or `db/migration/V*.sql` pattern → Flyway detected
   - `prisma/schema.prisma` exists → Prisma detected
   - `diesel.toml` exists → diesel-cli detected
   - `atlas.hcl` or `schema.hcl` exists → Atlas detected
3. Detection heuristics — tools requiring only CLAUDE.md documentation:
   - `alembic.ini` exists or `alembic/` directory exists → Alembic detected (Python ecosystem, pip-installed)
   - `drizzle.config.ts` or `drizzle.config.js` exists → Drizzle detected (npm-installed)
   - `knexfile.ts` or `knexfile.js` exists → Knex detected (npm-installed)
4. devenv.nix package additions (for tools with system dependencies):
   - Flyway: `pkgs.flyway`
   - Prisma: `pkgs.prisma-engines` (native engine binaries that prisma npm package needs)
   - diesel-cli: `pkgs.diesel-cli`, ensure `pkgs.postgresql.lib` (libpq) is in `buildInputs` or packages
   - Atlas: `pkgs.atlas`
5. CLAUDE.md section generation for each detected tool:
   - Flyway:
     ```markdown
     ## Database Migrations: Flyway
     - Migration directory: `db/migration/` (versioned SQL files: V1__description.sql)
     - Apply migrations: `flyway migrate`
     - Check status: `flyway info`
     - Naming convention: `V{version}__{description}.sql` (two underscores)
     ```
   - Prisma:
     ```markdown
     ## Database Migrations: Prisma
     - Schema: `prisma/schema.prisma`
     - Create migration: `npx prisma migrate dev --name <description>`
     - Apply migrations: `npx prisma migrate deploy`
     - Generate client: `npx prisma generate`
     - Reset database: `npx prisma migrate reset`
     ```
   - diesel-cli:
     ```markdown
     ## Database Migrations: Diesel
     - Migration directory: `migrations/`
     - Create migration: `diesel migration generate <name>`
     - Run migrations: `diesel migration run`
     - Revert last: `diesel migration revert`
     - Schema file: `src/schema.rs` (auto-generated)
     ```
   - Atlas:
     ```markdown
     ## Database Migrations: Atlas
     - Schema: `atlas.hcl`
     - Plan migration: `atlas schema diff`
     - Apply migration: `atlas schema apply`
     - Inspect database: `atlas schema inspect`
     ```
   - Alembic:
     ```markdown
     ## Database Migrations: Alembic
     - Config: `alembic.ini`
     - Migration directory: `alembic/versions/`
     - Create migration: `alembic revision --autogenerate -m "<description>"`
     - Apply migrations: `alembic upgrade head`
     - Revert last: `alembic downgrade -1`
     - Check current: `alembic current`
     ```
   - Drizzle:
     ```markdown
     ## Database Migrations: Drizzle
     - Config: `drizzle.config.ts`
     - Generate migration: `npx drizzle-kit generate`
     - Apply migrations: `npx drizzle-kit migrate`
     - Push schema (dev): `npx drizzle-kit push`
     - View studio: `npx drizzle-kit studio`
     ```
6. For each detected migration tool, include a "Migration Safety" sub-section in CLAUDE.md:
   ```markdown
   ### Migration Safety
   - Always create a new migration file; never modify existing applied migrations
   - Test migrations against a local database before applying to staging/production
   - Ensure migrations are reversible where possible
   ```

**Acceptance Criteria:**
- [ ] Detects Flyway from `flyway.conf` or versioned SQL pattern
- [ ] Detects Prisma from `prisma/schema.prisma`
- [ ] Detects diesel-cli from `diesel.toml`
- [ ] Detects Atlas from `atlas.hcl`
- [ ] Detects Alembic from `alembic.ini` or `alembic/` directory
- [ ] Detects Drizzle from `drizzle.config.ts`/`drizzle.config.js`
- [ ] Flyway, Prisma engines, diesel-cli, and Atlas added to devenv.nix packages
- [ ] diesel-cli gets libpq dependency in devenv.nix
- [ ] Alembic and Drizzle get CLAUDE.md documentation only (no devenv.nix packages)
- [ ] Each detected tool generates a CLAUDE.md section with commands and conventions
- [ ] Migration safety section included for all detected tools
- [ ] Module registers in ecosystem module registry

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 2.1 The Core Question` — gdev detects, does not choose migration tools
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 2.2 Migration Tools by Ecosystem` — per-ecosystem tool catalog with detection heuristics
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 2.3 Migration Tools Recommendation Summary` — install vs document-only classification
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 2.4 Key Insight` — value is friction removal (CLI + native deps + CLAUDE.md docs)
- `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md § 3. Database & Data Tools` — migration tool commonality assessment

**Status:** Not Started

---

## Summary

### Units Produced

| Unit | Phase | Category | Title |
|------|-------|----------|-------|
| 10.6 | 10 (amendment) | Shell/Workstation | Shell Fragment Directory & Init System |
| 10.7 | 10 (amendment) | Shell/Workstation | Modern Coreutils Installation via Nix Profile |
| 10.8 | 10 (amendment) | Shell/Workstation | Shell Aliases & Coreutils Configuration Fragments |
| 10.9 | 10 (amendment) | Shell/Workstation | Starship Prompt Configuration |
| 8.8 | 8 (amendment) | IDE Config | EditorConfig Generation |
| 8.9 | 8 (amendment) | IDE Config | VS Code Extensions Recommendation Generation |
| 7b.1 | 7b (new sub-phase) | Tool Detection | Git Platform Detection Module |
| 7b.2 | 7b (new sub-phase) | Tool Detection | Documentation Tools Detection Module |
| 7b.3 | 7b (new sub-phase) | Tool Detection | API Tools Detection Module |
| 7b.4 | 7b (new sub-phase) | Tool Detection | Database Migration Detection Module |

### Phase Split Recommendation

Phase 7 should be split into:
- **Phase 7a**: Language Ecosystem Modules — Tiers 2-4 (existing Units 7.1-7.4, 19 modules)
- **Phase 7b**: Non-Language Tool Detection Modules (new Units 7b.1-7b.4, 4 modules covering git, docs, API, DB migration)

Phase 7b depends on Phase 2 (module interface proven) and can run in parallel with Phase 7a since the modules are independent.

### Design Principles Applied

1. **Non-destructive**: Shell setup never modifies RC files. EditorConfig and extensions.json use hash tracking with diff-on-conflict. Fragment-based architecture isolates concerns.
2. **Detect, don't assume**: Every tool detection uses concrete project markers (config files, directory patterns, file globs). No tools are installed speculatively.
3. **Layer separation**: Personal tools (coreutils) go to Nix profile (system-wide). Project tools (gh, mkdocs, grpcurl) go to devenv.nix (per-project). Editor config (EditorConfig, extensions.json) goes to project root (version-controlled).
4. **CLAUDE.md as knowledge layer**: Tool detection modules don't just install CLIs — they document workflows for Claude Code, making the AI agent effective with the project's specific tooling.
