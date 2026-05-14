# devenv Tasks Documentation

- **Source URL**: https://devenv.sh/tasks/
- **Retrieval Date**: 2026-05-14

## Overview

Devenv tasks enable you to "form dependencies between code, executed in parallel." Introduced in version 1.2, tasks provide a declarative way to manage workflows with built-in caching and dependency resolution.

## Basic Task Definition

Tasks are defined in `devenv.nix` using the `tasks` attribute:

```nix
{ pkgs, ... }:
{
  tasks."myapp:hello" = {
    exec = ''echo "Hello, world!"'';
  };
}
```

Running tasks uses the command syntax:
```bash
devenv tasks run myapp:hello
```

## Namespace Support

Tasks use colon-separated naming conventions (e.g., `myapp:hello`). You can execute all tasks within a namespace by providing just the prefix:

```bash
devenv tasks run myapp
```

This runs all tasks matching that namespace prefix in parallel.

## Lifecycle Hooks

Two built-in lifecycle events allow tasks to integrate into devenv's execution flow:

- **`devenv:enterShell`**: Executes before entering the shell and before processes start
- **`devenv:enterTest`**: Runs before test execution; automatically depends on `devenv:enterShell`

Tasks hook into these events using the `before` attribute:

```nix
tasks = {
  "bash:hello" = {
    exec = "echo 'Hello world from bash!'";
    before = [ "devenv:enterShell" ];
  };
};
```

## Conditional Execution

### Status Command

Tasks can define a `status` command that acts as a guard. If the status command returns `0`, the exec is skipped:

```nix
tasks."myapp:migrations" = {
  exec = "db-migrate";
  status = "db-needs-migrations";
};
```

### File Modification Tracking

The `execIfModified` attribute monitors specific files and runs the task only when those files change. It supports glob patterns:

```nix
tasks."myapp:build" = {
  exec = "npm run build";
  execIfModified = [
    "src/**/*.ts"
    "*.json"
    "package.json"
    "src"
  ];
  cwd = "./frontend";
};
```

The system tracks both timestamps and content hashes to detect actual changes.

## Caching and Output Preservation

When tasks are skipped (via status checks or unchanged files), outputs from the most recent successful run are cached and passed to dependent tasks. This ensures dependent tasks receive consistent data even when parent tasks don't execute.

## Language Support

Tasks can specify a package for execution:

```nix
tasks."python:hello" = {
  exec = ''
    print("Hello world from Python!")
  '';
  package = config.languages.python.package;
};
```

## Data Flow: Inputs and Outputs

Tasks support JSON-based input/output passing through environment variables:

- **`$DEVENV_TASK_INPUT`**: JSON object containing task inputs
- **`$DEVENV_TASKS_OUTPUTS`**: JSON object with outputs from dependent tasks
- **`$DEVENV_TASK_OUTPUT_FILE`**: Writable file for task outputs in JSON format
- **`$DEVENV_TASK_EXPORTS_FILE`**: File for exporting environment variables to dependents

Example task with inputs/outputs:

```nix
tasks."myapp:mytask" = {
  exec = ''
    echo $DEVENV_TASK_INPUT > $DEVENV_ROOT/input.json
    echo '{ "output": 1 }' > $DEVENV_TASK_OUTPUT_FILE
    echo $DEVENV_TASKS_OUTPUTS > $DEVENV_ROOT/outputs.json
  '';
  input = {
    value = 1;
  };
};
```

### CLI Input Override

Tasks support input overrides from the command line:

```bash
devenv tasks run myapp:mytask --input value=42 --input name=hello
devenv tasks run myapp:mytask --input-json '{"value": 42, "name": "hello"}'
```

Values are auto-parsed as JSON when valid, otherwise treated as strings.

### Shell Messages

Tasks can display messages when entering the shell by writing to `$DEVENV_TASK_OUTPUT_FILE`:

```nix
tasks."myapp:info" = {
  exec = ''
    echo '{"devenv":{"messages":["Setup complete. Dashboard: http://localhost:3000"]}}' > "$DEVENV_TASK_OUTPUT_FILE"
  '';
  before = [ "devenv:enterShell" ];
};
```

## Process Integration

All processes defined in the `processes` attribute automatically become available as tasks with the `devenv:processes:` prefix, enabling dependencies between tasks and processes:

```nix
{
  processes.web-server = {
    exec = "python -m http.server 8080";
  };

  tasks."app:setup-data" = {
    exec = "echo 'Setting up data...'";
    before = [ "devenv:processes:web-server" ];
  };
}
```

Tasks can also run after processes complete using the `after` attribute for cleanup operations.

## Git Integration

Tasks can reference the git repository root path using `${config.git.root}`, particularly useful in monorepos:

```nix
{
  tasks."build:frontend" = {
    exec = "npm run build";
    cwd = "${config.git.root}/frontend";
  };
}
```

## Task Server Protocol

A proposal exists for defining tasks in alternative languages through the Task Server Protocol, enabling broader language integration beyond Nix.
