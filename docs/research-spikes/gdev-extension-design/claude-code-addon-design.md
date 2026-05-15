# Claude Code gdev Addon — Detailed Design

## 1. Overview

This document specifies the design for a `claudecode` gdev addon that bootstraps and manages Claude Code configuration for team projects. The addon follows gdev's standard addon pattern (generic struct, Configure/Initialize lifecycle, option pattern) and generates the full `.claude/` directory structure, `CLAUDE.md`, `.mcp.json`, and `.gitignore` entries from wizard-driven or CLI-flag-driven configuration.

The addon targets the project-level configuration layer (committed to git) and generates but does not commit user-local overrides. It does not touch managed/IT-enforced settings.

---

## 2. Addon Structure

### Package Layout

```
addons/claudecode/
  addon.go          # Addon definition, Configure(), Initialize()
  config.go         # Config struct and config keys
  commands.go       # CLI commands (init, update, add-skill, add-hook)
  bootstrap.go      # Bootstrap step registration
  wizard.go         # huh form construction and wizard flow
  templates.go      # Embedded template files and generation logic
  skills.go         # Team skill library management
  permissions.go    # Permission preset logic
  detect.go         # Project detection (language, framework, existing config)
  types.go          # Shared types (PermissionPreset, HookConfig, MCPServer, etc.)
  templates/        # Embedded template directory
    claude-md.tmpl
    settings-json.tmpl
    mcp-json.tmpl
    gitignore-append.tmpl
    rules/
      go-conventions.md
      typescript-conventions.md
      testing-conventions.md
      security-rules.md
    skills/
      deploy.md
      review-pr.md
      security-review.md
      generate-tests.md
      refactor.md
      db-migration.md
```

### Addon Registration

```go
package claudecode

import (
    "github.com/user/gdev/addons"
    "github.com/user/gdev/addons/bootstrap"
    "github.com/user/gdev/instance"
)

var addon = addons.Addon[Config]{
    Definition: addons.Definition{
        Name:        "claudecode",
        Description: func() string { return "Claude Code project configuration" },
        Initialize:  initialize,
    },
    Config: Config{},
}

func Configure(opts ...Option) {
    addon.CheckNotInitialized()
    for _, o := range opts {
        o(&addon.Config)
    }
    addon.RegisterIfNeeded()
}

func initialize() error {
    // Register CLI commands
    instance.AddCommands(
        claudeCmd(),      // parent: `qsdev claude`
        initCmd(),        // `qsdev claude init`
        updateCmd(),      // `qsdev claude update`
        addSkillCmd(),    // `qsdev claude add-skill <name>`
        addHookCmd(),     // `qsdev claude add-hook <type>`
        listSkillsCmd(),  // `qsdev claude list-skills`
    )

    // Register bootstrap steps
    bootstrap.WithSteps(
        detectStep(),
        instructionsStep(),
        permissionsStep(),
        skillsStep(),
        hooksStep(),
        mcpStep(),
        generateStep(),
    )

    return nil
}
```

---

## 3. Config Struct and Keys

### Config Type

```go
type Config struct {
    // Detection results (populated by wizard or flags)
    Languages     []Language       `yaml:"languages,omitempty"`
    Frameworks    []string         `yaml:"frameworks,omitempty"`
    BuildCommands []string         `yaml:"build_commands,omitempty"`
    TestCommands  []string         `yaml:"test_commands,omitempty"`
    LintCommands  []string         `yaml:"lint_commands,omitempty"`

    // CLAUDE.md generation
    ProjectDescription string      `yaml:"project_description,omitempty"`
    ArchitectureNotes  string      `yaml:"architecture_notes,omitempty"`
    CustomInstructions []string    `yaml:"custom_instructions,omitempty"`

    // Permissions
    PermissionPreset   PermissionPreset `yaml:"permission_preset,omitempty"`
    ExtraAllowPatterns []string         `yaml:"extra_allow_patterns,omitempty"`
    ExtraDenyPatterns  []string         `yaml:"extra_deny_patterns,omitempty"`
    SandboxEnabled     bool             `yaml:"sandbox_enabled,omitempty"`
    AllowedDomains     []string         `yaml:"allowed_domains,omitempty"`

    // Skills
    EnabledSkills []string `yaml:"enabled_skills,omitempty"`

    // Hooks
    EnabledHooks []HookPreset `yaml:"enabled_hooks,omitempty"`

    // MCP
    MCPServers []MCPServerConfig `yaml:"mcp_servers,omitempty"`

    // Team library
    SkillLibrarySource string `yaml:"skill_library_source,omitempty"`

    // Behavioral flags
    SkipDetection bool `yaml:"skip_detection,omitempty"`
}
```

### Supporting Types

```go
type Language struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version,omitempty"`
}

type PermissionPreset string

const (
    PermissionPresetMinimal    PermissionPreset = "minimal"
    PermissionPresetStandard   PermissionPreset = "standard"
    PermissionPresetPermissive PermissionPreset = "permissive"
    PermissionPresetCustom     PermissionPreset = "custom"
)

type HookPreset string

const (
    HookAutoFormat    HookPreset = "auto-format"
    HookSafetyBlock   HookPreset = "safety-block"
    HookPreCommit     HookPreset = "pre-commit"
    HookAuditLog      HookPreset = "audit-log"
)

type MCPServerConfig struct {
    Name    string            `yaml:"name"`
    Command string            `yaml:"command"`
    Args    []string          `yaml:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"`
}

type HookEntry struct {
    Matcher string `json:"matcher"`
    Command string `json:"command"`
}

type SettingsJSON struct {
    Permissions Permissions       `json:"permissions"`
    Sandbox     *SandboxConfig    `json:"sandbox,omitempty"`
    Hooks       map[string][]Hook `json:"hooks,omitempty"`
}

type Permissions struct {
    Allow []string `json:"allow"`
    Deny  []string `json:"deny"`
}

type SandboxConfig struct {
    WriteDeny  []string `json:"writeDeny,omitempty"`
    WriteAllow []string `json:"writeAllow,omitempty"`
    ReadDeny   []string `json:"readDeny,omitempty"`
    NetAllow   []string `json:"netAllow,omitempty"`
}

type Hook struct {
    Matcher string `json:"matcher"`
    Command string `json:"command"`
}
```

### Config Key Registration

```go
var configKey = config.NewKey[Config]("claudecode", Config{})

