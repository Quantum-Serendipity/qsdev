# qsdev devenv Addon — Design Specification

## 1. Overview

This document specifies the design of a gdev addon that scaffolds and manages devenv.sh developer environments. The addon follows gdev's `_template` pattern: a package-level `Addon[T]` generic struct with `Configure()` options and an `Initialize()` callback. It generates `devenv.yaml`, `devenv.nix`, `.envrc`, and `.gitignore` entries from wizard answers, persists configuration for re-generation, and exposes CLI commands for post-init modifications.

The addon targets devenv.sh native mode (not flake mode) because native mode is simpler, more featureful, and the recommended path per devenv.sh documentation.

---

## 2. Addon Structure

### Package Layout

```
addons/devenv/
  addon.go          # Addon definition, Configure(), Initialize()
  config.go         # Config struct and config key registration
  bootstrap.go      # Bootstrap step definitions
  commands.go       # CLI command registration (init, update, add-service, add-language)
  wizard.go         # huh form construction and execution
  detect.go         # Project type auto-detection
  generate.go       # File generation orchestration
  templates.go      # Embedded Go templates for devenv.nix, devenv.yaml, .envrc
  templates/        # Template files (embedded via go:embed)
    devenv.nix.tmpl
    devenv.yaml.tmpl
    envrc.tmpl
  types.go          # Shared types (LanguageConfig, ServiceConfig, etc.)
```

### Addon Definition

```go
package devenv

import (
    "github.com/user/gdev/addons"
    "github.com/user/gdev/addons/bootstrap"
    "github.com/user/gdev/config"
    "github.com/user/gdev/instance"
)

var addon = addons.Addon[Config]{
    Definition: addons.Definition{
        Name:        "devenv",
        Description: func() string { return "devenv.sh environment setup and management" },
        Initialize:  initialize,
    },
    Config: Config{},
}

type option func(*Config)

func Configure(opts ...option) {
    addon.CheckNotInitialized()
    for _, o := range opts {
        o(&addon.Config)
    }
    addon.RegisterIfNeeded()
}

func initialize() error {
    instance.AddCommands(devenvCmd())
    bootstrap.WithSteps(
        detectProjectStep(),
        languageConfigStep(),
        serviceSelectionStep(),
        devToolingStep(),
        generateFilesStep(),
    )
    return nil
}
```

### Configuration Options (Customization Phase)

Consumers calling `Configure()` from their `main()` can pre-set defaults:

```go
// WithDefaultLanguage sets the default language selection in the wizard.
func WithDefaultLanguage(lang string) option {
    return func(c *Config) { c.DefaultLanguage = lang }
}

// WithExtraPackages adds Nix packages to every generated devenv.nix.
func WithExtraPackages(pkgs ...string) option {
    return func(c *Config) { c.ExtraPackages = append(c.ExtraPackages, pkgs...) }
}

// WithDirenv controls whether .envrc generation is enabled by default.
func WithDirenv(enabled bool) option {
    return func(c *Config) { c.DirenvEnabled = enabled }
}

// WithTemplateOverride replaces a built-in template with a custom one.
// name is one of: "devenv.nix", "devenv.yaml", ".envrc"
func WithTemplateOverride(name string, tmpl string) option {
    return func(c *Config) { c.TemplateOverrides[name] = tmpl }
}
```

---

## 3. Config Struct and Persistence

### Config Key Registration

The addon registers a single config key `devenv` that persists the full environment configuration. This enables `qsdev devenv update` to regenerate files without re-running the wizard.

```go
package devenv

import "github.com/user/gdev/config"

// DevenvConfigKey implements config.ConfigKey[DevenvPersistedConfig].
type DevenvConfigKey struct{}

func (k DevenvConfigKey) Name() string { return "devenv" }

func (k DevenvConfigKey) New() DevenvPersistedConfig {
    return DevenvPersistedConfig{}
}

func (k DevenvConfigKey) NewFrom(value any) (DevenvPersistedConfig, error) {
    // YAML deserialization handled by go-yaml via config package
    // ...
}

func (k DevenvConfigKey) IsDefault(value DevenvPersistedConfig) bool {
    return len(value.Languages) == 0 && len(value.Services) == 0
}

func init() {
    config.AddKey(DevenvConfigKey{})
}
```

### Persisted Config Structure

Everything the wizard collects is saved here. This is the source of truth for `qsdev devenv update`.

```go
// DevenvPersistedConfig is saved to ~/.config/<appname>.yaml under the "devenv" key.
type DevenvPersistedConfig struct {
    // Project metadata
    ProjectName string `yaml:"project_name,omitempty"`
    ProjectType string `yaml:"project_type,omitempty"` // "go", "node", "python", "rust", "multi"

    // Language configurations
    Languages []LanguageConfig `yaml:"languages,omitempty"`

    // Service configurations
    Services []ServiceConfig `yaml:"services,omitempty"`

    // Dev tooling
    GitHooks    []string `yaml:"git_hooks,omitempty"`    // e.g., ["gofmt", "prettier"]
    DirenvEnabled bool   `yaml:"direnv_enabled,omitempty"`

    // Extra packages beyond what languages/services pull in
    ExtraPackages []string `yaml:"extra_packages,omitempty"`

    // Environment variables
    EnvVars map[string]string `yaml:"env_vars,omitempty"`

    // Custom scripts
    Scripts []ScriptConfig `yaml:"scripts,omitempty"`

    // Shell initialization code
    EnterShell string `yaml:"enter_shell,omitempty"`

    // Generation metadata
    LastGenerated string `yaml:"last_generated,omitempty"` // ISO 8601 timestamp
}

type LanguageConfig struct {
    Name           string            `yaml:"name"`              // "go", "javascript", "python", "rust", "typescript"
    Version        string            `yaml:"version,omitempty"` // e.g., "1.22", "3.11"
    PackageManager string            `yaml:"package_manager,omitempty"` // "npm", "pnpm", "yarn", "bun", "poetry", "uv"
    Extras         map[string]string `yaml:"extras,omitempty"`  // language-specific options
}

type ServiceConfig struct {
    Name     string            `yaml:"name"`              // "postgres", "redis", "mysql", etc.
    Version  string            `yaml:"version,omitempty"` // package version suffix, e.g., "16" for postgresql_16
    Settings map[string]string `yaml:"settings,omitempty"` // service-specific key-value config
}

type ScriptConfig struct {
    Name        string `yaml:"name"`
    Exec        string `yaml:"exec"`
    Description string `yaml:"description,omitempty"`
}
```

