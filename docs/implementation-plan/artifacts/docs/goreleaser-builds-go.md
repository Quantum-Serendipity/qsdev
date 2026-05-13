<!-- Source: https://goreleaser.com/customization/builds/go/ -->
<!-- Retrieved: 2026-05-12 -->

# GoReleaser Go Build Configuration

GoReleaser's Go builder supports comprehensive cross-compilation configuration. The primary build section accepts multiple build definitions as a YAML list, each with a unique `id`.

## Essential Configuration Fields

- `main`: Path to main.go or package (default: `.`). Supports `./...` to auto-discover.
- `binary`: Output binary name
- `dir`: Working directory (default: `.`)
- `flags`: Custom Go build flags
- `ldflags`: Linker flags with template support
- `env`: Custom environment variables

## Cross-Compilation Targets

- `goos`: Operating systems (default: darwin, linux, windows)
- `goarch`: Architectures (default: 386, amd64, arm64)
- `goarm`: ARM version (default: 6)
- `goamd64`: AMD64 level (default: v1)
- `ignore`: Combinations to exclude
- `targets`: Override matrix with explicit target list; `go_first_class` for latest stable first-class ports

## Template Variables

- `.Os` (GOOS), `.Arch` (GOARCH), `.Arm` (GOARM), `.Ext` (file extension), `.Target`

## Advanced Options

- `overrides`: Per-target field customization (essential for CGO)
- `hooks`: Pre/post-build commands
- `buildmode`: `c-shared` or `c-archive` for C library compilation
