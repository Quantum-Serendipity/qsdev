<!-- Source: https://github.com/fastcat/gdev (multiple interface files from repository) -->
<!-- Retrieved: 2026-05-12 -->
<!-- This document catalogs all pluggable extension points in gdev -->

# gdev Extension Interfaces

gdev provides multiple extension points, each with its own interface pattern.
This document catalogs every pluggable seam in the system.

## 1. Addon System (addons/)

The primary extension mechanism. Every addon follows this pattern:

```go
// Package-level unexported addon variable with typed config
var addon = addons.Addon[config]{
    Definition: addons.Definition{
        Name: "my-addon",
        Description: func() string { return "..." },
    },
    Config: config{},
}

func init() {
    addon.Definition.Initialize = initialize
}

// Functional options pattern for configuration
type config struct { /* fields */ }
type option func(*config)

func Configure(opts ...option) {
    addon.CheckNotInitialized()  // panics if already initialized
    for _, o := range opts { o(&addon.Config) }
    addon.RegisterIfNeeded()     // idempotent registration
}

func initialize() error {
    // Called during addons.Initialize(), before LockCustomizations()
    // Can still add commands, services, etc.
    return nil
}
```

**Key constraint:** `Configure()` is called by the consuming app's `main()`.
Addons are initialized in registration order. Addons can depend on other addons
by calling their `Configure()` from within their own `Configure()`.

## 2. CLI Commands (instance/)

```go
// Static commands
instance.AddCommands(cmds ...*cobra.Command)

// Deferred command construction
instance.AddCommandBuilders(fns ...func() *cobra.Command)

// Config sub-commands
cmd.AddConfigCommandBuilder(fns ...func() *cobra.Command)
```

Commands can be added during the customization phase (before `cmd.Main()`).
After `LockCustomizations()`, the command tree is frozen.

## 3. Resource System (resource/)

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

### Resource Wrappers

- `Anti(r Resource) Resource` — inverts Start to call Stop (cleanup resources)
- `Waiter(name, readyFunc) Resource` — blocks Start until ready function passes

### Resource Context (Dependency Injection)

```go
// Register a typed value initializer (once, during setup)
resource.AddContextEntry[T](func(context.Context) (T, error))

// Retrieve in resource Start/Stop/Ready
val := resource.ContextValue[T](ctx)
```

Type-keyed DI: each type T can have exactly one initializer. Values are lazily
initialized on first access. The docker addon uses this to inject `client.APIClient`.

## 4. Service System (service/)

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

### Service Modes

```go
const (
    ModeDefault  Mode = iota  // Run from artifact (container image)
    ModeLocal                 // Run from local source code
    ModeDebug                 // Run in debugger
    ModeDisabled              // Don't run at all
)
```

### Service Builder (Functional Options)

```go
svc := service.New("my-service",
    service.WithResources(r1, r2),                    // always-on
    service.WithModalResources(service.ModeDefault, r3), // default-only
    service.WithModalResources(service.ModeLocal, r4),   // local-only
    service.WithSource(repo, subdir, vcs, remote),       // source info
)
```

Modal resources are automatically converted to Anti resources when the service
runs in a different mode (ensuring cleanup of resources from other modes).

## 5. Stack Hooks (addons/stack/)

```go
type PreStartHook interface {
    Name() string
    LoadServices(context.Context) error
    BeforeServices(ctx context.Context, infra, svcs []service.Service) error
    Service(context.Context, service.Service) error
    AfterServices(ctx context.Context, infra, svcs []service.Service) error
}
```

Registration:
```go
// From a factory function
stack.AddPreStartHook(func() PreStartHook { ... })

// From a type where *T implements PreStartHook
stack.AddPreStartHookType[myHookState]()

// From individual functions
hook := stack.PreStartHookFuncs("name", loadFn, beforeFn, svcFn, afterFn)
```

Also: `stack.AddStartFlaggers(...)` for adding CLI flags to the start command.

## 6. Build Strategies (addons/build/)

```go
type Builder interface {
    Root() string
    BuildAll(context.Context, Options) error
    ValidSubdirs(context.Context) ([]string, error)
    BuildDirs(ctx context.Context, dirs []string, opts Options) error
}

type Detector func(root string) (Builder, error)
```

