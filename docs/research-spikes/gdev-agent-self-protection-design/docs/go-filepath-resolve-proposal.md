<!-- Source: https://github.com/golang/go/issues/37113 -->
<!-- Retrieved: 2026-05-15 -->

# filepath.Resolve Proposal - Replacing EvalSymlinks

## Core Problem

Go lacks a function to canonicalize paths in a standard way. `filepath.Abs` combined with `filepath.EvalSymlinks` fails to properly canonicalize paths on Windows when UNC paths are mounted as drive letters. The reporter states: "Go lacks a function to canonicalize paths in a standard way, so it isn't possible to produce results equivalent to a C program and still write code that works portably across systems."

## What EvalSymlinks Gets Wrong

When a UNC path is mounted as a drive letter and the current working directory is switched to that drive, calling `filepath.EvalSymlinks` on an absolute path returns a path using the drive letter instead of the canonical UNC path. The expected behavior would match Windows' `GetFinalPathNameByHandle` output.

## Proposed Solution

The proposal calls for adding a new function that "is explicitly defined to canonicalize paths in a way equivalent to the underlying operating system." This would replace reliance on the combination of `filepath.Abs` and `filepath.EvalSymlinks`.

## Key Context

- The issue references a prior discussion (#17084) where those functions were deemed sufficient, but practical experience proved otherwise
- The problem originated from Git LFS project needs for path canonicalization
- While Unix paths are simpler, Windows paths require more sophisticated handling

## Known Issues with EvalSymlinks

- Issue #30520: Incorrect traversal of relative paths
- Issue #40176: Endless loop when symlink resolves to itself
- Issue #29449: Fails when target is a file (not directory)
- Issue #23512: Fails with container-mapped directories
- Issue #42079: Fails with UNC share root paths on Windows

## Implications for gdev

On Linux, `filepath.EvalSymlinks` + `filepath.Abs` is generally reliable. The main gap is non-existent paths: `EvalSymlinks` returns an error if the path doesn't exist. For Write operations to new files, a fallback to `filepath.Clean` (lexical normalization) is needed.
