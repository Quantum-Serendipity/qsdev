# Build Your Own *dev Tool

qsdev is a framework you can fork and rebrand. Import it, set your branding, and ship a binary with the full feature set under your own name.

## Prerequisites

- Go 1.22+
- A GitHub repo for your tool (for self-update)

## Step 1: Create your module

```bash
mkdir acmedev && cd acmedev
go mod init github.com/acme-corp/acmedev
go get github.com/Quantum-Serendipity/qsdev@latest
go get github.com/spf13/cobra
```

## Step 2: Write main.go

```go
package main

import (
    "github.com/spf13/cobra"
    "fastcat.org/go/gdev/addons/bootstrap"
    "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
    "github.com/Quantum-Serendipity/qsdev/addons/devenv"
    "github.com/Quantum-Serendipity/qsdev/addons/devinit"
    "github.com/Quantum-Serendipity/qsdev/instance"
    "github.com/Quantum-Serendipity/qsdev/pkg/branding"
    _ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // all 30 ecosystems
)

func main() {
    // 1. Brand it yours
    instance.SetBranding(branding.Config{
        AppName:       "acmedev",
        ConfigFile:    ".acmedev.yaml",
        LocalConfig:   ".acmedev.local.yaml",
        StateDir:      ".acmedev",
        EnvLogVar:     "ACMEDEV_LOG",
        EnvLogDirVar:  "ACMEDEV_LOG_DIR",
        EnvNoUpdate:   "ACMEDEV_NO_UPDATE_CHECK",
        EnvPrefix:     "ACMEDEV_",
        LogFilePrefix: "acmedev-",
        TempPrefix:    ".acmedev-tmp-",
        GitHubOwner:   "acme-corp",
        GitHubRepo:    "acmedev",
    })

    // 2. Configure bootstrap steps
    bootstrap.Configure(
        bootstrap.WithSteps(
            devenv.InstallDevenvStep(),
            devenv.InstallDirenvStep(),
            claudecode.InstallClaudeStep(),
        ),
    )

    // 3. Tune addon defaults
    devenv.Configure(devenv.WithDefaultLanguages("go", "python"))
    claudecode.Configure(claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard))
    devinit.Configure(devinit.WithDetectProjectType(true))

    // 4. Add custom commands
    instance.AddCommands(acmeHelloCmd())

    // 5. Install the standard runtime and launch (in place of gdev's
    // cmd.Main: it also makes every command group reject unknown
    // subcommands, so a typo exits non-zero)
    instance.Main()
}

func acmeHelloCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "hello",
        Short: "A custom acmedev command",
        Run: func(cmd *cobra.Command, args []string) {
            cmd.Println("Hello from acmedev!")
        },
    }
}
```

That's the whole binary.

`instance.Main()` must be the last call in `main`. It installs the same runtime
qsdev itself runs with, then starts the CLI:

- the universal MCP server's framework adapters (`mcp serve`) and the
  external-log providers
