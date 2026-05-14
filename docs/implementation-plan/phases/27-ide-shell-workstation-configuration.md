# Phase 27: IDE, Shell & Workstation Configuration

## Goal

Add developer workstation personalization to gdev that lives at two distinct scopes: project-level (EditorConfig, VS Code extensions.json) and user-level (shell fragments, personal CLI tools via nix profile, Starship prompt integration). The key architectural principle is that shell tools belong at `~/.nix-profile` and are never forced into project scope, while EditorConfig rules belong in the repo and are always ecosystem-aware. gdev never modifies shell rc files directly — it generates fragments and prints clear source instructions.

## Dependencies

Phase 9 complete (cross-platform system detection, `gdev doctor`, `gdev setup` bootstrap step framework). Phase 10 complete (ecosystem addon system, per-ecosystem tool and configuration awareness).

## Phase Outputs

- `EditorConfig` generation in `gdev init` with ecosystem-aware rules and hash tracking
- `gdev enable vscode` command that generates `.vscode/extensions.json` with 14 ecosystem mappings
- `~/.qsdev/shell/` fragment directory system with bash/zsh/fish support
- `gdev setup --tools` installing 12 personal CLI tools via `nix profile install`
- Shell aliases fragment generated conditionally based on installed tools
- `~/.qsdev/shell/starship-gdev.toml` custom module showing gdev project status
- `gdev setup` interactive wizard for shell integration with non-interactive `--yes` mode

---

### Unit 27.1: EditorConfig Generation

**Description:** Generate a `.editorconfig` file on every `gdev init` with ecosystem-aware indent rules, tracking the generated file with the Phase 8 hash system and updating it via `gdev init --update`.

**Context:** EditorConfig is a project-level concern: it belongs in the repository alongside `devenv.nix` and `.pre-commit-config.yaml`. Unlike shell config (which is personal), EditorConfig rules represent team agreements on code formatting that should be consistent across all contributors. The rules are intentionally minimal and opinionated — gdev generates only the high-signal settings that editors actually act on, not every possible EditorConfig knob. The ecosystem mapping follows community convention: Go uses tabs (consistent with `gofmt`), everything else uses spaces with width varying by language community.

The generated file is tracked by the Phase 8 hash system. If a developer manually modifies `.editorconfig`, the modification flag is set and `gdev init --update` respects the override rather than stomping it. Section markers (introduced in Phase 12) are NOT used for `.editorconfig` because the entire file is gdev-owned — there is no human-edited section to preserve.

**Code-Grounded Note:** EditorConfig generation should be implemented as a new `Generate()` method on a `EditorConfigGenerator` struct in `internal/generators/editorconfig.go`, following the same interface pattern as existing generators. The Phase 8 `UpdatedFile` struct at the migration infrastructure layer tracks file hashes; EditorConfig plugs into that same tracking without special-casing. The ecosystem list comes from the `DetectedProject.Ecosystems` field produced by `internal/detect/detect.go`.

**Desired Outcome:** Every project initialized with gdev gets a correct `.editorconfig` with ecosystem-appropriate rules. The file is updated when ecosystems change and left alone when manually modified.

**Steps:**
1. Define the ecosystem-to-rule mapping in `internal/generators/editorconfig.go`:
   ```go
   var ecosystemEditorConfigRules = map[string]EditorConfigRules{
       "go": {
           IndentStyle: "tab",
           IndentSize:  "tab",
       },
       "python": {
           IndentStyle: "space",
           IndentSize:  "4",
       },
       "rust": {
           IndentStyle: "space",
           IndentSize:  "4",
       },
       "java": {
           IndentStyle: "space",
           IndentSize:  "4",
       },
       "kotlin": {
           IndentStyle: "space",
           IndentSize:  "4",
       },
       "javascript": {
           IndentStyle: "space",
           IndentSize:  "2",
       },
       "typescript": {
           IndentStyle: "space",
           IndentSize:  "2",
       },
   }

   // File-type overrides applied regardless of primary ecosystem.
   var fileTypeRules = map[string]EditorConfigRules{
       "*.json":     {IndentStyle: "space", IndentSize: "2"},
       "*.yaml":     {IndentStyle: "space", IndentSize: "2"},
       "*.yml":      {IndentStyle: "space", IndentSize: "2"},
       "*.md":       {IndentStyle: "space", IndentSize: "2", TrimTrailingWhitespace: false},
       "Makefile":   {IndentStyle: "tab"},
       "*.mk":       {IndentStyle: "tab"},
   }

   type EditorConfigRules struct {
       IndentStyle             string
       IndentSize              string
       TrimTrailingWhitespace  bool   // default true; false for Markdown
       InsertFinalNewline      bool   // always true
       Charset                 string // always utf-8
   }
   ```
2. Implement `GenerateEditorConfig(ecosystems []string) string`:
   - Start with the `[*]` root section: `charset = utf-8`, `end_of_line = lf`, `insert_final_newline = true`, `trim_trailing_whitespace = true`.
   - For each detected ecosystem (in alphabetical order for determinism), append a `[*.{ext}]` section with the ecosystem's rules.
   - For multi-ecosystem projects, produce one section per ecosystem file extension: `[*.go]` for Go, `[*.py]` for Python, etc.
   - Append the file-type override sections (`*.json`, `*.yaml`, `*.yml`, `*.md`, `Makefile`) unconditionally.
   - Output is deterministic: same inputs always produce identical output.
3. Produce a concrete example for a Go + TypeScript project:
   ```ini
   # EditorConfig — generated by gdev. Manual edits are preserved.
   # See https://editorconfig.org for format documentation.
   root = true

   [*]
   charset = utf-8
   end_of_line = lf
   insert_final_newline = true
   trim_trailing_whitespace = true

   # Go: tabs (gofmt standard)
   [*.go]
   indent_style = tab
   indent_size = tab

   # TypeScript
   [*.{ts,tsx}]
   indent_style = space
   indent_size = 2

   # JavaScript
   [*.{js,jsx,mjs,cjs}]
   indent_style = space
   indent_size = 2

   # JSON / YAML
   [*.{json,yaml,yml}]
   indent_style = space
   indent_size = 2

   # Markdown: preserve trailing whitespace (two spaces = line break)
   [*.md]
   indent_style = space
   indent_size = 2
   trim_trailing_whitespace = false

   # Makefiles: tabs required
   [Makefile,*.mk]
   indent_style = tab
   ```
