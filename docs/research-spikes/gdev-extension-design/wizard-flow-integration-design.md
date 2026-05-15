# Wizard Flow Integration Design

## How huh Forms Integrate with gdev's Bootstrap Step System

### Core Mapping

gdev's bootstrap system runs ordered steps. Each step can collect user input and perform actions. charmbracelet/huh provides form-based interactive UI. The integration:

```
Bootstrap Step  →  huh Form  →  Form Groups (pages)  →  Fields (questions)
```

One bootstrap step = one huh Form with multiple Groups. This gives us:
- Page-by-page progression within a step
- Validation per field and per group
- Conditional fields via huh's dynamic functions
- Theming and accessibility from huh

### The `qsdev init` Wizard Flow

#### Quick Path (1 question, <5 seconds)

```
┌─────────────────────────────────────────────┐
│  Welcome to myxdev!                         │
│                                             │
│  Detected: Go project (go.mod found)        │
│                                             │
│  ◆ Set up with recommended defaults?        │
│    ● Yes — Go 1.22, devenv.sh, Claude Code  │
│    ○ No, let me customize                   │
│    ○ Show me what "defaults" means first    │
└─────────────────────────────────────────────┘
```

If "Yes" → skip to plan preview → generate.
If "Show me" → display the default profile, then ask again.
If "No" → enter customization flow.

#### Customization Flow (5 groups, ~30 seconds)

**Group 1: Languages**
```
┌─────────────────────────────────────────────┐
│  Languages & Runtimes                [1/5]  │
│                                             │
│  ◆ Select languages:                        │
│    ✓ Go           (detected: go.mod)        │
│    ○ TypeScript                             │
│    ○ Python                                 │
│    ○ Rust                                   │
│    ○ Other...                               │
│                                             │
│  ◆ Go version:  [ 1.22 ▾ ]                 │
└─────────────────────────────────────────────┘
```

- Auto-detect existing project files and pre-select
- Version dropdowns populated from devenv.sh supported versions
- "Other..." opens a text input for nixpkgs package name

**Group 2: Services**
```
┌─────────────────────────────────────────────┐
│  Services                            [2/5]  │
│                                             │
│  ◆ Select services to run locally:          │
│    ○ PostgreSQL                             │
│    ○ MySQL/MariaDB                          │
│    ○ Redis/Valkey                           │
│    ○ Elasticsearch                          │
│    ○ RabbitMQ                               │
│    ○ Kafka                                  │
│    ○ MinIO (S3)                             │
│    ○ None                                   │
└─────────────────────────────────────────────┘
```

- Multi-select
- Service-specific follow-up questions only for selected services (e.g., PostgreSQL version, initial database name)

**Group 3: Dev Environment Options**
```
┌─────────────────────────────────────────────┐
│  Development Environment             [3/5]  │
│                                             │
│  ◆ Enable direnv (.envrc)?          [Yes]   │
│  ◆ Git hooks?                               │
│    ✓ pre-commit (lint)                      │
│    ○ pre-push (test)                        │
│    ○ commit-msg (conventional commits)      │
│  ◆ Additional packages:                     │
│    [ jq, ripgrep, fd               ]        │
└─────────────────────────────────────────────┘
```

**Group 4: Claude Code**
```
┌─────────────────────────────────────────────┐
│  Claude Code Setup                   [4/5]  │
│                                             │
│  ◆ Configure Claude Code?           [Yes]   │
│  ◆ Permission level:                        │
│    ● Standard — allow build/test/lint       │
│    ○ Restricted — read-only + approved cmds │
│    ○ Custom...                              │
│  ◆ Install team skills?                     │
│    ✓ deploy                                 │
│    ✓ security-review                        │
│    ○ review                                 │
│  ◆ Auto-format hook?                [Yes]   │
│  ◆ MCP servers?                             │
│    ○ GitHub                                 │
│    ○ None                                   │
└─────────────────────────────────────────────┘
```

**Group 5: Plan Preview & Confirm**
```
┌─────────────────────────────────────────────┐
│  Review & Confirm                    [5/5]  │
│                                             │
│  Will generate:                             │
│                                             │
│  devenv.yaml       inputs, Go 1.22          │
│  devenv.nix        Go, PostgreSQL, redis    │
│  .envrc            direnv auto-activate     │
│  CLAUDE.md         Go web service config    │
│  .claude/                                   │
│    settings.json   standard permissions     │
│    skills/         deploy, security-review  │
│    rules/          (none)                   │
│  .mcp.json         (none)                   │
│  .gitignore        +4 entries               │
│                                             │
│  ◆ Proceed?                                 │
│    ● Yes, generate files                    │
│    ○ Go back and edit                       │
│    ○ Cancel                                 │
└─────────────────────────────────────────────┘
```