func init() {
    config.AddKey(configKey)
}
```

The addon registers a single top-level config key `claudecode` in gdev's YAML config. All wizard answers persist under this key, enabling `qsdev claude update` to regenerate files from saved config without re-running the wizard.

---

## 4. Bootstrap Steps

The addon registers 7 ordered bootstrap steps. Each step uses a `charmbracelet/huh` form group for interactive mode and reads CLI flags for non-interactive mode.

### Step 1: Claude Code Detection

**Purpose:** Verify Claude Code is installed, determine version, detect existing configuration.

```go
func detectStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-detect",
        Description: "Detect Claude Code installation",
        Run: func(ctx context.Context) error {
            // 1. Check `claude --version` on PATH
            version, err := detectClaudeVersion()
            if err != nil {
                // Not installed: show install instructions, offer to skip
                form := huh.NewForm(huh.NewGroup(
                    huh.NewNote().
                        Title("Claude Code Not Found").
                        Description("Claude Code CLI is not installed or not on PATH.\n\n"+
                            "Install: npm install -g @anthropic-ai/claude-code\n\n"+
                            "You can continue to generate config files without it."),
                    huh.NewConfirm().
                        Title("Continue without Claude Code?").
                        Value(&continueWithout),
                ))
                if err := form.Run(); err != nil {
                    return err
                }
                if !continueWithout {
                    return bootstrap.ErrStepSkipped
                }
            }

            // 2. Detect existing .claude/ directory
            existingConfig := detectExistingConfig(projectRoot)

            // 3. Detect project languages and frameworks
            detected := detectProject(projectRoot)

            // 4. Store detection results in step context
            setDetectionResult(ctx, &DetectionResult{
                ClaudeVersion:  version,
                ExistingConfig: existingConfig,
                Languages:      detected.Languages,
                Frameworks:     detected.Frameworks,
                BuildCommands:  detected.BuildCommands,
                TestCommands:   detected.TestCommands,
                LintCommands:   detected.LintCommands,
            })

            return nil
        },
    }
}
```

**Detection heuristics:**
- `go.mod` present -> Go (parse version from `go.mod`)
- `package.json` present -> Node.js/TypeScript (check for `tsconfig.json`)
- `pyproject.toml` or `requirements.txt` -> Python
- `Cargo.toml` -> Rust
- `flake.nix` or `shell.nix` -> Nix project
- `.claude/` directory exists -> existing config (offer merge vs overwrite)
- `Makefile`, `Taskfile.yml`, `justfile` -> extract build/test commands

### Step 2: Project Instructions (CLAUDE.md)

**Purpose:** Configure what goes into the project's CLAUDE.md file.

```go
func instructionsStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-instructions",
        Description: "Configure project instructions for Claude Code",
        Run: func(ctx context.Context) error {
            det := getDetectionResult(ctx)
            cfg := &addon.Config

            var (
                description    string
                archNotes      string
                useDefaults    bool
            )

            // Infer sensible defaults from detection
            defaultDesc := inferProjectDescription(det)
            defaultInstructions := inferInstructions(det)

            form := huh.NewForm(
                huh.NewGroup(
                    huh.NewConfirm().
                        Title("Use detected project settings?").
                        Description(fmt.Sprintf(
                            "Detected: %s\nBuild: %s\nTest: %s",
                            joinLanguages(det.Languages),
                            strings.Join(det.BuildCommands, ", "),
                            strings.Join(det.TestCommands, ", "),
                        )).
                        Value(&useDefaults),
                ),
                // Only shown if useDefaults == false
                huh.NewGroup(
                    huh.NewInput().
                        Title("Project description").
                        Description("One-line description for CLAUDE.md header").
                        Placeholder(defaultDesc).
                        Value(&description),
                    huh.NewText().
                        Title("Architecture notes").
                        Description("Key architectural decisions Claude should know about").
                        CharLimit(2000).
                        Value(&archNotes),
                ).WithHideFunc(func() bool { return useDefaults }),
            )

            if err := form.Run(); err != nil {
                return err
            }

            if useDefaults {
                cfg.ProjectDescription = defaultDesc
                cfg.CustomInstructions = defaultInstructions
            } else {
                cfg.ProjectDescription = description
                cfg.ArchitectureNotes = archNotes
            }
            cfg.Languages = det.Languages
            cfg.BuildCommands = det.BuildCommands
            cfg.TestCommands = det.TestCommands
            cfg.LintCommands = det.LintCommands

            return nil
        },
    }
}
```

### Step 3: Permission Configuration

**Purpose:** Select permission preset and customize allow/deny lists.

The wizard presents three presets and an option to customize:

| Preset | Description | Allow patterns |
|--------|-------------|---------------|
| `minimal` | Read-only + explicit build commands | `Bash(go build *)`, `Bash(go test *)`, `Read(*)` |
| `standard` | Build, test, lint, git, common tools | Above + `Bash(git *)`, `Bash(make *)`, `Bash(npm run *)`, `Edit(*)`, `Write(*)` |
| `permissive` | Most tools allowed, dangerous ops denied | Above + `Bash(curl *)`, `Bash(docker *)`, deny `Bash(rm -rf /)` etc. |
| `custom` | User builds their own list | Opens multi-select of known patterns |

```go
func permissionsStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-permissions",
        Description: "Configure Claude Code permissions",
        Run: func(ctx context.Context) error {
            cfg := &addon.Config

            var preset PermissionPreset

            form := huh.NewForm(
                huh.NewGroup(
                    huh.NewSelect[PermissionPreset]().
                        Title("Permission level").
                        Description("Controls what tools Claude can use without asking").
                        Options(
                            huh.NewOption("Minimal — read + explicit build commands only", PermissionPresetMinimal),
                            huh.NewOption("Standard — build, test, lint, git, file editing", PermissionPresetStandard),
                            huh.NewOption("Permissive — most tools, dangerous ops blocked", PermissionPresetPermissive),
                            huh.NewOption("Custom — build your own list", PermissionPresetCustom),
                        ).
                        Value(&preset),
                ),
                // Custom preset: show pattern builder
                huh.NewGroup(
                    huh.NewMultiSelect[string]().
                        Title("Allowed tool patterns").
                        Description("Select which operations Claude can perform without confirmation").
                        OptionsFunc(func() []huh.Option[string] {
                            return buildPatternOptions(cfg)
                        }, cfg).
                        Value(&cfg.ExtraAllowPatterns),
                ).WithHideFunc(func() bool { return preset != PermissionPresetCustom }),
                // Sandbox configuration
                huh.NewGroup(
                    huh.NewConfirm().
                        Title("Enable filesystem sandbox?").
                        Description("Restricts Claude's filesystem and network access").
                        Value(&cfg.SandboxEnabled),
                ),
            )

            if err := form.Run(); err != nil {
                return err
            }

            cfg.PermissionPreset = preset
            return nil
        },
    }
}
```

### Step 4: Skills Selection

**Purpose:** Choose team skills to install from the embedded library or a remote source.

```go
func skillsStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-skills",
        Description: "Select Claude Code skills to install",
        Run: func(ctx context.Context) error {
            cfg := &addon.Config
            det := getDetectionResult(ctx)

            // Load available skills from embedded library + remote source
            available := loadAvailableSkills(cfg.SkillLibrarySource)

            // Pre-select skills that match detected project type
            recommended := recommendSkills(available, det)

            var selected []string

            form := huh.NewForm(huh.NewGroup(
                huh.NewMultiSelect[string]().
                    Title("Install team skills").
                    Description("Reusable workflows Claude can execute via /skill-name").
                    Options(buildSkillOptions(available, recommended)...).
                    Value(&selected),
            ))

            if err := form.Run(); err != nil {
                return err
            }

            cfg.EnabledSkills = selected
            return nil
        },
    }
}
```

### Step 5: Hooks Configuration

**Purpose:** Select event-triggered automation hooks.

```go
func hooksStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-hooks",
        Description: "Configure Claude Code hooks",
        Run: func(ctx context.Context) error {
            cfg := &addon.Config
            det := getDetectionResult(ctx)

            // Build hook options based on detected project type
            hookOptions := buildHookOptions(det)

            var selected []HookPreset

            form := huh.NewForm(huh.NewGroup(
                huh.NewMultiSelect[HookPreset]().
                    Title("Enable automation hooks").
                    Description("Hooks run automatically on Claude Code events").
                    Options(hookOptions...).
                    Value(&selected),
            ))

            if err := form.Run(); err != nil {
                return err
            }

            cfg.EnabledHooks = selected
            return nil
        },
    }
}
```

**Available hook presets by project type:**

| Hook | Event | Command | When offered |
|------|-------|---------|-------------|
| `auto-format` | `PostToolUse(Edit,Write)` | `gofmt -w $file` / `prettier --write $file` | Go / TS projects |
| `safety-block` | `PreToolUse(Bash)` | Block `rm -rf /`, `git push --force` etc. | Always |
| `pre-commit` | `PreToolUse(Bash(git commit*))` | Run linter/test suite | If lint/test commands detected |
| `audit-log` | `PostToolUse(*)` | Append to `.claude/audit.log` | Opt-in |

### Step 6: MCP Server Setup

**Purpose:** Configure Model Context Protocol server integrations.

```go
func mcpStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-mcp",
        Description: "Configure MCP server integrations",
        Run: func(ctx context.Context) error {
            cfg := &addon.Config

            // Offer known MCP servers
            knownServers := []MCPServerOption{
                {Name: "GitHub", Desc: "PR reviews, issue tracking", Pkg: "@anthropic-ai/mcp-github"},
                {Name: "Filesystem", Desc: "Extended filesystem operations", Pkg: "@anthropic-ai/mcp-filesystem"},
                {Name: "PostgreSQL", Desc: "Database queries and schema inspection", Pkg: "@anthropic-ai/mcp-postgres"},
                {Name: "Slack", Desc: "Channel messaging and search", Pkg: "@anthropic-ai/mcp-slack"},
                {Name: "Linear", Desc: "Issue tracking integration", Pkg: "@anthropic-ai/mcp-linear"},
                {Name: "Sentry", Desc: "Error tracking and monitoring", Pkg: "@anthropic-ai/mcp-sentry"},
            }

            var selected []string

            form := huh.NewForm(huh.NewGroup(
                huh.NewMultiSelect[string]().
                    Title("MCP server integrations").
                    Description("External tools Claude can access via MCP protocol").
                    Options(buildMCPOptions(knownServers)...).
                    Value(&selected),
            ))

            if err := form.Run(); err != nil {
                return err
            }

            cfg.MCPServers = buildMCPConfigs(selected, knownServers)
            return nil
        },
    }
}
```

### Step 7: File Generation

**Purpose:** Preview what will be generated, confirm, then write all files.

```go
func generateStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "claude-code-generate",
        Description: "Generate Claude Code configuration files",
        Run: func(ctx context.Context) error {
            cfg := &addon.Config

            // Build the generation plan
            plan := buildGenerationPlan(cfg)

            // Show plan preview
            var proceed bool
            form := huh.NewForm(huh.NewGroup(
                huh.NewNote().
                    Title("Generation Plan").
                    Description(plan.Summary()),
                huh.NewConfirm().
                    Title("Generate files?").
                    Value(&proceed),
            ))

            if err := form.Run(); err != nil {
                return err
            }

            if !proceed {
                return bootstrap.ErrStepSkipped
            }

            // Execute generation
            for _, file := range plan.Files {
                if err := file.Generate(); err != nil {
                    return fmt.Errorf("generating %s: %w", file.Path, err)
                }
            }

            // Save config for future updates
            config.Set(configKey, *cfg)
            config.SetDirty()

            return nil
        },
    }
}
```

---

## 5. Template Strategy

### CLAUDE.md Generation

Uses Go `text/template` with the project-specific config struct as data. The template is structured into sections that are conditionally included based on what the wizard detected/collected.

```go
//go:embed templates/claude-md.tmpl
var claudeMDTemplate string