4. Wire into `gdev init` generation pipeline:
   - Call `GenerateEditorConfig(project.Ecosystems)` after ecosystem detection.
   - Write `.editorconfig` to project root.
   - Register the file with Phase 8 hash tracking as a machine-owned file.
5. Handle update behavior in `gdev init --update`:
   - If hash matches expected: regenerate (ecosystem may have changed).
   - If hash differs (manual edit detected): skip with message "`.editorconfig` has been manually modified. Skipping regeneration. Use `--force-update` to overwrite."
6. Handle the no-ecosystem case:
   - If no ecosystems detected (empty project), generate the minimal `[*]` root section only.
   - Print a note in the init output: "EditorConfig generated with universal rules. Re-run `gdev init --update` after adding language files to get ecosystem-specific rules."
7. Write unit tests:
   - Go-only project produces tab-based rules for `*.go`.
   - Python project produces 4-space rules.
   - Multi-ecosystem (Go + TypeScript) produces correct sections for both.
   - JSON/YAML/Markdown file-type overrides always present regardless of ecosystem.
   - Output is deterministic (same inputs = same output, run twice).
   - Empty ecosystem list produces valid minimal EditorConfig.

**Acceptance Criteria:**
- [ ] `.editorconfig` generated on every `gdev init` with ecosystem-appropriate rules
- [ ] Go ecosystems produce `indent_style = tab` for `*.go` sections
- [ ] Python/Rust/Java/Kotlin produce `indent_style = space`, `indent_size = 4`
- [ ] JavaScript/TypeScript produce `indent_style = space`, `indent_size = 2`
- [ ] JSON, YAML, and Markdown file-type overrides always included
- [ ] Markdown section sets `trim_trailing_whitespace = false`
- [ ] Multi-ecosystem projects produce one section per ecosystem
- [ ] Generated file tracked by Phase 8 hash system
- [ ] Manual edits to `.editorconfig` detected and respected on `gdev init --update`
- [ ] Output is deterministic: identical inputs produce identical output
- [ ] Empty/no-ecosystem project generates valid minimal `[*]` section

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — ecosystem indent mapping, EditorConfig scope decision (project vs user)

**Status:** Not Started

---

### Unit 27.2: VS Code Extensions.json (Opt-in)

**Description:** Implement `gdev enable vscode` to generate `.vscode/extensions.json` with ecosystem-mapped extension IDs, tracked by the tool lifecycle system so `gdev disable vscode` removes it cleanly.

**Context:** VS Code extension recommendations are opt-in because not every team uses VS Code — it would be presumptuous to generate `.vscode/` in every project. The `gdev enable vscode` command follows the same tool lifecycle pattern as other addons: it registers the file with the Phase 12 file ownership registry, writes it with section markers, and `gdev disable vscode` removes it via shared-file surgery. `.vscode/settings.json` is explicitly NOT generated because developer settings preferences are too personal and varied to impose project-wide.

The `extensions.json` format is a `recommendations` array of extension IDs. The list is ecosystem-derived at generation time and reflects what gdev found when `gdev detect` was last run. The `anthropics.claude-code` extension is always included when the claudecode addon is active (installed by Phase 4). When claudecode is NOT active, `GitHub.copilot` is included as a fallback AI assistant, since developers on a project without the claudecode addon likely still want an AI coding assistant suggested.

**Code-Grounded Note:** The tool lifecycle system from Phase 12 provides `RegisterTool()`, `EnableTool()`, and `DisableTool()`. The vscode integration is a tool in this registry with a single owned file: `.vscode/extensions.json`. The file is entirely machine-owned (no human-edited sections), so section markers are used to allow `DisableTool()` to cleanly delete it. The claudecode addon state is readable via `addons/claudecode/state.go` (or equivalent).

**Desired Outcome:** Running `gdev enable vscode` generates a correct `.vscode/extensions.json` with ecosystem-appropriate recommendations. `gdev disable vscode` removes it cleanly. The file is regenerated correctly when `gdev init --update` runs.

**Steps:**
1. Define the ecosystem-to-extension mapping:
   ```go
   var ecosystemExtensions = map[string][]string{
       "go":         {"golang.go"},
       "python":     {"ms-python.python", "ms-python.vscode-pylance"},
       "rust":       {"rust-lang.rust-analyzer"},
       "java":       {"redhat.java", "vscjava.vscode-java-debug"},
       "kotlin":     {"fwcd.kotlin"},
       "javascript": {"dbaeumer.vscode-eslint", "esbenp.prettier-vscode"},
       "typescript": {"dbaeumer.vscode-eslint", "esbenp.prettier-vscode"},
       "nix":        {"jnoortheen.nix-ide"},
       "docker":     {"ms-azuretools.vscode-docker"},
       "terraform":  {"hashicorp.terraform"},
       "yaml":       {"redhat.vscode-yaml"},
       "toml":       {"tamasfe.even-better-toml"},
       "proto":      {"zxh404.vscode-proto3"},
       "shell":      {"mads-hartmann.bash-ide-vscode"},
   }

   // Always included when claudecode addon is active.
   const claudeCodeExtension = "anthropics.claude-code"
   // Included when claudecode addon is NOT active.
   const copilotExtension = "GitHub.copilot"
   ```
2. Implement `GenerateVSCodeExtensions(ecosystems []string, claudeCodeEnabled bool) []string`:
   - Collect extension IDs from each detected ecosystem (deduplicated).
   - If `claudeCodeEnabled`: append `anthropics.claude-code`.
   - If not `claudeCodeEnabled`: append `GitHub.copilot`.
   - Sort the list for determinism (alphabetical, except AI assistant always last).
   - Return the deduplicated, sorted list.
