---
source: https://raw.githubusercontent.com/devcontainers/spec/main/docs/specs/devcontainerjson-reference.md
retrieved: 2026-03-20
type: specification-reference
---

# Complete devcontainer.json Reference

## General Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `name` | string | unset | "A name for the dev container displayed in the UI" |
| `forwardPorts` | array | `[]` | Port numbers or host:port pairs to forward from container to local machine |
| `portsAttributes` | object | unset | Maps ports to default options; example: `{"3000": {"label": "Application port"}}` |
| `otherPortsAttributes` | object | unset | Default options for unmapped ports and ranges |
| `containerEnv` | object | unset | Environment variables applied to the container itself; static for container lifetime |
| `remoteEnv` | object | unset | Environment variables for tooling/subprocesses; updatable without rebuild |
| `remoteUser` | string | container default | Overrides user for tooling operations; see containerUser for container-wide setting |
| `containerUser` | string | `root` or final Dockerfile USER | User for all container operations |
| `updateRemoteUserUID` | boolean | `true` | On Linux, syncs user UID/GID with host to prevent permission issues |
| `userEnvProbe` | enum | `loginInteractiveShell` | Shell type for probing environment: `none`, `interactiveShell`, `loginShell`, `loginInteractiveShell` |
| `overrideCommand` | boolean | `true` (image) / `false` (compose) | Override default command with sleep loop to prevent shutdown |
| `shutdownAction` | enum | `stopContainer` (image) / `stopCompose` (compose) | Shutdown behavior: `none`, `stopContainer`, `stopCompose` |
| `init` | boolean | `false` | Enable tini init process for zombie process handling |
| `privileged` | boolean | `false` | Run container in privileged mode (security implications) |
| `capAdd` | array | `[]` | Add Linux capabilities; "most often used to add the `ptrace` capability" |
| `securityOpt` | array | `[]` | Set container security options; example: `["seccomp=unconfined"]` |
| `mounts` | string/object | unset | Additional mounts using Docker CLI mount flag syntax |
| `features` | object | unset | Dev Container Feature IDs and options to add |
| `overrideFeatureInstallOrder` | array | unset | Override automatic Feature installation ordering |
| `customizations` | object | unset | Product-specific properties per supporting tool documentation |

## Image or Dockerfile Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `image` | string | **required** | Container registry image name for dev container creation |
| `build.dockerfile` | string | **required** | Relative path to Dockerfile defining container contents |
| `build.context` | string | `"."` | Docker build directory relative to devcontainer.json |
| `build.args` | object | unset | Docker build-time argument key-value pairs |
| `build.options` | array | `[]` | Docker build command options |
| `build.target` | string | unset | Dockerfile build stage target to build |
| `build.cacheFrom` | string/array | unset | Image(s) to use as cache sources for build |
| `appPort` | integer/string/array | `[]` | Ports to publish (deprecated; use forwardPorts) |
| `workspaceMount` | string | auto | Custom local mount point; "Supports the same values as the Docker CLI" |
| `workspaceFolder` | string | auto | Default path when connecting to container |
| `runArgs` | array | `[]` | Docker CLI arguments applied when running container |

## Docker Compose Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `dockerComposeFile` | string/array | **required** | Path(s) to Docker Compose files relative to devcontainer.json |
| `service` | string | **required** | Service name to connect to after launching |
| `runServices` | array | all | Services to start; defaults to all services |
| `workspaceFolder` | string | `"/"` | Default container path for opening workspace |

## Lifecycle Scripts

| Property | Type | Execution Context | Description |
|----------|------|-------------------|-------------|
| `initializeCommand` | string/array/object | host machine | Runs during initialization; may execute multiple times |
| `onCreateCommand` | string/array/object | inside container | First setup command after initial container start |
| `updateContentCommand` | string/array/object | inside container | Second setup command when new content available |
| `postCreateCommand` | string/array/object | inside container | Final setup command after user assignment |
| `postStartCommand` | string/array/object | inside container | Runs each time container successfully starts |
| `postAttachCommand` | string/array/object | inside container | Runs each time tool attaches to container |
| `waitFor` | enum | n/a | Specifies which command to wait for; default is `updateContentCommand` |

**Note:** String format uses `/bin/sh`; array format executes without shell. Failure stops subsequent scripts.

## Host Requirements Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `hostRequirements.cpus` | integer | unset | Minimum required CPU cores; example: `{"cpus": 2}` |
| `hostRequirements.memory` | string | unset | Minimum RAM with suffix (tb/gb/mb/kb); example: `"4gb"` |
| `hostRequirements.storage` | string | unset | Minimum storage with suffix; example: `"32gb"` |

## Port Attributes Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `label` | string | unset | Display name in ports view |
| `protocol` | enum | unset | Protocol handling: `http`, `https` for web forwarding |
| `onAutoForward` | enum | `notify` | Auto-forward behavior: `notify`, `openBrowser`, `openBrowserOnce`, `openPreview`, `silent`, `ignore` |
| `requireLocalPort` | boolean | `false` | Require same port locally or notify if unavailable |
| `elevateIfNeeded` | boolean | `false` | Auto-elevate permissions for low ports (22, 80, 443) |

## Available Variables

| Variable | Scope | Description |
|----------|-------|-------------|
| `${localEnv:VARIABLE_NAME}` | any | Host environment variable value |
| `${containerEnv:VARIABLE_NAME}` | remoteEnv only | Running container environment variable |
| `${localWorkspaceFolder}` | any | Local folder path containing devcontainer.json |
| `${containerWorkspaceFolder}` | any | Container workspace path |
| `${localWorkspaceFolderBasename}` | any | Local folder name |
| `${containerWorkspaceFolderBasename}` | any | Container workspace folder name |
| `${devcontainerId}` | specific | Unique, rebuild-stable dev container identifier |
