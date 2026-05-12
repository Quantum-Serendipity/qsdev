<!-- Source: https://github.com/fastcat/gdev (multiple files from repository) -->
<!-- Retrieved: 2026-05-12 -->
<!-- This document assembles key source files from the gdev repository to document its architecture -->

# gdev Source Architecture

## Repository Structure

```
fastcat/gdev/
├── addons/           # All addon packages (the extension mechanism)
│   ├── _template/    # Template for creating new addons
│   ├── asdf/         # asdf version manager bootstrap
│   ├── bootstrap/    # System bootstrapping framework
│   ├── build/        # Source code build strategies
│   ├── containerd/   # containerd integration (separate go module)
│   ├── containers/   # Shared container helpers (not an addon)
│   ├── diags/        # Diagnostics collection
│   ├── docker/       # Docker integration (separate go module)
│   ├── docs/         # CLI documentation generation
│   ├── gcloud/       # Google Cloud CLI bootstrap
│   ├── gcs/          # Google Cloud Storage (separate go module)
│   ├── github/       # GitHub CLI bootstrap
│   ├── gocache/      # GOBUILDCACHE with pluggable backends (separate go module)
│   ├── golang/       # Go project build support
│   ├── k3s/          # K3S local kubernetes (separate go module)
│   ├── k8s/          # Kubernetes interaction (separate go module)
│   ├── mariadb/      # MariaDB in K8S (separate go module)
│   ├── nodejs/       # Node.js project build support
│   ├── pm/           # Process manager daemon
│   ├── postgres/     # PostgreSQL in K8S (separate go module)
│   ├── stack/        # Service stack orchestration
│   ├── tailscale/    # Tailscale integration
│   ├── uv/           # Python uv bootstrap
│   └── valkey/       # Valkey (Redis fork) in K8S (separate go module)
├── cmd/              # Main entrypoint (cmd.Main())
├── examples/         # Example applications
│   ├── custom-commands/  # Minimal: just custom CLI commands
│   ├── full-stack/       # Full app with K8S, Postgres, Go service
│   ├── gdev/             # All addons enabled (reference build)
│   └── stack/            # Multi-service with Go + Node.js
├── instance/         # App identity, version, command registration
├── internal/         # Lockdown mechanism (customization vs runtime phases)
├── lib/              # Shared utilities (config, httpx, shx, sys)
├── progress/         # Progress reporting
├── resource/         # Resource abstraction (start/stop/ready lifecycle)
└── service/          # Service abstraction (modal resources, source info)
```

## Go Workspace Structure (go.work)

Addons with heavy dependencies are broken into separate Go modules to avoid
bloating binaries that don't use them:

```
go 1.26.2

use (
    .
    ./addons/bootstrap/input
    ./addons/containerd
    ./addons/docker
    ./addons/gcs
    ./addons/gocache
    ./addons/gocache/gcs
    ./addons/gocache/s3
    ./addons/k3s
    ./addons/k8s
    ./addons/mariadb
    ./addons/postgres
    ./addons/valkey
    ./examples/full-stack
    ./examples/gdev
    ./examples/stack
    ./magefiles
)
```

## Core Addon Framework

### addons/addon.go — Addon[T] Generic Type

```go
type Addon[T any] struct {
    addonState
    Config     T
    Definition Definition
}

func (a *Addon[T]) RegisterIfNeeded() {
    if a.registered.CompareAndSwap(false, true) {
        Register(a)
    }
}

func (a *addonState) CheckNotInitialized() {
    internal.CheckCanCustomize()
    if a.initialized.Load() {
        panic(errors.New("addon already initialized"))
    }
}

func (a *addonState) CheckInitialized() {
    internal.CheckLockedDown()
    if !a.initialized.Load() {
        panic(errors.New("addon not initialized"))
    }
}
```

### addons/description.go — Registration System

```go
type Definition struct {
    Name        string
    Description func() string
    Initialize  func() error
}

var enabled = map[string]*registration{}

func Register[T any](a *Addon[T]) {
    if a.Definition.Name == "" {
        panic(fmt.Errorf("addon name required"))
    }
    internal.CheckCanCustomize()
    if _, ok := enabled[a.Definition.Name]; ok {
        panic(fmt.Errorf("addon %q already enabled", a.Definition.Name))
    }
    enabled[a.Definition.Name] = &registration{...}
    pending = append(pending, a.Definition.Name)
}
```

### addons/pending.go — Ordered Initialization

```go
var pending []string

func Initialize() {
    internal.CheckCanCustomize()
    for _, name := range pending {
        addonReg := enabled[name]
        if addonReg.state.initialized.Load() {
            continue
        }
        if addonReg.Initialize != nil {
            if err := addonReg.Initialize(); err != nil {
                panic(fmt.Errorf("failed to initialize addon %s: %w", addonReg.Name, err))
            }
        }
        addonReg.state.initialized.Store(true)
    }
    pending = nil
}
```

## Addon Template (addons/_template/addon.go)

```go
package template

import "fastcat.org/go/gdev/addons"

var addon = addons.Addon[config]{
    Definition: addons.Definition{
        Name: "template",
        Description: func() string {
            return "Template addon for creating new addons"
        },
    },
    Config: config{},
}

func init() {
    addon.Definition.Initialize = initialize
}

type config struct {
    // Add fields for your addon configuration here
}

type option func(*config)

func Configure(opts ...option) {
    addon.CheckNotInitialized()
    for _, o := range opts {
        o(&addon.Config)
    }
    addon.RegisterIfNeeded()
}

func initialize() error {
    // At this point configuration of all addons is frozen,
    // but you can still add stack services, resource context entries,
    // instance commands, etc.
    return nil
}
```

