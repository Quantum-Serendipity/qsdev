# Git Platform CLIs, Documentation Tools, and IDE Configuration Patterns

## Overview

This report covers three gap categories from the coverage matrix: **Category G (Git Platform CLIs)**, **Category H (Documentation & Diagramming)**, and **Category I (IDE/Editor Configuration)**. For each, we assess the tool landscape, nixpkgs availability, configuration requirements, detection heuristics, and whether gdev should integrate them. The IDE section revisits the explicit rejection from the DX polish spike with new evidence.

---

## 1. Git Platform CLIs & Extensions

### 1.1 Platform CLIs

#### gh (GitHub CLI)

- **Nixpkgs**: `gh`
- **What it does**: PR workflows, issue management, Actions debugging, release management, API queries, repository operations. It is GitHub's official CLI and brings the full GitHub surface area to the terminal.
- **How commonly needed**: Practically essential for any GitHub-hosted project. Claude Code's GitHub MCP server provides AI-assisted GitHub access, but `gh` is the human developer's primary tool for PR creation, CI debugging, code review, and release management. The two are complementary, not competing.
- **Configuration beyond install**: `gh auth login` (interactive or token-based). Supports multiple authenticated instances. Extensions ecosystem (`gh extension install`). Config stored in `~/.config/gh/`.
- **Detection heuristic**: `.github/` directory, or `git remote -v` showing `github.com`. Either signals a GitHub-hosted project.
- **Recommendation**: **Should install and configure.** gh is the strongest candidate in this entire category. gdev already configures a GitHub MCP server, which means it already assumes GitHub hosting for some projects. Installing gh and running `gh auth status` as part of `gdev doctor` is a natural extension. Detection: if `.github/` exists or remote points to github.com, add `gh` to devenv.nix packages and prompt for auth if not already authenticated.

#### glab (GitLab CLI)

- **Nixpkgs**: `glab`
- **What it does**: Merge requests, CI/CD pipeline management, issue management, repository operations. Officially maintained by GitLab (adopted from community project by profclems). Command structure mirrors `gh` closely — muscle memory transfers (swap `pr` for `mr`).
- **How commonly needed**: Essential for GitLab-hosted projects. GitLab is common in enterprise consulting engagements.
- **Configuration beyond install**: `glab auth login`. Supports multiple instances and self-hosted GitLab. Auto-detects authenticated hostname from git remotes.
- **Detection heuristic**: `.gitlab-ci.yml` file, or `git remote -v` showing `gitlab.com` or a known self-hosted GitLab instance.
- **Recommendation**: **Should install when detected.** Same logic as gh but for GitLab projects. Lower priority since GitHub dominates, but consulting engagements frequently use GitLab.

#### Bitbucket CLI

- **Nixpkgs**: Not in nixpkgs. Third-party tools exist:
  - `bkt` (bitbucket-cli) — mirrors gh ergonomics for Bitbucket Data Center and Cloud
  - `atlassian-cli` — Rust-based, covers Jira/Confluence/Bitbucket in one binary
- **How commonly needed**: Less common than gh/glab. Bitbucket is declining in market share but still present in enterprise Atlassian shops. App passwords are being deprecated (new creation blocked since September 2025, existing stop working June 2026), which complicates CLI auth.
- **Detection heuristic**: `git remote -v` showing `bitbucket.org`.
- **Recommendation**: **Skip for now.** No stable nixpkgs package, authentication model in flux, declining market share. If a consulting engagement uses Bitbucket, engineers can install tools manually. Revisit if a quality Bitbucket CLI lands in nixpkgs.

### 1.2 Git Extensions

