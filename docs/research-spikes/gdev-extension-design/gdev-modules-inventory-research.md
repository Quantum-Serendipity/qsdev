# gdev Modules & Extensions Inventory

## Overview

gdev ships 25+ built-in addons across 11 categories. Each addon follows the same pattern: a module-level `Addon[T]` generic struct, a `Configure()` function with option pattern for customization phase, and an `Initialize()` callback for post-lockdown setup.

## Addon Categories

### Category 1: Stack & Service Management

**stack** (`addons/stack/`)
- Core framework for service orchestration
- Extension points: `AddService()`, `AddInfrastructure()`, `AddPreStartHook()`, `PreStartHookFuncs()`
- Service modes: Default, Local, Debug, Disabled
- Commands: start, stop, apply

**pm (Process Manager)** (`addons/pm/`)
- Daemon for running local processes outside containers
- Extension: `WithTask(interval, timeout, runFunc)` for periodic background tasks
- Resource integration via `PMStaticInfra()`

### Category 2: Build Systems

**build** (`addons/build/`)
- Strategy-based build system for multiple project types
- Extension: `WithStrategy(name, Detector, supersedes[])` — pluggable build strategies
- Topological sort for strategy ordering; cycle detection

**golang** (`addons/golang/`)
- Go project building: `go-build` and `mage` strategies
- Mage supersedes go-build when detected

**nodejs** (`addons/nodejs/`)
- Node.js project building: `npm` and `rush` strategies
- Workspace configuration parsing for monorepos

### Category 3: Container & Kubernetes Infrastructure

**docker** (`addons/docker/`)
- Docker client support, resource management, bootstrap integration
- K3s provider available

**containerd** (`addons/containerd/`)
- Containerd support, similar structure to docker

**k8s** (`addons/k8s/`)
- Generic Kubernetes support
- Config: `WithContextFunc()`, `WithNamespace()`
- Waiters: `APIReadyWaiter()`, `NodeReadyWaiter()`

**k3s** (`addons/k3s/`)
- Local k3s cluster management
- Provider pattern: configurable backend (docker/containerd)
- Config: `WithProvider()`, `WithContext()`, `WithNamespace()`, `WithPath()`, `WithK3SArgs()`

### Category 4: Databases

**postgres** (`addons/postgres/`)
- PostgreSQL service in Kubernetes
- Config: `WithService(opts ...svcOpt)`
- Bootstrap integration: postgresql-client package

**mariadb** (`addons/mariadb/`)
- Similar pattern to postgres

**valkey** (`addons/valkey/`)
- Redis fork, similar pattern
- Config: `WithService(WithConfig(configStr))`

### Category 5: Go Build Cache (GOCACHEPROG)

**gocache** (`addons/gocache/`)
- `GOCACHEPROG` implementation with pluggable remote backends
- Factory pattern: `WithRemoteStorageFactory(RemoteStorageFactory)`
- Layering: write-through and read-through backends for chaining

**Sub-addons:** gocache-gcs, gocache-http, gocache-s3, gocache-sftp

### Category 6: Cloud & Storage

**gcs** (`addons/gcs/`) — GCS emulator and real GCS support
**gcloud** (`addons/gcloud/`) — Google Cloud CLI bootstrap and login
**github** (`addons/github/`) — GitHub CLI bootstrap and login

### Category 7: System Bootstrap

**bootstrap** (`addons/bootstrap/`)
- System initialization and software installation framework
- Config: `WithSteps()`, `WithChildCmds()`, `WithAlternatePlanCmd()`
- Step system with reboot support, skip handlers, headless mode
- Sub-components: apt, input (user interaction), textedit (file editing)
- Plan system: base plan + derived plans with exceptions

**Sub-addons:** asdf, uv (version/environment managers)

### Category 8: Diagnostics

**diags** (`addons/diags/`)
- Diagnostic collection and uploading
- Config: `WithSources()`, `WithSourceFuncs()`, `WithCollectorProvider()`
- Built-in collectors: TarFileCollector

### Category 9: Documentation

**docs** (`addons/docs/`) — Generate CLI docs (man pages, markdown) from cobra commands

### Category 10: Networking

**tailscale** (`addons/tailscale/`) — Tailscale VPN integration with bootstrap

### Category 11: Template

**_template** (`addons/_template/`) — Template for creating new addons

## Dependency Graph

```
stack (base)
  <- build (depends on stack)
    <- golang (depends on build)
    <- nodejs (depends on build)
  <- postgres (depends on stack)
  <- valkey (depends on stack)
  <- mariadb (depends on stack)
  <- k8s (independent, configured alongside)
    <- k3s (depends on k8s, pm)
      <- docker or containerd (selected provider)
  <- pm (process manager)

gocache (independent)
  <- gocache-gcs, gocache-http, gocache-s3, gocache-sftp

bootstrap (independent)
  <- asdf, uv, gcloud, github (sub-addons that add steps)

diags (independent, extensible)
docs (independent)
gcs (independent)
```

## Extension Points Summary

1. **Addon Configuration** — `Configure()` functions with `Option` pattern
2. **Build Strategies** — `build.WithStrategy(name, Detector, supersedes)`
3. **Bootstrap Steps** — `bootstrap.WithSteps(*Step)`, custom step factories
4. **Services** — `stack.AddService()`, `stack.AddInfrastructure()`
5. **Pre-start Hooks** — `stack.AddPreStartHook()`, lifecycle interception
6. **Commands** — `instance.AddCommands()`, `instance.AddCommandBuilders()`
7. **Context Entries** — `resource.AddContextEntry[T]()`
8. **Backend Factories** — `gocache.WithRemoteStorageFactory()`
9. **Diagnostic Sources** — `diags.WithSourceProvider()`
10. **Collectors** — `diags.WithCollectorProvider()`

## Key Design Principles

- Type-safe generic addons
- Explicit initialization phases with customization guards
- Configuration immutability after init
- Interface-based extensibility
- Factory patterns for pluggable backends
- Dependency injection via typed context entries
