# devenv.sh Processes
- **Source**: https://devenv.sh/processes/
- **Retrieved**: 2026-05-12

## Overview
Devenv provides built-in process management featuring supervision, socket activation, file watching, and dependency management capabilities.

## Basic Example
A simple process configuration includes:
- **silly-example**: `while true; do echo hello && sleep 1; done`
- **ping**: `ping localhost`
- **server**: Python HTTP server with configurable working directory

Commands include:
- `devenv up` - starts processes
- `devenv processes down` - stops detached processes
- `devenv processes wait --timeout 120` - waits for readiness (default 120 seconds)

## Dependencies
Processes support dependency management through `after` and `before` declarations. Dependency suffixes include:

**For processes:**
- `@started` - waits for process execution to begin
- `@ready` (default) - awaits readiness probe passage
- `@completed` - waits for process completion regardless of exit code

**For tasks:**
- `@started` - execution begins
- `@succeeded` (default) - exit code 0
- `@completed` - finishes regardless of exit code

## Pre-built Services
Devenv includes pre-configured services: PostgreSQL, Redis, MySQL, MongoDB, and Elasticsearch among others, each with "sensible defaults, health checks, and proper initialization scripts."

## Restart Policies (devenv 2.0+)
Three policies control restart behavior:
- `on_failure` (default) - restart on non-zero exit
- `always` - restart on any exit
- `never` - never restart

Configuration includes `max` setting for restart limits (default: 5, null for unlimited).

## Ready Probes (devenv 2.0+)

**Exec Probe**: Runs shell commands checking readiness via exit code 0:
```
ready = { exec = "pg_isready -d template1"; }
```

**HTTP Probe**: Polls HTTP endpoints:
```
ready = { http.get = { port = 8080; path = "/health"; }; }
```

Supports optional `host` (127.0.0.1 default) and `scheme` (http default).

**Notify Probe**: Uses systemd-style notifications where "Your process should send `READY=1` to the socket path in `$NOTIFY_SOCKET`"

**Probe Timing Options**:
- `initial_delay` - seconds before first probe (default: 0)
- `period` - seconds between probes (default: 10)
- `probe_timeout` - seconds before probe times out (default: 1)
- `success_threshold` - consecutive successes needed (default: 1)
- `failure_threshold` - consecutive failures before unhealthy (default: 3)

TCP connectivity checks apply automatically when `listen` sockets or allocated `ports` are configured without explicit probes.

## File Watching (devenv 2.0+)
Automatically restarts processes on file changes:
- `paths` - directories to monitor
- `extensions` - file types to watch
- `ignore` - patterns to exclude

## Socket Activation (devenv 2.0+)
Enables process manager socket binding before process startup for zero-downtime restarts and lazy startup.

Socket types include:
- TCP sockets: `kind = "tcp"` with address specification
- Unix stream sockets: `kind = "unix_stream"` with path

Environment variables passed to processes:
- `LISTEN_FDS` - number of file descriptors
- `LISTEN_PID` - process ID accepting sockets
- `LISTEN_FDNAMES` - colon-separated socket names

File descriptors start at 3, maintaining systemd socket activation compatibility.

## Watchdog (devenv 2.0+)
Systemd-compatible monitoring requiring periodic `WATCHDOG=1` signals to notify socket. Configuration includes:
- `usec` - timeout duration in microseconds
- `require_ready` - enforcement only after READY=1 (default: true)

## Git Integration
Processes can reference repository root via `${config.git.root}`, useful in monorepo environments for setting working directories.

Processes are automatically available as tasks, enabling pre and post hooks.

## Automatic Port Allocation (devenv 2.0+)
Devenv automatically allocates free ports preventing conflicts:

Configuration uses `ports.<name>.allocate` with base port numbers. The system "will find a free port starting from that base, incrementing until one is available."

Resolved ports access via `config.processes.<name>.ports.<port>.value`.

**Benefits**:
- Running multiple projects simultaneously
- CI environments with parallel test execution
- Shared development machines with multiple developers

### Strict Port Mode
Set `strict_ports: true` in `devenv.yaml` to fail on port conflicts rather than auto-increment. CLI flags override config:
- `devenv up --strict-ports`
- `devenv up --no-strict-ports`

## Alternative Process Managers
Default native process manager can switch to:
- **process-compose** - Feature-rich external manager with TUI
- **overmind** - Procfile-based with tmux integration
- **honcho** - Python Foreman port
- **hivemind** - Simple Procfile manager
- **mprocs** - TUI process manager

Configuration: `process.manager.implementation = "process-compose"`