| Tool | Nixpkgs | Purpose | Config Needed | Detection | Recommendation |
|------|---------|---------|--------------|-----------|---------------|
| git-lfs | `git-lfs` | Large file versioning | `git lfs install` (one-time) | `.gitattributes` with `filter=lfs` | **Auto-install when detected.** Zero-config after initial setup. LFS repos won't work without it. |
| git-crypt | `git-crypt` | Transparent file encryption in git | GPG key management | `.gitattributes` with `filter=git-crypt`, `.git-crypt/` directory | **Install when detected.** Required to work with encrypted repos. Note: has known issues with Nix flakes (encrypted files remain encrypted during build). |
| git-secret | `git-secret` | GPG-based file encryption | GPG key ring | `.gitsecret/` directory | **Install when detected.** Less common than git-crypt. Similar purpose. |
| git-filter-repo | `git-filter-repo` | Repository history rewriting | None (one-shot tool) | N/A — used ad-hoc | **Skip.** Not a daily-driver tool. Used for one-time history surgery. |

### 1.3 Git TUI Tools

| Tool | Nixpkgs | Purpose | Popularity | Recommendation |
|------|---------|---------|-----------|---------------|
| lazygit | `lazygit` | Full-featured terminal UI for git — staging, committing, rebasing, branch management | Very high. Active community, extensive tutorials. Keyboard-driven, visual layer over git. | **Strong candidate for optional install.** Most popular git TUI. "Makes you productive from day 1." |
| tig | `tig` | Lightweight ncurses viewer for git history, blame, diffs | Moderate. Focused on viewing, not editing. Lighter than lazygit. | **Optional.** Good complement to lazygit (viewer vs. manager). |
| gitui | `gitui` | Rust-based terminal UI, similar to lazygit | Moderate. Faster than lazygit in benchmarks but smaller community. | **Skip.** Overlaps with lazygit. Pick one TUI. |

**Recommendation**: Offer lazygit as an opt-in via `gdev enable lazygit` or include in a "git productivity" bundle. Do not auto-install — TUI preference is personal.

### 1.4 Git Productivity Tools

| Tool | Nixpkgs | Purpose | Config Needed | Recommendation |
|------|---------|---------|--------------|---------------|
| delta | `delta` | Syntax-highlighting pager for git diffs. Line numbers, side-by-side view, themes. | `~/.gitconfig` pager settings. Home Manager has `programs.git.delta.enable`. | **Strong candidate.** Transforms every `git diff`, `git log`, `git show`. Config is simple and non-invasive. |
| diff-so-fancy | `diff-so-fancy` | Prettier git diffs (predecessor to delta) | `~/.gitconfig` pager | **Skip.** Delta supersedes it. |
| git-absorb | `git-absorb` | Auto-fixup: assigns hunks to the correct commit in a branch | None | **Optional.** Niche but powerful for stacked workflow users. |
| git-branchless | `git-branchless` | Stacked diffs workflow, undo, smartlog for git | `.git/hooks/` integration | **Skip for gdev.** Stacked diffs is a workflow choice, not a tool requirement. Opinionated. Companies like Meta use stacked diffs but it requires team buy-in. |

**Key finding on delta**: Delta is the strongest candidate in this sub-category. It requires only 3 lines of `.gitconfig` to activate and improves every git diff operation. gdev could add delta to devenv.nix and generate the gitconfig fragment as part of the devenv enterShell hook. Home Manager already has first-class delta support (`programs.git.delta`), which validates the "generate config" pattern.

### 1.5 Summary: Git Platform CLI Recommendations

**Tier 1 — Should integrate (detect and install):**
- `gh` — when `.github/` exists or remote is github.com
- `git-lfs` — when `.gitattributes` contains `filter=lfs`

**Tier 2 — Should integrate when detected:**
- `glab` — when `.gitlab-ci.yml` exists or remote is gitlab.com
- `git-crypt` — when `.git-crypt/` exists
- `git-secret` — when `.gitsecret/` exists