### What Gets Saved vs What Does Not

**Saved to qsdev config** (survives across sessions):
- All wizard answers (languages, services, hooks, packages, env vars, scripts)
- Generation timestamp
- Project type detection result

**Not saved** (derived at generation time):
- The generated file contents themselves (regenerated from config)
- Nix input URLs (derived from language/service selections)
- Template text (embedded in the binary)

---

## 4. Bootstrap Steps

The addon registers five bootstrap steps that execute during `qsdev bootstrap`. Each step uses charmbracelet/huh forms for interactive input and supports headless mode via CLI flags.

### Step 1: Project Type Detection

Scans the working directory for project markers and either auto-selects or asks the user.

```go
func detectProjectStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "devenv-detect-project",
        Description: "Detect project type for devenv.sh setup",
        Run: func(ctx context.Context) error {
            detected := detectProject()
            if len(detected) == 1 && !isCustomize(ctx) {
                // Single language detected, auto-confirm
                state.ProjectType = detected[0]
                return nil
            }
            // Multiple or none detected — ask
            return runDetectionForm(ctx, detected)
        },
        Skip: func(ctx context.Context) bool {
            // Skip if --project-type flag was provided
            return flagProjectType != ""
        },
    }
}
```

**Detection logic** (`detect.go`):

```go
type DetectionResult struct {
    Language   string
    Confidence string // "high", "medium"
    Evidence   string // e.g., "found go.mod"
}

func detectProject() []DetectionResult {
    var results []DetectionResult

    // Go: go.mod or go.sum
    if fileExists("go.mod") {
        results = append(results, DetectionResult{"go", "high", "found go.mod"})
    }

    // Node/TypeScript: package.json
    if fileExists("package.json") {
        lang := "javascript"
        if fileExists("tsconfig.json") {
            lang = "typescript"
        }
        results = append(results, DetectionResult{lang, "high", "found package.json"})
    }

    // Python: pyproject.toml, setup.py, requirements.txt
    if fileExists("pyproject.toml") || fileExists("setup.py") || fileExists("requirements.txt") {
        results = append(results, DetectionResult{"python", "high", "found pyproject.toml"})
    }

    // Rust: Cargo.toml
    if fileExists("Cargo.toml") {
        results = append(results, DetectionResult{"rust", "high", "found Cargo.toml"})
    }

    return results
}
```

**huh form for detection**:

```go
func runDetectionForm(ctx context.Context, detected []DetectionResult) error {
    var projectTypes []string

    // Build options with detection hints
    options := []huh.Option[string]{
        huh.NewOption("Go", "go"),
        huh.NewOption("JavaScript / TypeScript", "javascript"),
        huh.NewOption("Python", "python"),
        huh.NewOption("Rust", "rust"),
    }
    // Mark detected languages
    for i, opt := range options {
        for _, d := range detected {
            if opt.Value == d.Language {
                options[i] = huh.NewOption(
                    fmt.Sprintf("%s (detected: %s)", opt.Key, d.Evidence), d.Language,
                )
            }
        }
    }

    form := huh.NewForm(
        huh.NewGroup(
            huh.NewMultiSelect[string]().
                Title("Select languages for this project").
                Description("Space to toggle, Enter to confirm").
                Options(options...).
                Value(&projectTypes),
        ),
    )

    if err := form.Run(); err != nil {
        return err
    }

    state.Languages = projectTypes
    return nil
}
```

### Step 2: Language Configuration

For each selected language, asks version and ecosystem-specific questions.

```go
func languageConfigStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "devenv-language-config",
        Description: "Configure language versions and package managers",
        Run: func(ctx context.Context) error {
            var groups []*huh.Group

            for _, lang := range state.Languages {
                groups = append(groups, languageGroup(lang))
            }

            if len(groups) == 0 {
                return nil
            }

            return huh.NewForm(groups...).Run()
        },
        Skip: func(ctx context.Context) bool {
            return len(state.Languages) == 0
        },
    }
}
```

**Language-specific groups**:

