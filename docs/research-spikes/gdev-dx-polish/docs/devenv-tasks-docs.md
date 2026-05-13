<!-- Source: https://devenv.sh/tasks/ -->
<!-- Retrieved: 2026-05-12 -->

# DevEnv Tasks Documentation

## Task Definition Basics

Tasks in devenv allow you to "form dependencies between code, executed in parallel." The fundamental structure uses a namespace:task naming convention:

```nix
tasks."myapp:hello" = {
  exec = ''echo "Hello, world!"'';
};
```

Tasks can be executed individually or by namespace prefix, running all tasks sharing that prefix.

## Lifecycle Hooks

DevEnv provides two built-in lifecycle events:

- **`devenv:enterShell`**: Executes before shell entry (`devenv shell`) and before processes start (`devenv up`)
- **`devenv:enterTest`**: Runs before test execution, automatically inheriting all shell setup tasks

Tasks hook into these events using the `before` attribute:

```nix
tasks."bash:hello" = {
  exec = "echo 'Hello world from bash!'";
  before = [ "devenv:enterShell" ];
};
```

Many devenv modules automatically register dependencies with these events -- for example, git hooks automatically tie into `devenv:enterShell`.

## Dependency Ordering

Tasks support explicit ordering through:

- **`before`**: Declares tasks that must complete before a specific event/task
- **`after`**: Specifies execution after another task/process completes

This enables complex startup sequences and cleanup operations.

## Process Integration

All processes defined in the `processes` configuration become available as tasks with the `devenv:processes:` prefix. This allows:

- Running individual processes as standalone tasks
- Defining task dependencies on processes
- Executing setup tasks before processes start or cleanup tasks afterward

## Input/Output Handling

Tasks exchange data through environment variables:

- `$DEVENV_TASK_INPUT`: JSON object containing task inputs
- `$DEVENV_TASKS_OUTPUTS`: JSON from dependent tasks
- `$DEVENV_TASK_OUTPUT_FILE`: Writable file for output JSON
- `$DEVENV_TASK_EXPORTS_FILE`: File for exporting environment variables to dependents

CLI overrides are possible via `--input` and `--input-json` flags.

## Performance Optimization

Tasks support conditional execution via:

- **`status`**: Returns 0 to skip expensive `exec` commands; outputs are cached from prior successful runs
- **`execIfModified`**: Monitors files/glob patterns; skips if unchanged since last run, preserving previous outputs for dependents

## Additional Features

**Shell Messages**: Tasks (v2.1+) can display informational output by writing to `$DEVENV_TASK_OUTPUT_FILE` with a `devenv.messages` array.

**Git Integration**: Tasks reference repository root via `${config.git.root}`, enabling monorepo support regardless of `devenv.nix` location.

**Language Support**: Tasks execute using specified packages, allowing execution in any language via the `package` attribute.
