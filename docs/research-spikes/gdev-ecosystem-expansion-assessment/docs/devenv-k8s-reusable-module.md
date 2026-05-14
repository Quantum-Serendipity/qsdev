# devenv-k8s: Reusable devenv with Kubernetes tools

- **Source**: https://github.com/LCOGT/devenv-k8s
- **Retrieved**: 2026-05-14

## Content

Repository structure indicates these key files exist:
- `flake.nix` - Main Nix flake configuration
- `flake-module.nix` - Modular flake setup
- `deploy.nix` - Deployment configuration
- `skaffold-builder.nix` - Skaffold integration

## Import Pattern

The documentation demonstrates importing this as a flake module:

Add the following to your `flake.nix`:
```nix
inputs = {
  ...
  devenv-k8s.url = "github:LCOGT/devenv-k8s/v1";
}
```

## Language Composition

- **Nix: 91.1%**
- **Shell: 8.9%**

Note: The GitHub landing page doesn't display the actual package contents or configuration details. The source files would need to be viewed directly for specific K8s tool lists.