3. Implement `GenerateExtensionsJson(extensions []string) string`:
   - Produce valid `.vscode/extensions.json` JSON with `recommendations` array.
   - Include a comment header (as a `// comment` above the JSON, or as a `_comment` key — VS Code supports `//` comments in `.jsonc` format, which extensions.json uses).
   - Example output:
     ```json
     {
       // Generated by gdev. Run 'gdev enable vscode' to regenerate.
       "recommendations": [
         "golang.go",
         "jnoortheen.nix-ide",
         "ms-azuretools.vscode-docker",
         "anthropics.claude-code"
       ]
     }
     ```
4. Register `vscode` as a tool in the Phase 12 lifecycle registry:
   ```go
   ToolRegistration{
       Name:         "vscode",
       Description:  "VS Code extension recommendations (.vscode/extensions.json)",
       OwnedFiles:   []string{".vscode/extensions.json"},
       EnableFunc:   enableVSCode,
       DisableFunc:  disableVSCode,
       Category:     ToolCategoryIDE,
   }
   ```
5. Implement `enableVSCode(projectRoot string, state *ProjectState) error`:
   - Create `.vscode/` directory if it does not exist.
   - Generate extensions list from `state.DetectedProject.Ecosystems`.
   - Write `.vscode/extensions.json`.
   - Register file ownership.
   - Print: "Generated `.vscode/extensions.json` with N extension recommendations."
6. Implement `disableVSCode(projectRoot string) error`:
   - Remove `.vscode/extensions.json`.
   - If `.vscode/` directory is now empty, offer to remove it.
   - Release file ownership from registry.
7. Handle update behavior:
   - `gdev init --update` regenerates `.vscode/extensions.json` if the vscode tool is enabled.
   - Regeneration detects newly-added ecosystems (e.g., Docker added later) and adds their extensions.
8. Write unit tests:
   - Go project produces `["golang.go", "anthropics.claude-code"]` when claudecode enabled.
   - Go project produces `["GitHub.copilot", "golang.go"]` when claudecode disabled.
   - Multi-ecosystem project deduplicates extension IDs (TypeScript and JavaScript both add eslint — only one entry).
   - Output JSON is valid and parses without error.
   - `gdev disable vscode` removes the file; empty `.vscode/` directory offered for removal.

**Acceptance Criteria:**
- [ ] `gdev enable vscode` generates `.vscode/extensions.json` with ecosystem-mapped extension IDs
- [ ] 14 extension mappings implemented (go, python, rust, java, kotlin, javascript, typescript, nix, docker, terraform, yaml, toml, proto, shell)
- [ ] `anthropics.claude-code` included when claudecode addon is active
- [ ] `GitHub.copilot` included when claudecode addon is NOT active
- [ ] Extension list is deduplicated and deterministically sorted
- [ ] `.vscode/settings.json` is NOT generated
- [ ] `gdev disable vscode` removes `.vscode/extensions.json` cleanly
- [ ] `gdev init --update` regenerates when vscode tool is enabled
- [ ] File ownership tracked by Phase 12 tool lifecycle registry
- [ ] Output is valid `.jsonc` parseable by VS Code

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — VS Code opt-in decision, extension ID mappings, settings.json exclusion rationale

**Status:** Not Started

---

### Unit 27.3: Shell Fragment System (`gdev setup --shell`)

**Description:** Implement the shell fragment system that writes per-shell configuration files to `~/.qsdev/shell/` and prints `source` instructions for the user to manually add to their rc file. gdev never modifies `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish` directly.

**Context:** Modifying shell rc files automatically is a common source of bugs: duplicate entries on re-run, conflicts with existing config, difficult cleanup, and user surprise when their shell suddenly behaves differently. gdev follows the manual-sourcing pattern used by nix itself (`~/.nix-profile/etc/profile.d/nix.sh`) and many other tools: generate fragments, print instructions, let the user decide. The `~/.qsdev/shell/` directory (rather than a hidden dotfile) is used to make the fragments discoverable.

The fragment system must handle three shells with different syntax: bash/zsh (POSIX-compatible, largely shared), and fish (entirely different syntax, no `export VAR=val`, uses `set -gx`). Idempotency is guaranteed by regenerating fragments in place — there is no append-to-file step that could duplicate entries.

**Code-Grounded Note:** The `gdev setup` command is the entry point established in Phase 9 for machine-level bootstrap. `--shell` is a new flag to Phase 9's setup command. The `~/.qsdev/` root directory is used by other gdev user-level outputs (Starship config in Unit 27.6). Create it once in a shared `ensureQsdevDir()` helper.

**Desired Outcome:** Running `gdev setup --shell` creates `~/.qsdev/shell/` with per-shell fragment files and prints clear source instructions. Re-running is safe and idempotent. Fragments only reference tools that are actually installed.

**Steps:**
1. Define the fragment directory structure:
   ```
   ~/.qsdev/shell/
   ├── gdev.bash       # bash/zsh combined fragment
   ├── gdev.zsh        # zsh-specific additions (completion, zoxide)
   └── gdev.fish       # fish-specific fragment
   ```
2. Implement `ensureQsdevDir() error`:
   - Creates `~/.qsdev/` and `~/.qsdev/shell/` with permissions `0700` if they do not exist.
   - Idempotent: `MkdirAll` semantics.
3. Implement fragment content generation in `internal/shell/fragments.go`:
   ```go
   type ShellFragmentOptions struct {
       InstalledTools  []string  // tools actually present on PATH
       Shell           string    // "bash", "zsh", "fish"
   }

   func GenerateBashFragment(opts ShellFragmentOptions) string
   func GenerateZshFragment(opts ShellFragmentOptions) string
   func GenerateFishFragment(opts ShellFragmentOptions) string
   ```
