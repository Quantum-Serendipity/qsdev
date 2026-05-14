---
source: Multiple web searches (jetify.com/docs/devbox/configuration, memo.d.foundation)
retrieved: 2026-03-20
---

# Devbox devbox.json Configuration Reference

## Configuration Structure

Your devbox configuration is stored in a devbox.json file, located in your project's root directory.

## Main Configuration Fields

### packages
A list or map of Nix packages that should be installed in your Devbox shell and containers. These packages will only be installed and available within your shell, and will have precedence over any packages installed in your local machine.

### env
A map of key-value pairs that should be set as Environment Variables when activating devbox shell, running a script with devbox run, or starting a service.

### shell
The Shell object defines init hooks and scripts. Two fields are supported:
- **init_hook**: Run a set of commands every time you start a devbox shell
- **scripts**: Commands that can be run using `devbox run`

### include
Use include to add extra configurations:
- Pull plugins from GitHub
- Use local plugins
- Activate built-in plugins

## Full Example

```json
{
  "packages": ["rustup@latest", "libiconv@latest"],
  "env": {
    "PROJECT_DIR": "$PWD"
  },
  "shell": {
    "init_hook": [
      ". conf/set-env.sh",
      "rustup default stable",
      "cargo fetch"
    ],
    "scripts": {
      "build-docs": "cargo doc",
      "start": "cargo run",
      "run_test": ["cargo test -- --show-output"]
    }
  },
  "include": [
    "github:org/repo/ref?dir=<path-to-plugin>",
    "path:path/to/plugin.json",
    "plugin:php-config"
  ]
}
```

## Plugin System

Plugins are defined as Go JSON Template files with fields for:
- name, version, description
- env (environment variables)
- create_files (file generation)
- init_hook (initialization commands)
- Services defined via process-compose.yaml

Plugins activate when a developer runs devbox shell, runs a script with devbox run, or starts a service using devbox services start|restart.

## Services (via process-compose)

Devbox uses Process Compose to run services and background processes. Plugins can add services by including a process-compose.yaml file, which will be automatically detected by Devbox. Commands:
- `devbox services up` — start all services (add -b for background)
- `devbox services ls` — list running services
- `devbox services stop` — stop services

Services started in the background will continue running even if the current shell is closed.
