<!-- Source: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck -->
<!-- Retrieved: 2026-05-15 -->

# Govulncheck Technical Reference

## Supported Modes

### 1. Source Code Analysis (default)

```bash
$ cd my-module
$ govulncheck ./...
```

- Analyzes source code from the module directory
- Uses Go version from the `go` command on PATH
- Supports build tag specification: `-tags` flag
- Includes test files with `-test` flag

### 2. Binary Analysis

```bash
$ govulncheck -mode binary $HOME/go/bin/my-go-program
```

- Uses binary's symbol table to detect vulnerable functions
- Checks transitive dependencies and main module functions
- Omits call stacks (requires source code)
- Limited to Go 1.18+ for standard library vulnerabilities

### 3. Binary Extraction

```bash
$ govulncheck -mode extract <binary>
```

- Produces minimal information blob (smaller than binary)
- Blob can be analyzed with `-mode binary`
- Contents/representation not intended for end-user interpretation

## Call Graph Analysis Mechanism

### Reachability Detection

Govulncheck performs static analysis of call chains to determine if vulnerable functions are reachable.

Output example:
```
main.go:[line]:[column]: mypackage.main calls golang.org/x/text/language.Parse
```

### Enhanced Tracing

- `-show traces`: Displays full call stacks for each vulnerability
- `-show verbose`: Includes progress messages and detailed findings

## Output Formats

1. **Default Text Format**: Brief summary with call stack summary
2. **JSON Streaming**: Structured format for tool integration (`golang.org/x/vuln/internal/govulncheck`)
3. **SARIF Format**: `-format sarif` — OASIS Static Analysis Results Interchange Format
4. **OpenVEX Format**: `-format openvex` — Vulnerability EXchange format per OpenVEX spec

## Command-Line Flags

| Flag | Purpose |
|------|---------|
| `-mode` | `source`, `binary`, or `extract` |
| `-tags` | Build tags (comma-separated) |
| `-test` | Include test files |
| `-show` | `traces` or `verbose` output |
| `-db` | Custom vulnerability database URL |
| `-json` | JSON output (exits successfully regardless) |
| `-format` | `sarif` or `openvex` output format |

## Vulnerability Database Integration

**Default Database**: `https://vuln.go.dev`
**Custom Database**: Use `-db` flag — must implement Go vuln database specification.
**Privacy**: Database requests contain only module paths with known vulnerabilities, not code or program properties.

## Exit Codes

| Condition | Exit Code |
|-----------|-----------|
| No vulnerabilities found | 0 (success) |
| Vulnerabilities detected | Non-zero (failure) |
| With `-json`, `-format sarif`, or `-format openvex` | 0 (success, regardless) |

## Limitations

### Binary Analysis Limitations

1. No symbol information for binaries built with Go < 1.18 (stdlib only)
2. False positives possible: code may be in binary but unreachable
3. No call graphs due to absent detailed call information in binaries
4. Symbol extraction failure falls back to reporting all module dependencies

### Source Analysis Limitations

1. Function pointers & interfaces: conservative analysis (false positives/inaccurate stacks)
2. Reflect package: vulnerable code accessed via reflect is invisible to static analysis
3. Unsafe package: may cause false negatives
4. No silencing mechanism: cannot suppress specific findings (GitHub issue #61211)
5. Build configuration dependency: different configs may have different vulnerabilities

## Vulnerability Detection Logic

Core process:
1. Identifies all module dependencies in the Go program
2. Queries vulnerability database for known issues
3. Uses static analysis to determine if vulnerable code is imported and reachable through call graph
4. Narrows results to only exploitable vulnerabilities

For binaries:
- Symbol table analysis identifies vulnerable functions
- Only reports vulnerabilities for functions actually present in binary
- Cannot determine unreachability from symbol information alone
