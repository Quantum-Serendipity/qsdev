# devenv.sh Tasks
- **Source**: https://devenv.sh/tasks/
- **Retrieved**: 2026-05-12

## Core Functionality

Tasks in devenv (introduced in v1.2) enable dependency-based code execution with parallel processing. They follow a directed acyclic graph (DAG) model for managing task relationships.

## Basic Task Definition

Tasks are defined within `devenv.nix` using the `tasks` attribute:

```nix
{ pkgs, ... }:
{
  tasks."myapp:hello" = {
    exec = ''echo "Hello, world!"'';
  };
}
```

Execution via CLI: `devenv tasks run myapp:hello`

## Namespace Execution (v1.7+)

Running a namespace prefix executes all matching tasks: `devenv tasks run myapp` will run `myapp:hello`, `myapp:build`, and `myapp:test` in parallel where possible.

## Lifecycle Events

Two built-in lifecycle tasks control execution timing:

- **`devenv:enterShell`**: Runs before interactive shell entry and process startup
- **`devenv:enterTest`**: Runs before test execution (depends on `devenv:enterShell`)

Tasks hook into these using the `before` attribute:

```nix
tasks."bash:hello" = {
  exec = "echo 'Hello world from bash!'";
  before = [ "devenv:enterShell" ];
};
```

## Language-Specific Execution

Tasks can execute code in different languages by specifying a `package`:

```nix
tasks."python:hello" = {
  exec = ''print("Hello world from Python!")'';
  package = config.languages.python.package;
};
```

## Conditional Execution with Status Checks

The `status` command prevents expensive operations from re-running unnecessarily:

```nix
tasks."myapp:migrations" = {
  exec = "db-migrate";
  status = "db-needs-migrations";
};
```

When `status` returns `0`, `exec` is skipped. Outputs from the most recent successful run are cached and passed to dependent tasks.

## File-Based Conditional Execution

The `execIfModified` attribute monitors specific files and only runs tasks when those files change:

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

The system detects changes via modification times and content hashes. Tasks are skipped if files haven't genuinely changed, with previous outputs preserved for dependent tasks.

## Input/Output Handling

Tasks exchange data through JSON and environment variables:

- **`$DEVENV_TASK_INPUT`**: JSON object containing task inputs
- **`$DEVENV_TASKS_OUTPUTS`**: JSON from dependent tasks
- **`$DEVENV_TASK_OUTPUT_FILE`**: Writable file for task output JSON
- **`$DEVENV_TASK_EXPORTS_FILE`**: File for exporting environment variables

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

## Shell Messages (v2.1+)

Tasks can display messages when entering the shell by writing to `$DEVENV_TASK_OUTPUT_FILE`:

```nix
tasks."myapp:info" = {
  exec = ''echo '{"devenv":{"messages":["Setup complete. Dashboard: http://localhost:3000"]}}' > "$DEVENV_TASK_OUTPUT_FILE"'';
  before = [ "devenv:enterShell" ];
};
```

## CLI Input Passing (v2.0+)

Override or add inputs from the command line:

```bash
devenv tasks run myapp:mytask --input value=42 --input name=hello
devenv tasks run myapp:mytask --input-json '{"value": 42, "name": "hello"}'
```

Values are automatically parsed as JSON when valid; otherwise they're treated as strings.

## Processes as Tasks (v1.4+)

All defined processes become automatically available as tasks with the `devenv:processes:` prefix:

```nix
processes.web-server = {
  exec = "python -m http.server 8080";
};

tasks."app:setup-data" = {
  exec = "echo 'Setting up data...'";
  before = [ "devenv:processes:web-server" ];
};
```

Tasks can run after process completion using the `after` attribute for cleanup operations.

## Git Integration (v1.10+)

Tasks reference the git repository root via `${config.git.root}`, useful for monorepo configurations:

```nix
tasks."build:frontend" = {
  exec = "npm run build";
  cwd = "${config.git.root}/frontend";
};
```

## Task Server Protocol

A proposed SDK allows defining tasks in languages beyond Nix, referenced in the project's issue tracking system.