```go
func languageGroup(lang string) *huh.Group {
    switch lang {
    case "go":
        return goLanguageGroup()
    case "javascript", "typescript":
        return jsLanguageGroup()
    case "python":
        return pythonLanguageGroup()
    case "rust":
        return rustLanguageGroup()
    default:
        return genericLanguageGroup(lang)
    }
}

func goLanguageGroup() *huh.Group {
    cfg := &LanguageConfig{Name: "go"}

    return huh.NewGroup(
        huh.NewSelect[string]().
            Title("Go version").
            Options(
                huh.NewOption("1.24 (latest)", "1.24"),
                huh.NewOption("1.23", "1.23"),
                huh.NewOption("1.22", "1.22"),
                huh.NewOption("System default", ""),
            ).
            Value(&cfg.Version),
    ).WithHideFunc(func() bool { return !slices.Contains(state.Languages, "go") })
}

func jsLanguageGroup() *huh.Group {
    cfg := &LanguageConfig{Name: "javascript"}
    var pkgMgr string

    return huh.NewGroup(
        huh.NewSelect[string]().
            Title("Node.js package manager").
            Options(
                huh.NewOption("pnpm", "pnpm"),
                huh.NewOption("npm", "npm"),
                huh.NewOption("yarn", "yarn"),
                huh.NewOption("bun", "bun"),
            ).
            Value(&pkgMgr),
    ).WithHideFunc(func() bool {
        return !slices.Contains(state.Languages, "javascript") &&
            !slices.Contains(state.Languages, "typescript")
    })
}

func pythonLanguageGroup() *huh.Group {
    cfg := &LanguageConfig{Name: "python"}

    return huh.NewGroup(
        huh.NewSelect[string]().
            Title("Python version").
            Options(
                huh.NewOption("3.12 (latest)", "3.12"),
                huh.NewOption("3.11", "3.11"),
                huh.NewOption("3.10", "3.10"),
                huh.NewOption("System default", ""),
            ).
            Value(&cfg.Version),
        huh.NewSelect[string]().
            Title("Python package manager").
            Description("poetry and uv are mutually exclusive in devenv.sh").
            Options(
                huh.NewOption("uv (recommended)", "uv"),
                huh.NewOption("poetry", "poetry"),
                huh.NewOption("pip (no manager)", "pip"),
            ).
            Value(&cfg.PackageManager),
    ).WithHideFunc(func() bool { return !slices.Contains(state.Languages, "python") })
}

func rustLanguageGroup() *huh.Group {
    cfg := &LanguageConfig{Name: "rust"}

    return huh.NewGroup(
        huh.NewSelect[string]().
            Title("Rust channel").
            Options(
                huh.NewOption("stable", "stable"),
                huh.NewOption("beta", "beta"),
                huh.NewOption("nightly", "nightly"),
                huh.NewOption("nixpkgs (default)", "nixpkgs"),
            ).
            Value(&cfg.Extras["channel"]),
    ).WithHideFunc(func() bool { return !slices.Contains(state.Languages, "rust") })
}
```

### Step 3: Service Selection

Multi-select for databases, caches, and queues, with per-service configuration.

```go
func serviceSelectionStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "devenv-services",
        Description: "Select development services (databases, caches, queues)",
        Run: func(ctx context.Context) error {
            var serviceNames []string

            form := huh.NewForm(
                huh.NewGroup(
                    huh.NewMultiSelect[string]().
                        Title("Select services").
                        Description("These run via devenv up").
                        Options(
                            huh.NewOption("PostgreSQL", "postgres"),
                            huh.NewOption("MySQL / MariaDB", "mysql"),
                            huh.NewOption("MongoDB", "mongodb"),
                            huh.NewOption("Redis (Valkey)", "redis"),
                            huh.NewOption("Memcached", "memcached"),
                            huh.NewOption("Elasticsearch", "elasticsearch"),
                            huh.NewOption("RabbitMQ", "rabbitmq"),
                            huh.NewOption("NATS", "nats"),
                            huh.NewOption("Kafka", "kafka"),
                            huh.NewOption("None", "none"),
                        ).
                        Value(&serviceNames),
                ),
                // Conditional: PostgreSQL config
                postgresConfigGroup(&serviceNames),
                // Conditional: Redis config
                redisConfigGroup(&serviceNames),
            )

            return form.Run()
        },
    }
}

func postgresConfigGroup(selected *[]string) *huh.Group {
    var dbName string
    return huh.NewGroup(
        huh.NewInput().
            Title("PostgreSQL initial database name").
            Placeholder("myapp_dev").
            Value(&dbName),
    ).WithHideFunc(func() bool {
        return !slices.Contains(*selected, "postgres")
    })
}
```

### Step 4: Dev Tooling

Git hooks (auto-suggested based on languages), direnv toggle, extra packages.

```go
func devToolingStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "devenv-dev-tooling",
        Description: "Configure git hooks, direnv, and extra packages",
        Run: func(ctx context.Context) error {
            suggestedHooks := suggestHooks(state.Languages)
            var selectedHooks []string
            var enableDirenv bool = true
            var extraPkgs []string

            form := huh.NewForm(
                huh.NewGroup(
                    huh.NewMultiSelect[string]().
                        Title("Git hooks (pre-commit)").
                        Description("Auto-suggested based on your languages").
                        Options(hookOptions(suggestedHooks)...).
                        Value(&selectedHooks),
                    huh.NewConfirm().
                        Title("Enable direnv integration?").
                        Description("Creates .envrc for automatic shell activation").
                        Value(&enableDirenv),
                    huh.NewMultiSelect[string]().
                        Title("Extra CLI tools").
                        Options(
                            huh.NewOption("jq", "jq"),
                            huh.NewOption("ripgrep", "ripgrep"),
                            huh.NewOption("curl", "curl"),
                            huh.NewOption("httpie", "httpie"),
                            huh.NewOption("grpcurl", "grpcurl"),
                            huh.NewOption("just (task runner)", "just"),
                        ).
                        Value(&extraPkgs),
                ),
            )

            return form.Run()
        },
    }
}

// suggestHooks returns hooks appropriate for the selected languages,
// pre-checked in the multi-select.
func suggestHooks(languages []string) []HookSuggestion {
    var hooks []HookSuggestion

    for _, lang := range languages {
        switch lang {
        case "go":
            hooks = append(hooks,
                HookSuggestion{"gofmt", true},
                HookSuggestion{"govet", true},
            )
        case "javascript", "typescript":
            hooks = append(hooks,
                HookSuggestion{"prettier", true},
                HookSuggestion{"eslint", true},
            )
        case "python":
            hooks = append(hooks,
                HookSuggestion{"ruff", true},
                HookSuggestion{"mypy", false},
            )
        case "rust":
            hooks = append(hooks,
                HookSuggestion{"rustfmt", true},
                HookSuggestion{"clippy", true},
            )
        }
    }
    // Always available
    hooks = append(hooks,
        HookSuggestion{"shellcheck", false},
        HookSuggestion{"nixfmt", false},
    )
    return hooks
}
```