func generateClaudeMD(cfg *Config) (string, error) {
    tmpl, err := template.New("claude-md").Parse(claudeMDTemplate)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, cfg); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

**Template structure** (`templates/claude-md.tmpl`):

```
# CLAUDE.md

{{ .ProjectDescription }}

## Build Commands

{{ range .BuildCommands -}}
- `{{ . }}`
{{ end }}

## Test Commands

{{ range .TestCommands -}}
- `{{ . }}`
{{ end }}
{{ if .LintCommands }}
## Lint Commands

{{ range .LintCommands -}}
- `{{ . }}`
{{ end }}
{{ end }}
{{ if .ArchitectureNotes }}
## Architecture

{{ .ArchitectureNotes }}
{{ end }}
{{ range .Languages }}
## {{ .Name }} Conventions

{{ call $.ConventionsFor .Name }}
{{ end }}
{{ if .CustomInstructions }}
## Additional Instructions

{{ range .CustomInstructions -}}
- {{ . }}
{{ end }}
{{ end }}
```

### settings.json Generation

Uses Go structs marshaled to JSON. No templates -- the structure is fully typed.

```go
func generateSettingsJSON(cfg *Config) ([]byte, error) {
    settings := SettingsJSON{
        Permissions: buildPermissions(cfg),
    }

    if cfg.SandboxEnabled {
        settings.Sandbox = buildSandboxConfig(cfg)
    }

    if len(cfg.EnabledHooks) > 0 {
        settings.Hooks = buildHooks(cfg)
    }

    return json.MarshalIndent(settings, "", "  ")
}

func buildPermissions(cfg *Config) Permissions {
    var allow, deny []string

    switch cfg.PermissionPreset {
    case PermissionPresetMinimal:
        allow = []string{"Read(*)"}
        for _, cmd := range cfg.BuildCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }
        for _, cmd := range cfg.TestCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }

    case PermissionPresetStandard:
        allow = []string{
            "Read(*)", "Edit(*)", "Write(*)",
            "Bash(git *)",
        }
        for _, cmd := range cfg.BuildCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }
        for _, cmd := range cfg.TestCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }
        for _, cmd := range cfg.LintCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }

    case PermissionPresetPermissive:
        allow = []string{
            "Read(*)", "Edit(*)", "Write(*)",
            "Bash(git *)", "Bash(make *)", "Bash(docker *)",
            "Bash(curl *)", "Bash(wget *)",
        }
        for _, cmd := range cfg.BuildCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }
        for _, cmd := range cfg.TestCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }
        for _, cmd := range cfg.LintCommands {
            allow = append(allow, fmt.Sprintf("Bash(%s)", cmd))
        }
        deny = []string{
            "Bash(rm -rf /)",
            "Bash(git push --force *)",
            "Bash(chmod -R 777 *)",
        }

    case PermissionPresetCustom:
        allow = cfg.ExtraAllowPatterns
    }

    // Merge any extra patterns from custom additions
    if cfg.PermissionPreset != PermissionPresetCustom {
        allow = append(allow, cfg.ExtraAllowPatterns...)
    }
    deny = append(deny, cfg.ExtraDenyPatterns...)

    return Permissions{Allow: allow, Deny: deny}
}
```

