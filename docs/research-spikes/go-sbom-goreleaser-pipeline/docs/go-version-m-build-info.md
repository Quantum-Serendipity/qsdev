<!-- Source: https://appliedgo.net/spotlight/get-build-information-of-a-go-binary/ -->
<!-- Retrieved: 2026-05-15 -->

# Go Build Information Extraction via `go version -m`

## Command Usage
The `go version -m` flag, when passed with a binary path, retrieves embedded build metadata without executing the binary.

## Output Fields

The example output demonstrates these information categories:

**Binary Metadata:**
- Go version used for compilation (e.g., "go1.21.5")
- Module path and version hash

**Dependencies:**
Prefixed with "dep", listing all module dependencies with their versions and hashes (e.g., BurntSushi/toml v1.2.1, google/go-cmp v0.5.9)

**Build Settings:**
Prefixed with "build", capturing:
- Build mode (`-buildmode=exe`)
- Compiler type (`-compiler=gc`)
- Runtime configuration (`DefaultGODEBUG=panicnil=1`)
- Environment variables: `CGO_ENABLED`, `GOARCH`, `GOOS` (darwin, arm64 in example)

## Example Output

```
/path/to/binary: go1.21.5
	path    golang.org/x/tools/gopls
	mod     golang.org/x/tools/gopls  v0.14.2  h1:sIw6vjZiuQ9S7s0auUUkHlWgsCkKZFWDHmrge8LYsnc=
	dep     github.com/BurntSushi/toml  v1.2.1  h1:9F2/+DoOYIOksmaJFPw1tGFy1eDnIJXg+UHjuD8lTak=
	dep     github.com/google/go-cmp  v0.5.9  h1:O2Tfq5qg4qc4AmwVlvv0oLiVAGB7enBSJ2x2DqQFi38=
	build   -buildmode=exe
	build   -compiler=gc
	build   CGO_ENABLED=0
	build   GOARCH=arm64
	build   GOOS=darwin
```

## Programmatic Access

Build information is also accessible programmatically via `runtime/debug.ReadBuildInfo()` for the currently running binary, or `debug/buildinfo.ReadFile()` for external binaries.
