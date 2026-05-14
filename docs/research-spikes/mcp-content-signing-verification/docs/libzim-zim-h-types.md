# libzim zim.h — Type Definitions and Integrity Check Enum

- **Source URL**: https://raw.githubusercontent.com/openzim/libzim/refs/heads/main/include/zim/zim.h
- **Retrieved**: 2026-05-14

## Type Definitions

- **entry_index_type**: `uint32_t` for entry indexing
- **cluster_index_type**: `uint32_t` for cluster indexing
- **blob_index_type**: `uint32_t` for blob indexing within clusters
- **size_type**: `uint64_t` for measuring sizes
- **offset_type**: `uint64_t` for file offsets

## Compression Methods

```cpp
enum class Compression {
  None = 1,
  Zstd = 5
}
```

Zstandard (value 5) is the currently supported compression method. Intermediate values represent deprecated techniques.

## MIME Type Constant

`MimeHtmlTemplate = "text/x-zim-htmltemplate"` — template markup format.

## Integrity Check Types

The `IntegrityCheck` enum defines validation operations:
- CHECKSUM
- DIRENT_PTRS (PathPtrList offset validation)
- DIRENT_ORDER (directory entry sorting)
- TITLE_INDEX (title index validation)
- CLUSTER_PTRS (cluster pointer validation)
- CLUSTERS_OFFSETS (internal cluster offsets)
- DIRENT_MIMETYPES (MIME type validation)
- COUNT (total check count indicator)

## Configuration & Access Structures

**OpenConfig**: Manages archive opening behavior, supporting Xapian database preloading and dirent range preloading.

**FdInput**: Represents file descriptor-based data access with offset and size parameters.

**ItemDataDirectAccessInfo**: Enables direct file access bypassing the library for uncompressed data, storing filename and offset information.