- the release version stamped at build time (see [Version injection](#version-injection))
- the project's committed `.acmedev/defaults.yaml` catalog layer
- the standard `self-update`, `logs` and `report` commands
- the `--debug` flag and a redacting session log, written to the project's
  `.acmedev/logs/` or to `~/.acmedev/logs/` (override with `ACMEDEV_LOG_DIR`)
- the background update check against your GitHub releases

Calling gdev's `cmd.Main()` directly skips all of this: `mcp serve` finds no
framework adapters, nothing is logged, and unknown subcommands at the root and
in `config` print help and exit 0 (see strict command dispatch below). If you
do need to call `cmd.Main()` yourself, call `instance.DefaultRuntime()` first
and its `Finish()` method afterwards:

```go
rt := instance.DefaultRuntime()
cmd.Main()
rt.Finish()
```

## Step 3: Build and run

```bash
go build -o acmedev .
./acmedev --help
./acmedev init        # runs the full setup wizard
./acmedev hello       # your custom command
```

## Customization

### Adding custom commands

Use `instance.AddCommands()` with standard Cobra commands, or `instance.AddCommandBuilders()` for deferred registration (useful when commands depend on config loaded at runtime):

```go
instance.AddCommandBuilders(func() *cobra.Command {
    return &cobra.Command{
        Use: "deploy",
        RunE: func(cmd *cobra.Command, args []string) error {
            // access runtime config here
            return nil
        },
    }
})
```

### Adding custom ecosystem modules

Implement `ecosystem.EcosystemModule`. At minimum you need detection logic; stub the rest until you need them:

```go
type InternalToolModule struct{}

func (m *InternalToolModule) Name() string          { return "internaltool" }
func (m *InternalToolModule) DisplayName() string   { return "Internal Tool" }
func (m *InternalToolModule) Tier() int             { return 3 }
func (m *InternalToolModule) Detect(root string) ecosystem.DetectionResult {
    if _, err := os.Stat(filepath.Join(root, ".internaltool.json")); err == nil {
        return ecosystem.DetectionResult{Detected: true, Confidence: ecosystem.ConfidenceCertain}
    }
    return ecosystem.DetectionAbsent()
}
// ... stub remaining interface methods
```

To ask the user for settings in the `init` wizard, also implement
`ecosystem.WizardFieldProvider`. Each `WizardField.Key` must be the setting
your module reads: `types.SettingVersion` for `ModuleConfig.Version`,
`types.SettingPackageManager` for `ModuleConfig.PackageManager`, or the
`ModuleConfig.Extras` key otherwise. Give every input field a `Placeholder`
example.

Register via init (auto-discovery):

```go
func init() {
    ecosystem.RegisterModule(&InternalToolModule{})
}
```

Or explicitly:

```go
instance.AddEcosystemModules(&InternalToolModule{})
```

### Choosing which built-in modules to include

Import all 30 at once:

```go
_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
```

Or pick individual ones:

```go
_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
```

No blank import = module not included. Your binary, your choice.

### Configuring addons

Each addon exposes `Configure()` with functional options:

```go
// devenv — devenv.sh environment generation
devenv.Configure(
    devenv.WithDefaultLanguages("go", "rust"),
    devenv.WithDefaultServices("postgres", "redis"),
)

// claudecode — Claude Code agent configuration
claudecode.Configure(
    claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard),
)

// devinit — orchestration wizard
devinit.Configure(
    devinit.WithDetectProjectType(true),
)
```

### Version injection

`instance.Main()` reports the version stamped into `instance.VersionPackage`,
which the self-update check and generated state also read. Stamp that package,
not your own `main`:

```bash
go build -ldflags "-X github.com/Quantum-Serendipity/qsdev/internal/version.version=v1.2.3 \
  -X github.com/Quantum-Serendipity/qsdev/internal/version.commit=$(git rev-parse --short HEAD)" .
```

Development builds (no stamp) report `dev`. A version set explicitly with
`instance.SetVersionOverride(version, commit)` before `instance.Main()` takes
precedence over the stamp for `version` and `self-update`; the background
update notice, bug reports and session logs still read the stamped version, so
keep the two in step.

## What you get for free

By importing qsdev, your tool ships with:

- **30 ecosystem modules** — Go, JavaScript/TypeScript, Python, Rust, Java, .NET, Ruby, PHP, Swift, Scala, Elixir, Haskell, Zig, C/C++, Dart, Clojure, Lua, Perl, R, Shell, PowerShell, Nix, Containers, Terraform, Helm, Ansible, Bazel, AWS, GCP, Azure
- **Supply chain security** — lockfile enforcement, age-gating, registry pinning, deny rules per ecosystem
- **AI agent configuration** — Claude Code permissions, PreToolUse hooks, skill scaffolding
- **devenv.sh generation** — languages, services, packages, pre-commit hooks, all from one config
- **Fragment accumulation** — multi-addon file composition with 5 merge modes (replace, append, section, merge-JSON, merge-YAML)
- **Lifecycle hooks** — PostCollect and PostResolve hooks for custom addon integration
- **Tool behavior system** — two-phase tool registration: YAML catalog for metadata, Go functions (EnableFunc, DisableFunc, GenerateFunc, DetectFunc) for behavior
- **Hook execution sandboxing** — bubblewrap + landlock + seccomp isolation with 5 degradation tiers
- **Self-update** — GitHub release checking, in-place binary update
- **Strict command dispatch** — `instance.Main()` makes the root and every command group, including ones your tool adds, reject an unknown subcommand with a non-zero exit and a "Did you mean" suggestion, so a typo in a CI step (`acmedev chek`) fails instead of printing help and exiting 0. A tool that still launches through gdev's `cmd.Main` keeps this only for the qsdev addon command groups; switch to `instance.Main()` to cover the root, `config` and your own groups