### .claude/skills/*.md Generation

Skills are embedded markdown files copied verbatim or with minimal variable substitution. Each skill file has YAML frontmatter.

```go
//go:embed templates/skills/*
var skillTemplates embed.FS

func generateSkills(cfg *Config, destDir string) error {
    for _, skillName := range cfg.EnabledSkills {
        content, err := skillTemplates.ReadFile(
            filepath.Join("templates/skills", skillName+".md"),
        )
        if err != nil {
            return fmt.Errorf("skill %s not found in embedded library: %w", skillName, err)
        }

        destPath := filepath.Join(destDir, ".claude", "skills", skillName+".md")
        if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
            return err
        }
        if err := os.WriteFile(destPath, content, 0o644); err != nil {
            return err
        }
    }
    return nil
}
```

### .claude/rules/*.md Generation

Rules are conditionally included based on detected languages. They are standalone instruction files scoped to file paths.

```go
//go:embed templates/rules/*
var ruleTemplates embed.FS

func generateRules(cfg *Config, destDir string) error {
    // Map languages to their convention rules
    ruleMapping := map[string][]string{
        "Go":         {"go-conventions.md"},
        "TypeScript": {"typescript-conventions.md"},
        "Python":     {"python-conventions.md"},
    }

    // Always include security rules
    rulesToInstall := []string{"security-rules.md"}

    for _, lang := range cfg.Languages {
        if rules, ok := ruleMapping[lang.Name]; ok {
            rulesToInstall = append(rulesToInstall, rules...)
        }
    }

    for _, ruleName := range rulesToInstall {
        content, err := ruleTemplates.ReadFile(
            filepath.Join("templates/rules", ruleName),
        )
        if err != nil {
            continue // Skip if rule template doesn't exist
        }

        destPath := filepath.Join(destDir, ".claude", "rules", ruleName)
        if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
            return err
        }
        if err := os.WriteFile(destPath, content, 0o644); err != nil {
            return err
        }
    }
    return nil
}
```

### .mcp.json Generation

Structured JSON from typed structs. Each MCP server is an entry in a `mcpServers` map.

```go
type MCPFile struct {
    MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

type MCPServerEntry struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env,omitempty"`
}

func generateMCPJSON(cfg *Config) ([]byte, error) {
    mcpFile := MCPFile{
        MCPServers: make(map[string]MCPServerEntry),
    }

    for _, server := range cfg.MCPServers {
        mcpFile.MCPServers[server.Name] = MCPServerEntry{
            Command: server.Command,
            Args:    server.Args,
            Env:     server.Env,
        }
    }

    return json.MarshalIndent(mcpFile, "", "  ")
}
```

### .gitignore Additions

Appends patterns to existing `.gitignore` (or creates one). Checks for existing patterns before appending to avoid duplicates.

```go
var gitignorePatterns = []string{
    "",
    "# Claude Code local config (not committed)",
    ".claude/settings.local.json",
    "CLAUDE.local.md",
    ".claude/audit.log",
}

func appendGitignore(projectRoot string) error {
    path := filepath.Join(projectRoot, ".gitignore")
    existing, _ := os.ReadFile(path) // ok if doesn't exist

    var toAdd []string
    for _, pattern := range gitignorePatterns {
        if !strings.Contains(string(existing), pattern) {
            toAdd = append(toAdd, pattern)
        }
    }

    if len(toAdd) == 0 {
        return nil
    }

    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = f.WriteString(strings.Join(toAdd, "\n") + "\n")
    return err
}
```

---

## 6. Config Persistence

All wizard answers persist to gdev's YAML config under the `claudecode` key. This enables:

- `qsdev claude update` -- regenerate all files from saved config
- `qsdev claude init` -- re-run wizard with previous answers as defaults
- CI/headless mode -- read config from checked-in qsdev config

**Example saved config (`~/.config/gdev.yaml`):**

```yaml
claudecode:
  languages:
    - name: Go
      version: "1.22"
  frameworks:
    - gin
    - sqlc
  build_commands:
    - go build ./...
  test_commands:
    - go test ./...
  lint_commands:
    - golangci-lint run
  project_description: "REST API service for inventory management"
  permission_preset: standard
  sandbox_enabled: true
  allowed_domains:
    - api.github.com
    - pkg.go.dev
  enabled_skills:
    - review-pr
    - generate-tests
    - security-review
  enabled_hooks:
    - auto-format
    - safety-block
  mcp_servers:
    - name: github
      command: npx
      args: ["@anthropic-ai/mcp-github"]
  skill_library_source: "git@github.com:myorg/gdev-skills.git"
