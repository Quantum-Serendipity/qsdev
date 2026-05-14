# Rust zim Crate Documentation
> Source: https://docs.rs/zim/latest/zim/
> Retrieved: 2026-05-14

## Version & Licensing
zim 0.4.0 -- dual licensed Apache-2.0 and MIT.

## Core Purpose
A pure-Rust library for reading ZIM files (primarily used to store wikis like Wikipedia).

## API Components

**Structs:**
- `Zim` - represents a ZIM file
- `DirectoryEntry` - holds metadata about articles
- `Cluster` - manages blobs of data
- `Uuid` - identifier type

**Enums:**
- `Error` - error handling
- `Namespace` - separates different directory entry types
- `MimeType` - represents MIME types
- `Target` - redirect target type

**Type Aliases:**
- `Result` - convenience type

## Decompression Support
- XZ2 (xz2 ^0.1)
- Zstandard (zstd ^0.12)

## Other Dependencies
- Rayon (parallel processing)
- memmap (memory-mapped file access)
- MD5 hashing
- Progress indicators for CLI

## Documentation Coverage
Only 31.48% documented -- API coverage is incomplete.

## Key Limitation
No full-text search support (no Xapian integration). Can read entries and decompress clusters but cannot perform search queries within ZIM files.
