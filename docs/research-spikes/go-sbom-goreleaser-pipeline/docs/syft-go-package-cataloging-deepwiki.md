<!-- Source: https://deepwiki.com/anchore/syft/3.2-go-package-cataloging -->
<!-- Retrieved: 2026-05-15 -->

# Syft Go Package Cataloging: Comprehensive Overview

## Cataloger Architecture

Syft employs a dual-approach Go cataloging system with two primary cataloger types:

1. **Binary Cataloger** - Extracts package data from compiled Go executables
2. **Module Cataloger** - Analyzes go.mod files for dependency information

## Go.mod Scanning Process

The module cataloger leverages two complementary approaches:

**Direct File Parsing**: Reads go.mod files to extract declared dependencies and versions directly from the module specification.

**Go Toolchain Integration**: Utilizes `golang.org/x/tools/go/packages` for "deep source analysis" that provides "comprehensive dependency information but requires the Go toolchain to be available at analysis time."

## Binary Scanning Metadata Extraction

The binary cataloger extracts embedded build information from Go executables through several mechanisms:

**Version Detection Strategy** employs a three-tiered hierarchy:
- **ldflags parsing** (highest priority) - Recognizes patterns like `-X main.version=1.0.0`
- **Binary content scanning** (medium priority) - Uses regex pattern matching
- **VCS pseudo-versions** (lowest priority) - Derives from build settings

The document notes the system "recognizes common build flag patterns" for version injection during compilation.

## License Resolution System

The cataloger implements sophisticated license discovery across four sources:

| Source | Method | Priority |
|--------|--------|----------|
| Embedded in scan target | `findLicensesInSource()` | Highest |
| Local mod cache (`$GOPATH/pkg/mod`) | `getLicensesFromLocal()` | Medium-High |
| Vendor directory | `getLicensesFromLocalVendor()` | Medium |
| Remote proxies | `getLicensesFromRemote()` | Lowest |

The system respects Go proxy configuration including NOPROXY patterns.

## Package Metadata & PURL Generation

Packages generate Package URLs following golang specifications:

```
pkg:golang/namespace/name@version#subpath
```

Examples demonstrate namespace and subpath extraction: `pkg:golang/github.com/coreos/go-systemd@v22.1.0#v22` extracts the major version as a subpath.

## Known Architecture Details

The golang cataloger directory includes specialized modules:
- `cataloger.go` - Orchestration logic
- `parse_go_binary.go` & `scan_binary.go` - Binary extraction
- `parse_go_mod.go` - Module file parsing
- `licenses.go` - License resolution
- `package.go` - Metadata structure generation

## Source vs. Binary Output Differences

The document distinguishes `GolangBinaryEntry` metadata for binaries versus `GolangModuleEntry` and `GolangSourceEntry` structures for module-based analysis, reflecting fundamentally different information availability.

## Limitations & Edge Cases

The documentation does not explicitly detail known limitations, though the requirement for "Go toolchain availability" for comprehensive module analysis represents a practical constraint.
