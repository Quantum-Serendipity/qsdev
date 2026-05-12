<!-- Source: https://github.com/fastcat/gdev/tree/main/examples -->
<!-- Retrieved: 2026-05-12 -->

# gdev Example Applications

## 1. Minimal: Custom Commands (examples/custom-commands/)

The simplest possible gdev app: just adds a custom CLI command.

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

**Key patterns:**
- `instance.SetAppName()` before `cmd.Main()`
- Commands can be added in `init()` or `main()` before `cmd.Main()`
- No addons needed for basic CLI extension

## 2. Multi-Service Stack (examples/stack/)

Two services (Go + Node.js) with Docker and local mode support.

```go
func main() {
    instance.SetAppName("sdev")
    pm.Configure()
    docker.Configure()
    build.Configure()
    golang.Configure()
    nodejs.Configure()

    stack.AddService(
        myGoService("svc1", svc1Repo, svc1Subdir,
            "ghcr.io/fastcat/gdev/svc1",
            service.WithSource(svc1Repo, svc1Subdir, "", ""),
        ),
    )
    stack.AddService(
        myNodeService("svc2", svc2Repo, svc2Subdir,
            "ghcr.io/fastcat/gdev/svc2",
            service.WithSource(svc2Repo, svc2Subdir, "", ""),
        ),
    )

    cmd.Main()
}
```

**Key patterns:**
- Helper functions (`myGoService`, `myNodeService`) for repeated service patterns
- Each service has ModeDefault (docker container) and ModeLocal (process manager) resources
- `service.WithSource()` enables build detection and auto-compilation
- Addons configured in dependency order (pm, docker, build, golang, nodejs)

### Service Factory Pattern

```go
func myGoService(
    name string,
    _repo, subDir string,
    imageName string,
    opts ...service.BasicOpt,
) service.Service {
    allOpts := []service.BasicOpt{
        service.WithModalResources(
            service.ModeDefault,
            docker.Container(name, imageName).WithPorts("8080"),
        ),
        service.WithModalResources(
            service.ModeLocal,
            resource.PMStatic(api.Child{
                Name: name + "-local",
                Main: api.Exec{
                    Cmd:  "go",
                    Args: []string{"run", filepath.Join(".", subDir)},
                },
            }),
        ),
    }
    allOpts = append(allOpts, opts...)
    return service.New(name, allOpts...)
}
```

## 3. Full Reference Build (examples/gdev/)

Enables nearly all addons. Shows the maximum configuration surface.

```go
func main() {
    instance.SetAppName("gdev")

    bootstrap.Configure(
        apt.WithPackages("Select Go packages for install", "golang"),
        apt.WithPackages("Select git packages for install", "git", "git-lfs", ...),
        bootstrap.WithSteps(shellRCSteps()...),
        bootstrap.WithSteps(input.UserInfoStep()),
        bootstrap.WithSteps(apt.PublicSourceInstallSteps(...)...),
        apt.WithPackages("Select desktop tools", "firefox", "google-chrome-stable", ...),
        bootstrap.WithSteps(github.GHLoginStep(github.GHLoginOpts{})),
    )
    pm.Configure()
    k8s.Configure()
    containerd.Configure()
    docker.Configure()
    k3s.Configure(
        k3s.WithProvider(docker.K3SProvider()),
        k3s.WithK3SArgs("--service-node-port-range=1024-65535"),
    )
    postgres.Configure(postgres.WithService())
    valkey.Configure(valkey.WithService(
        valkey.WithConfig("maxmemory 100mb", "maxmemory-policy allkeys-lru"),
    ))
    build.Configure()
    nodejs.Configure()
    golang.Configure()
    gocache_sftp.Configure()
    gocache_gcs.Configure()
    gocache_http.Configure()
    gocache_s3.Configure(gocache_s3.WithRegion("us-east-1"))
    gocache.Configure(gocache.WithDefaultRemotes(
        "gs://gdev-go-build-cache/v1",
        "s3://gdev-go-build-cache/v1",
    ))
    gcs.Configure(gcs_k8s.WithK8SService())
    diags.Configure(
        diags.WithDefaultFileCollector(),
        diags.WithDefaultSources(),
        diags.WithSourceFuncs(customSource),
        diags.WithSourceProvider(pm.DiagsSources()),
        diags.WithSourceProvider(k8s.DiagsSources()),
    )
    docs.Configure()

    cmd.Main()
}
```

**Key patterns:**
- Addons can be configured with nested options from sub-packages
- Multiple gocache backends configured independently before gocache itself
- Bootstrap steps composed from multiple sources (apt packages, custom steps, login steps)
- Diags sources composed from addon-provided and custom sources
- k3s configured to use Docker as its container runtime backend

## 4. Full-Stack App (examples/full-stack/)

Real application (ent-blog CMS) demonstrating:
- K3S with Docker backend
- PostgreSQL with auto-created database
- Kubernetes Deployment + Service resources in default mode
- Local process manager in local mode
- Atlas database migrations as init container
- Health checks for readiness probes

```go
func main() {
    instance.SetAppName("eb-dev")
    pm.Configure()
    k3s.Configure(
        k3s.WithProvider(docker.K3SProvider()),
        k3s.WithK3SArgs("--service-node-port-range=1024-65535"),
    )
    build.Configure()
    golang.Configure()
    postgres.Configure(postgres.WithService(
        postgres.WithNodePort(55432),
        postgres.WithInitDBs("ent-blog"),
    ))

    stack.AddService(service.New(
        svcName,
        service.WithSource(svcRepo, svcSubdir, "git", "https://github.com/fastcat/gdev.git"),
        service.WithModalResourceFuncs(service.ModeDefault, svcDefaultResources),
        service.WithModalResourceFuncs(service.ModeLocal, svcLocalResources),
    ))

    cmd.Main()
}
```

**Key patterns:**
- `WithModalResourceFuncs` for complex resource construction (returns []resource.Resource)
- Kubernetes apply configurations built programmatically
- Process manager with health checks for local mode
- Environment variables injected from Kubernetes secrets
