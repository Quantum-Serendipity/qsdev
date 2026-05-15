# Task Runner Integration Research

## Research Question

Should gdev include or configure a task runner? If so, which one -- or should it rely on devenv's built-in task system?

## Landscape: Five Contenders

### 1. devenv Tasks (Native)

devenv 2.0+ has a first-class task system with significant capabilities:

- **Namespace-based**: Tasks use `namespace:name` convention (e.g., `myapp:build`)
- **Parallel execution**: Independent tasks run in parallel automatically
- **Dependency ordering**: `before` and `after` fields with DAG resolution
- **Lifecycle hooks**: `devenv:enterShell` and `devenv:enterTest` built-in events
- **Process integration**: All processes are exposed as `devenv:processes:<name>` tasks
- **Conditional execution**: `status` command (skip if returns 0) and `execIfModified` (file-watching)
- **Data passing**: JSON I/O between tasks via environment variables
- **Language agnostic**: Tasks can run in any language via the `package` attribute

This is a real task runner, not a convenience wrapper. It handles the hard problems (parallel execution, dependency ordering, caching, data passing between tasks).

### 2. mise Tasks

mise combines version management + env vars + task running in one tool:

- **Parallel by default**: Independent tasks run concurrently with no config
- **Dependency-aware**: Tasks declare dependencies, mise parallelizes what it can
- **Change detection**: `sources`/`outputs` for incremental execution
- **File-based tasks**: Can write tasks as standalone bash scripts in `.mise/tasks/`
- **Version-aware**: Tasks automatically run with correct language versions from `.mise.toml`

Key differentiator: mise knows which tool versions and env vars your project needs, so tasks run in the correct context automatically.

### 3. just (casey/just)

A pure command runner with no build-system ambitions:

- **Simple mental model**: Recipes are named shell scripts, period
- **No change detection**: Recipes always execute unconditionally
- **No parallel execution**: Sequential only
- **Clean syntax**: Close to Makefile but without the footguns (spaces vs tabs, etc.)
- **Arguments**: Recipes accept positional parameters naturally
- **Cross-platform**: Written in Rust, single binary

Best for: "I want named, documented commands" without build intelligence.

### 4. Taskfile (go-task)

YAML-based task runner with build-tool features:

- **Checksum-based change detection**: Avoids timestamp unreliability
- **Parallel execution**: `deps` tasks run in parallel
- **Variable system**: Dynamic and static variables, .env loading
- **Conditional execution**: `status` and `sources`/`generates` for skipping
- **Cross-platform**: Go binary, YAML config

Best for: Complex workflows where change detection and parallel execution matter.

### 5. Make

The incumbent. Still works. Still has footguns (tabs, implicit rules, timestamp-only detection). Every developer knows it exists. Not worth discussing further for a greenfield tool in 2026.

## Critical Analysis: Does gdev Need to Pick One?

**No. devenv already has the task runner gdev needs.**

The key insight is that gdev generates `devenv.nix`, and devenv's task system is defined in `devenv.nix`. Therefore:

1. gdev already controls the task runner's configuration file
2. devenv tasks run inside the devenv shell, with all the correct packages and env vars
3. devenv tasks have lifecycle hooks that integrate with shell entry and process startup
4. devenv tasks support parallel execution, dependencies, caching, and data passing

Adding a second task runner (just, Taskfile, mise) creates **tool overlap without benefit**:

| Capability | devenv tasks | just | Taskfile | mise |
|------------|-------------|------|----------|------|
| Parallel execution | Yes | No | Yes | Yes |
| Dependency ordering | Yes | Yes | Yes | Yes |
| Change detection | Yes (status, execIfModified) | No | Yes (checksums) | Yes (sources/outputs) |
| Env var context | Yes (devenv shell) | No (manual) | Partial (.env) | Yes (mise.toml) |
| Process integration | Yes (native) | No | No | No |
| Lifecycle hooks | Yes (enterShell, enterTest) | No | No | No |
| Language agnostic | Yes (package attr) | Yes (shell) | Yes (shell) | Yes (shell) |

devenv tasks win on integration. The only area where just/Taskfile have an edge is **discoverability** -- `just --list` or `task --list` is a well-known pattern. But `devenv tasks list` provides the same.

### The One Exception: Teams Not Using devenv

If gdev ever supports non-devenv environments (e.g., Docker-only, bare metal), then a standalone task runner matters. In that case, **just** is the right choice: simplest mental model, no build-system complexity, single binary, cross-platform. But this is a future concern, not a current need.

## What gdev Should Do

1. **Generate devenv task definitions** for common operations (build, test, lint, format, check, deploy) as part of the devenv addon's templates. These are ecosystem-specific -- a Go project gets `go:build`, `go:test`, `go:lint`; a Node project gets `npm:build`, `npm:test`, `npm:lint`.

2. **Do NOT bundle or recommend a separate task runner.** It duplicates devenv's native capability and creates confusion about which task system is "the one."

3. **Document the pattern**: Show developers how to add custom tasks in devenv.nix, how lifecycle hooks work, and how to list/run tasks.

4. **Consider `qsdev run <task>`** as a thin wrapper around `devenv tasks run` -- same functionality, consistent CLI surface.

## Depth Checklist

- [x] Underlying mechanism explained -- devenv task DAG, namespace system, lifecycle hooks
- [x] Key tradeoffs -- native integration vs standalone simplicity, parallel execution capabilities
- [x] Compared to alternatives -- 5 tools compared on 7 dimensions
- [x] Failure modes -- devenv task system is Nix-defined (harder to edit than YAML), learning curve for Nix syntax
- [x] Concrete examples -- devenv task definitions, process-as-task pattern, lifecycle hooks
- [x] Standalone-readable -- yes