4. Bash/zsh fragment content (only includes sections for installed tools):
   ```bash
   # gdev shell fragment — generated by 'gdev setup --shell'
   # Source this file from ~/.bashrc or ~/.zshrc:
   #   source ~/.qsdev/shell/gdev.bash

   # fzf key bindings (if fzf is installed)
   [ -f ~/.nix-profile/share/fzf/key-bindings.bash ] && \
     source ~/.nix-profile/share/fzf/key-bindings.bash

   # zoxide init (if zoxide is installed)
   command -v zoxide >/dev/null 2>&1 && eval "$(zoxide init bash)"

   # starship init (if starship is installed)
   command -v starship >/dev/null 2>&1 && eval "$(starship init bash)"

   # delta as git pager (if delta is installed)
   command -v delta >/dev/null 2>&1 && git config --global core.pager delta 2>/dev/null || true

   # bat as MANPAGER (if bat is installed)
   command -v bat >/dev/null 2>&1 && export MANPAGER="sh -c 'col -bx | bat -l man -p'" && export MANROFFOPT="-c"
   ```
5. Fish fragment content:
   ```fish
   # gdev shell fragment — generated by 'gdev setup --shell'
   # Source this file from ~/.config/fish/config.fish:
   #   source ~/.qsdev/shell/gdev.fish

   # fzf key bindings
   if test -f ~/.nix-profile/share/fzf/key-bindings.fish
       source ~/.nix-profile/share/fzf/key-bindings.fish
   end

   # zoxide init
   if command -v zoxide >/dev/null 2>&1
       zoxide init fish | source
   end

   # starship init
   if command -v starship >/dev/null 2>&1
       starship init fish | source
   end

   # bat as MANPAGER
   if command -v bat >/dev/null 2>&1
       set -gx MANPAGER "sh -c 'col -bx | bat -l man -p'"
       set -gx MANROFFOPT "-c"
   end
   ```
6. Implement the `--shell` flag on `gdev setup`:
   - Detect current shell from `$SHELL` env var if no `--shell` flag given.
   - Accept explicit `--shell bash`, `--shell zsh`, `--shell fish`.
   - `--shell all` generates fragments for all three shells.
   - Default (no flag): detect from `$SHELL`, generate for detected shell.
7. Print source instructions after generating:
   ```
   Shell fragments written to ~/.qsdev/shell/

   Add the following to your shell rc file:

   For bash (~/.bashrc):
     source ~/.qsdev/shell/gdev.bash

   For zsh (~/.zshrc):
     source ~/.qsdev/shell/gdev.bash
     source ~/.qsdev/shell/gdev.zsh

   For fish (~/.config/fish/config.fish):
     source ~/.qsdev/shell/gdev.fish
   ```
8. Implement idempotency:
   - Fragment files are written with `os.WriteFile` (overwrite, not append).
   - Content is regenerated from scratch each run based on currently installed tools.
   - Re-running safely updates fragments if new tools have been installed since last run.
9. Write unit tests:
   - Bash fragment with all tools installed contains all expected sections.
   - Bash fragment with no tools installed contains only the header comment.
   - Fish fragment uses fish syntax (`set -gx`, `if command -v`).
   - Fragment generation is idempotent (second run produces identical output).
   - Shell detection from `$SHELL` env var.

**Acceptance Criteria:**
- [ ] `gdev setup --shell` creates `~/.qsdev/shell/` and writes fragment files
- [ ] `~/.bashrc`, `~/.zshrc`, and `~/.config/fish/config.fish` are NEVER modified directly
- [ ] Fragments generated for bash, zsh, and fish with shell-appropriate syntax
- [ ] Each fragment section is conditional on the relevant tool being installed
- [ ] Source instructions printed after generation for each shell
- [ ] Idempotent: re-running `gdev setup --shell` regenerates fragments without duplication
- [ ] `--shell` flag accepts `bash`, `zsh`, `fish`, `all`; defaults to detected shell from `$SHELL`
- [ ] Fish fragment uses fish syntax (`set -gx`, `if command -v`, `end` blocks)
- [ ] Bash/zsh fragment uses POSIX `command -v` guards, not `which`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — manual-sourcing pattern, `~/.qsdev/shell/` directory choice, fish vs bash/zsh split

**Status:** Not Started

---

### Unit 27.4: Personal CLI Tools via Nix Profile

**Description:** Implement `gdev setup --tools` to install 12 personal CLI tools to `~/.nix-profile` via `nix profile install`, with confirmation prompt, skip-existing detection, and `gdev doctor` integration to suggest the command when tools are missing.

**Context:** Personal shell tools (ripgrep, fd, bat, fzf, etc.) are personal infrastructure — they are not per-project dependencies and should not appear in any `devenv.nix`. They belong at `~/.nix-profile` where they are available in every shell session regardless of which project directory the developer is in. `nix profile install` is the correct mechanism on NixOS/nix-on-any-OS: it installs to the user profile, is declarative-enough for the use case, and does not require root.

The install command is never forced: gdev shows what would be installed, checks what is already present, and asks for confirmation. Developers who already have some tools (via system packages, homebrew, cargo, etc.) should not have gdev blindly reinstall them; already-present-on-PATH tools are skipped.

**Code-Grounded Note:** The `gdev doctor` command from Phase 9 checks for tool presence and reports missing tools. Phase 27 extends `gdev doctor` to check for the personal tool set and emit a suggestion if tools are missing. The `gdev setup --tools` implementation calls `nix profile install nixpkgs#<name>` for each tool that is not already on PATH. All 12 tool names must map to their correct nixpkgs attribute names.

**Desired Outcome:** `gdev setup --tools` installs missing personal CLI tools to `~/.nix-profile` in a single command. Re-running is safe. `gdev doctor` reports missing tools with a clear suggestion.

