<!-- Source: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck -->
<!-- Retrieved: 2026-05-15 -->

# Govulncheck: Comprehensive Overview

## Purpose
Govulncheck reports known vulnerabilities that affect Go code using static analysis of source code or binary symbol tables. It narrows down reports to only vulnerabilities that could actually affect the application.

## Vulnerability Database
- **Default**: https://vuln.go.dev (Go vulnerability database)
- **Custom**: Use `-db` flag to specify alternative databases
- **Privacy**: Database requests contain only module paths with known vulnerabilities, not code or program properties

## Operating Modes

### 1. Source Code Analysis (Default)
```bash
cd my-module
govulncheck ./...
```
- Analyzes Go source code using the Go version from the PATH
- Supports build tag customization via `-tags` flag
- Can include test files with `-test` flag

### 2. Binary Analysis Mode
```bash
govulncheck -mode binary $HOME/go/bin/my-go-program
```
- Uses binary's symbol table to identify vulnerable functions
- Checks transitive dependencies and main module functions
- **Limitation**: Omits call stacks (requires source code)
- Works best when precise binary module version is known

### 3. Binary Extraction Mode
```bash
govulncheck -mode extract <binary>
```
- Extracts minimal information needed for analysis
- Produces a smaller blob than the original binary
- Blob output can be analyzed with `-mode binary`

## Output Formats

### 1. Standard Text Output (Default)
### 2. JSON Streaming (-format json)
### 3. SARIF Format (-format sarif)
### 4. OpenVEX Format (-format openvex)

## Key Differences from Standard Vulnerability Scanners

| Aspect | Govulncheck | Standard Scanners |
|--------|-------------|-------------------|
| **Scope** | Go-specific | Multi-language |
| **Analysis** | Static analysis of reachability | Dependency matching |
| **False Positives** | Reduced (reachability checks) | Higher (all dependencies) |
| **Call Stacks** | Provided (source mode) | Usually not |
| **Binary Support** | Yes, symbol-based | Limited |

## Limitations

1. **Function Pointers & Interfaces**: Conservative analysis may cause false positives
2. **Reflection**: Calls via reflect package are invisible to static analysis
3. **Binary Analysis**: No detailed call graphs; may report false positives for unreachable code
4. **Legacy Binaries**: Pre-Go 1.18 binaries report only standard library vulnerabilities
5. **Symbol Extraction**: If symbol information can't be extracted, reports vulnerabilities for all dependent modules
