# devenv Codespaces/devcontainer Integration
- **Source**: https://devenv.sh/integrations/codespaces-devcontainer/
- **Retrieved**: 2026-05-12

## Setup Process

The integration is remarkably straightforward. You enable it by setting a single configuration option in your `devenv.nix` file:

```nix
{ pkgs, ... }:

{
    devcontainer.enable = true;
}
```

After enabling this setting, run `devenv shell` and the tooling will automatically generate a `.devcontainer.json` file for you.

## Deployment

The generated configuration file should then be committed to your repository and pushed. This allows GitHub Codespaces to automatically detect and use your development environment setup.

## Documentation Limitations

The provided documentation is quite minimal on this topic. It covers only the basic activation steps but doesn't detail:

- What specific configurations are included in the auto-generated `.devcontainer.json`
- How Nix operates within the containerized environment
- What isolation mechanisms are provided
- Advanced customization options for the devcontainer setup

For more comprehensive information about advanced devcontainer configuration, container behavior, and Nix integration specifics, you would likely need to consult additional resources or the project's GitHub repository directly.
