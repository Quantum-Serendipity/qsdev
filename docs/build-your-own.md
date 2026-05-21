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
    "fastcat.org/go/gdev/cmd"
    "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
    "github.com/Quantum-Serendipity/qsdev/addons/devenv"
    "github.com/Quantum-Serendipity/qsdev/addons/devinit"
    "github.com/Quantum-Serendipity/qsdev/instance"
    "github.com/Quantum-Serendipity/qsdev/pkg/branding"
    _ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // all 27 ecosystems
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

    // 5. Launch
    cmd.Main()
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

Import all 27 at once:

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

Wire your release version at build time:

```go
instance.SetVersionOverride(version, commit)
```

Then build with:

```bash
go build -ldflags "-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD)" .
```

## What you get for free

By importing qsdev, your tool ships with:

- **27 ecosystem modules** — Go, JavaScript/TypeScript, Python, Rust, Java, .NET, Ruby, PHP, Swift, Scala, Elixir, Haskell, Zig, C/C++, Dart, Clojure, Lua, Perl, R, Shell, PowerShell, Nix, Docker, Terraform, Helm, Ansible, Bazel
- **Supply chain security** — lockfile enforcement, age-gating, registry pinning, deny rules per ecosystem
- **AI agent configuration** — Claude Code permissions, PreToolUse hooks, skill scaffolding
- **devenv.sh generation** — languages, services, packages, pre-commit hooks, all from one config
- **Self-update** — GitHub release checking, in-place binary update
