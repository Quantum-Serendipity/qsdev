---
source: multiple web searches
retrieved: 2026-03-20
type: web-search-synthesis
sources:
  - https://containers.dev/supporting
  - https://medium.com/cloudnativepub/dev-containers-vs-code-vs-jetbrains-ides-207556d81bfe
  - https://www.jetbrains.com/help/idea/connect-to-devcontainer.html
  - https://www.jetbrains.com/help/idea/dev-container-limitations.html
  - https://codeberg.org/esensar/nvim-dev-container
  - https://github.com/erichlf/devcontainer-cli.nvim
  - https://github.com/debdutdeb/devcontainer.nvim
  - https://devpod.sh/docs/getting-started/quickstart-vim
---

# Dev Container IDE Support Beyond VS Code

## VS Code (Reference Implementation)
- Full-featured support via the "Dev Containers" extension (formerly "Remote - Containers")
- Most complete implementation of the spec
- Automatic Features support, lifecycle hooks, Docker Compose integration
- Port forwarding, settings sync, extensions inside container

## JetBrains IDEs (IntelliJ IDEA, GoLand, etc.)

### Supported
- Basic devcontainer.json support for creating/connecting to dev containers
- Docker and Docker Compose-based containers
- Features support
- Lifecycle hooks

### Limitations (as of 2025)
- Code completion for devcontainer.json is limited — only `label` supported in port attributes
- Minimal host requirements (`hostRequirements`) not supported
- Windows-based dev container images not supported
- Cannot create a Dev Container from a running backend connection (e.g., SSH remote)
- User experience described as cumbersome compared to VS Code — VS Code is "Rebuild and Reopen in Container" while JetBrains asks to specify a Git repo
- General sentiment: "getting closer but still feels cumbersome and prone to failure"

## Neovim

### Available Plugins
1. **nvim-dev-container** (esensar) — requires NeoVim 0.12.0+, commands: DevcontainerStart, DevcontainerAttach, DevcontainerExec, DevcontainerStop
2. **devcontainer-cli.nvim** (erichlf) — wraps the devcontainer CLI, supports :DevcontainerExec and :DevcontainerConnect
3. **devcontainer.nvim** (debdutdeb) — fork of nvim-dev-container

### Limitations
- Community-maintained, not official
- Author of nvim-dev-container notes they "haven't been using the plugin much lately"
- Some "difficult issues persisting since the first version"
- Less mature than VS Code's integration

## DevPod
- Editor-agnostic open-source tool from Loft Labs
- Supports VS Code, JetBrains, and Neovim/Vim
- Provider model for local/cloud flexibility
- Wraps the devcontainer spec with additional orchestration

## Other Supporting Tools (from containers.dev/supporting)
- **GitHub Codespaces** — cloud-hosted dev containers
- **DevPod** — open-source, editor-agnostic
- **CodeSandbox** — browser-based
- **Coder** — self-hosted remote development
- **DevZero** — cloud dev environments
- **Daytona** — self-hosted/cloud dev environment manager
