# CLI Wizard/Installer UX Patterns for Guided Project Setup

## Executive Summary

This report surveys the landscape of CLI wizard and installer UX patterns used in development tooling, with the goal of informing how a gdev extension should implement guided setup for devenv.sh and Claude Code configuration. The research covers six categories: classic generator frameworks (Yeoman), modern scaffolding CLIs (create-*), template-based generators (cookiecutter/copier), lightweight scaffolding (degit), interactive prompt UI libraries (JS and Go ecosystems), and cross-cutting UX patterns (progressive disclosure, opinionated defaults). The key finding is that the ecosystem has converged on a pattern we call **"opinionated menu with escape hatches"** — strong defaults presented first, with progressive customization available on demand, and full CLI flag override for non-interactive/CI use. For gdev specifically, the charmbracelet/huh library is the strongest fit for the Go ecosystem, and gdev's existing bootstrap step system already provides the structural foundation that huh's Form > Group > Field hierarchy would map onto cleanly.

---

## 1. Yeoman (yo) — The Classic Generator Framework

**Source:** `docs/yeoman-composability.md`, `docs/yeoman-runtime-context.md`

### How It Works

Yeoman generators are Node.js classes with methods that map to lifecycle phases. The run loop executes methods in a fixed priority order:

1. **initializing** — State checks, configuration retrieval
2. **prompting** — User input collection via `this.prompt()` (wraps Inquirer.js)
3. **configuring** — Metadata file creation, project setup
4. **default** — Unnamed methods land here
5. **writing** — File generation (uses mem-fs for in-memory file system)
6. **conflicts** — Internal conflict resolution
7. **install** — Dependency installation (npm, bower)
8. **end** — Cleanup and completion messages

### Composability

Generators compose via `composeWith()`. When two generators compose, their methods interleave by priority group: all generators run `prompting` in order, then all run `writing` in order, etc. This is powerful but creates ordering constraints — a generator cannot easily depend on another generator's prompting output during its own prompting phase.

### Defaults vs Customization

Yeoman uses `{store: true}` on prompts to persist answers in a machine-global storage. Next run, the stored value becomes the default. This is preference persistence, not progressive disclosure — all questions are always asked.

### Strengths
- **Composability**: Multiple generators work together through the priority queue
- **Full lifecycle**: Clear separation between prompting, writing, and installing
- **Rich ecosystem**: Thousands of generators existed at peak
- **In-memory FS**: Virtual file system prevents partial writes on failure