**Steps:**
1. Define the personal tool registry:
   ```go
   type PersonalTool struct {
       Name        string   // binary name on PATH
       NixAttr     string   // nixpkgs attribute, e.g., "ripgrep"
       Description string
       CheckCmd    string   // command to verify installed, e.g., "rg --version"
   }

   var PersonalTools = []PersonalTool{
       {Name: "rg",       NixAttr: "ripgrep",   Description: "Fast regex search (grep replacement)"},
       {Name: "fd",       NixAttr: "fd",         Description: "Fast file finder (find replacement)"},
       {Name: "bat",      NixAttr: "bat",        Description: "Syntax-highlighted cat replacement"},
       {Name: "fzf",      NixAttr: "fzf",        Description: "Fuzzy finder for shell history and files"},
       {Name: "jq",       NixAttr: "jq",         Description: "JSON processor"},
       {Name: "yq",       NixAttr: "yq-go",      Description: "YAML/JSON/TOML processor"},
       {Name: "delta",    NixAttr: "delta",       Description: "Syntax-highlighted git diff"},
       {Name: "eza",      NixAttr: "eza",         Description: "Modern ls replacement"},
       {Name: "zoxide",   NixAttr: "zoxide",      Description: "Smarter cd with frecency tracking"},
       {Name: "starship", NixAttr: "starship",    Description: "Cross-shell prompt"},
       {Name: "sops",     NixAttr: "sops",        Description: "Secrets file encryption"},
       {Name: "age",      NixAttr: "age",         Description: "Simple file encryption"},
   }
   ```
2. Implement tool presence detection:
   ```go
   func checkToolPresence(tool PersonalTool) (installed bool, path string) {
       path, err := exec.LookPath(tool.Name)
       if err != nil {
           return false, ""
       }
       return true, path
   }
   ```
3. Implement `gdev setup --tools` command flow:
   - Check each tool with `checkToolPresence`.
   - Categorize: `alreadyInstalled []PersonalTool`, `toInstall []PersonalTool`.
   - Print a table showing status:
     ```
     Personal CLI Tools
     ==================
     rg (ripgrep)      ✓ already installed (/home/user/.nix-profile/bin/rg)
     fd                ✓ already installed
     bat               ✗ not found — will install nixpkgs#bat
     fzf               ✗ not found — will install nixpkgs#fzf
     ...

     Will install 4 tools via nix profile install. Proceed? [y/N]
     ```
   - If `--yes` flag: skip prompt, proceed automatically.
   - If no tools to install: print "All tools already installed." and exit 0.
4. Implement the installation loop:
   ```go
   func installPersonalTool(tool PersonalTool) error {
       cmd := exec.Command("nix", "profile", "install",
           fmt.Sprintf("nixpkgs#%s", tool.NixAttr))
       cmd.Stdout = os.Stdout
       cmd.Stderr = os.Stderr
       return cmd.Run()
   }
   ```
   - Install tools sequentially (nix profile install can conflict when run in parallel).
   - On each success: print `✓ Installed <name>`.
   - On failure: print error, continue with remaining tools, report failures at end.
   - After all installs: print summary and suggest `gdev setup --shell` to add aliases.
5. Integrate into `gdev doctor`:
   - Add a "Personal Tools" check category to `gdev doctor` output.
   - For each tool not found on PATH: report as `MISSING`.
   - If any tools are missing: add suggestion "Run `gdev setup --tools` to install missing personal CLI tools."
   - `gdev doctor --json` includes personal tool status in JSON output.
6. Handle the case where `nix` is not available:
   - If `nix` binary not found on PATH: print "Nix is not installed. This tool requires Nix package manager. See https://nixos.org/download."
   - Exit with non-zero status.
7. Write unit tests (using mock exec for `nix profile install`):
   - All tools present: reports all installed, no install prompt.
   - Mix of present/missing: correct split, correct nix attributes in install commands.
   - `--yes` flag skips confirmation.
   - nix not found: clear error message.
   - `gdev doctor` includes personal tools check.

**Acceptance Criteria:**
- [ ] `gdev setup --tools` checks all 12 personal tools for PATH presence
- [ ] Tools already installed (via any mechanism) are skipped, not reinstalled
- [ ] Correct `nixpkgs#<attr>` names used: `ripgrep`, `fd`, `bat`, `fzf`, `jq`, `yq-go`, `delta`, `eza`, `zoxide`, `starship`, `sops`, `age`
- [ ] Confirmation prompt shown before installing, listing what will be installed
- [ ] `--yes` flag skips confirmation (for scripted setup)
- [ ] Sequential installation (no parallel nix profile install)
- [ ] `gdev doctor` reports missing personal tools with `gdev setup --tools` suggestion
- [ ] `nix` not on PATH produces clear error with installation link
- [ ] Already-all-installed case exits cleanly without prompting

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — personal tool list, `~/.nix-profile` vs project scope rationale, nix profile install approach

**Status:** Not Started

---

### Unit 27.5: Shell Aliases Fragment

**Description:** Generate the aliases fragment in `~/.qsdev/shell/` that sets up modern CLI tool aliases, with conditional generation based on installed tools and bat/delta special handling for MANPAGER and git pager configuration.

**Context:** Aliases like `alias cat=bat` are convenience that developers must opt into — they are not imposed globally. The fragment system from Unit 27.3 provides the delivery mechanism; this unit defines the alias content and the conditional logic. A key principle: an alias is only generated if the target tool is actually installed. Generating `alias cat=bat` when bat is not installed would break `cat` for the developer.

bat as MANPAGER and delta as git pager are slightly more complex: they require environment variable exports (`MANPAGER`, `MANROFFOPT`) and a global git config change. These are generated in the aliases fragment (not the shell init fragment from Unit 27.3) because they are more like configuration than initialization.

**Code-Grounded Note:** The aliases fragment is a separate file from the init fragment (Unit 27.3). The init fragment handles tool initialization (`eval "$(zoxide init bash)"`); the aliases fragment handles convenience wrappers. Both are sourced from the same `source ~/.qsdev/shell/gdev.bash` line — the two files are concatenated at source time. Alternatively, the init fragment can `source` the aliases fragment. Either approach is valid; the former is simpler.

**Desired Outcome:** Developers who source the gdev shell fragment get modern CLI aliases for installed tools. No alias is generated for a tool that is not installed.