## App Lifecycle (cmd/main.go)

```go
func Main() {
    addons.Initialize()           // Run all addon Initialize() functions
    internal.LockCustomizations() // No more configuration changes allowed
    config.Initialize()           // Load persisted config
    // ... signal handling, cobra root command execution
}
```

## Two-Phase Lifecycle

The system enforces a strict two-phase lifecycle:

1. **Customization Phase** — `Configure()` calls, addon registration, service
   registration, command registration. Guarded by `CheckCanCustomize()`.

2. **Runtime Phase** — After `LockCustomizations()`. Services start, commands
   execute. Guarded by `CheckLockedDown()`.

Attempting to customize during runtime or instantiate during customization
causes panics.

## Resource System (resource/)

```go
type Resource interface {
    ID() string
    Start(context.Context) error
    Stop(context.Context) error
    Ready(context.Context) (bool, error)
}

type ContainerResource interface {
    Resource
    ContainerImages(context.Context) ([]string, error)
}
```

Resources also support:
- **Anti resources** — `Anti(r)` wraps a resource so Start() calls Stop() (for cleanup)
- **Waiter resources** — `Waiter(name, readyFunc)` blocks during start until ready
- **Context-based DI** — `AddContextEntry[T](initializer)` for type-keyed dependency injection

## Service System (service/)

```go
type Service interface {
    Name() string
    Resources(context.Context) ([]resource.Resource, error)
    HasModal(Mode) bool
}

type ServiceWithSource interface {
    Service
    LocalSource(context.Context) (root, subDir string, err error)
    RemoteSource(context.Context) (vcs, repo string, err error)
    UsesSourceInMode(mode Mode) bool
}
```

Modes: `ModeDefault`, `ModeLocal`, `ModeDebug`, `ModeDisabled`

Services are composed using functional options:
- `WithResources(...)` — always-on resources
- `WithModalResources(mode, ...)` — mode-specific resources (converted to Anti in other modes)
- `WithSource(...)` — source code location info

## Stack Addon (addons/stack/)

Central orchestration addon that manages services and infrastructure:

- `AddService(svc)` — register an application service
- `AddInfrastructure(svc)` — register an infrastructure service (started first)
- `AddPreStartHook(factory)` — register pre-start hooks for the stack lifecycle

### PreStartHook Interface

```go
type PreStartHook interface {
    Name() string
    LoadServices(context.Context) error
    BeforeServices(ctx context.Context, infra, svcs []service.Service) error
    Service(context.Context, service.Service) error
    AfterServices(ctx context.Context, infra, svcs []service.Service) error
}
```

Start order: infra first, then app services. Each resource in each service is
started sequentially, then waited on for readiness.

## Build Strategy System (addons/build/)

Pluggable build detection with priority ordering:

```go
type Builder interface {
    Root() string
    BuildAll(context.Context, Options) error
    ValidSubdirs(context.Context) ([]string, error)
    BuildDirs(ctx context.Context, dirs []string, opts Options) error
}

type Detector func(root string) (Builder, error)
```

Strategies are registered with `WithStrategy(name, detector, supersedes)`.
The `supersedes` list enables topological sorting so more specific strategies
(e.g., `mage` superseding `go`) take precedence.

## Instance Package (instance/)

App-level identity and command registration:

```go
func SetAppName(name string)              // Set before Main()
func AppName() string
func AddCommands(cmds ...*cobra.Command)  // Add CLI commands
func AddCommandBuilders(fns ...func() *cobra.Command)
```

## Diagnostics Addon (addons/diags/)

Pluggable diagnostics collection:

```go
// Source providers add data to the collection
type SourceProvider func(context.Context) ([]Source, error)

// Collector provider creates the output destination
type CollectorProvider func(context.Context) (Collector, error)
```

Configure with:
- `WithSources(...)`, `WithSourceFuncs(...)`, `WithSourceProvider(...)` — add data sources
- `WithCollectorProvider(...)` — set output mechanism
- `WithDefaultFileCollector()` — use temp file output
- `WithDefaultSources()` — add baseline collectors

## GOBUILDCACHE Addon (addons/gocache/)

Pluggable remote storage backends:

```go
type RemoteStorageFactory interface {
    Name() string
    Want(url string) bool
    New(url string) (ReadonlyStorageBackend, error)
}
```

Configure with `WithRemoteStorageFactory(f)` and `WithDefaultRemotes(urls...)`.
Built-in backends: HTTP, GCS, S3, SFTP. Supports write-through and read-through
layering of multiple backends.

## Example: Minimal Custom Commands App

```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
    "fastcat.org/go/gdev/cmd"
    "fastcat.org/go/gdev/instance"
)

func main() {
    instance.SetAppName("edev")
    cmd.Main()
}

func init() {
    instance.AddCommands(&cobra.Command{
        Use: "custom",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("this is the custom command")
        },
    })
}
```

## Example: Full Stack App (ent-blog)

Demonstrates a real application with:
- K3S with Docker backend
- PostgreSQL in Kubernetes with auto-init
- Go service with Kubernetes Deployment + Service resources in default mode
- Local process manager resources in local mode
- Build detection via golang addon
- Source code reference for auto-building

## GitHub Issues (as of 2026-05-12)

- Issue #6: build: bazel support [open]
- Issue #5: nodejs: lerna support [open]
- Issue #4: nodejs: turbo support [open]
- Issue #2: release: get semrel to tag submodules [open]

No discussions found. No wiki. No CONTRIBUTING.md, ARCHITECTURE.md, or DESIGN.md files.
