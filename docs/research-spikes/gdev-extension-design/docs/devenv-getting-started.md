# devenv.sh Getting Started Guide
- **Source**: https://devenv.sh/getting-started/
- **Retrieved**: 2026-05-12

## Installation

### Step 1: Install Nix

The documentation provides platform-specific instructions:

- **Linux**: `sh <(curl -L https://nixos.org/nix/install) --daemon`
- **macOS**: Uses the Nix installer with `curl -sSfL https://artifacts.nixos.org/nix-installer | sh -s -- install`
- **Windows (WSL2)**: `sh <(curl -L https://nixos.org/nix/install)`
- **Docker**: `docker run -it nixos/nix`

The guide recommends the newer Nix installer over the classic version. For macOS users, upgrading Bash is advised since the system ships with an older version. Two installation methods are provided: `nix-env` for newcomers or Nix profiles for experienced users.

### Step 2: Install devenv

Installation varies by user type:

- **Newcomers**: `nix-env --install --attr devenv -f https://github.com/NixOS/nixpkgs/tarball/nixpkgs-unstable`
- **Nix profiles users**: `nix profile install nixpkgs#devenv`
- **NixOS/nix-darwin users**: Add `pkgs.devenv` to `configuration.nix` or `home.nix`

### Step 3: Configure GitHub Access Token (Optional)

To prevent rate-limiting from GitHub API calls, users should create a token at the GitHub settings page with no extra permissions and add it to `~/.config/nix/nix.conf` with the format: `access-tokens = github.com=<GITHUB_TOKEN>`

## Initial Setup

Run `devenv init` to scaffold a new project. This command creates three files:
- `devenv.nix`
- `devenv.yaml`
- `.gitignore`

## Available Commands

The guide lists these primary commands:

- `devenv shell` — activates the developer environment
- `devenv test` — builds the environment and validates checks
- `devenv search <NAME>` — finds packages in Nixpkgs
- `devenv update` — pins inputs into `devenv.lock`
- `devenv gc` — removes unused environments
- `devenv up` — starts processes
- `devenv processes down` — halts background processes
- `devenv info` — displays environment details
- `devenv build <attr>` — builds specified attributes
- `devenv eval <attr>` — evaluates attributes as JSON
- `devenv repl` — launches interactive Nix REPL
- `devenv tasks run <task>` — executes tasks
- `devenv container build|copy|run` — manages containers
- `devenv inputs add <name> <url>` — adds inputs to configuration
- `devenv lsp` — starts language server
- `devenv mcp` — launches MCP server for AI assistants

## Updating

**Updating devenv CLI**: Use `nix-env --upgrade` for newcomers or `nix profile upgrade devenv` for profile-based installations.

**Updating project inputs**: Run `devenv update` to refresh pinned dependencies stored in `devenv.lock`.

## Additional Resources

The guide directs readers to documentation sections on inputs, composing configurations, and writing devenv.nix files for deeper learning.