```

**Config key behavior:**
- Default values (empty struct) are omitted from the YAML file
- `config.Get[Config](configKey)` retrieves the current config at any time
- `config.Set(configKey, cfg)` + `config.SetDirty()` persists changes
- Config survives gdev upgrades -- it's user data, not addon data

---

## 7. CLI Commands

### `qsdev claude init`

Runs the full wizard outside of bootstrap. This is for projects that already have gdev set up but want to add Claude Code config.

```go
func initCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "init",
        Short: "Initialize Claude Code configuration",
        Long:  "Run the Claude Code setup wizard to generate project configuration.",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg := config.Get[Config](configKey)

            // Run detection
            det := detectProject(".")

            // Run wizard steps sequentially
            if err := runInstructionsWizard(&cfg, det); err != nil {
                return err
            }
            if err := runPermissionsWizard(&cfg, det); err != nil {
                return err
            }
            if err := runSkillsWizard(&cfg, det); err != nil {
                return err
            }
            if err := runHooksWizard(&cfg, det); err != nil {
                return err
            }
            if err := runMCPWizard(&cfg); err != nil {
                return err
            }

            // Generate files
            return generateAll(&cfg, ".")
        },
    }

    // Non-interactive flags (see section 8)
    addNonInteractiveFlags(cmd)

    return cmd
}
```

### `qsdev claude update`

Regenerates all files from saved config. No wizard. Useful after editing the gdev YAML config directly or after a template update.

```go
func updateCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "update",
        Short: "Regenerate Claude Code config from saved settings",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg := config.Get[Config](configKey)
            if cfg.IsEmpty() {
                return fmt.Errorf("no saved Claude Code config found; run 'qsdev claude init' first")
            }
            return generateAll(&cfg, ".")
        },
    }
}
```

### `qsdev claude add-skill <name>`

Adds a single skill from the library without re-running the full wizard.

```go
func addSkillCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "add-skill <name>",
        Short: "Add a skill from the team library",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            skillName := args[0]
            cfg := config.Get[Config](configKey)

            // Check if already installed
            for _, s := range cfg.EnabledSkills {
                if s == skillName {
                    return fmt.Errorf("skill %q already installed", skillName)
                }
            }

            // Verify skill exists in library
            available := loadAvailableSkills(cfg.SkillLibrarySource)
            if !available.Has(skillName) {
                return fmt.Errorf("skill %q not found; run 'qsdev claude list-skills' to see available", skillName)
            }

            // Install it
            if err := installSkill(skillName, available, "."); err != nil {
                return err
            }

            // Update config
            cfg.EnabledSkills = append(cfg.EnabledSkills, skillName)
            config.Set(configKey, cfg)
            config.SetDirty()
            return config.Save()
        },
    }
}
```

### `qsdev claude add-hook <type>`

Adds a hook preset to the settings.json.

```go
func addHookCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "add-hook <type>",
        Short: "Add a hook preset (auto-format, safety-block, pre-commit, audit-log)",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            hookName := HookPreset(args[0])

            // Validate hook name
            if !isValidHookPreset(hookName) {
                return fmt.Errorf("unknown hook %q; valid: auto-format, safety-block, pre-commit, audit-log", args[0])
            }

            cfg := config.Get[Config](configKey)

            // Check for duplicates
            for _, h := range cfg.EnabledHooks {
                if h == hookName {
                    return fmt.Errorf("hook %q already enabled", args[0])
                }
            }

            cfg.EnabledHooks = append(cfg.EnabledHooks, hookName)

            // Regenerate settings.json with the new hook
            if err := regenerateSettingsJSON(&cfg, "."); err != nil {
                return err
            }

            config.Set(configKey, cfg)
            config.SetDirty()
            return config.Save()
        },
    }
}
```

### `qsdev claude list-skills`

Lists available skills from the embedded library and any configured remote source.

```go
func listSkillsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list-skills",
        Short: "List available skills from the team library",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg := config.Get[Config](configKey)
            available := loadAvailableSkills(cfg.SkillLibrarySource)

            fmt.Println("Available skills:")
            for _, skill := range available.All() {
                installed := ""
                if slices.Contains(cfg.EnabledSkills, skill.Name) {
                    installed = " (installed)"
                }
                fmt.Printf("  %-20s %s%s\n", skill.Name, skill.Description, installed)
            }
            return nil
        },
    }
}
```

---

## 8. Non-Interactive Mode

Every wizard question has a CLI flag equivalent. When any flag is provided, that question is skipped in the wizard. When `--yes` is passed, all questions use defaults/flags and no interactive prompts appear.

```go
func addNonInteractiveFlags(cmd *cobra.Command) {
    // Global
    cmd.Flags().Bool("yes", false, "Accept all defaults, no interactive prompts")

    // Step 2: Instructions
    cmd.Flags().String("description", "", "Project description for CLAUDE.md")
    cmd.Flags().String("arch-notes", "", "Architecture notes for CLAUDE.md")
    cmd.Flags().StringSlice("languages", nil, "Languages (e.g., Go,TypeScript)")
    cmd.Flags().StringSlice("build-cmd", nil, "Build commands (e.g., 'go build ./...')")
    cmd.Flags().StringSlice("test-cmd", nil, "Test commands (e.g., 'go test ./...')")
    cmd.Flags().StringSlice("lint-cmd", nil, "Lint commands")

    // Step 3: Permissions
    cmd.Flags().String("permission-preset", "standard", "Permission preset: minimal|standard|permissive|custom")
    cmd.Flags().StringSlice("allow", nil, "Extra permission allow patterns")
    cmd.Flags().StringSlice("deny", nil, "Extra permission deny patterns")
    cmd.Flags().Bool("sandbox", false, "Enable filesystem sandbox")
    cmd.Flags().StringSlice("allowed-domains", nil, "Allowed network domains for sandbox")

    // Step 4: Skills
    cmd.Flags().StringSlice("skills", nil, "Skills to install (comma-separated names)")
    cmd.Flags().Bool("no-skills", false, "Skip skills installation")

    // Step 5: Hooks
    cmd.Flags().StringSlice("hooks", nil, "Hook presets to enable (auto-format,safety-block,...)")
    cmd.Flags().Bool("no-hooks", false, "Skip hooks configuration")

    // Step 6: MCP
    cmd.Flags().StringSlice("mcp", nil, "MCP servers to configure (github,postgres,...)")
    cmd.Flags().Bool("no-mcp", false, "Skip MCP configuration")

    // Team library
    cmd.Flags().String("skill-source", "", "Git URL for team skill library")
}
```

**CI/headless usage:**

```bash
# Fully non-interactive with explicit choices
qsdev claude init --yes \
    --languages=Go \
    --build-cmd="go build ./..." \
    --test-cmd="go test ./..." \
    --permission-preset=standard \
    --skills=review-pr,generate-tests \
    --hooks=auto-format,safety-block \
    --no-mcp

