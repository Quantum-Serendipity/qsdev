<!-- Source: https://devenv.sh/blog/2025/07/25/devenv-devlog-processes-are-now-tasks/ -->
<!-- Retrieved: 2026-05-12 -->

# Devenv Processes Are Now Tasks

## What Changed

Devenv now exposes all processes as tasks named `devenv:processes:<name>`. This integration allows developers to orchestrate startup and shutdown sequences by running tasks before or after processes execute.

## How Tasks and Processes Interact

Previously, processes ran independently. Now, "process-compose runs processes through `devenv-tasks run --mode all devenv:processes:<name>`" rather than executing them directly. This approach maintains existing process functionality while enabling task capabilities.

## Lifecycle Hooks

The update supports two key scenarios:

**Before execution:** Tasks can run setup operations (like database migrations) before a process starts. Example: a `db:migrate` task marked with `before = [ "devenv:processes:backend" ]` runs migrations before the backend service launches.

**After execution:** Tasks can perform cleanup operations after a process stops, such as removing temporary files or process artifacts.

## Dependency Ordering

Tasks use `before` and `after` fields to define ordering. The `--mode all` flag ensures both dependency directions execute, preserving expected lifecycle behavior.

## Workflow Implications

This addresses a "frequently requested feature for orchestrating the startup sequence." Developers gain finer control over environment initialization and teardown without manual orchestration scripts.