Registration (via addon option):
```go
build.Configure(
    build.WithStrategy("mage", detectMage, []string{"go-build"}),
)
```

The `supersedes` list enables priority ordering via topological sort. First
matching detector wins.

Example from golang addon:
```go
build.Configure(
    build.WithStrategy("go-build", detectGoBuild, nil),
    build.WithStrategy("mage", detectMage, []string{"go-build"}),
)
```

## 7. Diagnostics (addons/diags/)

### Source Interface
```go
type Source interface {
    Collect(ctx context.Context, collector Collector) error
}

type SourceFunc func(ctx context.Context, collector Collector) error
```

### Collector Interface
```go
type Collector interface {
    Begin(ctx context.Context) error
    Collect(ctx context.Context, name string, contents io.Reader) error
    AddError(ctx context.Context, item string, err error) error
    Finalize(ctx context.Context, collectErr error) error
    Destination() string
}
```

### Source Provider (dynamic sources)
```go
type SourceProvider func(context.Context) ([]Source, error)
```

Registration:
```go
diags.Configure(
    diags.WithSources(s1, s2),          // static sources
    diags.WithSourceFuncs(fn1, fn2),    // function sources
    diags.WithSourceProvider(provider),  // dynamic provider
    diags.WithCollectorProvider(cp),     // output destination
)
```

## 8. GOBUILDCACHE Backends (addons/gocache/)

### Storage Backend Interfaces
```go
type ReadonlyStorageBackend interface {
    io.Closer
    ReadActionEntry(id []byte) (*ActionEntry, error)
    CheckOutputFile(a ActionEntry) (string, error)
    OpenOutputFile(a ActionEntry) (io.ReadCloser, error)
}

type StorageBackend interface {
    ReadonlyStorageBackend
    WriteOutput(a ActionEntry, body io.Reader) (string, error)
    WriteActionEntry(a ActionEntry) error
}
```

### Remote Storage Factory
```go
type RemoteStorageFactory interface {
    Name() string
    Want(uri string) bool
    New(uri string) (ReadonlyStorageBackend, error)
}
```

Registration:
```go
gocache.Configure(
    gocache.WithRemoteStorageFactory(myFactory),
    gocache.WithDefaultRemotes("gs://bucket/prefix"),
)
```

Backends can be layered: write-through, read-through, and readonly chaining.

## 9. Bootstrap Steps (addons/bootstrap/)

```go
type Step struct {
    name   string
    run    func(*Context) error
    // ...
}

func NewStep(name string, run func(*Context) error, opts ...StepOpt) *Step
```

Step options:
- `AfterSteps(names...)` — dependency ordering
- `BeforeSteps(names...)` — reverse dependency
- `SimFunc(fn)` — dry-run simulation
- `SkipFunc(fn)` — conditional skip

Registration:
```go
bootstrap.Configure(
    bootstrap.WithSteps(step1, step2),
    bootstrap.WithChildCmds(subCmd),
    bootstrap.WithAlternatePlanCmd("headless", planFactory, customize),
)
```

Steps are topologically sorted based on their dependency declarations.

## 10. Config System (lib/config/)

```go
type ConfigKey[T any] interface {
    Name() string
    New() T
    NewFrom(value any) (T, error)
    IsDefault(value T) bool
}
```

Registration:
```go
config.AddKey(myConfigKey{})     // during init
val := config.Get(myConfigKey{}) // during runtime
```

Used by the service mode system to persist user's service mode selections.

## Summary of Extension Points

| Extension Point     | When to Use                          | Registration Phase |
|---------------------|--------------------------------------|--------------------|
| Addon               | New major capability                 | main() Configure() |
| CLI Command         | New user-facing commands             | Customization      |
| Resource            | New start/stop/ready lifecycle       | Runtime            |
| Resource Context DI | Shared clients/connections           | Customization      |
| Service             | New app or infra component           | Customization      |
| PreStartHook        | Stack lifecycle interception         | Customization      |
| Build Strategy      | New build tool support               | Configure()        |
| Diags Source        | New diagnostic data collection       | Configure()        |
| Diags Collector     | New diagnostic output destination    | Configure()        |
| GoBuildCache Backend| New remote cache storage             | Configure()        |
| Bootstrap Step      | New machine setup step               | Configure()        |
| Config Key          | New persistent configuration         | init()             |