# Accept all detected defaults
qsdev claude init --yes

# Override just one thing, defaults for the rest
qsdev claude init --yes --permission-preset=permissive
```

---

## 9. Team Skill Library

### Storage and Versioning

The team skill library supports two sources, checked in priority order:

1. **Embedded skills** -- Compiled into the gdev binary via `//go:embed`. These are the default library and ship with every addon release. Updated when the addon binary is rebuilt.

2. **Remote git repository** -- Configured via `skill_library_source` in qsdev config or `--skill-source` flag. The addon clones/pulls the repo to a local cache (`~/.cache/gdev/skill-library/`) and reads skill files from it. This is how teams maintain their own skills.

### Remote Library Structure

```
skills-repo/
  skills/
    deploy.md
    review-pr.md
    my-team-workflow.md
  manifest.yaml          # Metadata: name, description, tags, min-version
```

**manifest.yaml:**

```yaml
skills:
  - name: deploy
    description: "Deploy to staging/production via CI pipeline"
    tags: [ops, ci]
    min_gdev_version: "0.5.0"
  - name: review-pr
    description: "Structured PR review with checklist"
    tags: [review, quality]
  - name: my-team-workflow
    description: "Team-specific sprint review workflow"
    tags: [team, process]
```

### Skill Resolution

```go
type SkillLibrary struct {
    embedded embed.FS
    remote   *RemoteLibrary // nil if no remote configured
}

func (lib *SkillLibrary) Resolve(name string) ([]byte, error) {
    // Remote takes priority -- team overrides trump embedded defaults
    if lib.remote != nil {
        if content, err := lib.remote.Read(name); err == nil {
            return content, nil
        }
    }

    // Fall back to embedded
    return lib.embedded.ReadFile(filepath.Join("templates/skills", name+".md"))
}

func (lib *SkillLibrary) All() []SkillMeta {
    // Merge embedded + remote, remote wins on name collision
    result := make(map[string]SkillMeta)

    // Embedded first
    entries, _ := lib.embedded.ReadDir("templates/skills")
    for _, e := range entries {
        name := strings.TrimSuffix(e.Name(), ".md")
        result[name] = SkillMeta{Name: name, Source: "embedded"}
    }

    // Remote overlays
    if lib.remote != nil {
        for _, skill := range lib.remote.Manifest().Skills {
            result[skill.Name] = SkillMeta{
                Name:        skill.Name,
                Description: skill.Description,
                Source:      "remote",
                Tags:        skill.Tags,
            }
        }
    }

    return maps.Values(result)
}
```

### Cache Management

```go
type RemoteLibrary struct {
    repoURL  string
    cacheDir string // ~/.cache/gdev/skill-library/<hash>
}

func (r *RemoteLibrary) Sync() error {
    if _, err := os.Stat(r.cacheDir); os.IsNotExist(err) {
        // Clone
        return exec.Command("git", "clone", "--depth=1", r.repoURL, r.cacheDir).Run()
    }
    // Pull
    cmd := exec.Command("git", "-C", r.cacheDir, "pull", "--ff-only")
    return cmd.Run()
}
```

The cache is refreshed on `qsdev claude init`, `qsdev claude add-skill`, and `qsdev claude list-skills`. A `--offline` flag skips the sync for air-gapped environments.

---

## 10. Template Examples

### Generated CLAUDE.md for a Go Web Service

This is what `qsdev claude init` produces for a Go project using Gin, sqlc, and PostgreSQL with the `standard` permission preset:

```markdown
# CLAUDE.md

REST API service for inventory management built with Go, Gin, and sqlc.

## Build Commands

- `go build ./...`
- `go build -o bin/server ./cmd/server`

## Test Commands

- `go test ./...`
- `go test -race -count=1 ./...`

## Lint Commands

- `golangci-lint run`

## Architecture

- HTTP layer: Gin router in `cmd/server/`, handlers in `internal/api/`
- Database: PostgreSQL with sqlc-generated queries in `internal/db/`
- Config: Environment variables loaded via envconfig in `internal/config/`
- Migrations: goose in `migrations/`

## Go Conventions

- Use `internal/` for non-exported packages
- Error handling: return errors, don't panic. Wrap with `fmt.Errorf("context: %w", err)`
- Tests: table-driven tests in `_test.go` files alongside source
- Naming: follow Go conventions (MixedCaps, not snake_case)
- Dependencies: use standard library where possible before reaching for third-party

## Database Conventions

- All schema changes go through goose migrations in `migrations/`
- Queries are defined in `internal/db/queries/` as sqlc SQL files
- Run `sqlc generate` after modifying queries
- Never modify generated files in `internal/db/sqlc/`
```

### Generated settings.json (Standard Preset, Go Project)

```json
{
  "permissions": {
    "allow": [
      "Read(*)",
      "Edit(*)",
      "Write(*)",
      "Bash(git *)",
      "Bash(go build *)",
      "Bash(go build -o *)",
      "Bash(go test *)",
      "Bash(golangci-lint run)"
    ],
    "deny": []
  },
  "sandbox": {
    "writeDeny": [
      "/etc",
      "/usr"
    ],
    "netAllow": [
      "api.github.com",
      "pkg.go.dev",
      "proxy.golang.org"
    ]
  },
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "command": "gofmt -w $CLAUDE_FILE_PATH"
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "command": "echo \"$CLAUDE_TOOL_INPUT\" | grep -qE '(rm -rf /|git push --force|chmod -R 777)' && exit 1 || exit 0"
      }
    ]
  }
}
```

