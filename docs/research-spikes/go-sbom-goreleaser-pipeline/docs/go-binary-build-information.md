# Getting Build Information from Go Binaries

- **Source**: https://appliedgo.net/spotlight/get-build-information-of-a-go-binary/
- **Retrieved**: 2026-05-15

## The go version -m Command

"If you pass the `-m` flag along with the path to a Go binary, `go version` lists the Go version used to build the binary, the dependencies, and the build flags used."

## What Information Is Displayed

Running this command reveals:
- The Go version that compiled the binary
- Module path and version information
- All dependencies with their versions and hashes
- Build flags (such as buildmode, compiler type, and environment variables like GOARCH and GOOS)

## Example Output

The article demonstrated the command with `go version -m $(which gopls)`, showing output that included dependencies like google/go-cmp, golang.org/x/tools, and build settings like CGO_ENABLED=0 and GOARCH=arm64.

## Programmatic Access

Build information can be accessed programmatically via the `runtime/debug` package's `ReadBuildInfo()` function.

The BuildInfo struct includes:
- GoVersion: the version of the Go toolchain that built the binary
- Path: the package path of the main package
- Main: describes the module that contains the main package
- Deps: describes all the dependency modules
- Settings: describes the build settings used to build the binary

## Build Info Storage

The build info structure is available as a separate section in ELF and Mach-O files. The section name is `.go.buildinfo`.
