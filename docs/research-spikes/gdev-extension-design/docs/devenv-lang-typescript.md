# devenv.sh TypeScript Language Configuration
- **Source**: https://devenv.sh/supported-languages/typescript/
- **Retrieved**: 2026-05-12

## Configuration Options

### languages.typescript.enable
- **Type:** boolean
- **Default:** `false`
- **Description:** "Whether to enable tools for TypeScript development."

### languages.typescript.lsp.enable
- **Type:** boolean
- **Default:** `true`
- **Description:** "Whether to enable TypeScript Language Server."

### languages.typescript.lsp.package
- **Type:** package
- **Default:** `pkgs.typescript-language-server`
- **Description:** "The TypeScript language server package to use."

Source module: https://github.com/cachix/devenv/blob/main/src/modules/languages/typescript.nix