**Steps:**
1. Define the alias map:
   ```go
   type Alias struct {
       Name        string   // alias name
       Target      string   // alias expansion
       RequiresTool string  // tool binary that must be on PATH
       BashSyntax  string   // for bash/zsh
       FishSyntax  string   // for fish (uses `abbr`)
   }

   var DefaultAliases = []Alias{
       {Name: "ll",   Target: "eza -la",  RequiresTool: "eza",  BashSyntax: `alias ll='eza -la'`,        FishSyntax: `abbr -a ll 'eza -la'`},
       {Name: "cat",  Target: "bat",      RequiresTool: "bat",  BashSyntax: `alias cat='bat'`,            FishSyntax: `abbr -a cat 'bat'`},
       {Name: "grep", Target: "rg",       RequiresTool: "rg",   BashSyntax: `alias grep='rg'`,            FishSyntax: `abbr -a grep 'rg'`},
       {Name: "find", Target: "fd",       RequiresTool: "fd",   BashSyntax: `alias find='fd'`,            FishSyntax: `abbr -a find 'fd'`},
       {Name: "diff", Target: "delta",    RequiresTool: "delta", BashSyntax: `alias diff='delta'`,        FishSyntax: `abbr -a diff 'delta'`},
   }
   ```
2. Implement `GenerateAliasesFragment(shell string, installed []string) string`:
   - Accept a list of installed tool binary names.
   - For each alias, include it only if `RequiresTool` is in `installed`.
   - Include bat MANPAGER config if `bat` is in `installed`.
   - Include delta git pager config if `delta` is in `installed`.
3. Bat MANPAGER section (bash/zsh):
   ```bash
   # bat as MANPAGER
   if command -v bat >/dev/null 2>&1; then
     export MANPAGER="sh -c 'col -bx | bat -l man -p'"
     export MANROFFOPT="-c"
   fi
   ```
4. Delta git pager section (bash/zsh/fish):
   ```bash
   # delta as git pager
   if command -v delta >/dev/null 2>&1; then
     git config --global core.pager delta 2>/dev/null || true
     git config --global interactive.diffFilter "delta --color-only" 2>/dev/null || true
   fi
   ```
5. Fish uses `abbr` instead of `alias` for interactive abbreviations:
   ```fish
   # Modern CLI aliases (fish abbreviations)
   if command -v eza >/dev/null 2>&1
       abbr -a ll 'eza -la'
   end
   if command -v bat >/dev/null 2>&1
       abbr -a cat 'bat'
       set -gx MANPAGER "sh -c 'col -bx | bat -l man -p'"
       set -gx MANROFFOPT "-c"
   end
   ```
6. Integrate the aliases fragment into the `gdev setup --shell` command from Unit 27.3:
   - Run tool presence check for all tools in `DefaultAliases` before generating.
   - Include aliases section in the main fragment file (not a separate file).
   - Label the section clearly: `# Aliases (only for installed tools)`.
7. Write unit tests:
   - All tools installed: all aliases present in output.
   - No tools installed: aliases section is empty/absent.
   - Only bat installed: `cat` alias and MANPAGER export present, others absent.
   - Only delta installed: `diff` alias and git config present.
   - Fish output uses `abbr -a` syntax, not `alias`.

**Acceptance Criteria:**
- [ ] `ll` → `eza -la` alias generated only when `eza` is installed
- [ ] `cat` → `bat` alias generated only when `bat` is installed
- [ ] `grep` → `rg` alias generated only when `rg` is installed
- [ ] `find` → `fd` alias generated only when `fd` is installed
- [ ] `diff` → `delta` alias generated only when `delta` is installed
- [ ] bat MANPAGER (`MANPAGER`, `MANROFFOPT`) exported when bat is installed
- [ ] delta git pager (`core.pager`, `interactive.diffFilter`) configured when delta is installed
- [ ] Fish fragment uses `abbr -a` for interactive abbreviations instead of `alias`
- [ ] No alias generated for any tool that is not installed
- [ ] Aliases section integrated into main shell fragment from Unit 27.3

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — alias list, bat MANPAGER configuration, delta git pager setup, fish abbr vs alias distinction

**Status:** Not Started

---

### Unit 27.6: Starship gdev Module

**Description:** Generate a `~/.qsdev/shell/starship-gdev.toml` Starship config fragment that shows a gdev project indicator in the shell prompt, and generate or update `~/.config/starship.toml` to include the gdev module.

**Context:** Starship is the cross-shell prompt configured in TOML. It supports `[custom.X]` modules that run shell commands and display their output in the prompt. The gdev module detects `.gdev.yaml` in the current directory (or any parent) and displays the project name plus compliance level when a client profile is active. This gives developers an at-a-glance signal that they are inside a gdev-managed project.

The critical constraint: gdev must never overwrite an existing `~/.config/starship.toml`. If one exists, print merge instructions and show the exact TOML block to add. If no `~/.config/starship.toml` exists, generate a minimal one that includes the gdev module. This respects developer prompt customization investment.

**Code-Grounded Note:** Starship `[custom.X]` modules use a `command` field that runs a shell command and a `when` field that controls display. The `when` command should check for `.gdev.yaml` existence. The `format` uses Starship's format string syntax. The generated `starship-gdev.toml` is the source of truth; the merge into `~/.config/starship.toml` adds an `[include]` directive (supported in Starship 1.x) or adds the `[custom.gdev]` block inline.

**Desired Outcome:** After `gdev setup --shell`, developers with Starship see a gdev project indicator in their prompt. Developers with existing Starship configs receive merge instructions instead of having their config overwritten.

**Steps:**
1. Define the Starship gdev module:
   ```toml
   # ~/.qsdev/shell/starship-gdev.toml
   # gdev project indicator for Starship prompt.
   # To use: add 'custom.gdev' to your [format] string in ~/.config/starship.toml,
   # or use the [include] directive: add the line below to your starship.toml:
   #   "$config_dir/../../.qsdev/shell/starship-gdev.toml"

   [custom.gdev]
   description = "gdev-managed project indicator"
   command = """
   gdev status --prompt 2>/dev/null || true
   """
   when = "test -f .gdev.yaml || test -f ../.gdev.yaml || test -f ../../.gdev.yaml"
   format = "[$output]($style) "
   style = "bold cyan"
   shell = ["bash", "--noprofile", "--norc"]
   ```
