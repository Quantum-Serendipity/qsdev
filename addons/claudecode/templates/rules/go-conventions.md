# Go Conventions

## Error handling
- Always wrap errors with context using `fmt.Errorf("doing X: %w", err)`.
- Return errors rather than calling `log.Fatal` or `os.Exit` in library code.
- Use sentinel errors (`var ErrNotFound = errors.New(...)`) for expected conditions
  callers need to check. Use custom error types for carrying structured data.

## Naming
- Use MixedCaps (PascalCase for exported, camelCase for unexported). Never use
  underscores in Go identifiers except in test function names.
- Acronyms should be all caps: `HTTPClient`, `userID`, `xmlParser`.
- Package names are lowercase, single-word, and match the directory name.

## Project structure
- Use `internal/` packages for code that must not be imported by external modules.
- Keep `package main` thin: parse flags, wire dependencies, call `run()`.
- Group imports in three blocks: stdlib, external, internal (separated by blank lines).

## Functions
- Pass `context.Context` as the first parameter when the function performs I/O
  or may need cancellation.
- Avoid `init()` functions. Prefer explicit initialization in `main()` or
  constructors.
- Keep functions short and focused. If a function exceeds ~40 lines, consider
  extracting helpers.

## Testing
- Use table-driven tests with `t.Run` subtests.
- Use `t.Helper()` in test helper functions.
- Use `t.Parallel()` for tests that do not share mutable state.
- Prefer stdlib testing over third-party assertion libraries when practical.

## Dependencies
- Prefer the standard library when it provides equivalent functionality.
- Run `go vet ./...` and `golangci-lint run` before committing.
- Keep `go.mod` tidy: run `go mod tidy` after dependency changes.