### Implementation: huh Form Construction

```go
func buildInitForm(detected DetectedProject, defaults Profile) *huh.Form {
    var answers WizardAnswers
    
    // Pre-populate from detection
    answers.Languages = detected.Languages
    
    quickGroup := huh.NewGroup(
        huh.NewSelect[string]().
            Title("Set up with recommended defaults?").
            Options(
                huh.NewOption("Yes — "+defaults.Summary(), "yes"),
                huh.NewOption("No, let me customize", "customize"),
                huh.NewOption("Show defaults first", "show"),
            ).
            Value(&answers.QuickChoice),
    )
    
    langGroup := huh.NewGroup(
        huh.NewMultiSelect[string]().
            Title("Select languages").
            Options(languageOptions(detected)...).
            Value(&answers.SelectedLanguages),
        // Version selects added dynamically per language
    ).WithHideFunc(func() bool {
        return answers.QuickChoice == "yes"
    })
    
    servicesGroup := huh.NewGroup(
        huh.NewMultiSelect[string]().
            Title("Select services to run locally").
            Options(serviceOptions()...).
            Value(&answers.SelectedServices),
    ).WithHideFunc(func() bool {
        return answers.QuickChoice == "yes"
    })
    
    devenvGroup := huh.NewGroup(
        huh.NewConfirm().
            Title("Enable direnv (.envrc)?").
            Value(&answers.Direnv),
        huh.NewMultiSelect[string]().
            Title("Git hooks").
            Options(hookOptions(answers.SelectedLanguages)...).
            Value(&answers.GitHooks),
        huh.NewInput().
            Title("Additional packages (comma-separated)").
            Value(&answers.ExtraPackages),
    ).WithHideFunc(func() bool {
        return answers.QuickChoice == "yes"
    })
    
    claudeGroup := huh.NewGroup(
        huh.NewConfirm().
            Title("Configure Claude Code?").
            Value(&answers.ClaudeCode),
        huh.NewSelect[string]().
            Title("Permission level").
            Options(
                huh.NewOption("Standard — allow build/test/lint", "standard"),
                huh.NewOption("Restricted — read-only + approved", "restricted"),
                huh.NewOption("Custom...", "custom"),
            ).
            Value(&answers.PermissionLevel),
        huh.NewMultiSelect[string]().
            Title("Install team skills").
            Options(skillOptions()...).
            Value(&answers.Skills),
    ).WithHideFunc(func() bool {
        return answers.QuickChoice == "yes" || !answers.ClaudeCode
    })
    
    previewGroup := huh.NewGroup(
        huh.NewNote().
            Title("Review & Confirm").
            Description(buildPlanPreview(answers)),
        huh.NewConfirm().
            Title("Proceed?").
            Value(&answers.Confirmed),
    )
    
    return huh.NewForm(
        quickGroup,
        langGroup,
        servicesGroup,
        devenvGroup,
        claudeGroup,
        previewGroup,
    ).WithTheme(huh.ThemeDracula())
}
```

### Progressive Disclosure Mechanics

Three levels of detail, controlled by `WithHideFunc`:

1. **Quick path**: Only Group 1 (quick select) and Group 5 (preview) are visible
2. **Standard customize**: Groups 1-5 visible, with sane defaults pre-filled
3. **Advanced**: Within each group, an "Advanced options..." confirm field can reveal additional fields (e.g., Nix overlay configuration, custom devenv imports, sandbox network rules)

```go
// Within a group, advanced fields hidden by default
huh.NewConfirm().
    Title("Show advanced options?").
    Value(&answers.ShowAdvanced),

huh.NewInput().
    Title("Custom Nix overlay URL").
    Value(&answers.NixOverlay).
    WithHideFunc(func() bool { return !answers.ShowAdvanced }),
```

### Non-Interactive / CI Mode

Every wizard question maps to a CLI flag. The form is never shown when flags provide all answers.

```bash
# Full non-interactive
qsdev init \
    --lang go --go-version 1.22 \
    --lang typescript --ts-version 22 \
    --service postgres --service redis \
    --direnv \
    --git-hooks pre-commit \
    --claude-code \
    --claude-permissions standard \
    --claude-skills deploy,security-review \
    --yes

# Profile shortcut (equivalent to above)
qsdev init --profile go-web --yes

# Partial flags + interactive for the rest
qsdev init --lang go --service postgres
# → wizard opens but pre-fills Go and PostgreSQL, asks remaining questions
```