### Generated .mcp.json (GitHub Integration)

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": [
        "@anthropic-ai/mcp-github"
      ],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

### Generated .claude/skills/review-pr.md

```markdown
---
name: review-pr
description: Structured pull request review with checklist
---

Review the current pull request thoroughly:

1. Read the PR diff using `git diff main...HEAD`
2. Check each changed file for:
   - Correctness: Does the logic do what's intended?
   - Error handling: Are errors properly handled and propagated?
   - Tests: Are new code paths covered by tests?
   - Performance: Any obvious performance concerns?
   - Security: Input validation, injection risks, secret exposure?
3. Run the test suite: `go test ./...`
4. Run the linter: `golangci-lint run`
5. Summarize findings with specific file/line references
6. Rate: approve, request-changes, or comment
```

---

## 11. File Generation Plan Preview

Before writing any files, the generate step shows a summary:

```
  Will generate:

    CREATE  CLAUDE.md                        Project instructions (Go, Gin, sqlc)
    CREATE  .claude/settings.json            Permissions (standard) + hooks (2)
    CREATE  .claude/skills/review-pr.md      PR review skill
    CREATE  .claude/skills/generate-tests.md Test generation skill
    CREATE  .claude/rules/go-conventions.md  Go coding conventions
    CREATE  .claude/rules/security-rules.md  Security rules
    CREATE  .mcp.json                        MCP servers (github)
    APPEND  .gitignore                       Add Claude Code local patterns

  Existing files will NOT be overwritten without confirmation.
```

For projects with existing configuration, the plan shows merge operations:

```
    MERGE   .claude/settings.json            Add 3 new allow patterns
    SKIP    CLAUDE.md                        Already exists (use --force to overwrite)
    CREATE  .claude/skills/deploy.md         Deploy skill
```

---

## 12. Merge and Update Strategy

The `qsdev claude update` command regenerates config from saved state. For files that may have been hand-edited, the addon uses different strategies per file type:

| File | Update strategy | Rationale |
|------|----------------|-----------|
| `CLAUDE.md` | Regenerate only if `--force` flag is passed; otherwise warn and skip | Free-form text; users customize heavily after generation |
| `.claude/settings.json` | Three-way merge: load existing, apply saved config changes, preserve unknown keys | Structured JSON; users may add patterns manually |
| `.claude/skills/*.md` | Overwrite matching skill files; leave unrecognized skills untouched | Skills are discrete units; team library is the source of truth |
| `.claude/rules/*.md` | Same as skills: overwrite known, leave unknown | Same logic |
| `.mcp.json` | Merge: add new servers, update existing, leave manual additions | Structured JSON; manual entries should survive |
| `.gitignore` | Append-only: add missing patterns, never remove | Safe -- only adds lines |

The three-way merge for `settings.json` works by comparing the "last generated" state (stored as a hash in qsdev config) against the current file to detect manual edits, then applying the new generated state on top.

```go
type GeneratedFileState struct {
    Hash string `yaml:"hash"` // SHA256 of last generated content
}

func mergeSettingsJSON(existing, generated []byte, lastHash string) ([]byte, error) {
    currentHash := sha256Hex(existing)

    if currentHash == lastHash {
        // No manual edits since last generation -- safe to overwrite
        return generated, nil
    }

    // Manual edits detected -- merge
    var existingSettings, generatedSettings map[string]any
    json.Unmarshal(existing, &existingSettings)
    json.Unmarshal(generated, &generatedSettings)

    merged := deepMerge(existingSettings, generatedSettings)
    return json.MarshalIndent(merged, "", "  ")
}
```

---

## 13. Project Detection Details

The `detect.go` module inspects the project root to populate default values. This runs before any wizard questions, enabling the wizard to pre-fill answers and offer "use detected defaults?" as the first question.

```go
type DetectionResult struct {
    ClaudeVersion  string
    ExistingConfig ExistingConfigState
    Languages      []Language
    Frameworks     []string
    BuildCommands  []string
    TestCommands   []string
    LintCommands   []string
}

type ExistingConfigState struct {
    HasClaudeDir     bool
    HasClaudeMD      bool
    HasSettingsJSON  bool
    HasMCPJSON       bool
    HasGitignore     bool
}

func detectProject(root string) *DetectionResult {
    det := &DetectionResult{}

    // Language detection
    if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
        goVersion := parseGoModVersion(root)
        det.Languages = append(det.Languages, Language{Name: "Go", Version: goVersion})
        det.BuildCommands = append(det.BuildCommands, "go build ./...")
        det.TestCommands = append(det.TestCommands, "go test ./...")
        if hasFile(root, ".golangci.yml") || hasFile(root, ".golangci.yaml") {
            det.LintCommands = append(det.LintCommands, "golangci-lint run")
        }
    }

    if hasFile(root, "package.json") {
        det.Languages = append(det.Languages, Language{Name: "Node.js"})
        if hasFile(root, "tsconfig.json") {
            det.Languages = append(det.Languages, Language{Name: "TypeScript"})
        }
        pkg := parsePackageJSON(root)
        for _, script := range []string{"build", "test", "lint"} {
            if cmd, ok := pkg.Scripts[script]; ok {
                switch script {
                case "build":
                    det.BuildCommands = append(det.BuildCommands, "npm run build")
                    _ = cmd // the actual command is opaque via npm run
                case "test":
                    det.TestCommands = append(det.TestCommands, "npm test")
                case "lint":
                    det.LintCommands = append(det.LintCommands, "npm run lint")
                }
            }
        }
    }

    if hasFile(root, "pyproject.toml") || hasFile(root, "requirements.txt") {
        det.Languages = append(det.Languages, Language{Name: "Python"})
    }

    if hasFile(root, "Cargo.toml") {
        det.Languages = append(det.Languages, Language{Name: "Rust"})
        det.BuildCommands = append(det.BuildCommands, "cargo build")
        det.TestCommands = append(det.TestCommands, "cargo test")
    }

    // Framework detection (Go-specific)
    if hasImport(root, "github.com/gin-gonic/gin") {
        det.Frameworks = append(det.Frameworks, "gin")
    }
    if hasImport(root, "github.com/sqlc-dev/sqlc") || hasDir(root, "sqlc") {
        det.Frameworks = append(det.Frameworks, "sqlc")
    }

    // Makefile / Taskfile detection
    if hasFile(root, "Makefile") {
        det.BuildCommands = append(det.BuildCommands, "make build")
        det.TestCommands = append(det.TestCommands, "make test")
    }
    if hasFile(root, "Taskfile.yml") {
        det.BuildCommands = append(det.BuildCommands, "task build")
        det.TestCommands = append(det.TestCommands, "task test")
    }

    // Existing config detection
    det.ExistingConfig = ExistingConfigState{
        HasClaudeDir:    hasDir(root, ".claude"),
        HasClaudeMD:     hasFile(root, "CLAUDE.md"),
        HasSettingsJSON: hasFile(root, ".claude/settings.json"),
        HasMCPJSON:      hasFile(root, ".mcp.json"),
        HasGitignore:    hasFile(root, ".gitignore"),
    }

    return det
}
```

