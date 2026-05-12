<!-- Source: https://pkg.go.dev/fastcat.org/go/gdev/addons -->
<!-- Retrieved: 2026-05-12 -->
<!-- Note: Content may be AI-summarized by the WebFetch tool; verify against raw source if critical -->

# addons Package Documentation

## Overview
The `addons` package is part of `fastcat.org/go/gdev` (v0.14.0) and provides a framework for customizing the `xdev` tool through built-in addons. The package is licensed under Apache-2.0 and is designed to be permissive enough that users can fork and customize addons as needed.

## Functions

### func Initialize()
```go
func Initialize()
```
Initializes the addon system.

### func Register[T any](a *Addon[T])
```go
func Register[T any](a *Addon[T])
```
Registers an addon with the system.

## Types

### type Addon[T any]
```go
type Addon[T any] struct {
    Config     T
    Definition Definition
    // contains filtered or unexported fields
}
```

#### Methods

**func (a *Addon) CheckInitialized()**
Checks that the addon has been initialized and therefore it is safe to assume that its configuration is final and that it has registered anything it needs with the rest of the system. If not, it panics. Includes a call to `instance.CheckLockedDown`.

**func (a *Addon) CheckNotInitialized()**
Checks that the addon has not been initialized yet and therefore it is safe to apply customizations/configuration changes to it. If not, it panics. Includes a call to `instance.CheckCanCustomize`.

**func (a *Addon[T]) RegisterIfNeeded()**
Conditionally registers the addon if it hasn't been registered yet.

### type Definition
```go
type Definition struct {
    Name        string
    Description func() string
    Initialize  func() error
}
```
Defines the structure and initialization behavior of an addon.

### type Description
```go
type Description struct {
    Name        string
    Description string
}
```

**func Enabled() []Description**
Returns a slice of descriptions for all enabled addons.

## Package Details

- **Version:** v0.14.0
- **Published:** Mar 30, 2026
- **License:** Apache-2.0 (Redistributable)
- **Valid go.mod:** Yes
- **Imports:** 8 packages
- **Imported by:** 12 packages
- **Repository:** github.com/fastcat/gdev
- **Stability:** Not yet at major version v1 (considered pre-stable)