### Step 5: File Generation

Shows a plan preview, asks for confirmation, then generates all files.

```go
func generateFilesStep() *bootstrap.Step {
    return &bootstrap.Step{
        Name:        "devenv-generate-files",
        Description: "Generate devenv.nix, devenv.yaml, .envrc, and .gitignore",
        Run: func(ctx context.Context) error {
            plan := buildGenerationPlan(state)

            // Plan preview
            var confirm bool
            preview := formatPlanPreview(plan)

            form := huh.NewForm(
                huh.NewGroup(
                    huh.NewNote().
                        Title("Files to generate").
                        Description(preview),
                    huh.NewConfirm().
                        Title("Proceed?").
                        Value(&confirm),
                ),
            )

            if err := form.Run(); err != nil {
                return err
            }
            if !confirm {
                return fmt.Errorf("generation cancelled by user")
            }

            // Generate files
            if err := generateDevenvYaml(plan); err != nil {
                return fmt.Errorf("generating devenv.yaml: %w", err)
            }
            if err := generateDevenvNix(plan); err != nil {
                return fmt.Errorf("generating devenv.nix: %w", err)
            }
            if plan.DirenvEnabled {
                if err := generateEnvrc(); err != nil {
                    return fmt.Errorf("generating .envrc: %w", err)
                }
            }
            if err := updateGitignore(plan); err != nil {
                return fmt.Errorf("updating .gitignore: %w", err)
            }

            // Persist config
            persistedConfig := toPersistedConfig(state)
            persistedConfig.LastGenerated = time.Now().Format(time.RFC3339)
            config.Set(DevenvConfigKey{}, persistedConfig)
            config.SetDirty()
            config.Save()

            return nil
        },
    }
}
```

---

## 5. Template Strategy

### devenv.yaml: YAML Marshaling

`devenv.yaml` is simple structured data. Generate it with Go's `gopkg.in/yaml.v3` marshaler, not text templates. This avoids YAML formatting bugs and guarantees valid output.

```go
type DevenvYaml struct {
    Inputs      map[string]DevenvInput `yaml:"inputs"`
    Imports     []string               `yaml:"imports,omitempty"`
    Nixpkgs     *NixpkgsConfig         `yaml:"nixpkgs,omitempty"`
    Clean       *CleanConfig           `yaml:"clean,omitempty"`
}

type DevenvInput struct {
    URL     string `yaml:"url"`
    Follows string `yaml:"follows,omitempty"`
}

type NixpkgsConfig struct {
    AllowUnfree              bool     `yaml:"allow_unfree,omitempty"`
    PermittedUnfreePackages  []string `yaml:"permitted_unfree_packages,omitempty"`
}

func buildDevenvYaml(plan *GenerationPlan) DevenvYaml {
    yaml := DevenvYaml{
        Inputs: map[string]DevenvInput{
            "nixpkgs": {URL: "github:cachix/devenv-nixpkgs/rolling"},
        },
    }

    // Add git-hooks input when any hooks are enabled
    if len(plan.GitHooks) > 0 {
        yaml.Inputs["git-hooks"] = DevenvInput{
            URL: "github:cachix/git-hooks.nix",
        }
    }

    // Add language-specific overlay inputs
    for _, lang := range plan.Languages {
        if input, ok := languageInputs(lang); ok {
            yaml.Inputs[input.Name] = DevenvInput{URL: input.URL}
        }
    }

    return yaml
}

// languageInputs returns additional devenv.yaml inputs required
// when pinning a specific language version.
func languageInputs(lang LanguageConfig) (DevenvInput, bool) {
    switch {
    case lang.Name == "python" && lang.Version != "":
        // Python version pinning requires the nixpkgs-python input
        return DevenvInput{URL: "github:cachix/nixpkgs-python"}, true
    default:
        // Go, Rust, Node use overlays fetched via config.lib.getInput (automatic)
        return DevenvInput{}, false
    }
}
```

### devenv.nix: text/template

`devenv.nix` is Nix language code. It requires `text/template` with careful handling of Nix syntax (attribute sets, string interpolation escaping, conditional blocks).

**Template design principles:**
- Each configuration dimension (languages, services, packages, hooks, scripts, env vars) is an independent template block
- Blocks are only rendered when they have content (no empty `services = {};` stanzas)
- Nix string escaping: `''${var}` for shell variable interpolation inside Nix multiline strings
- Comments mark sections for human readability

**Key template helpers:**

