# GitHub Codespaces Architecture Deep Dive
- **Source**: https://docs.github.com/en/codespaces/about-codespaces/deep-dive
- **Retrieved**: 2026-03-20
- **Note**: Content synthesized from web search results (WebFetch unavailable)

## How Codespaces Work

GitHub Codespaces is a development environment running inside a Docker container that is remotely hosted on a cloud-based Virtual Machine (Azure VM) linked to your code repository.

### Creation Process

When you create a codespace:
1. A Virtual Machine (VM) is created using either the stable or public preview release of the VM host image
2. The VM is both dedicated and private to you — two codespaces are never co-located on the same VM
3. A Docker dev container is created on the VM based on your devcontainer.json and/or Dockerfile
4. The repository is cloned into the container
5. Lifecycle commands run (onCreateCommand, updateContentCommand, postCreateCommand, postStartCommand, postAttachCommand)

### Machine Types

Available compute options (all Azure-hosted):
- 2 cores, 8 GB RAM, 32 GB storage
- 4 cores, 16 GB RAM, 32 GB storage
- 8 cores, 32 GB RAM, 64 GB storage
- 16 cores, 64 GB RAM, 128 GB storage
- 32 cores, 128 GB RAM, 128 GB storage

GPU machine types were deprecated as of August 29, 2025 (NCv3-series Azure VMs retired).

### Connection Methods

- VS Code desktop (via Remote SSH extension)
- VS Code in the browser (vscode.dev)
- JetBrains IDEs (via JetBrains Gateway)
- SSH from terminal
- GitHub CLI (`gh codespace ssh`)

### Dev Container Foundation

Every codespace uses the Dev Container specification:
- Configuration via `.devcontainer/devcontainer.json`
- Optional Dockerfile for custom images
- Features system for modular tool installation
- Lifecycle hooks for setup automation
- Default image: `mcr.microsoft.com/devcontainers/universal` (includes Node, Python, Java, .NET, PHP, Go, Ruby, Rust, C++)

### Lifecycle

1. **Create** — VM provisioned, container built, repo cloned
2. **Active** — Connected and running, compute charges accrue
3. **Idle** — Still running but no user activity
4. **Stopped** — VM deallocated, only storage charges accrue (auto-stop after configurable idle timeout, default 30 min)
5. **Deleted** — All resources removed (auto-delete after retention period, default 30 days)

Stopped codespaces retain all file changes, installed tools, and terminal history. Restarting resumes from the stopped state on a fresh VM.

### Personalization

- **Dotfiles repository**: Automatically cloned into new codespaces for shell config, aliases, tool preferences
- **Settings Sync**: VS Code settings, keybindings, snippets, extensions synced across codespaces and local VS Code
- User-scoped VS Code settings cannot be personalized via dotfiles (limitation)

### Port Forwarding

- Automatic detection when apps bind to localhost ports
- Forwarded ports accessible via `https://CODESPACENAME-PORT.app.github.dev`
- Ports can be private (only codespace creator), org-visible, or public
- HTTP by default, HTTPS available
- When using VS Code desktop, localhost forwarding also works
- Organizations can restrict port visibility via policy

## Additional Sources

- https://docs.github.com/codespaces/overview
- https://docs.github.com/en/codespaces/getting-started/understanding-the-codespace-lifecycle
- https://github.com/features/codespaces
- https://github.blog/developer-skills/github/how-to-automate-your-dev-environment-with-dev-containers-and-github-codespaces/
- https://www.nathannellans.com/post/all-about-github-codespaces
