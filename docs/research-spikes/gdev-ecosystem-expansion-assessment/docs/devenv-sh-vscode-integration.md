# devenv.sh VS Code Integration
- **Source**: https://devenv.sh/editor-support/vscode/
- **Retrieved**: 2026-05-14

## Official Extension

Primary integration mechanism is a dedicated extension: **devenv extension for VS Code**, available on the Visual Studio Marketplace (datakurre.devenv).

## Configuration File Generation

The documentation does not specify whether devenv automatically generates VS Code configuration files or workspace settings. The approach is minimal — install the extension, and it handles the integration.

## IDE Configuration Approach

Minimal configuration philosophy — users are directed to install the extension rather than receiving detailed setup instructions. The extension handles integration rather than manual configuration steps.

## DevContainer Generation

Cachix's devenv now supports automatically generating a .devcontainer.json file, which gives a more convenient and consistent way to use Nix with any Dev Container Spec supporting tool or service.
