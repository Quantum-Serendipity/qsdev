# gdev Extension Architecture Design

## Decision: Separate Addons with Shared Init Orchestration

### Architecture

Three addons, each following gdev's standard pattern:

```
addons/
├── devenv/          # devenv.sh environment management
├── claudecode/      # Claude Code AI assistant configuration
└── devinit/         # Orchestration: combined init wizard + shared logic
```

### Rationale

**Why separate `devenv` and `claudecode` addons:**
- Follows gdev's established pattern: each addon owns one concern (postgres owns PostgreSQL, docker owns Docker, etc.)
- Teams may want devenv.sh without Claude Code, or vice versa
- Each addon registers its own bootstrap steps, config keys, and commands independently
- Go module boundaries keep dependency trees clean (claudecode doesn't pull in Nix-related deps)

**Why a `devinit` orchestration addon:**
- The combined init wizard (project type → languages → services → dev tools → AI config) spans both addons
- `devinit` provides `gdev init` command that runs a unified huh wizard, then delegates to devenv and claudecode for file generation
- Without devinit, teams that want both addons would need to run two separate wizards
- devinit is optional — teams can use devenv and claudecode independently via their own commands

### Addon Composition Model

```go
// In the team's main.go:
func main() {
    instance.SetAppName("myxdev")
    
    // Core gdev addons
    bootstrap.Configure(...)
    build.Configure(...)
    
    // New addons — order doesn't matter, gdev handles initialization
    devenv.Configure(
        devenv.WithDefaultLanguages("go", "typescript"),
        devenv.WithDefaultServices("postgres", "redis"),
        devenv.WithDirenv(true),
    )
    claudecode.Configure(
        claudecode.WithSkillLibrary(embeddedSkills),
        claudecode.WithDefaultPermissions(teamPermissions),
        claudecode.WithDefaultHooks(teamHooks),
    )
    devinit.Configure(
        devinit.WithDetectProjectType(true),
        devinit.WithPlanPreview(true),
    )
    
    cmd.Main()
}
```

### Extension Points Used by Each Addon

| Extension Point | devenv | claudecode | devinit |
|----------------|--------|------------|---------|
| Addon registration | ✅ | ✅ | ✅ |
| Config keys | ✅ (languages, services, direnv) | ✅ (permissions, skills, hooks) | ✅ (last-used profile) |
| CLI commands | `gdev devenv init/update/add-*` | `gdev claude init/update/add-*` | `gdev init` |
| Bootstrap steps | devenv setup steps | claude code setup steps | combined wizard step |
| Context entries | DevenvConfig (shared state) | ClaudeConfig (shared state) | — |

### Data Flow

```
User runs: gdev init
    │
    ▼
devinit addon's init command
    │
    ├── Phase 1: Detection
    │   ├── Scan for go.mod, package.json, Cargo.toml, etc.
    │   ├── Scan for existing devenv.nix, .claude/, etc.
    │   └── Build default answers from detection
    │
    ├── Phase 2: Wizard (huh forms)
    │   ├── Group 1: Quick vs Custom (1 question)
    │   ├── Group 2: Languages & versions (conditional)
    │   ├── Group 3: Services (conditional)
    │   ├── Group 4: Dev tools — devenv options (git hooks, scripts, direnv)
    │   ├── Group 5: Claude Code options (permissions, skills, hooks)
    │   └── Group 6: Plan preview + confirm
    │
    ├── Phase 3: Generation
    │   ├── Call devenv.Generate(wizardAnswers) → devenv.yaml, devenv.nix, .envrc
    │   ├── Call claudecode.Generate(wizardAnswers) → CLAUDE.md, .claude/*, .mcp.json
    │   └── Update .gitignore
    │
    └── Phase 4: Post-generation
        ├── Save answers to gdev config (for re-run)
        ├── Print summary of generated files
        └── Suggest next steps ("run devenv shell to activate")
```

### Config Key Namespacing

Each addon owns a config namespace:

```yaml
# ~/.config/myxdev.yaml
devenv:
  languages:
    - name: go
      version: "1.22"
    - name: typescript
      version: "22"
  services:
    - postgres
    - redis
  direnv: true
  git_hooks:
    - pre-commit

claudecode:
  permissions:
    allow:
      - "Bash(npm run *)"
      - "Bash(go test *)"
  skills:
    - deploy
    - security-review
  hooks:
    auto_format: true
  mcp_servers:
    - github

devinit:
  last_profile: "go-web-service"
  last_run: "2026-05-12"
```

### Command Hierarchy

```
gdev init                      # devinit: combined wizard
gdev init --yes                # devinit: accept defaults
gdev init --profile go-web     # devinit: use named profile

gdev devenv init               # devenv: standalone wizard
gdev devenv update             # devenv: re-generate from config
gdev devenv add-service redis  # devenv: add service post-init
gdev devenv add-language rust  # devenv: add language post-init

gdev claude init               # claudecode: standalone wizard
gdev claude update             # claudecode: re-generate from config
gdev claude add-skill deploy   # claudecode: add skill from library
gdev claude add-hook format    # claudecode: add standard hook
```

### Inter-Addon Communication

devinit needs to pass wizard answers to devenv and claudecode. Two approaches:

**Option A: Context entries (recommended)**
- devinit writes a `WizardAnswers` struct to resource context
- devenv and claudecode read from context during generation
- Follows gdev's established DI pattern

**Option B: Direct function calls**
- devinit imports devenv and claudecode packages
- Calls `devenv.Generate(answers)` and `claudecode.Generate(answers)` directly
- Simpler but creates compile-time dependency

**Recommendation: Option B** — devinit already depends on both addons conceptually. Context DI is for runtime service sharing; this is a build-time orchestration concern. Keep it simple.

### Profile System

devinit supports named profiles — pre-configured answer sets for common project types:

```go
devinit.Configure(
    devinit.WithProfile("go-web", Profile{
        Languages: []Language{{Name: "go", Version: "1.22"}},
        Services:  []string{"postgres", "redis"},
        Direnv:    true,
        ClaudeCode: true,
        Skills:    []string{"deploy", "security-review"},
    }),
    devinit.WithProfile("ts-fullstack", Profile{
        Languages: []Language{
            {Name: "typescript", Version: "22"},
            {Name: "go", Version: "1.22"},
        },
        Services:  []string{"postgres", "redis", "elasticsearch"},
        Direnv:    true,
        ClaudeCode: true,
        Skills:    []string{"deploy", "review"},
    }),
)
```

Usage: `gdev init --profile go-web` skips the wizard entirely.

### Bootstrap Integration

When used via `gdev bootstrap` (full system setup), the addons register steps in the bootstrap plan:

```go
// devenv addon registers:
bootstrap.Configure(
    bootstrap.WithSteps(
        devenv.InstallDevenvStep(),     // ensures devenv CLI is installed
        devenv.InstallDirenvStep(),     // ensures direnv is installed
    ),
)

// claudecode addon registers:
bootstrap.Configure(
    bootstrap.WithSteps(
        claudecode.InstallClaudeStep(), // ensures claude CLI is installed
    ),
)
```

These are system-level installation steps, separate from per-project init. Bootstrap handles "install the tools"; init handles "configure them for this project."

### File Ownership Rules

Clear ownership prevents conflicts:

| File | Owner | Created by |
|------|-------|------------|
| `devenv.yaml` | devenv addon | devenv.Generate() |
| `devenv.nix` | devenv addon | devenv.Generate() |
| `.envrc` | devenv addon | devenv.Generate() |
| `CLAUDE.md` | claudecode addon | claudecode.Generate() |
| `.claude/settings.json` | claudecode addon | claudecode.Generate() |
| `.claude/skills/*` | claudecode addon | claudecode.Generate() |
| `.claude/rules/*` | claudecode addon | claudecode.Generate() |
| `.mcp.json` | claudecode addon | claudecode.Generate() |
| `.gitignore` | devinit addon | devinit (appends entries from both) |

### Error Handling

- Detection failures (can't read go.mod, etc.) → fall back to manual selection, don't abort
- Generation failures → report which file failed, roll back all generated files
- Partial existing config → merge mode (see migration strategy doc)
- Missing tool (devenv not installed) → offer to install via bootstrap step, or skip
