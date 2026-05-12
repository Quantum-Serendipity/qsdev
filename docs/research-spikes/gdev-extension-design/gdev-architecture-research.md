# gdev Architecture Deep Research

## Overall Architecture

**Language & Framework:**
- Written in Go (1.26.1)
- Uses Cobra for CLI command structure
- Primary entry point: `cmd.Main()` in the `cmd` package
- Uses YAML configuration (via `go-yaml`) stored in `${HOME}/.config/<appname>.yaml`

**Core Design Pattern:**
gdev is fundamentally a toolkit/framework for building developer experience tools (`xdev`). It's NOT a standalone application, but rather a library + CLI that users customize by:
1. Creating their own Go `main()` function
2. Configuring addons
3. Calling `cmd.Main()`

**Key Lifecycle:**
- **Customization Phase** (before `cmd.Main()`): Addons and instances register themselves
- **Lockdown** (triggered by `cmd.Main()`): Customizations freeze, initialization runs
- **Runtime**: Commands execute with locked-down configuration

---

## Module/Plugin/Extension System

### Addon System (Primary Extension Mechanism)

Located in `/addons` directory with 20+ built-in addons. Core interface in `addons/addon.go`:

```go
type Addon[T any] struct {
    Config T
    Definition Definition
}

type Definition struct {
    Name        string
    Description func() string
    Initialize  func() error
}
```

### Extension Points

1. **Registration Pattern:**
   - Each addon is a package with a module-level `addon` variable
   - Addons register via `addon.RegisterIfNeeded()` in their `Configure()` function
   - Uses "option" pattern for configuration: `Configure(opts ...option)`

2. **Addon Lifecycle:**
   - `Configure()`: Customization phase — register config options, depends on other addons
   - `Initialize()`: After lockdown — register commands, hooks, services

3. **Key Extension Points:**
   - **Commands**: Via `instance.AddCommands()` or `instance.AddCommandBuilders()`
   - **Services**: Via `stack.AddService()` or `stack.AddInfrastructure()`
   - **Config Keys**: Via `config.AddKey()` for YAML persistence
   - **Build Strategies**: Via `build.WithStrategy()` for custom build patterns
   - **Pre-Start Hooks**: Via `stack.AddPreStartHook()` to customize startup
   - **Bootstrap Steps**: Via `bootstrap.WithSteps()` for installation workflows

4. **Customization Guards:**
   - `internal.CheckCanCustomize()`: Panics if called after lockdown
   - `internal.CheckLockedDown()`: Panics if called before lockdown
   - Ensures clean separation between customization and runtime

---

## Configuration Model

### Config Key Registration

```go
type ConfigKey[T any] interface {
    Name() string
    New() T
    NewFrom(value any) (T, error)
    IsDefault(value T) bool
}
```

- Addons call `config.AddKey()` to register custom config keys
- Each key must implement serialization/deserialization

### Storage
- YAML-based, located at `${HOME}/.config/<appname>.yaml`
- Default values omitted from saved file
- Comments preserved on load/save
- Loaded on app startup in `config.Initialize()`

### Access
- Type-safe via generics: `config.Get[T](key)`
- `config.SetDirty()` marks config for saving
- `config.Save()` persists to disk

### Module Contribution
- Each addon can register config keys via `config.AddKey()`
- Configuration frozen before initialization starts

---

## Command Structure

### Entry Points
- Root command from `cmd/Root()` combines version info + addon commands
- Addons add commands via `instance.AddCommands()` or `instance.AddCommandBuilders()`

### Command Categories
- **Built-in**: `version`, `bootstrap`, `start`, `stop`, `addons`, `config`, `build`
- **Addon-specific**: Each addon can add custom commands (e.g., `config <addon-name>`)
- **Stack commands**: `start`, `stop` from stack addon

### Implementation
- Uses Cobra for all CLI management
- Commands added during initialization phase
- Supports nested commands and persistent flags

---

## Integration with External Tools

### devenv.sh/Nix Integration
No explicit devenv.sh or Nix integration found in core gdev code. Existing integrations include:
- Container runtimes (Docker, containerd, K3s)
- Kubernetes integration via K8s addon
- Local process management via process manager addon
- Build strategies for Go, Node.js

### External Tool Integration
- **Docker/containerd addon**: Container execution
- **K3s addon**: Kubernetes cluster management
- **Bootstrap addon**: Apt packages, shell scripting, text file editing
- **Build addon**: Pluggable build strategies (go-build, mage, npm, etc.)

---

## Key Abstractions & Interfaces

### Service Interface (`service/service.go`)
```go
type Service interface {
    Name() string
    Resources(context.Context) ([]resource.Resource, error)
    HasModal(Mode) bool
}
```
- Build services via `service.New()` with option pattern
- Support modal resources (different resources per mode)
- Modes: Default, Local, Debug, Disabled

### Resource Interface (`resource/resource.go`)
```go
type Resource interface {
    ID() string
    Start(context.Context) error
    Stop(context.Context) error
    Ready(context.Context) (bool, error)
}
```
- Containers, K8s objects, local processes implement this
- Anti-resources for stopping resources not needed in a mode

### Build Strategy Pattern (`addons/build/`)
```go
type Builder interface {
    Root() string
    BuildAll(ctx context.Context, opts Options) error
    ValidSubdirs(ctx context.Context) ([]string, error)
    BuildDirs(ctx context.Context, dirs []string, opts Options) error
}
```
- Strategies detected or specified for project types
- Supersedes order resolved topologically

### Pre-Start Hooks (`addons/stack/hooks.go`)
```go
type PreStartHook interface {
    Name() string
    LoadServices(ctx context.Context) error
    BeforeServices(ctx, infra, svcs) error
    Service(ctx, svc) error
    AfterServices(ctx, infra, svcs) error
}
```
- Customization points in stack startup flow

### Bootstrap Steps (`addons/bootstrap/`)
- Custom steps for installation workflows
- Context-aware execution with support for skipping, reboots, etc.

---

## Template & Example Pattern

### Creating Custom Addons

The `addons/_template` addon provides the template:

```go
var addon = addons.Addon[config]{
    Definition: addons.Definition{
        Name: "myname",
        Description: func() string { return "..." },
        Initialize: initialize,
    },
    Config: config{ /* fields */ },
}

func Configure(opts ...option) {
    addon.CheckNotInitialized()
    for _, o := range opts { o(&addon.Config) }
    addon.RegisterIfNeeded()
}

func initialize() error {
    // Register commands, services, hooks here
    return nil
}
```

**Real Examples:**
- `postgres/` addon: Infrastructure service with K8s resources
- `build/` addon: Strategy registration and command integration
- `bootstrap/` addon: Complex multi-step workflows

---

## Design Principles

gdev is built on strong principles:
- **Customization Phase Separation**: Prevents runtime misconfiguration
- **Interface-Based Design**: Type-safe plugins via Go generics
- **Composition Over Inheritance**: Addons combine to build xdev tools
- **No Magic**: Explicit registration and initialization
- **Tree Shakeable**: Unused dependencies don't bloat the binary