### Weaknesses
- **Complexity**: Requires deep Node.js knowledge and npm packaging to author generators
- **Always asks everything**: No progressive disclosure — every prompt fires every time
- **Composability footguns**: File system conflicts between composed generators were never fully solved (Issue #658 drove a complete FS rewrite in v0.18)
- **Effectively deprecated**: The project still exists but is rarely used for new tooling. Modern tools favor simpler approaches.

### Why Yeoman Declined

Three factors: (1) the overhead of publishing generators to npm was too high for most use cases; (2) composability never worked cleanly enough to justify the complexity; (3) modern `create-*` tools proved that a single-purpose CLI with good defaults beats a general-purpose generator framework.

### Relevance to gdev

**High relevance for the lifecycle model.** Yeoman's priority queue (prompting → configuring → writing → installing) maps directly to gdev's bootstrap step system. The lesson is: define clear phases, let addons interleave within phases, but don't try to build a general-purpose generator framework. gdev's approach of "steps with skip handlers" is already better-scoped than Yeoman's run loop.

---

## 2. Modern Scaffolding CLIs (create-* tools)

**Sources:** `docs/create-next-app-docs.md`, `docs/create-t3-app-overview.md`, `docs/sv-create-docs.md`

### The Pattern

Modern scaffolding CLIs follow a consistent flow: **prompt → copy → transform → install → initialize**. They are single-purpose (one framework, one stack) and ship as `npm create` packages invoked via `npx`.

### create-next-app (Next.js)

The gold standard for progressive disclosure in a scaffolding CLI:

```
What is your project named? my-app
Would you like to use the recommended Next.js defaults?
    Yes, use recommended defaults  ← first option, most users stop here
    No, reuse previous settings    ← preference persistence
    No, customize settings         ← progressive disclosure gate
```

Only if you choose "customize settings" do you see the 8 detailed questions. Every prompt has a corresponding CLI flag (`--ts`, `--tailwind`, `--no-eslint`), enabling fully non-interactive use. The `--yes` flag accepts all defaults or stored preferences.

Notable: create-next-app v16+ now generates `AGENTS.md` and `CLAUDE.md` by default — the ecosystem recognizes AI coding agents as first-class project consumers.

### create-t3-app

Follows the "curated menu" philosophy. Three design axioms:

1. **Solve specific problems** — Include only what addresses concrete stack challenges
2. **Responsible innovation** — Bet on things trivial to remove (tRPC yes, novel DB no)
3. **Typesafety is non-negotiable** — The one thing that's never optional

Each technology (Prisma, tRPC, NextAuth, Tailwind) is independently selectable. The CLI generates different boilerplate depending on the combination selected. This is a scaffolding tool, not a framework — once generated, it's your code.

### sv create (SvelteKit)

Offers three templates (minimal, demo, library) and an add-ons system. The `--add` flag enables composing additional tooling (eslint, prettier) during creation. Has `--no-add-ons` to suppress the interactive add-ons prompt entirely.

### Common Patterns Across create-* Tools

| Pattern | Description | Adoption |
|---------|-------------|----------|
| **Recommended defaults first** | The "just works" option is always the first choice | create-next-app, sv create |
| **Preference persistence** | Remember previous answers for next run | create-next-app |
| **Progressive disclosure** | Detailed options hidden behind a gate | create-next-app |
| **CLI flags for everything** | Every interactive prompt has a non-interactive equivalent | All |
| **Modular selection** | Toggle individual features on/off | create-t3-app |
| **Package manager detection** | Auto-detect or let user choose npm/pnpm/yarn/bun | All |
| **Post-scaffold actions** | Install deps, init git, print next steps | All |

### Relevance to gdev

**Very high relevance.** The gdev extension should adopt create-next-app's three-tier disclosure: (1) accept all defaults, (2) reuse previous settings, (3) customize. The create-t3-app model of independent feature toggles maps well to devenv.sh configuration (enable Go? enable Node? enable Postgres?). Every wizard question must have a CLI flag equivalent for headless/CI mode — gdev's bootstrap already supports headless mode.

---

## 3. Template-Based Generators (Cookiecutter / Copier)

**Sources:** `docs/copier-comparisons.md`, `docs/cookiecutter-hooks.md`

### Cookiecutter

A cross-platform tool that generates projects from templates using Jinja2 templating. Variables are defined in `cookiecutter.json`, and the user is prompted for each one. Hooks run at three points:

1. **pre_prompt** — Before prompts (prerequisite checks)
2. **pre_gen_project** — After prompts, before file generation (input validation)
3. **post_gen_project** — After generation (cleanup, conditional file deletion)

If a hook exits non-zero, the generated directory is cleaned up. Hooks support Jinja templating, enabling conditional behavior based on user choices.

### Copier

Evolved beyond scaffolding into "lifecycle management." Key differentiator: **template updates**. When a template is updated (new version tag), Copier can re-apply the template to an existing project, merging changes. This addresses the fundamental weakness of one-shot scaffolders: projects drift from their template immediately after generation.

Feature comparison:

| Feature | Copier | Cookiecutter | Yeoman |
|---------|--------|-------------|--------|
| Config format | YAML | JSON | JavaScript |
| Template engine | Jinja | Jinja | EJS |
| Updates/migrations | Yes (native) | No (Cruft adds it) | No |
| Hook types | Task + context | Task + context | Task only |
| Loop generation | Yes | No | No |
| Template location | Git/bundle/folder | Git/Hg/Zip/subfolder | npm |

### Relevance to gdev

**Medium relevance for the update story.** One-shot scaffolding is the easy part. The hard question is: what happens when the gdev extension's template for devenv.nix or CLAUDE.md evolves? Copier's approach (version-tagged templates, diff-based updates) is the most mature answer. gdev extensions should consider whether generated config files need an update path or are truly one-shot.

---

## 4. Degit — Lightweight Scaffolding

**Source:** `docs/degit-readme.md`

### How It Works

Degit downloads the latest commit of a git repo as a tarball — no `.git` directory, no history. It's a copy tool, not a generator. No templating, no prompting, no hooks (beyond a basic `degit.json` actions file for post-clone file manipulation).

### Strengths
- Extremely fast (tarball download, cached)
- Zero configuration for template authors
- Works with any git hosting (GitHub, GitLab, BitBucket, Sourcehut)
- Supports targeting branches, tags, commits, and subdirectories

### Weaknesses
- No variable substitution or templating
- No prompting — template is used as-is
- No update mechanism
- Actions system is minimal (clone, remove only)

### Relevance to gdev

**Low direct relevance, but instructive as the minimalist extreme.** Degit proves that for some use cases, a plain file copy with no wizard is the right answer. If gdev's default devenv.nix or CLAUDE.md templates are good enough for 80% of projects, a degit-style "just copy the files" mode should be the fast path, with the wizard reserved for customization.

---

## 5. Interactive Prompt UI Libraries

### JavaScript Ecosystem

**Source:** `docs/clack-vs-inquirer-vs-enquirer.md`

The JS ecosystem has converged on three tiers:

| Library | Bundle Size | Best For | Key Feature |
|---------|------------|----------|-------------|
| **@clack/prompts** | ~2KB | Standard wizards | `group()` API with centralized cancellation |
| **Enquirer** | ~100KB | Advanced prompts | 15+ prompt types (autocomplete, scale, etc.) |
| **Ink** | ~150KB+React | Stateful TUIs | React components in terminal |

**@clack/prompts** is the modern default. Its `group()` API is particularly relevant — it runs prompts sequentially and returns all values as an object, with a centralized `onCancel` callback. Cancellation returns `Symbol('cancel')` rather than undefined, forcing explicit handling. TypeScript-native, ESM-native, beautifully styled out of the box.

**Inquirer.js** is the legacy standard (powers Yeoman). Still actively maintained but feels heavy compared to clack. The modular `@inquirer/prompts` rewrite addressed bundle size but the API remained verbose.

### Go Ecosystem

**Sources:** `docs/charmbracelet-huh-library.md`, `docs/go-interactive-cli-prompts.md`, `docs/go-cli-packages-evaluation.md`

| Library | Status | Best For | Key Feature |
|---------|--------|----------|-------------|
| **huh** (charmbracelet) | Active, v2 | Forms/wizards | Form > Group > Field hierarchy, dynamic forms |
| **bubbletea** | Active | Full TUI apps | Elm Architecture for terminals |
| **survey** | Archived | Simple prompts | Was the standard, now deprecated |
| **promptui** | Maintained | Simple prompts | Prompt + Select, integrates with Cobra |
| **go-prompt** | Maintained | REPL-style | Tab completion, suggestions |

**charmbracelet/huh is the clear winner for gdev.** It provides:

- **Form > Group > Field hierarchy** — Groups are "pages" in a wizard. Fields are inputs within a page. This maps directly to gdev bootstrap steps.
- **7 field types**: Input, Text, Select, MultiSelect, Confirm, FilePicker, Note
- **Dynamic forms**: `TitleFunc`, `OptionsFunc`, `DescriptionFunc` recompute when watched variables change. This enables conditional questions (e.g., "Which Node version?" only appears if Node was selected).
- **Built-in validation**: `ValidateNotEmpty()`, `ValidateMinLength()`, custom validators
- **Accessibility mode**: Falls back to plain text prompts for screen readers (set via `ACCESSIBLE` env var)
- **Themes**: Charm, Dracula, Catppuccin, Base16, Default — or custom
- **Layout**: Single column, multi-column, grid
- **Standalone execution**: Individual fields can run without a form wrapper
- **Bubble Tea integration**: Forms can be embedded in larger TUI applications
- **Spinner**: Built-in spinner for post-wizard async operations

Example of the wizard pattern in huh:

```go
form := huh.NewForm(
    // Page 1: Project basics
    huh.NewGroup(
        huh.NewInput().Title("Project name").Value(&name),
        huh.NewSelect[string]().Title("Language").
            Options(huh.NewOptions("Go", "TypeScript", "Python")...).
            Value(&lang),
    ),
    // Page 2: Conditional on language selection
    huh.NewGroup(
        huh.NewSelect[string]().
            TitleFunc(func() string { return lang + " version" }, &lang).
            OptionsFunc(func() []huh.Option[string] {
                return versionsFor(lang)
            }, &lang).
            Value(&version),
    ),
)
err := form.Run()
```

**survey** (AlecAivazis) was the previous standard but is archived. Its README redirects to Bubble Tea. Do not use for new projects.

**promptui** is still maintained and integrates well with Cobra, but lacks multi-select, dynamic forms, and the modern UX polish of huh.

---

## 6. Opinionated Defaults vs Flexibility

### The Spectrum

Tools fall on a spectrum from fully opinionated to fully flexible:

```
Fully Opinionated                                    Fully Flexible
    degit ←── create-next-app ←── create-t3-app ←── cookiecutter ←── Yeoman
   (no questions)  (defaults+gate)  (curated menu)   (all templated) (code anything)
```

### What Works Best

The ecosystem has converged on what we might call the **"opinionated menu with escape hatches"** pattern:

1. **Strong defaults that work for 80% of users** — The "just press Enter" path produces a working, well-configured project
2. **A gate before detailed customization** — "Would you like to customize?" prevents overwhelming new users
3. **Independent feature toggles** — Each optional feature is a yes/no decision, not a complex configuration form
4. **CLI flags for everything** — Non-interactive mode for CI, scripting, and power users
5. **Preference persistence** — Remember choices for next time

### Progressive Disclosure Patterns

Progressive disclosure in CLI wizards takes three forms:

| Pattern | How It Works | Example |
|---------|-------------|---------|
| **Accept-defaults gate** | Single question gates access to all detailed options | create-next-app "use recommended defaults?" |
| **Expanding sections** | Advanced options appear only after basic ones are answered | Yeoman's priority groups |
| **Conditional questions** | Questions appear based on previous answers | huh's `OptionsFunc` with watched variables |

### The Right Number of Questions

Analysis of successful scaffolding tools suggests:

- **0 questions** (degit): Only works when the template is the product (Svelte starter)
- **1-3 questions**: Ideal for focused tools (project name + 1-2 key decisions)
- **4-8 questions**: Maximum for a comfortable wizard experience
- **9+ questions**: Requires grouping into pages/steps or progressive disclosure

create-next-app asks 1 question for default users, up to 9 for customizers. create-t3-app asks 4-6. Both feel fast. Yeoman generators that asked 15+ questions felt like tax forms.

### Relevance to gdev

For devenv.sh setup, the wizard should ask:
1. **Project name** (auto-detected from directory)
2. **Language(s)** (multi-select: Go, TypeScript, Python, Rust, etc.)
3. Per-language version (conditional, only for selected languages)
4. **Services** (multi-select: PostgreSQL, Redis, etc.) — only if relevant
5. **Accept defaults for everything else?** (confirm gate)

That's 3-5 questions for the common case, expandable to more via the gate. For Claude Code, it's even simpler — likely just "Enable Claude Code configuration? [Y/n]" plus optionally selecting which MCP servers to configure.

---

## 7. Cross-Cutting Analysis

### Pattern Taxonomy

Across all surveyed tools, wizard patterns fall into four architectural approaches:

| Approach | Examples | Mechanism | Best For |
|----------|----------|-----------|----------|
| **Phase-based lifecycle** | Yeoman, gdev bootstrap | Fixed phases (prompt → configure → write → install) | Complex multi-addon composition |
| **Sequential prompts** | create-*, clack group() | Linear question flow, output at end | Single-purpose scaffolding |
| **Template expansion** | cookiecutter, copier | Variables → Jinja → files | Config-driven generation |
| **File copy** | degit | Download → extract | Zero-config templates |

### How gdev Should Combine Them

gdev's bootstrap system is already a phase-based lifecycle. The extension design should:

1. **Use huh for the prompt UI layer** — Form > Group > Field maps to bootstrap's step hierarchy
2. **Use sequential prompts within each bootstrap step** — Each step collects its own inputs
3. **Use template expansion for file generation** — Go's `text/template` for devenv.nix, CLAUDE.md, etc.
4. **Support file copy as a fast path** — Pre-built templates for common stacks (Go+Postgres, TypeScript+Tailwind, etc.)

### Failure Modes

| Failure Mode | Seen In | Mitigation |
|-------------|---------|------------|
| Asking too many questions | Yeoman generators | Progressive disclosure, sensible defaults |
| No non-interactive mode | Early scaffolders | CLI flags for every prompt, headless support |
| No update path | create-*, cookiecutter | Copier-style version tracking (or accept one-shot) |
| Composed generators conflict | Yeoman | Isolated step outputs, no shared file system |
| Wizard output not inspectable | Most tools | Dry-run/plan preview before writing (like `terraform plan`) |

### The "Plan Preview" Pattern

One pattern that's underexplored in the scaffolding space but well-established in infrastructure tooling (Terraform, Nix): **show the user what will be generated before generating it.** gdev's bootstrap already has a plan system (base plan + derived plans with exceptions). The wizard should:

1. Collect all inputs
2. Show a summary: "I will create: devenv.nix (Go 1.22, PostgreSQL 16), CLAUDE.md (with MCP servers: postgres), .claude/settings.json"
3. Ask for confirmation
4. Generate files

This maps to create-next-app's pattern of showing the selected configuration before proceeding, and to Terraform's `plan` → `apply` flow.

---

## 8. Recommendations for gdev Extension Design

### Recommended UI Library: charmbracelet/huh

- Native Go, actively maintained, excellent API design
- Form > Group > Field hierarchy maps to gdev's bootstrap steps
- Dynamic forms enable conditional wizard paths
- Accessibility mode for screen reader users
- Theming for consistent visual identity across gdev addons
- Spinner for post-wizard operations (file generation, dependency installation)

### Recommended UX Pattern: Opinionated Menu with Escape Hatches

```
$ gdev init

  Welcome to gdev! Let's set up your development environment.

  Project: my-project (detected from directory)

  ┌ Quick Setup
  │
  ◆ Use recommended defaults for Go project?
  │  ● Yes — Go 1.22, devenv.sh, Claude Code, standard packages
  │  ○ No, customize
  │
  └ Creating devenv.nix, CLAUDE.md, .claude/settings.json...
     Done! Run `gdev start` to begin.
```

For the "customize" path:

```
  ┌ Languages
  │
  ◆ Select languages (space to toggle, enter to confirm)
  │  ☑ Go
  │  ☐ TypeScript
  │  ☐ Python
  │  ☐ Rust
  │
  ◆ Go version
  │  ● 1.22 (latest stable)
  │  ○ 1.21
  │  ○ 1.20
  │
  ├ Services
  │
  ◆ Select services
  │  ☐ PostgreSQL
  │  ☐ Redis (Valkey)
  │  ☐ MariaDB
  │
  ├ Dev Tools
  │
  ◆ Configure Claude Code? Yes
  ◆ Configure pre-commit hooks? Yes
  │
  ├ Review
  │
  │  Will create:
  │    devenv.nix    — Go 1.22, standard packages
  │    CLAUDE.md     — Project instructions
  │    .claude/      — Settings, permissions
  │
  ◆ Proceed? Yes
  │
  └ Done!
```

### Recommended Architecture

```
gdev bootstrap step
    └── huh Form
            ├── Group 1: Quick vs Custom
            ├── Group 2: Languages (conditional)
            ├── Group 3: Services (conditional)
            ├── Group 4: Dev Tools
            └── Group 5: Review & Confirm
                    └── File generation via Go templates
```

### Non-Interactive Mode

Every wizard question must have a CLI flag equivalent:

```bash
gdev init --lang=go --go-version=1.22 --services=postgres \
          --claude-code --no-precommit --yes
```

This maps to gdev bootstrap's existing headless mode support.

### Key Design Principles (distilled from ecosystem survey)

1. **Detect before asking** — Auto-detect project name, existing languages, current Go version. Pre-fill answers.
2. **Defaults should be the right answer** — If you detect a Go project, default to Go enabled. If `.git` exists, default git init to off.
3. **Group related questions** — Languages together, services together, dev tools together. One group per wizard page.
4. **Show what you'll do before doing it** — Plan preview before file generation.
5. **Make it re-runnable** — `gdev init` on an existing project should detect existing config and offer to update/merge, not overwrite.
6. **Keep the wizard under 30 seconds** — If the happy path takes longer, you're asking too much.

---

## Sources

All raw source material saved to `docs/`:

| File | Source | Topic |
|------|--------|-------|
| `yeoman-composability.md` | yeoman.io | Yeoman composability system |
| `yeoman-runtime-context.md` | yeoman.io | Yeoman run loop and priority groups |
| `copier-comparisons.md` | copier.readthedocs.io | Copier vs Cookiecutter vs Yeoman |
| `cookiecutter-hooks.md` | cookiecutter.readthedocs.io | Cookiecutter hook lifecycle |
| `create-next-app-docs.md` | nextjs.org | create-next-app CLI documentation |
| `create-t3-app-overview.md` | github.com/t3-oss, create.t3.gg | create-t3-app design philosophy |
| `sv-create-docs.md` | svelte.dev | Svelte CLI sv create command |
| `clack-vs-inquirer-vs-enquirer.md` | pkgpulse.com | JS prompt library comparison |
| `go-interactive-cli-prompts.md` | dev.to/tidalcloud | Go CLI prompt approaches |
| `charmbracelet-huh-library.md` | pkg.go.dev | huh library full documentation |
| `go-cli-packages-evaluation.md` | medium.com | survey/promptui/go-prompt comparison |
| `degit-readme.md` | github.com/Rich-Harris | degit scaffolding tool |
| `cli-ux-patterns.md` | medium.com | CLI UX patterns (first-run wizard, etc.) |