2. Implement `gdev status --prompt` (a new output mode of `gdev status`):
   - Output a short string: project name from `.gdev.yaml` (if set), or directory basename.
   - If client profile active: append compliance level indicator, e.g., `myproject [strict]`.
   - If health checks recently ran and found issues: append `⚠` symbol.
   - If no `.gdev.yaml` found in current or parent dirs: output nothing (empty string, exit 0).
   - Must complete in under 50ms (Starship prompt impact).
3. Implement `GenerateStarshipGdevToml() string`:
   - Returns the content of `~/.qsdev/shell/starship-gdev.toml` as a string.
   - The content is a constant template (not dynamically generated per-project).
4. Implement the `~/.config/starship.toml` handling:
   ```go
   func integrateStarshipConfig() error {
       starshipConfig := filepath.Join(os.Getenv("HOME"), ".config", "starship.toml")
       qsdevModule := filepath.Join(os.Getenv("HOME"), ".qsdev", "shell", "starship-gdev.toml")

       // Write the gdev module file regardless
       if err := os.WriteFile(qsdevModule, []byte(starshipGdevToml), 0644); err != nil {
           return err
       }

       if _, err := os.Stat(starshipConfig); err == nil {
           // Existing config: print merge instructions
           printStarshipMergeInstructions(starshipConfig, qsdevModule)
           return nil
       }

       // No existing config: generate a minimal one that includes gdev module
       minimal := generateMinimalStarshipToml(qsdevModule)
       if err := os.WriteFile(starshipConfig, []byte(minimal), 0644); err != nil {
           return err
       }
       fmt.Printf("Generated ~/.config/starship.toml with gdev module.\n")
       return nil
   }
   ```
5. Implement `printStarshipMergeInstructions`:
   - Print a clear message explaining that the existing config was not modified.
   - Show the TOML `[include]` directive to add.
   - Show the `[custom.gdev]` block to add if the user prefers inline.
   - Example output:
     ```
     Existing ~/.config/starship.toml was not modified.

     To add the gdev prompt module, choose one of:

     Option 1 — Include directive (add to top of your starship.toml):
       "$config_dir/../../.qsdev/shell/starship-gdev.toml"

     Option 2 — Inline (add [custom.gdev] block from):
       cat ~/.qsdev/shell/starship-gdev.toml
     ```
6. Generate a minimal `~/.config/starship.toml` when none exists:
   ```toml
   # ~/.config/starship.toml — generated by gdev setup --shell
   # Starship prompt configuration. See https://starship.rs for full options.

   format = """
   $directory\
   $git_branch\
   $git_status\
   ${custom.gdev}\
   $character"""

   [character]
   success_symbol = "[➜](bold green)"
   error_symbol = "[➜](bold red)"

   "$config_dir/../../.qsdev/shell/starship-gdev.toml"
   ```
7. Write unit tests:
   - `gdev status --prompt` in a gdev-managed dir returns project name.
   - `gdev status --prompt` in a non-gdev dir returns empty string.
   - `gdev status --prompt` with active client profile includes compliance level.
   - `integrateStarshipConfig` writes module file regardless of existing config.
   - Existing `starship.toml`: module file written, existing config untouched, instructions printed.
   - No `starship.toml`: minimal config generated with gdev module included.
   - `gdev status --prompt` completes in under 50ms.

**Acceptance Criteria:**
- [ ] `~/.qsdev/shell/starship-gdev.toml` generated with `[custom.gdev]` module
- [ ] `gdev status --prompt` outputs project name (and compliance level when client profile active)
- [ ] `gdev status --prompt` outputs empty string in non-gdev directories
- [ ] `gdev status --prompt` completes in under 50ms
- [ ] Existing `~/.config/starship.toml` is NEVER overwritten
- [ ] When no `~/.config/starship.toml`: minimal config generated that includes gdev module
- [ ] When `~/.config/starship.toml` exists: merge instructions printed with both include and inline options
- [ ] Starship module file written to `~/.qsdev/shell/starship-gdev.toml` unconditionally

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — Starship `[custom.X]` module design, no-overwrite policy, `gdev status --prompt` command spec

**Status:** Not Started

---

### Unit 27.7: Shell Integration Wizard

**Description:** Implement the `gdev setup` interactive flow for shell integration: detect the current shell, ask which components to enable, show previews, offer to install missing tools, and support non-interactive `--yes` mode.

**Context:** `gdev setup` as established in Phase 9 is the bootstrap command for machine-level setup. Unit 27.7 adds the interactive orchestration layer that ties together shell fragments (Unit 27.3), personal tools (Unit 27.4), aliases (Unit 27.5), and Starship integration (Unit 27.6). The wizard is the user-facing entry point; the individual units are the implementation. Following Phase 6's wizard pattern (huh form library), the wizard is pleasant to use but always skippable with `--yes`.

The key UX decisions: show the user what will happen before doing it, detect the current shell automatically, and offer to install missing tools inline rather than making the user run a second command.

**Code-Grounded Note:** Phase 6 established `huh` (charmbracelet/huh) as the form library. The shell wizard uses huh `Select` and `MultiSelect` forms. Phase 9's `gdev setup` command already has the cobra command structure; this unit extends its `Run` function.

**Desired Outcome:** `gdev setup` without flags launches an interactive wizard that configures shell integration. `gdev setup --shell --tools --yes` configures everything non-interactively for CI or scripted onboarding.

**Steps:**
1. Implement the interactive wizard in `cmd/setup.go`:
   ```go
   func runSetupInteractive() error {
       // Step 1: Detect current shell
       detectedShell := detectShell() // from $SHELL env var

       // Step 2: Check tool presence
       toolStatus := checkPersonalToolPresence(PersonalTools)
       missingTools := filterMissing(toolStatus)

       // Step 3: Check starship
       hasStarship := isOnPath("starship")
       hasExistingStarshipConfig := fileExists(starshipConfigPath())

       // Step 4: Present choices
       var form huh.Form
       // ... huh form asking:
       //   - Which shell(s) to configure (multi-select, pre-selected = detected)
       //   - Install missing personal tools? (confirm, shown only if missingTools > 0)
       //   - Set up Starship gdev module? (confirm, shown only if starship installed)

       // Step 5: Execute chosen actions
   }
   ```
