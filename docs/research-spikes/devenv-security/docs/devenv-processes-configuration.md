# devenv.sh Processes Configuration
- **Source**: https://devenv.sh/processes/
- **Retrieved**: 2026-05-12

## Available Options

Devenv processes support extensive configuration:

- **Execution**: `exec` command with working directory (`cwd`)
- **Dependencies**: `after`/`before` with suffixes (`@started`, `@ready`, `@succeeded`, `@completed`)
- **Restart policies**: `on_failure` (default), `always`, or `never` with max restart limits
- **Readiness probes**: exec commands, HTTP endpoints, or systemd-style notifications
- **File watching**: automatic restart on file changes with path/extension filters
- **Socket activation**: TCP and Unix domain sockets pre-bound before process start
- **Watchdog monitoring**: periodic health signals via notify socket
- **Port allocation**: automatic free port discovery with optional strict mode
- **Environment**: access to git root via `${config.git.root}`

## Process Management and Isolation

Processes run under supervision with these characteristics:

"Devenv provides built-in process management with supervision, socket activation, file watching, and dependency management." They operate as independent executables rather than isolated containers -- no namespace or cgroup isolation is mentioned.

Port allocation prevents conflicts through automatic discovery rather than namespace segregation. Socket activation enables "zero-downtime restarts and lazy process startup."

Multiple alternative process managers are available: process-compose, overmind, honcho, hivemind, and mprocs.

## Security Implications

Documentation does not explicitly address security isolations. Notable considerations:

- **No explicit isolation**: Processes share the same user/environment space
- **Notify socket access**: Processes receive `$NOTIFY_SOCKET` paths for readiness signaling
- **File descriptor passing**: Socket activation passes FDs 3+ via `LISTEN_FDS` (systemd-compatible but unauthenticated)
- **Environment variable exposure**: Git roots and port assignments visible to all processes