```go
var templateFuncs = template.FuncMap{
    // nixList renders a Go string slice as a Nix list: [ pkgs.git pkgs.curl ]
    "nixList": func(items []string) string {
        if len(items) == 0 {
            return "[ ]"
        }
        parts := make([]string, len(items))
        for i, item := range items {
            parts[i] = "pkgs." + item
        }
        return "[ " + strings.Join(parts, " ") + " ]"
    },
    // nixBool renders a Go bool as a Nix bool
    "nixBool": func(b bool) string {
        if b {
            return "true"
        }
        return "false"
    },
    // nixString renders a Go string as a Nix string literal
    "nixString": func(s string) string {
        escaped := strings.ReplaceAll(s, `\`, `\\`)
        escaped = strings.ReplaceAll(escaped, `"`, `\"`)
        escaped = strings.ReplaceAll(escaped, "${", "\\${")
        return `"` + escaped + `"`
    },
    // indent adds n spaces of indentation to each line
    "indent": func(n int, s string) string {
        pad := strings.Repeat(" ", n)
        lines := strings.Split(s, "\n")
        for i, line := range lines {
            if line != "" {
                lines[i] = pad + line
            }
        }
        return strings.Join(lines, "\n")
    },
    // hasAny returns true if the slice is non-empty
    "hasAny": func(items any) bool {
        v := reflect.ValueOf(items)
        return v.Len() > 0
    },
}
```

### devenv.nix Template

```
{{/* templates/devenv.nix.tmpl */}}
{ pkgs, lib, config, ... }:

{
{{- /* ---- Packages ---- */}}
{{- if hasAny .Packages }}
  # Development tools
  packages = {{ nixList .Packages }};
{{- end }}

{{- /* ---- Environment Variables ---- */}}
{{- if hasAny .EnvVars }}

  # Environment variables
  {{- range $key, $val := .EnvVars }}
  env.{{ $key }} = {{ nixString $val }};
  {{- end }}
{{- end }}

{{- /* ---- Languages ---- */}}
{{- range .Languages }}

  # {{ .Name | title }}
  languages.{{ .NixName }}.enable = true;
  {{- if .Version }}
  languages.{{ .NixName }}.version = {{ nixString .Version }};
  {{- end }}
  {{- range .ExtraLines }}
  {{ . }}
  {{- end }}
{{- end }}

{{- /* ---- Services ---- */}}
{{- range .Services }}

  # {{ .Name | title }}
  services.{{ .NixName }} = {
    enable = true;
    {{- range .ConfigLines }}
    {{ . }}
    {{- end }}
  };
{{- end }}

{{- /* ---- Scripts ---- */}}
{{- if hasAny .Scripts }}

  # Scripts
  {{- range .Scripts }}
  scripts.{{ .Name }}.exec = ''
    {{ .Exec }}
  '';
  {{- if .Description }}
  scripts.{{ .Name }}.description = {{ nixString .Description }};
  {{- end }}
  {{- end }}
{{- end }}

{{- /* ---- Git Hooks ---- */}}
{{- if hasAny .GitHooks }}

  # Git hooks (pre-commit)
  {{- range .GitHooks }}
  git-hooks.hooks.{{ . }}.enable = true;
  {{- end }}
{{- end }}

{{- /* ---- Shell Init ---- */}}
{{- if .EnterShell }}

  enterShell = ''
    {{ .EnterShell | indent 4 }}
  '';
{{- end }}
}
```

### .envrc Template

The `.envrc` file is a trivial shell script. Hardcode it rather than template it:

```go
const envrcContent = `# Auto-generated by qsdev devenv addon
# Run 'direnv allow' to activate

eval "$(devenv direnvrc)"
use devenv
`
```

### .gitignore Entries

Append these entries if not already present (do not overwrite existing `.gitignore`):

```
# devenv.sh
.devenv
.pre-commit-config.yaml
.direnv
```

The addon reads the existing `.gitignore` (if any), checks for each entry, and appends only missing ones.

---

## 6. Template Data Assembly

The template receives a `TemplateData` struct assembled from wizard state. This struct mediates between the wizard's user-facing model and the template's Nix-facing model.

```go
type TemplateData struct {
    Packages   []string
    EnvVars    map[string]string
    Languages  []LanguageTemplateData
    Services   []ServiceTemplateData
    Scripts    []ScriptTemplateData
    GitHooks   []string
    EnterShell string
}

type LanguageTemplateData struct {
    Name       string   // Display name: "Go", "Python"
    NixName    string   // Nix attribute: "go", "python", "javascript"
    Version    string   // Optional version pin
    ExtraLines []string // Additional Nix lines for this language
}

type ServiceTemplateData struct {
    Name        string   // Display name
    NixName     string   // Nix attribute: "postgres", "redis", "mysql"
    ConfigLines []string // Nix attribute lines inside the service block
}

