# DevPod — devcontainer.json Support (Official Documentation)
- **Source**: https://devpod.sh/docs/developing-in-workspaces/devcontainer-json (fetched from raw GitHub)
- **Retrieved**: 2026-03-20

DevPod uses the open `devcontainer.json` standard (https://containers.dev/) to allow users to customize their development containers. Development containers are Docker containers that provide a user with a fully featured development environment. Within DevPod, this container is created based on the underlying provider either locally, in a remote virtual machine or even in a Kubernetes cluster. DevPod makes sure that no matter where you use this configuration the developer experience stays the same.

## Compatibility with VS Code & Codespaces

The same format is used by VS Code for their development containers and by Github for their Codespaces. This makes it easy to reuse existing configurations and tooling around this standard within DevPod.

## Unsupported Properties (as of latest release)

- userEnvProbe
- waitFor
- Parallel lifecycle scripts

## devcontainer.json Locations

- `.devcontainer/devcontainer.json`
- `.devcontainer.json`
- `.devcontainer/my-other-folder/devcontainer.json`

Custom path: `devpod up github.com/my-org/my-repo --devcontainer-path ./path/to/file.json`

## Auto-Detection

If DevPod doesn't find any configuration for the project, it will automatically detect the programming language and provide a sane default configuration.

## Features Support

DevPod supports devcontainer Features — reusable Dockerfile parts that get merged into the Dockerfile upon creation. Custom HTTP headers for feature downloads can be configured in `customizations.devpod.featureDownloadHTTPHeaders`.

## Development Flow

Changes to devcontainer.json can be applied on the fly via:
```
devpod up my-workspace --recreate
```

This applies ALL new configurations including Dockerfile changes, new mounts, new features, etc. DevPod only replaces the existing container if the command succeeds. Changes in the overlay layer (non-volumes) will be lost; mounted paths are preserved.
