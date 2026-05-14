# devenv.sh Helm Language Module Documentation

- **Source**: https://devenv.sh/languages/helm/
- **Retrieved**: 2026-05-14

## Core Configuration Options

### Enable Helm Support
`languages.helm.enable` - A boolean flag (default: `false`) that activates Helm development tools. Set to `true` to include Helm in your environment.

### Package Selection
`languages.helm.package` - Specifies which Helm distribution to use. The default is `pkgs.kubernetes-helm`, allowing customization of the Helm binary.

### Language Server Protocol Support

**Enable LSP:** `languages.helm.lsp.enable` is enabled by default (`true`), providing IDE integration for Helm chart development.

**LSP Package:** `languages.helm.lsp.package` defaults to `pkgs.helm-ls`, the Helm language server implementation.

### Plugin Management
`languages.helm.plugins` - A list of plugin names from `pkgs.kubernetes-helmPlugins`. These plugins are symlinked into a single directory and exposed via the `HELM_PLUGINS` environment variable.

**Supported plugins include:**
- helm-secrets
- helm-diff
- helm-unittest

## Example Configuration

```nix
languages.helm = {
  enable = true;
  lsp.enable = true;
  plugins = ["helm-secrets" "helm-diff" "helm-unittest"];
};
```