type ScriptTemplateData struct {
    Name        string
    Exec        string
    Description string
}
```

**Language-to-template mapping** (the core translation logic):

```go
func languageToTemplate(lang LanguageConfig) LanguageTemplateData {
    td := LanguageTemplateData{
        Name:    lang.Name,
        NixName: lang.Name,
        Version: lang.Version,
    }

    switch lang.Name {
    case "go":
        // Go needs GOPATH/bin on PATH via enterShell
        // delve is useful for debugging
        td.ExtraLines = []string{
            "languages.go.delve.enable = true;",
        }

    case "javascript":
        td.NixName = "javascript"
        switch lang.PackageManager {
        case "pnpm":
            td.ExtraLines = []string{
                "languages.javascript.pnpm.enable = true;",
                "languages.javascript.pnpm.install.enable = true;",
            }
        case "yarn":
            td.ExtraLines = []string{
                "languages.javascript.yarn.enable = true;",
                "languages.javascript.yarn.install.enable = true;",
            }
        case "bun":
            td.ExtraLines = []string{
                "languages.javascript.bun.enable = true;",
                "languages.javascript.bun.install.enable = true;",
            }
        default: // npm
            td.ExtraLines = []string{
                "languages.javascript.npm.install.enable = true;",
            }
        }

    case "typescript":
        // TypeScript is typically enabled alongside javascript
        td.NixName = "typescript"

    case "python":
        switch lang.PackageManager {
        case "uv":
            td.ExtraLines = []string{
                "languages.python.uv.enable = true;",
                "languages.python.uv.sync.enable = true;",
            }
        case "poetry":
            td.ExtraLines = []string{
                "languages.python.poetry.enable = true;",
                "languages.python.poetry.install.enable = true;",
            }
        }

    case "rust":
        if channel, ok := lang.Extras["channel"]; ok && channel != "nixpkgs" {
            td.ExtraLines = []string{
                fmt.Sprintf(`languages.rust.channel = "%s";`, channel),
            }
        }
    }

    return td
}
```

**Service-to-template mapping**:

```go
func serviceToTemplate(svc ServiceConfig) ServiceTemplateData {
    td := ServiceTemplateData{
        Name:    svc.Name,
        NixName: svc.Name,
    }

    switch svc.Name {
    case "postgres":
        if v, ok := svc.Settings["version"]; ok {
            td.ConfigLines = append(td.ConfigLines,
                fmt.Sprintf("package = pkgs.postgresql_%s;", v),
            )
        }
        if db, ok := svc.Settings["initial_db"]; ok {
            td.ConfigLines = append(td.ConfigLines,
                fmt.Sprintf(`initialDatabases = [{ name = "%s"; }];`, db),
            )
        }

    case "redis":
        td.NixName = "redis"
        // Redis/Valkey has minimal config — enable is usually sufficient

    case "mysql":
        td.NixName = "mysql"
        if db, ok := svc.Settings["initial_db"]; ok {
            td.ConfigLines = append(td.ConfigLines,
                fmt.Sprintf(`initialDatabases = [{ name = "%s"; }];`, db),
            )
        }

    case "mongodb":
        td.NixName = "mongodb"

    case "elasticsearch":
        td.NixName = "elasticsearch"

    case "rabbitmq":
        td.NixName = "rabbitmq"

    case "nats":
        td.NixName = "nats"
    }

    return td
}
```

---

## 7. Commands

The addon registers a `devenv` command group with four subcommands.

```go
func devenvCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "devenv",
        Short: "Manage devenv.sh environment configuration",
    }

    cmd.AddCommand(
        initCmd(),
        updateCmd(),
        addServiceCmd(),
        addLanguageCmd(),
    )

    return cmd
}
```

### `qsdev devenv init`

Runs the full wizard outside of bootstrap. Identical logic to the bootstrap steps, but invoked standalone.

```go
func initCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "init",
        Short: "Initialize devenv.sh environment (interactive wizard)",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Check for existing files
            if fileExists("devenv.nix") && !flagForce {
                var overwrite bool
                huh.NewConfirm().
                    Title("devenv.nix already exists. Overwrite?").
                    Value(&overwrite).Run()
                if !overwrite {
                    return fmt.Errorf("cancelled — use 'qsdev devenv update' to regenerate from saved config")
                }
            }

            // Run the same wizard steps as bootstrap
            if err := runDetectionForm(cmd.Context(), detectProject()); err != nil {
                return err
            }
            // ... remaining steps ...
            return nil
        },
    }

    // Non-interactive flags
    cmd.Flags().StringSliceVar(&flagLanguages, "lang", nil, "Languages to enable (go,javascript,python,rust,typescript)")
    cmd.Flags().StringVar(&flagProjectType, "project-type", "", "Project type override")
    cmd.Flags().StringSliceVar(&flagServices, "services", nil, "Services to enable (postgres,redis,mysql,...)")
    cmd.Flags().BoolVar(&flagDirenv, "direnv", true, "Generate .envrc for direnv")
    cmd.Flags().StringSliceVar(&flagHooks, "hooks", nil, "Git hooks to enable (gofmt,prettier,eslint,...)")
    cmd.Flags().BoolVar(&flagYes, "yes", false, "Accept all defaults (non-interactive)")
    cmd.Flags().BoolVar(&flagForce, "force", false, "Overwrite existing files without prompting")

    return cmd
}
```

### `qsdev devenv update`

Re-generates files from saved config without running the wizard. Useful after manually editing the qsdev config YAML, or when the addon's templates have been updated.

```go
func updateCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "update",
        Short: "Regenerate devenv files from saved configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            persisted := config.Get[DevenvPersistedConfig](DevenvConfigKey{})
            if persisted.IsDefault() {
                return fmt.Errorf("no saved devenv configuration — run 'qsdev devenv init' first")
            }

            plan := configToGenerationPlan(persisted)

            if err := generateDevenvYaml(plan); err != nil {
                return err
            }
            if err := generateDevenvNix(plan); err != nil {
                return err
            }
            if persisted.DirenvEnabled {
                if err := generateEnvrc(); err != nil {
                    return err
                }
            }

            persisted.LastGenerated = time.Now().Format(time.RFC3339)
            config.Set(DevenvConfigKey{}, persisted)
            config.SetDirty()
            return config.Save()
        },
    }
}
```

### `qsdev devenv add-service <name>`

Adds a service to the existing configuration and regenerates.

```go
func addServiceCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "add-service <name>",
        Short: "Add a service to the devenv configuration",
        Args:  cobra.ExactArgs(1),
        ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            return []string{"postgres", "redis", "mysql", "mongodb", "elasticsearch",
                "rabbitmq", "nats", "kafka", "memcached"}, cobra.ShellCompDirectiveNoFileComp
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            serviceName := args[0]
            persisted := config.Get[DevenvPersistedConfig](DevenvConfigKey{})

            // Check for duplicates
            for _, s := range persisted.Services {
                if s.Name == serviceName {
                    return fmt.Errorf("service %q already configured", serviceName)
                }
            }

            // Run service-specific config form (if any)
            svcConfig := ServiceConfig{Name: serviceName}
            if form := serviceConfigForm(serviceName, &svcConfig); form != nil {
                if err := form.Run(); err != nil {
                    return err
                }
            }

            persisted.Services = append(persisted.Services, svcConfig)
            config.Set(DevenvConfigKey{}, persisted)
            config.SetDirty()
            config.Save()

            // Regenerate
            plan := configToGenerationPlan(persisted)
            return generateAll(plan)
        },
    }
}
```

### `qsdev devenv add-language <name>`

Adds a language to the existing configuration and regenerates. Same pattern as `add-service`.

```go
func addLanguageCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "add-language <name>",
        Short: "Add a language to the devenv configuration",
        Args:  cobra.ExactArgs(1),
        ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            return []string{"go", "javascript", "typescript", "python", "rust"}, cobra.ShellCompDirectiveNoFileComp
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            langName := args[0]
            persisted := config.Get[DevenvPersistedConfig](DevenvConfigKey{})

            for _, l := range persisted.Languages {
                if l.Name == langName {
                    return fmt.Errorf("language %q already configured", langName)
                }
            }

            // Run language-specific config form
            langConfig := LanguageConfig{Name: langName}
            group := languageGroup(langName)
            if err := huh.NewForm(group).Run(); err != nil {
                return err
            }

            persisted.Languages = append(persisted.Languages, langConfig)
            config.Set(DevenvConfigKey{}, persisted)
            config.SetDirty()
            config.Save()

            plan := configToGenerationPlan(persisted)
            return generateAll(plan)
        },
    }
}
```

---

## 8. Non-Interactive Mode

Every wizard question maps to a CLI flag. When a flag is set, the corresponding wizard question is skipped and the flag value is used directly.

| Wizard Question | CLI Flag | Type | Default |
|---|---|---|---|
| Project languages | `--lang` | `[]string` | auto-detected |
| Go version | `--go-version` | `string` | latest stable |
| Node.js package manager | `--node-pkg-mgr` | `string` | `pnpm` |
| Python version | `--python-version` | `string` | latest stable |
| Python package manager | `--python-pkg-mgr` | `string` | `uv` |
| Rust channel | `--rust-channel` | `string` | `stable` |
| Services | `--services` | `[]string` | none |
| PostgreSQL initial DB | `--pg-db` | `string` | `<projectname>_dev` |
| Git hooks | `--hooks` | `[]string` | auto-suggested |
| Enable direnv | `--direnv` | `bool` | `true` |
| Extra packages | `--packages` | `[]string` | `[git]` |
| Accept all defaults | `--yes` | `bool` | `false` |
| Overwrite existing | `--force` | `bool` | `false` |

**Resolution order**: CLI flag > saved config > auto-detected default > hardcoded default.

When `--yes` is set, all questions resolve to their default values without prompting. This enables CI and scripted usage:

```bash
# Fully non-interactive: Go project with Postgres
qsdev devenv init --lang=go --go-version=1.24 --services=postgres --pg-db=myapp_dev --yes