**Tier 3 — Opt-in via `gdev enable`:**
- `lazygit` — personal preference TUI
- `delta` — diff improvement (could argue for Tier 2 since it's non-invasive)
- `tig` — lightweight git viewer
- `git-absorb` — power user tool

**Skip:**
- Bitbucket CLI (no nixpkgs, auth model in flux)
- gitui (overlaps with lazygit)
- diff-so-fancy (superseded by delta)
- git-filter-repo (one-shot tool)
- git-branchless (opinionated workflow)

---

## 2. Documentation & Diagramming Tools

### 2.1 Diagramming Tools

#### Mermaid CLI (mmdc)

- **Nixpkgs**: `mermaid-cli` (provides `mmdc` binary)
- **What it does**: Renders Mermaid diagram syntax to SVG/PNG/PDF from the command line.
- **Why it matters**: Mermaid has the broadest platform support of any text-to-diagram tool. GitHub, GitLab, Notion, and dozens of other platforms render Mermaid natively in markdown — no plugins or export steps needed. This makes it the default choice for diagrams-as-code in repositories.
- **Configuration**: None beyond install. Optionally a `mermaid.config.json` for theme customization.
- **Detection**: `.md` files containing ` ```mermaid ` code blocks. Or `mermaid.config.json` in repo root.
- **Recommendation**: **Install when mermaid content detected.** The CLI is needed for CI rendering, local preview, and PDF generation. If mermaid blocks exist in the repo, the developer likely needs `mmdc`.

#### D2

- **Nixpkgs**: `d2` (v0.7.1)
- **What it does**: Modern diagram scripting language. Go binary, no runtime dependencies. Better aesthetics than Mermaid for complex diagrams, friendly error messages, dark mode, autoformatting.
- **Configuration**: None beyond install.
- **Detection**: `*.d2` files in the repository.
- **Recommendation**: **Install when detected.** Growing adoption, single binary, zero config.

#### PlantUML

- **Nixpkgs**: `plantuml`
- **What it does**: The most powerful UML diagramming tool. Supports every UML diagram type plus many non-UML types. Used heavily in enterprise architecture documentation.
- **Configuration**: Requires Java runtime. This is the primary friction point.
- **Detection**: `*.puml`, `*.plantuml` files, or `plantuml` references in documentation config.
- **Recommendation**: **Install when detected, but flag Java dependency.** The Java requirement means it pulls in a significant dependency. Worth it only for projects that actually use PlantUML.

#### Graphviz

- **Nixpkgs**: `graphviz`
- **What it does**: The classic graph rendering engine (`dot`, `neato`, `fdp`, etc.). Many other tools depend on it (PlantUML can use it, Terraform graph outputs dot format).
- **Configuration**: None.
- **Detection**: `*.dot`, `*.gv` files, or as a dependency of PlantUML/other tools.
- **Recommendation**: **Install when detected or as PlantUML dependency.**

#### Excalidraw

- **Not a CLI tool.** Excalidraw is a browser-based whiteboard. The `.excalidraw` file format is JSON and can be version-controlled. VS Code has an Excalidraw extension. No CLI rendering tool exists.
- **Recommendation**: **Skip for gdev CLI.** If IDE config is reconsidered, the VS Code Excalidraw extension could be recommended.

### 2.2 Documentation Generators

| Tool | Nixpkgs | Stack | Detection | Consulting Relevance | Recommendation |
|------|---------|-------|-----------|---------------------|---------------|
| mdbook | `mdbook` | Rust | `book.toml` | Used by Rust ecosystem, internal knowledge bases | Install when detected |
| mkdocs | `mkdocs` | Python | `mkdocs.yml` | Very common in enterprise. Material theme is dominant. | Install when detected |
| mkdocs-material | `python3xxPackages.mkdocs-material` | Python | `mkdocs.yml` with `theme: material` | The de facto mkdocs theme | Install alongside mkdocs when detected |
| Hugo | `hugo` | Go | `hugo.toml`, `config.toml` with Hugo markers | Common for company blogs, docs sites | Install when detected |
| Docusaurus | Not in nixpkgs (npm package) | Node.js | `docusaurus.config.js` | Common in open-source, React ecosystem | Skip (npm install handles it) |
| Sphinx | `sphinx` (via Python packages) | Python | `conf.py` with Sphinx markers | Python ecosystem standard | Install when detected |

**Key finding**: Documentation generators are project-level dependencies that already have clear detection heuristics (config files). gdev can detect them and add the right packages to devenv.nix. The pattern mirrors ecosystem module detection — if `mkdocs.yml` exists, add `mkdocs` and detected plugins to packages.

### 2.3 ADR (Architecture Decision Record) Tools

| Tool | Nixpkgs | What it Does | Status |
|------|---------|-------------|--------|
| adr-tools | `adr-tools` (likely) | CLI to create/update ADRs in markdown format. The original ADR tool. | Mature, low maintenance. Simple bash scripts. |
| log4brains | Not in nixpkgs (npm package) | ADR management + static site publication. Hot reload, multi-project support. | Active. Better DX than adr-tools. |
| MADR | Not a tool — it's a template format | Markdown ADR template standard | Widely adopted template format |

**Context**: gdev already has a `write-adr` skill in Phase 14 that generates ADRs via Claude Code. This means the Claude Code agent is the primary ADR authoring tool, not a CLI. The question is whether to also install a CLI for non-Claude workflows.

**Recommendation**: **Low priority.** The write-adr skill covers ADR creation. If a project uses log4brains for ADR publication (detected by `log4brains.yml`), install it. Otherwise skip — the skill is sufficient.

### 2.4 Markdown Quality Tools

| Tool | Nixpkgs | Purpose | Detection | Recommendation |
|------|---------|---------|-----------|---------------|
| markdownlint-cli | `markdownlint-cli` | Markdown linting (rule-based) | `.markdownlint.json`, `.markdownlint.yml` | **Install when config detected.** Already covered by pre-commit hooks in Phase 12 if configured. |
| markdownlint-cli2 | `markdownlint-cli2` | Faster, config-driven variant | `.markdownlint-cli2.jsonc` | Same as above, pick whichever the project uses. |
| mdformat | Likely in nixpkgs | Markdown formatter (opinionated) | `.mdformat.toml` | Install when detected |
| markdown-link-check | Not confirmed in nixpkgs (npm) | Validates links in markdown | CI config references | Low priority — CI tool, not daily-driver |

### 2.5 Summary: Documentation Tool Recommendations

**Tier 1 — Detect and install:**
- `mermaid-cli` — when mermaid code blocks detected in .md files
- mkdocs/mkdocs-material — when `mkdocs.yml` detected
- mdbook — when `book.toml` detected
- markdownlint-cli/cli2 — when config file detected

**Tier 2 — Detect and install:**
- `d2` — when `*.d2` files detected
- `plantuml` + `graphviz` — when `*.puml`/`*.plantuml` files detected
- Hugo — when `hugo.toml` detected
- Sphinx — when Sphinx `conf.py` detected

**Tier 3 — Low priority:**
- ADR tools (write-adr skill covers this)
- log4brains (only if config detected)
- markdown-link-check (CI-focused)

**Architecture fit**: Documentation tools fit naturally in the **devenv addon** as part of ecosystem detection. When gdev scans a project, it already looks for `package.json`, `go.mod`, etc. Adding `mkdocs.yml`, `book.toml`, `*.d2` to the detection matrix is a straightforward extension. The tools go into `devenv.nix` packages.

---

## 3. IDE/Editor Configuration Patterns

### 3.1 The Current Rejection

The DX polish spike rejected "IDE config beyond Claude Code" with the rationale: **"Too personal, too variable."** The specific reasoning was:

> IDE configuration is deeply personal and highly variable. VS Code alone has thousands of settings. gdev cannot know whether a developer uses VS Code, Neovim, Zed, Helix, or Emacs. Claude Code is special because gdev's security model requires specific Claude Code configuration (deny rules, hooks, permissions).

However, the same spike also noted: **"For VS Code, generate `.vscode/extensions.json` (recommended extensions) at most — and only if the user opts in."** This suggests the rejection is about avoiding *opinionated, comprehensive* IDE config, not about avoiding *all* IDE-adjacent files.

### 3.2 What Other Tools Do

#### devenv.sh
- Offers a VS Code extension (`datakurre.devenv`) that provides integration
- Can generate `.devcontainer.json` for Dev Container Spec compatibility
- Does NOT generate `.vscode/settings.json` or `.vscode/extensions.json`
- Relies on direnv + nix-direnv for environment activation in VS Code

#### mise (formerly rtx)
- Does NOT auto-generate IDE configurations
- Provides three integration methods: shims in PATH, direct SDK selection (JetBrains), plugin-based (VS Code `mise-vscode`)
- VS Code extension auto-configures other extensions to use mise-managed tool paths
- Explicitly documents per-editor setup but doesn't automate it

#### devcontainers
- DOES generate IDE config — by design. The `customizations.vscode.extensions` array in `devcontainer.json` lists extensions to auto-install
- The `customizations.vscode.settings` object configures workspace settings
- JetBrains support via `customizations.jetbrains.plugins`
- This is the most opinionated IDE config pattern in wide use — and teams accept it because it's scoped to the container

#### Nix + direnv + VS Code
- Common pattern: direnv activates nix shell, VS Code direnv extension reads environment
- No VS Code config generation — relies on extension discovery
- NixOS Wiki documents manual setup patterns

**Key finding**: No major dev environment tool auto-generates VS Code workspace config. They all provide *integration extensions* and rely on the developer to configure their editor. The exception is devcontainers, where IDE config is explicitly part of the contract.

### 3.3 The Spectrum of IDE Configuration

Not all "IDE config" is equally opinionated or risky. There's a clear spectrum:

| Level | File | What it Does | Risk | Universal? |
|-------|------|-------------|------|-----------|
| 1 (Safe) | `.editorconfig` | Indent style, line endings, charset, trailing whitespace | Near-zero. Supported by every major editor natively or via plugin. No editor lock-in. | Yes — editor-agnostic |
| 2 (Safe) | `.vscode/extensions.json` | Recommends extensions. Users see a prompt, can ignore it. | Very low. Non-mandatory. Users can decline. | VS Code only |
| 3 (Moderate) | `.vscode/settings.json` | Workspace-level settings (formatOnSave, default formatter, etc.) | Low-moderate. Can override user preferences. Precedence hierarchy means user settings win. | VS Code only |
| 4 (Opinionated) | `.vscode/launch.json`, `.vscode/tasks.json` | Debug configurations, task definitions | Moderate. Useful but project-specific. | VS Code only |
| 5 (High risk) | Full IDE config (keybindings, themes, UI layout) | Personal preferences | High. This is what people object to. | No |
| 6 (Parallel) | `.devcontainer.json` | Complete containerized dev environment with IDE config | Low (opt-in, container-scoped). | devcontainer-supporting editors |

The DX polish spike rejection is correct for levels 4-5. But levels 1-2 are not "too personal" or "too variable" — they're standardization tools that reduce friction.

### 3.4 Analysis: What's Actually Harmful?

**Generating `.editorconfig`**:
- Harm: Essentially none. EditorConfig is supported natively by Visual Studio, all JetBrains IDEs, and VS Code (via near-universal extension). It handles only mechanical formatting: indent size, line endings, charset, trailing whitespace. These are not personal preferences — they're project standards that prevent noisy diffs.
- Benefit: Eliminates mixed-indentation commits, inconsistent line endings, trailing whitespace noise. Critical for cross-platform teams (Windows CRLF vs Unix LF).
- Precedent: Nearly every mature open-source project includes `.editorconfig`. The Go, Rust, and Python ecosystems effectively standardize on specific formatting anyway.

**Generating `.vscode/extensions.json`**:
- Harm: Minimal. It creates a *recommendation*, not a requirement. VS Code shows a notification banner saying "This workspace has extension recommendations." The developer can click "Install All," install selectively, or dismiss entirely. It never auto-installs anything.
- Benefit: New team members get a curated list of project-relevant extensions on first open. Reduces "how do I set up my editor for this project?" questions. For a consulting org where engineers rotate between projects, this is especially valuable.
- Precedent: Widely used in industry. Atomic Object, multiple open-source projects, and enterprise teams all commit this file.

**Generating `.vscode/settings.json`**:
- Harm: Moderate. Can override user preferences for that workspace. However, VS Code's precedence hierarchy means user settings still win for most things. The main risk is generating settings that conflict with what a developer already has.
- Benefit: Ensures formatOnSave uses the project's formatter, file associations are correct, and linter settings match CI.
- Precedent: Less commonly committed than extensions.json. Teams that commit it usually limit it to formatter and linter configuration.

### 3.5 The Extension Pack Pattern

Teams can create a VS Code extension pack — a marketplace-published meta-extension that bundles recommended extensions. When installed, it automatically installs all bundled extensions.

- **How it works**: A `package.json` with `extensionPack` array listing extension IDs. Compiled to a `.vsix` file and published to the VS Code marketplace (or distributed privately).
- **Benefit over extensions.json**: One-click install of the entire team's tooling stack. New extensions added to the pack auto-install on update.
- **Drawback**: Requires marketplace publishing (or private distribution). More maintenance than a JSON file.
- **Relevance to gdev**: gdev could generate an extensions.json for immediate benefit, and the consulting org could optionally publish a Highspring extension pack for firm-wide standards. These are complementary, not competing.

### 3.6 The Devcontainer Generation Option

devenv.sh can now generate `.devcontainer.json`. This is notable because devcontainer.json includes IDE configuration (extensions, settings) as part of the container spec. If gdev generates a devcontainer.json (which is already within its "file generation" mandate), IDE configuration comes along for free.

This offers a path to IDE config that doesn't feel like "gdev is configuring your IDE" — it feels like "gdev generates a devcontainer, and the devcontainer configures the IDE." The distinction matters psychologically even if the outcome is identical.

### 3.7 Neovim/Helix/Zed — LSP Configuration via devenv

For non-VS Code editors, the most useful thing gdev can do is ensure the right LSP servers are in `devenv.nix` packages. When `typescript-language-server`, `gopls`, `rust-analyzer`, etc. are available in the Nix environment, editors that use LSP (Neovim, Helix, Zed, Emacs) automatically discover them via PATH.

This is already partially achieved by devenv ecosystem modules — they add language tools to the environment. gdev's contribution would be ensuring LSP servers are included alongside compilers/runtimes.

**Recommendation**: When generating devenv.nix ecosystem packages, include the corresponding LSP server. This benefits all editor users without generating any editor-specific config.

### 3.8 Reconsidered Recommendations

Based on the analysis, the IDE config rejection should be **narrowed, not reversed**. The rejection is correct for comprehensive IDE configuration (themes, keybindings, UI layout, debug configs). It should be relaxed for:

**Always generate (non-controversial):**
- `.editorconfig` — Editor-agnostic, universally supported, handles mechanical formatting only. Should be part of `gdev init` output alongside `devenv.nix`. Properties to include: root=true, charset=utf-8, end_of_line=lf, insert_final_newline=true, trim_trailing_whitespace=true, plus language-specific indent rules based on detected ecosystems.

**Generate on opt-in (`gdev enable vscode`):**
- `.vscode/extensions.json` — Recommended extensions based on detected ecosystems. Maps ecosystem to extensions: TypeScript -> `dbaeumer.vscode-eslint` + `esbenp.prettier-vscode`, Python -> `ms-python.python` + `charliermarsh.ruff`, Rust -> `rust-lang.rust-analyzer`, Go -> `golang.go`, etc.
- `.vscode/settings.json` — Limited to formatter and linter configuration matching the project's toolchain. formatOnSave=true, default formatter per language, linter settings matching pre-commit config.

**Generate as part of devcontainer (if devcontainer support is added):**
- `.devcontainer.json` — Full IDE config (extensions, settings) scoped to the container. devenv.sh already supports this.

**Include in devenv.nix packages (always, no opt-in needed):**
- LSP servers corresponding to detected language ecosystems. Benefits all editor users.

**Never generate:**
- Keybindings, themes, UI layout, debug configurations, editor-specific plugin config.
- JetBrains `.idea/` settings (too complex, too variable).
- Neovim `init.lua` or Helix `config.toml` (deeply personal).

### 3.9 Proposed `gdev enable vscode` Flow

1. Detect language ecosystems already identified by gdev
2. Map ecosystems to VS Code extensions (maintained lookup table)
3. Generate `.vscode/extensions.json` with recommendations + comments explaining each
4. Generate `.vscode/settings.json` with only: formatOnSave, default formatter per language, linter settings
5. Add `.vscode/` to detection for `gdev doctor` ("VS Code workspace config is present/outdated")
6. `gdev update` regenerates if ecosystems change

This follows gdev's existing pattern: detect -> generate -> maintain. The key safeguard is opt-in — `gdev enable vscode` is explicit user intent.

---

## 4. Cross-Cutting Analysis

### 4.1 Architecture Fit

All three categories fit within the **devenv addon**:

- **Git platform CLIs** → Added to `devenv.nix` packages based on remote detection
- **Documentation tools** → Added to `devenv.nix` packages based on config file detection
- **IDE config** → Generated files (`.editorconfig`, `.vscode/*`) alongside `devenv.nix`

No new addon is needed. The detection and generation patterns mirror what gdev already does for language ecosystems.

### 4.2 Detection Heuristic Summary

| Signal | Tools to Add |
|--------|-------------|
| `.github/` or github.com remote | `gh` |
| `.gitlab-ci.yml` or gitlab.com remote | `glab` |
| `.gitattributes` with `filter=lfs` | `git-lfs` |
| `.git-crypt/` | `git-crypt` |
| `.gitsecret/` | `git-secret` |
| `*.md` with ` ```mermaid ` blocks | `mermaid-cli` |
| `*.d2` files | `d2` |
| `*.puml` or `*.plantuml` | `plantuml`, `graphviz` |
| `mkdocs.yml` | `mkdocs`, `mkdocs-material` (if theme detected) |
| `book.toml` | `mdbook` |
| `hugo.toml` or Hugo markers | `hugo` |
| `.markdownlint.json` / `.markdownlint.yml` | `markdownlint-cli` |
| (always) | `.editorconfig` generation |
| `gdev enable vscode` | `.vscode/extensions.json`, `.vscode/settings.json` |

### 4.3 Priority Ranking

1. **gh** — Highest impact single tool. Most projects are GitHub-hosted. Enables PR workflows, CI debugging, code review from terminal.
2. **`.editorconfig` generation** — Zero-controversy, universal benefit. Should be part of initial `gdev init`.
3. **git-lfs auto-detection** — Repos with LFS won't work without it. Binary detection, binary fix.
4. **Documentation tool detection** (mkdocs, mdbook, mermaid) — Natural extension of ecosystem detection.
5. **glab** — Same value as gh for GitLab projects.
6. **`gdev enable vscode`** — Opt-in, high value for VS Code users (dominant editor).
7. **delta** — Nice-to-have productivity improvement. Opt-in.
8. **lazygit** — Nice-to-have TUI. Opt-in.
9. **LSP servers in devenv.nix** — Benefits all editor users. Low effort, high leverage.

### 4.4 Open Questions

- Should `gh` authentication be part of `gdev setup` or handled separately? (`gh auth login` is interactive and requires a browser.)
- Should `.editorconfig` be generated with ecosystem-specific indent rules (e.g., 2-space for JS/TS, 4-space for Python, tabs for Go) or stick to universal defaults?
- For `gdev enable vscode`, should there be a `gdev enable jetbrains` equivalent, or is JetBrains config too complex?
- Should delta configuration go in `devenv.nix` enterShell (project-scoped) or be a user-level recommendation?
- How does devenv.sh's `.devcontainer.json` generation interact with gdev's potential `.vscode/` generation? Are they complementary or competing?

---

## Depth Checklist

- [x] Underlying mechanism explained — detection heuristics, nixpkgs availability, configuration requirements for each tool
- [x] Key tradeoffs and limitations — IDE config spectrum from safe to risky, Java dependency for PlantUML, Bitbucket auth deprecation
- [x] Compared to alternatives — devenv.sh, mise, devcontainers IDE config approaches compared; tool alternatives within each category (delta vs diff-so-fancy, lazygit vs tig vs gitui)
- [x] Failure modes and edge cases — git-crypt + Nix flakes conflict, Bitbucket app password deprecation, .vscode/settings.json precedence conflicts
- [x] Concrete examples — VS Code extensions.json format, EditorConfig properties, devcontainer customizations pattern
- [x] Standalone-readable — yes, sufficient for implementation decisions without consulting sources
