<!-- Source: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck -->
<!-- Retrieved: 2026-05-12 -->

# Govulncheck Documentation

## Overview
Govulncheck reports known vulnerabilities that affect Go code using static analysis of source code or binary symbol tables. It queries the Go vulnerability database at https://vuln.go.dev by default.

## Command-Line Usage

### Basic Syntax
```bash
# Analyze source code
cd my-module
govulncheck ./...

# Analyze compiled binary
govulncheck -mode binary $HOME/go/bin/my-go-program

# Extract minimal binary info for analysis
govulncheck -mode extract <binary>
```

### Key Flags

| Flag | Description |
|------|-------------|
| `-mode` | Execution mode: `source` (default), `binary`, or `extract` |
| `-db` | Custom vulnerability database URL (must implement Go vuln database spec) |
| `-tags` | Comma-separated build tags to process |
| `-test` | Include test files in analysis |
| `-show traces` | Display full call stacks for each vulnerability |
| `-show verbose` | Include progress messages and detailed findings |
| `-format` | Output format specification (see below) |

## Output Format Options

The `-format` flag controls output format:

### Supported Formats
- **text** (default): Human-readable output with vulnerability summaries
- **json**: Streaming JSON format
- **sarif**: Static Analysis Results Interchange Format (OASIS specification)
- **openvex**: Vulnerability EXchange (VEX) format (OpenVEX spec)

### Format Flag Usage
```bash
govulncheck -format json ./...
govulncheck -format sarif ./...
govulncheck -format openvex ./...
```

**Note:** JSON, SARIF, and OpenVEX formats exit with code 0 regardless of vulnerabilities found, unlike default text format.

## Output Structure

### Example Text Output
```
main.go:[line]:[column]: mypackage.main calls golang.org/x/text/language.Parse
```

Shows:
- File location and call site
- Package and function name
- Called vulnerable function

### JSON Streaming Format
- Detailed in: `golang.org/x/vuln/internal/govulncheck`
- Structured for integration with CI/CD systems

### SARIF Format
- Complies with OASIS Static Analysis Results Interchange Format specification
- Details in: `golang.org/x/vuln/internal/sarif`

### OpenVEX Format
- Follows OpenVEX specification: https://github.com/openvex/spec
- Details in: `golang.org/x/vuln/internal/openvex`

## Exit Codes

| Exit Code | Condition |
|-----------|-----------|
| **0** | No vulnerabilities found, OR using `-format json/sarif/openvex` |
| **non-zero** | Vulnerabilities detected (text format only) |

## Known Limitations

1. **Function pointers & interfaces**: Conservative analysis may produce false positives or inaccurate call stacks
2. **Reflect package**: Calls via reflection are invisible to static analysis
3. **Unsafe package**: May result in false negatives
4. **Binary analysis**: No detailed call graphs available; may report false positives
5. **Silencing findings**: Not supported (see https://go.dev/issue/61211)
6. **Pre-Go 1.18 binaries**: Only standard library vulnerabilities reported
7. **Symbol extraction failure**: Reports vulnerabilities for all transitive dependencies

## Privacy
Requests to the vulnerability database contain only module paths with known vulnerabilities, not code or program properties.