# Re-use saved config from previous run
qsdev devenv init --yes

# CI: generate from explicit flags, overwrite existing
qsdev devenv init --lang=go,typescript --services=postgres,redis --force --yes
```

**Implementation pattern**: Each bootstrap step checks flag values before creating huh forms:

```go
func (s *wizardState) resolveLanguages() []string {
    if len(flagLanguages) > 0 {
        return flagLanguages
    }
    persisted := config.Get[DevenvPersistedConfig](DevenvConfigKey{})
    if len(persisted.Languages) > 0 {
        return languageNames(persisted.Languages)
    }
    detected := detectProject()
    if len(detected) > 0 {
        return detectedLanguageNames(detected)
    }
    return nil // will prompt
}
```

---

## 9. Template Examples

### Example: Go + PostgreSQL devenv.nix Output

Given wizard answers: Go 1.24, PostgreSQL 16 with initial DB "myapp_dev", gofmt hook, direnv enabled, extra packages [git, curl, jq].

**Generated `devenv.nix`:**

```nix
{ pkgs, lib, config, ... }:

{
  # Development tools
  packages = [ pkgs.git pkgs.curl pkgs.jq ];

  # Go
  languages.go.enable = true;
  languages.go.version = "1.24";
  languages.go.delve.enable = true;

  # PostgreSQL
  services.postgres = {
    enable = true;
    package = pkgs.postgresql_16;
    initialDatabases = [{ name = "myapp_dev"; }];
  };

  # Git hooks (pre-commit)
  git-hooks.hooks.gofmt.enable = true;
}
```

**Generated `devenv.yaml`:**

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  git-hooks:
    url: github:cachix/git-hooks.nix
```

**Generated `.envrc`:**

```bash
# Auto-generated by qsdev devenv addon
# Run 'direnv allow' to activate

eval "$(devenv direnvrc)"
use devenv
```

### Example: TypeScript + pnpm + Redis devenv.nix Output

Given: JavaScript with pnpm, TypeScript, Redis, prettier + eslint hooks.

**Generated `devenv.nix`:**