Implementation:

```go
func (c *initCommand) Run(cmd *cobra.Command, args []string) error {
    answers := c.answersFromFlags(cmd.Flags())
    
    if !answers.IsComplete() && !c.nonInteractive {
        // Run wizard for missing answers
        form := buildInitForm(c.detected, c.defaults)
        form.WithAccessible(c.accessible)
        if err := form.Run(); err != nil {
            return err
        }
    } else if !answers.IsComplete() && c.nonInteractive {
        // Fill remaining with defaults
        answers.FillDefaults(c.defaults)
    }
    
    if !answers.Confirmed {
        return nil // user cancelled
    }
    
    return c.generate(answers)
}
```

### gdev Bootstrap Step Integration

When running `qsdev bootstrap` (full system setup), the init wizard is NOT automatically included — bootstrap handles system-level tool installation, not per-project config. But teams can add an init step:

```go
// In the team's main.go, optionally:
bootstrap.Configure(
    bootstrap.WithSteps(
        devenv.InstallDevenvStep(),      // system: install devenv CLI
        claudecode.InstallClaudeStep(),   // system: install claude CLI
        // devinit.ProjectSetupStep(),    // per-project: only if desired in bootstrap
    ),
)
```

The normal workflow is:
1. `qsdev bootstrap` → installs system tools (devenv, claude, direnv)
2. `cd my-project && qsdev init` → configures project environment

### Detection Engine

Before the wizard runs, detect existing project state:

```go
type DetectedProject struct {
    // Language detection
    HasGoMod       bool
    GoVersion      string   // from go.mod
    HasPackageJSON bool
    NodeVersion    string   // from .nvmrc or package.json engines
    HasCargoToml   bool
    HasPyProject   bool
    
    // Existing config detection
    HasDevenvNix   bool
    HasDevenvYaml  bool
    HasClaudeDir   bool
    HasClaudeMd    bool
    HasEnvrc       bool
    HasMcpJson     bool
    
    // Git detection
    IsGitRepo      bool
    HasGitHooks    bool
    RemoteURL      string   // for inferring project name
}

func Detect(projectRoot string) DetectedProject {
    // Stat files, parse go.mod/package.json for versions
    // Fast — no network calls, just filesystem reads
}
```

Detection results pre-populate wizard defaults and trigger merge mode when existing config is found.

### Merge Mode (Existing Projects)

When existing config is detected:

```
┌─────────────────────────────────────────────┐
│  Existing configuration detected!           │
│                                             │
│  Found: devenv.nix, CLAUDE.md               │
│                                             │
│  ◆ How should we handle existing files?     │
│    ● Merge — add new config, keep existing  │
│    ○ Overwrite — replace with fresh config  │
│    ○ Skip — only generate missing files     │
│    ○ Cancel                                 │
└─────────────────────────────────────────────┘
```

Merge strategy per file type:
- **devenv.yaml**: Deep merge YAML (add new inputs, preserve existing)
- **devenv.nix**: Cannot auto-merge Nix code — show diff, let user accept/reject
- **CLAUDE.md**: Append new sections, preserve existing content
- **.claude/settings.json**: Deep merge JSON (union permission lists)
- **.claude/skills/**: Add new skill files, don't touch existing
- **.gitignore**: Append only missing entries

### Theming

Use a consistent theme across all wizard forms. huh supports built-in themes (Charm, Dracula, Catppuccin, Base16) and custom themes. Teams can configure this:

```go
devinit.Configure(
    devinit.WithTheme(huh.ThemeCatppuccin()),
)
```

### Accessibility

huh's accessibility mode (`WithAccessible(true)`) falls back to simple text prompts for screen readers. Detect via environment:

```go
accessible := os.Getenv("ACCESSIBLE") != "" || 
              os.Getenv("NO_COLOR") != ""
form.WithAccessible(accessible)
```

### Post-Generation Output

After file generation, print a clear summary:

```
✓ Generated 7 files:

  devenv.yaml         Nix inputs, Go 1.22 overlay
  devenv.nix          Go, PostgreSQL 16, Redis
  .envrc              direnv auto-activation
  CLAUDE.md           Go web service instructions
  .claude/settings.json  standard permissions (12 allow rules)
  .claude/skills/deploy.md
  .claude/skills/security-review.md

Next steps:
  1. Run `devenv shell` to activate the environment
  2. Run `direnv allow` to enable auto-activation
  3. Review CLAUDE.md and customize for your project
```