---

## 14. Error Handling

All wizard steps return errors that the bootstrap framework handles. Specific error types:

| Error | Meaning | Bootstrap behavior |
|-------|---------|-------------------|
| `bootstrap.ErrStepSkipped` | User chose to skip this step | Continue to next step |
| `huh.ErrUserAborted` | User pressed Ctrl+C / Escape | Abort entire wizard, no files generated |
| `os.ErrPermission` | Can't write to target directory | Report error, suggest fix |
| Generation errors | Template execution failure | Report which file failed, partial state is safe (atomic writes) |

File writes use atomic write-and-rename to prevent partial output:

```go
func atomicWrite(path string, content []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".gdev-claude-*")
    if err != nil {
        return err
    }
    tmpPath := tmp.Name()
    defer os.Remove(tmpPath) // cleanup on failure

    if _, err := tmp.Write(content); err != nil {
        tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    if err := os.Chmod(tmpPath, perm); err != nil {
        return err
    }
    return os.Rename(tmpPath, path)
}
```

---

## 15. Integration with Other gdev Addons

The `claudecode` addon integrates with other gdev addons during the customization phase:

### With `bootstrap`

The primary integration. All wizard steps register as bootstrap steps, so `qsdev bootstrap` includes Claude Code setup alongside other addon setup (Docker, K3s, database, etc.).

### With `golang` / `nodejs`

If the `golang` or `nodejs` addon is also configured, the `claudecode` addon can read their config to pre-populate language versions and build commands rather than re-detecting.

```go
func Configure(opts ...Option) {
    addon.CheckNotInitialized()
    for _, o := range opts {
        o(&addon.Config)
    }

    // Pull language info from sibling addons if available
    if golangAddon := addons.Get("golang"); golangAddon != nil {
        addon.Config.Languages = append(addon.Config.Languages,
            Language{Name: "Go", Version: golangAddon.Config.Version})
    }

    addon.RegisterIfNeeded()
}
```

### With `github`

If the `github` addon is configured (provides `gh` CLI bootstrap), the `claudecode` addon can offer the GitHub MCP server integration with pre-filled auth configuration.

---

## 16. Usage in a Custom gdev Binary

Teams use the addon by importing it in their custom `main.go`:

```go
package main

import (
    "myorg/gdev/addons/claudecode"
    "myorg/gdev/cmd"
)

func main() {
    // Configure Claude Code with team defaults
    claudecode.Configure(
        claudecode.WithPermissionPreset(claudecode.PermissionPresetStandard),
        claudecode.WithSkillLibrary("git@github.com:myorg/gdev-skills.git"),
        claudecode.WithDefaultSkills("review-pr", "generate-tests", "security-review"),
        claudecode.WithDefaultHooks(claudecode.HookAutoFormat, claudecode.HookSafetyBlock),
        claudecode.WithMCPServer(claudecode.MCPServerConfig{
            Name:    "github",
            Command: "npx",
            Args:    []string{"@anthropic-ai/mcp-github"},
        }),
    )

    cmd.Main()
}
```

### Option Functions

```go
type Option func(*Config)

func WithPermissionPreset(preset PermissionPreset) Option {
    return func(c *Config) { c.PermissionPreset = preset }
}

func WithSkillLibrary(url string) Option {
    return func(c *Config) { c.SkillLibrarySource = url }
}

func WithDefaultSkills(skills ...string) Option {
    return func(c *Config) { c.EnabledSkills = skills }
}

func WithDefaultHooks(hooks ...HookPreset) Option {
    return func(c *Config) { c.EnabledHooks = hooks }
}

func WithMCPServer(server MCPServerConfig) Option {
    return func(c *Config) { c.MCPServers = append(c.MCPServers, server) }
}

func WithProjectDescription(desc string) Option {
    return func(c *Config) { c.ProjectDescription = desc }
}

func WithExtraAllowPatterns(patterns ...string) Option {
    return func(c *Config) { c.ExtraAllowPatterns = append(c.ExtraAllowPatterns, patterns...) }
}

func WithSandbox(enabled bool, domains ...string) Option {
    return func(c *Config) {
        c.SandboxEnabled = enabled
        c.AllowedDomains = domains
    }
}
```

This lets teams bake their organizational defaults into the gdev binary. The wizard still runs, but the defaults reflect team standards rather than generic ones. Developers override at wizard time or via `CLAUDE.local.md` and `settings.local.json` after generation.

---

## 17. Open Design Questions

These items need resolution during implementation:

1. **CLAUDE.md update strategy** -- The current design skips CLAUDE.md on update unless `--force` is passed. An alternative is to use marker comments (`<!-- gdev:start -->` / `<!-- gdev:end -->`) to delimit the generated sections, allowing updates to those sections while preserving user additions outside the markers. This adds complexity but solves the "template evolved but user customized" problem.

2. **Hook command portability** -- Hook commands like `gofmt -w $CLAUDE_FILE_PATH` assume the tool is on PATH. Should the addon verify tool availability during generation, or just document the requirement?

3. **Skill versioning** -- When a remote skill library updates a skill that's already installed in a project, should `qsdev claude update` overwrite the local copy? The current design says yes (team library is source of truth), but some teams may want to pin skill versions per project.

4. **Multi-project monorepo support** -- A monorepo with Go backend and TypeScript frontend may need different CLAUDE.md sections or path-scoped rules. The current design detects all languages at root level. Path-scoped rules partially address this, but a `--subdir` flag or monorepo-aware detection may be needed.

5. **Managed settings integration** -- The addon currently targets project-level config only. Should it also support generating managed settings (for IT/security teams) via a separate command like `qsdev claude managed-settings`?