```nix
{ pkgs, lib, config, ... }:

{
  # Development tools
  packages = [ pkgs.git ];

  # JavaScript
  languages.javascript.enable = true;
  languages.javascript.pnpm.enable = true;
  languages.javascript.pnpm.install.enable = true;

  # TypeScript
  languages.typescript.enable = true;

  # Redis
  services.redis = {
    enable = true;
  };

  # Git hooks (pre-commit)
  git-hooks.hooks.prettier.enable = true;
  git-hooks.hooks.eslint.enable = true;
}
```

### Example: Python + uv + PostgreSQL + MongoDB

Given: Python 3.12 with uv, PostgreSQL 16, MongoDB, ruff + mypy hooks.

**Generated `devenv.nix`:**

```nix
{ pkgs, lib, config, ... }:

{
  # Development tools
  packages = [ pkgs.git ];

  # Python
  languages.python.enable = true;
  languages.python.version = "3.12";
  languages.python.uv.enable = true;
  languages.python.uv.sync.enable = true;

  # PostgreSQL
  services.postgres = {
    enable = true;
    package = pkgs.postgresql_16;
    initialDatabases = [{ name = "myapp_dev"; }];
  };

  # MongoDB
  services.mongodb = {
    enable = true;
  };

  # Git hooks (pre-commit)
  git-hooks.hooks.ruff.enable = true;
  git-hooks.hooks.mypy.enable = true;
}
```

**Generated `devenv.yaml`** (note the extra nixpkgs-python input for version pinning):

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  git-hooks:
    url: github:cachix/git-hooks.nix
  nixpkgs-python:
    url: github:cachix/nixpkgs-python
```

---

## 10. Edge Cases and Safety

### Existing File Handling

The addon must handle pre-existing files gracefully:

- **`devenv.nix` exists**: Prompt for overwrite confirmation (skip with `--force`). Never silently overwrite.
- **`devenv.yaml` exists**: Same as above.
- **`.envrc` exists**: Read existing content. If it already contains `use devenv`, skip generation. If it contains other direnv config, append rather than overwrite.
- **`.gitignore` exists**: Append missing entries only. Never replace the file.

### Nix String Escaping

Shell variables inside Nix multiline strings (`'' ... ''`) require `''${var}` escaping. The template must handle this for any user-provided content that might contain `$`:

```go
func nixMultilineEscape(s string) string {
    s = strings.ReplaceAll(s, "''", "'''")
    s = strings.ReplaceAll(s, "${", "''${")
    return s
}
```

### Git-Hooks Input Dependency

Whenever any git hook is enabled, the addon must ensure `devenv.yaml` includes the `git-hooks` input. This is enforced in `buildDevenvYaml()` (see Section 5), not left to the template. Omitting this input causes a devenv evaluation failure.

### Mutually Exclusive Options

Python's `poetry.install.enable` and `uv.sync.enable` cannot both be true. The wizard enforces this by presenting them as a single-select (not multi-select) for the Python package manager. The `add-language` command also validates this constraint.

### Language Version Input Dependencies

Pinning specific language versions may require additional inputs in `devenv.yaml`:
- **Python version pin**: Requires `nixpkgs-python` input
- **Go version pin**: Uses `config.lib.getInput` internally (no extra input needed)
- **Rust channel**: Handled by the Rust module's overlay (no extra input needed)

The `languageInputs()` function (Section 5) encapsulates this knowledge.

### First-Run Warning

After generating files, print a note about first-run latency:

```
Files generated successfully.

Note: The first 'devenv shell' or 'direnv allow' will download and
build all dependencies. This can take 5-30 minutes depending on your
configuration. Subsequent activations are cached and near-instant.
```

---

## 11. Dependency on Other Addons

The devenv addon is intentionally independent. It does not depend on stack, build, k8s, or other infrastructure addons. It generates static files that devenv.sh manages independently.

However, it can compose with other addons when they are present:

- **bootstrap addon**: The devenv addon registers its steps with `bootstrap.WithSteps()`. If bootstrap is not configured, the steps are simply not run during `qsdev bootstrap`, but the `qsdev devenv init` command still works standalone.
- **build addon**: If both devenv and build addons are active, the devenv addon does not interfere with build strategies. devenv.sh manages its own build/dev tooling independently of gdev's build system.

The addon does not register any services with gdev's stack system. devenv.sh manages its own service lifecycle via `devenv up`.

---

## 12. Future Extensions

These are out of scope for the initial implementation but should inform the design to avoid painting into corners:

1. **Imports composition**: For monorepos, generate per-directory `devenv.nix` modules and wire them together via `imports` in the root `devenv.yaml`. The `TemplateData` struct supports this by being scoped to a single module.

2. **Process definitions**: The wizard could grow a "custom processes" step for teams that run non-service workloads (API servers, watchers, compilers). The `TemplateData.Processes` field is reserved but not yet populated by the wizard.

3. **Container generation**: A `qsdev devenv containerize` command could add OCI container configuration to an existing `devenv.nix`. This is a natural extension of the `add-service` pattern.

4. **SecretSpec integration**: For teams using secrets management, a `qsdev devenv add-secrets` command could configure the secretspec block in `devenv.yaml` and create the `secretspec.toml` manifest.

5. **Template updates (Copier-style)**: When the addon's templates evolve, `qsdev devenv update` regenerates from saved config. This is a simple but effective update mechanism. A more sophisticated approach would diff the generated output against the current file and offer a merge, but the regenerate-from-config approach is sufficient for v1.

6. **Profile support**: devenv.sh supports profiles for environment variants. The addon could grow `qsdev devenv add-profile <name>` to manage these.
