# devenv.sh Go Language Configuration
- **Source**: https://devenv.sh/supported-languages/go/
- **Retrieved**: 2026-05-12

## Core Option

**languages.go.enable** - "Whether to enable tools for Go development." Type: boolean, Default: false

## Go Package Selection

**languages.go.package** - "The Go package to use." Type: package, Default: pkgs.go.

**languages.go.version** - "The Go version to use. This automatically sets the `languages.go.package` using go-overlay." Type: null or string, Default: null. Example value: "1.22.0"

## Debugging Support

**languages.go.delve.enable** - "Whether to enable Delve debugger." Type: boolean, Default: true

**languages.go.delve.package** - "The Delve package to use." Type: package, Default: pkgs.delve

**languages.go.enableHardeningWorkaround** - "Enable hardening workaround required for Delve debugger." Type: boolean, Default: false.

## Language Server Protocol

**languages.go.lsp.enable** - "Whether to enable Go Language Server." Type: boolean, Default: true

**languages.go.lsp.package** - "The Go language server package to use." Type: package, Default: pkgs.gopls

All options are declared within the go.nix module located in the devenv GitHub repository's languages directory.