2. Form structure:
   ```go
   huh.NewSelect[string]().
       Title("Which shell are you using?").
       Options(
           huh.NewOption("bash", "bash"),
           huh.NewOption("zsh", "zsh"),
           huh.NewOption("fish", "fish"),
           huh.NewOption("All shells", "all"),
       ).
       Value(&selectedShell)

   huh.NewConfirm().
       Title(fmt.Sprintf("Install %d missing tools via nix profile install?", len(missingTools))).
       Description(formatToolList(missingTools)).
       Value(&installTools)
   ```
3. Show a preview of what will be generated before writing:
   ```
   Shell Integration Setup
   =======================
   Shell:    zsh (detected from $SHELL)
   Actions:
     ✓ Generate ~/.qsdev/shell/gdev.bash
     ✓ Generate ~/.qsdev/shell/gdev.zsh
     ✓ Install 3 tools: bat, delta, eza
     ✓ Set up Starship gdev module

   Source instructions will be printed after setup.
   Proceed? [Y/n]
   ```
4. Implement non-interactive mode flags:
   - `gdev setup --shell`: configure shell fragments for detected shell, skip tool install.
   - `gdev setup --tools`: install missing tools only.
   - `gdev setup --shell --tools`: both, with confirmation prompts.
   - `gdev setup --shell --tools --yes`: both, fully non-interactive.
   - `gdev setup --shell zsh`: force zsh even if `$SHELL` differs.
5. Offer to install Starship if not found:
   ```
   Starship prompt is not installed.
   Install starship via nix profile install? [Y/n]
   ```
   - If yes: run `nix profile install nixpkgs#starship` inline.
   - After install: proceed with Starship integration (Unit 27.6).
6. Print a summary at the end:
   ```
   Setup Complete
   ==============
   ✓ Shell fragments written to ~/.qsdev/shell/
   ✓ 3 tools installed: bat, delta, eza
   ✓ Starship gdev module written to ~/.qsdev/shell/starship-gdev.toml

   Next steps:
   1. Add to ~/.zshrc:
        source ~/.qsdev/shell/gdev.bash
        source ~/.qsdev/shell/gdev.zsh
   2. Reload your shell: exec zsh

   Tip: Run 'gdev doctor' to verify your environment.
   ```
7. Write integration tests:
   - Non-interactive `--yes` with `--shell zsh --tools`: runs without prompts.
   - Interactive flow: wizard displays, choices respected.
   - Shell detection from `$SHELL` env var.
   - Starship not installed: offer displayed.
   - All already set up: "Nothing to do" message.

**Acceptance Criteria:**
- [ ] `gdev setup` (no flags) launches interactive wizard with shell and tool choices
- [ ] Shell auto-detected from `$SHELL` env var, pre-selected in wizard
- [ ] Preview of actions shown before writing
- [ ] `--shell bash|zsh|fish|all` flag bypasses shell selection question
- [ ] `--tools` flag triggers personal tool installation
- [ ] `--yes` flag makes all prompts non-interactive (accepts all defaults)
- [ ] Missing starship offered for install inline during wizard
- [ ] Summary with source instructions printed after completion
- [ ] `gdev setup` is idempotent: running twice produces same end state
- [ ] `gdev doctor` suggested in final output

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/ide-shell-config-research.md` — wizard flow design, non-interactive flag combinations
- `phases/06-wizard-orchestration.md` — huh form library usage, non-interactive mode pattern

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### New Files

| File | Purpose |
|------|---------|
| `internal/generators/editorconfig.go` | EditorConfig generation logic |
| `internal/generators/vscode.go` | VS Code extensions.json generation |
| `internal/shell/fragments.go` | Shell fragment generation (bash/zsh/fish) |
| `internal/shell/aliases.go` | Alias fragment content and conditional logic |
| `internal/shell/starship.go` | Starship module generation and config integration |
| `internal/tools/personal.go` | Personal tool registry and `nix profile install` logic |
| `cmd/setup_shell.go` | `gdev setup --shell/--tools` command implementation |

### Existing Commands to Extend

| Command | Extension |
|---------|-----------|
| `gdev init` | Call EditorConfig generator after ecosystem detection |
| `gdev doctor` | Add personal tools check category |
| `gdev status` | Add `--prompt` mode for Starship integration |
| `gdev setup` | Add `--shell`, `--tools`, `--yes` flags; add interactive wizard |
| `gdev enable` | Register `vscode` tool in lifecycle registry |
| `gdev disable` | Handle `vscode` tool cleanup |

### Scope Boundary

These units operate entirely at user (`~/.nix-profile`, `~/.qsdev/`) or project (`.editorconfig`, `.vscode/`) scope. They never touch system paths or require root. The nix channel/registry used by `nix profile install` is the user's configured nixpkgs, not a gdev-controlled channel.

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] `.editorconfig` generated on `gdev init` and verified for correctness across all supported ecosystems
- [ ] `gdev enable vscode` / `gdev disable vscode` round-trip produces and removes `.vscode/extensions.json` cleanly
- [ ] `gdev setup --shell` generates `~/.qsdev/shell/` fragments without modifying any rc file
- [ ] `gdev setup --tools` installs only missing tools; already-installed tools skipped
- [ ] Shell fragments contain only aliases for installed tools (verified with no-tools and all-tools scenarios)
- [ ] `gdev setup --shell --tools --yes` completes non-interactively (for CI/scripted onboarding)
- [ ] Starship gdev module works in bash, zsh, and fish (manual verification)
- [ ] Existing `~/.config/starship.toml` not modified; merge instructions printed
- [ ] `gdev status --prompt` returns output in under 50ms
- [ ] `gdev doctor` reports missing personal tools with actionable suggestion
