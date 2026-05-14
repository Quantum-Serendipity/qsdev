# Gozim (akhenakh/gozim)
> Source: https://github.com/akhenakh/gozim
> Retrieved: 2026-05-14

## Description
A Go native implementation for ZIM files, used primarily for offline Wikipedia copies. Enables reading and serving ZIM format archives.

## Key Features
- Native Go implementation for ZIM file format parsing
- HTTP server functionality for browsing ZIM content
- Full-text search capabilities via Bleve
- Support for multiple compression formats
- Cross-compilation support

## Compression Handling
- **XZ compression**: Implemented via both CGO wrapper (xz_cgo_reader.go) and pure Go library (xz_reader.go)
- **Zstandard compression**: Dedicated reader (zstd_reader.go)
- Pure Go XZ implementation is "around ~2.5x slower" but enables builds without CGO

## Search & Indexing
Uses **Bleve** for full-text search with LevelDB recommended as the storage backend. Optional index files can be pre-built using the `gozimindex` command-line tool.

## Repository Statistics
- **Stars**: 216
- **Forks**: 38
- **Language**: Go (64%), HTML (33.1%), Makefile (1.8%)
- **License**: MIT
- **Release**: v1.0 (January 2, 2015)

## Dependencies
- liblzma-dev (for XZ compression, optional with pure Go fallback)
- Bleve search library
- go.rice (for embedding static assets)

## Executables
- `gozimhttpd`: HTTP server for browsing ZIM files
- `gozimindex`: Index builder for search functionality
