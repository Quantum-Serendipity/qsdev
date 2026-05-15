<!-- Source: https://pkg.go.dev/debug/buildinfo -->
<!-- Retrieved: 2026-05-15 -->

# debug/buildinfo Package Details

## Overview
The `debug/buildinfo` package provides access to information embedded in Go binaries about how they were built, including the Go toolchain version and module dependencies (for binaries built in module mode).

## Key Components

### Type: BuildInfo
```go
type BuildInfo = debug.BuildInfo
```
- A type alias for `runtime/debug.BuildInfo`

### Functions

#### ReadFile(name string)
```go
func ReadFile(name string) (info *BuildInfo, err error)
```
- Returns build information embedded in a Go binary file at a given path
- Most information only available for binaries built with module support

#### Read(r io.ReaderAt)
```go
func Read(r io.ReaderAt) (*BuildInfo, error)
```
- Returns build information from a Go binary accessed through a ReaderAt interface

## Available Data (BuildInfo fields)
- **GoVersion** - The Go toolchain version used
- **Path** - Module path information
- **Main** - Main module details
- **Deps** - Module dependencies (path, version, hash)
- **Settings** - Build configuration settings (GOARCH, GOOS, CGO_ENABLED, vcs info, ldflags, etc.)

## Limitations
- Most detailed information (modules, settings) only available for binaries built with **module mode enabled**
- Legacy binaries without module support have limited information available

## Package Status
- **Version**: go1.26.3
- **License**: BSD-3-Clause
