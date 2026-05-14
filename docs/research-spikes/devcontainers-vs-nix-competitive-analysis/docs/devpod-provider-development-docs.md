# DevPod — Provider Development & Agent Documentation
- **Sources**:
  - https://devpod.sh/docs/developing-providers/quickstart (raw GitHub)
  - https://devpod.sh/docs/developing-providers/agent (raw GitHub)
- **Retrieved**: 2026-03-20

## Provider Development Quickstart

DevPod providers are small CLI programs defined through a `provider.yaml` that DevPod interacts with to bring up workspaces. Providers are standalone programs parsed through a manifest called `provider.yaml`.

### Minimal Provider Example

```yaml
name: my-first-provider
version: v0.0.1
agent:
  path: ${DEVPOD}
exec:
  command: |-
    sh -c "${COMMAND}"
```

### Provider.yaml Key Sections

- **exec**: Defines what commands DevPod should execute to interact with the environment
- **options**: Defines user-configurable options
- **binaries**: Defines additional helper binaries needed
- **agent**: Defines agent configuration (drivers, auto-inactivity, credentials injection)

### Exec Commands

- **command** (required): How to run a command in the environment. DevPod uses this to inject itself and route communication through STDIO.
- **init** (optional): Validate options, check prerequisites
- **create** (optional): Create a machine (makes it a machine provider)
- **delete** (optional): Delete a machine
- **start** (optional): Start a stopped machine
- **stop** (optional): Stop a machine
- **status** (optional): Get machine status (Running, Busy, Stopped, NotFound)

### Non-Machine vs Machine Providers

- Non-machine providers: only `command` and optionally `init`
- Machine providers: all commands available for full VM lifecycle management

## Provider Agent

When DevPod connects through a Provider to an environment, it injects itself into the environment to handle:
- Deploying the container
- Forwarding credentials
- SSH server
- Auto-shutdown after a period of inactivity

### Agent Configuration

```yaml
agent:
  path: ${DEVPOD}
  driver: docker  # default: docker
  inactivityTimeout: 10m
  containerInactivityTimeout: 10m
  injectGitCredentials: ${INJECT_GIT_CREDENTIALS}
  injectDockerCredentials: ${INJECT_DOCKER_CREDENTIALS}
  binaries:
    MY_BINARY:
      - os: linux
        arch: amd64
        path: https://url-to-binary.com
        checksum: shasum-of-binary
  exec:
    shutdown: |-
      ${MY_BINARY} stop
```

### Auto-Inactivity Stop

**Non-Machine Providers**: DevPod can automatically kill the container by terminating the process with pid 1. The timeout is configured through `agent.containerInactivityTimeout`. DevPod starts a process within the container to track activity and kills itself when the user hasn't connected for the given duration. This does not erase state — it only stops the container.

**Machine Providers**: Killing just the container is not enough since VMs still generate costs. DevPod provides `agent.exec.shutdown` to shut down or delete unused machines. DevPod automatically tracks when a user is connected.

### Auto-Shutdown Examples from Official Providers

- **Azure**: Uses `shutdown -t now` as `agent.exec.shutdown`
- **AWS**: Generates temporary token from local `aws` CLI, uses it to shutdown via AWS API
- **GCloud**: Generates temporary token from local `gcloud` CLI, uses it to shutdown via Google Cloud API
- **DigitalOcean**: Deletes entire machine on inactivity (stopped machines are still billed), preserves state in extra volume
