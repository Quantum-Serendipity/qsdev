# Config Template Engine Design

## Problem

The qsdev init wizard collects answers and must produce 7+ files in 4 different formats: Nix code, YAML, JSON, and Markdown. Each format has different constraints, and a unified approach to generation ensures consistency, testability, and maintainability.

## Generation Strategy Per Format

### Nix Code (devenv.nix) — `text/template`

Nix is a functional language with its own syntax. There's no Go library for constructing Nix ASTs, so `text/template` is the right approach — it produces text that happens to be valid Nix.

**Why not struct marshaling:** Nix has no standard data interchange format. The language syntax IS the format.

**Why not string concatenation:** Unreadable, untestable, no conditional blocks.

**Custom template functions required:**

```go
var nixFuncs = template.FuncMap{
    "nixList": func(items []string) string {
        // [ pkgs.go pkgs.gopls pkgs.delve ]
        quoted := make([]string, len(items))
        for i, item := range items {
            quoted[i] = "pkgs." + item
        }
        return "[ " + strings.Join(quoted, " ") + " ]"
    },
    "nixString": func(s string) string {
        // Escape Nix string interpolation: ${} → \${}
        escaped := strings.ReplaceAll(s, `${`, `\${`)
        return `"` + escaped + `"`
    },
    "nixBool": func(b bool) string {
        if b { return "true" }
        return "false"
    },
    "nixMultiline": func(s string) string {
        // Escape Nix multiline string interpolation: ${} → ''${}
        return strings.ReplaceAll(s, "${", "''${")
    },
    "indent": func(n int, s string) string {
        pad := strings.Repeat(" ", n)
        lines := strings.Split(s, "\n")
        for i, line := range lines {
            if line != "" { lines[i] = pad + line }
        }
        return strings.Join(lines, "\n")
    },
    "hasAny": func(items []string) bool {
        return len(items) > 0
    },
}
```

**Template structure:** One base template with sub-templates per section:

```go
//go:embed templates/devenv.nix.tmpl
var devenvNixTemplate string

// templates/devenv.nix.tmpl
// {{ define "languages" }}...{{ end }}
// {{ define "services" }}...{{ end }}
// {{ define "packages" }}...{{ end }}
// etc.
```

**Testing strategy:** Generate known inputs → snapshot test against expected .nix output. Also run `nix-instantiate --parse` on generated output to verify syntax validity.

### YAML (devenv.yaml) — Struct Marshaling

YAML has a standard Go library (`gopkg.in/yaml.v3`). Struct marshaling is safer than templates — guarantees valid YAML syntax, handles escaping automatically, and the Go type system prevents structural errors.

```go
type DevenvYAML struct {
    Inputs  map[string]DevenvInput `yaml:"inputs,omitempty"`
    Imports []string               `yaml:"imports,omitempty"`
    Nixpkgs NixpkgsConfig          `yaml:"nixpkgs,omitempty"`
}

type DevenvInput struct {
    URL     string            `yaml:"url"`
    Follows map[string]string `yaml:"follows,omitempty"`
}

func generateDevenvYAML(config DevenvPersistedConfig) ([]byte, error) {
    doc := buildYAMLDoc(config) // construct typed struct
    return yaml.Marshal(doc)
}
```

**Why not templates:** YAML's indentation sensitivity makes templates fragile. A typo in template indentation produces valid YAML with wrong semantics — silent bugs.

### JSON (settings.json, .mcp.json) — Struct Marshaling

Same rationale as YAML. Go's `json.MarshalIndent` produces valid, readable JSON from typed structs.

```go
type SettingsJSON struct {
    Permissions *Permissions  `json:"permissions,omitempty"`
    Sandbox     *SandboxConfig `json:"sandbox,omitempty"`
    Hooks       *HooksConfig  `json:"hooks,omitempty"`
    Attribution *Attribution  `json:"attribution,omitempty"`
}

func generateSettingsJSON(config ClaudePersistedConfig) ([]byte, error) {
    doc := buildSettingsDoc(config)
    return json.MarshalIndent(doc, "", "  ")
}
```

**Why not templates:** JSON has strict syntax (commas, brackets). Template-based JSON generation is a common source of trailing comma bugs and bracket mismatches.

### Markdown (CLAUDE.md, skills, rules) — `text/template`

Markdown is free-form text with sections controlled by project type, languages, and team conventions. Templates are the right approach.

```go
//go:embed templates/claude-md.tmpl
var claudeMdTemplate string

// templates/claude-md.tmpl:
// # CLAUDE.md
//
// ## Project Overview
// {{ .ProjectDescription }}
//
// {{ if .HasGo }}
// ## Go Conventions
// {{ template "go-conventions" . }}
// {{ end }}
// ...
```

**Why not struct marshaling:** There's no "markdown struct." The output is prose with conditional sections.

**Skill and rule files:** Copied verbatim from `embed.FS`, not templated. Skills are complete markdown files maintained in a library; the addon copies them to `.claude/skills/`. No generation needed.

## Unified Template Data Model

Both addons feed from the same wizard answers. A shared data model ensures consistency:

```go
type WizardAnswers struct {
    // Project detection
    ProjectName string
    ProjectRoot string
    Detected    DetectedProject
    
    // Language choices
    Languages []LanguageChoice
    
    // Service choices
    Services []ServiceChoice
    
    // devenv options
    Direnv       bool
    GitHooks     []string
    ExtraPackages []string
    EnvVars      map[string]string
    
    // Claude Code options
    ClaudeCode      bool
    PermissionLevel string
    Skills          []string
    Hooks           HookChoices
    MCPServers      []string
}

type LanguageChoice struct {
    Name           string // "go", "typescript", "python", "rust"
    Version        string // "1.22", "22", "3.12", etc.
    PackageManager string // "npm", "pnpm", "yarn", "poetry", "uv"
    Extras         []string // language-specific packages
}

type ServiceChoice struct {
    Name     string // "postgres", "redis", "mysql", etc.
    Version  string // optional version pin
    Settings map[string]string // service-specific config
}
```

Each addon transforms `WizardAnswers` into its own template data:

```go
// devenv addon
func (a *devenvAddon) templateDataFrom(answers WizardAnswers) DevenvTemplateData { ... }

// claudecode addon
func (a *claudecodeAddon) templateDataFrom(answers WizardAnswers) ClaudeTemplateData { ... }
```

## Template Organization

All templates live in `embed.FS` within each addon's package:

```
addons/devenv/
├── templates/
│   ├── devenv.nix.tmpl       # main devenv.nix template
│   ├── languages/
│   │   ├── go.nix.tmpl       # Go-specific Nix block
│   │   ├── typescript.nix.tmpl
│   │   ├── python.nix.tmpl
│   │   └── rust.nix.tmpl
│   ├── services/
│   │   ├── postgres.nix.tmpl
│   │   ├── redis.nix.tmpl
│   │   └── ...
│   └── envrc.tmpl            # .envrc template

addons/claudecode/
├── templates/
│   ├── claude-md.tmpl        # CLAUDE.md template
│   ├── conventions/
│   │   ├── go.md.tmpl        # Go conventions section
│   │   ├── typescript.md.tmpl
│   │   └── python.md.tmpl
│   └── hooks/
│       ├── auto-format.json.tmpl
│       └── safety.json.tmpl
├── skills/                   # copied verbatim, not templated
│   ├── deploy.md
│   ├── security-review.md
│   └── review.md
├── rules/                    # copied verbatim
│   ├── go-rules.md
│   └── typescript-rules.md
```

## Generation Pipeline

```go
type GeneratedFile struct {
    Path     string // relative to project root
    Content  []byte
    Mode     os.FileMode
    Strategy MergeStrategy // overwrite | append | merge | skip
}

type Generator interface {
    Generate(answers WizardAnswers) ([]GeneratedFile, error)
}

// devinit orchestrates:
func (d *devinitAddon) generate(answers WizardAnswers) error {
    var allFiles []GeneratedFile
    
    if devenvFiles, err := d.devenv.Generate(answers); err == nil {
        allFiles = append(allFiles, devenvFiles...)
    }
    if claudeFiles, err := d.claude.Generate(answers); err == nil {
        allFiles = append(allFiles, claudeFiles...)
    }
    allFiles = append(allFiles, d.gitignoreFile(allFiles))
    
    // Plan preview
    if d.planPreview {
        d.showPlanPreview(allFiles)
    }
    
    // Write atomically
    return d.writeFiles(allFiles)
}
```

## Atomic Writes

All files written atomically (write temp file → rename) to avoid partial generation on errors:

```go
func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(dir, ".gdev-*")
    if err != nil {
        return err
    }
    defer os.Remove(tmp.Name()) // cleanup on failure
    
    if _, err := tmp.Write(content); err != nil {
        tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    if err := os.Chmod(tmp.Name(), mode); err != nil {
        return err
    }
    return os.Rename(tmp.Name(), path)
}
```

## Validation

Post-generation validation catches template bugs before the user sees broken config:

| File | Validation |
|------|-----------|
| devenv.nix | `nix-instantiate --parse` (syntax check, no eval) |
| devenv.yaml | YAML round-trip (marshal → unmarshal) |
| settings.json | JSON round-trip |
| .mcp.json | JSON round-trip |
| CLAUDE.md | None (free-form markdown) |
| .envrc | `bash -n` (syntax check) |

Validation is best-effort — if the tool isn't installed (e.g., `nix-instantiate`), skip and warn.
