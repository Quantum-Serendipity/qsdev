# devenv Integration with Codespaces/devcontainer

- **Source URL**: https://devenv.sh/integrations/codespaces-devcontainer/
- **Retrieval Date**: 2026-05-14

## Configuration Method

To enable devcontainer support, you add a simple toggle to your `devenv.nix` file:

```nix
{ pkgs, ... }:
{
    devcontainer.enable = true;
}
```

## What Gets Generated

When you run `devenv shell` after enabling this option, the tool automatically creates a `.devcontainer.json` file. This file is auto-generated and ready for use with GitHub Codespaces.

## Integration Workflow

The process is straightforward:

1. Enable the devcontainer option in your configuration
2. Execute `devenv shell` to trigger file generation
3. Commit the resulting `.devcontainer.json` to your Git repository
4. Push to GitHub to activate Codespaces support

## Integration Mechanism

The integration functions by having devenv generate the necessary devcontainer specification file, eliminating the need to manually write this configuration. This allows developers to leverage devenv's declarative environment setup within GitHub's cloud development environment.

The documentation emphasizes simplicity -- just "flip a toggle" and the system handles containerization details automatically.
