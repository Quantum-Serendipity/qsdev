<!-- Source: https://devenv.sh/processes/ -->
<!-- Retrieved: 2026-05-12 -->

# Devenv Process Management Documentation

## Overview

Devenv provides integrated process supervision with dependency management, health checking, and file watching capabilities. Processes are defined in `devenv.nix` and managed through the `devenv up` command.

## Starting and Stopping Processes

**Starting processes:**
```
$ devenv up
```

**Stopping detached processes:**
```
$ devenv processes down
```

**Waiting for readiness (useful in CI):**
```
$ devenv processes wait --timeout 120
```

The default timeout is 120 seconds.

## Process Dependencies

Processes can declare dependencies using `after` and `before` directives. The platform distinguishes between process and task dependencies:

**Process dependency suffixes:**
- `@started` — begins execution
- `@ready` (default) — "readiness probe to pass"
- `@completed` — finishes regardless of exit code

**Task dependency suffixes:**
- `@started` — begins execution
- `@succeeded` (default) — exits with code 0
- `@completed` — finishes regardless of outcome

## Health Checking (Ready Probes)

Three probe types detect process readiness:

**Exec probes:** Run shell commands; exit code 0 indicates readiness.

**HTTP probes:** Poll an HTTP endpoint for readiness with configurable port, path, host, and scheme.

**Notify probes:** Use systemd-style readiness notification where processes send `READY=1` to the `$NOTIFY_SOCKET`.

All probe types support timing configuration:
- `initial_delay` — seconds before first probe (default: 0)
- `period` — seconds between probes (default: 10)
- `probe_timeout` — seconds before probe times out (default: 1)
- `success_threshold` — consecutive successes needed (default: 1)
- `failure_threshold` — consecutive failures before unhealthy (default: 3)

## Restart Policies

Three restart strategies control process behavior after exit:

- `on_failure` (default) — restart only on non-zero exit
- `always` — restart on any exit
- `never` — never restart

Configuration includes maximum restart attempts (default: 5, null for unlimited).

## Socket Activation

Socket activation allows the process manager to bind sockets before starting your process for zero-downtime restarts and lazy startup.

Supported socket kinds: TCP and Unix domain streams.

Processes receive:
- `LISTEN_FDS` — number of file descriptors
- `LISTEN_PID` — accepting process ID
- `LISTEN_FDNAMES` — colon-separated socket names

This implements systemd socket activation compatibility.

## File Watching

Processes can automatically restart when monitored files change. Configuration includes:
- `paths` — directories to monitor
- `extensions` — file types to watch
- `ignore` — patterns to exclude

## Watchdog Monitoring

Enable systemd-compatible watchdog monitoring where processes must periodically send `WATCHDOG=1` to the notify socket or face termination and restart.

## Port Management

Automatic port allocation finds free ports starting from a configured base, preventing conflicts. When port 8080 is taken, devenv automatically tries 8081, 8082, etc.

**Strict port mode** fails instead of auto-allocating when conflicts occur.

## Process Managers

The native process manager is default. Alternatives include:
- process-compose — Feature-rich external process manager with TUI
- overmind — Procfile-based with tmux integration
- honcho, hivemind, mprocs — Additional implementations

## Pre-built Services

Devenv includes many pre-configured services with proper process management including PostgreSQL, Redis, MySQL, MongoDB, and Elasticsearch with sensible defaults and health checks.
