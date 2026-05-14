# ZIM File Format Specification (from libzim source code)
> Sources: 
>   https://raw.githubusercontent.com/openzim/libzim/main/include/zim/zim.h
>   https://raw.githubusercontent.com/openzim/libzim/main/src/fileheader.h
> Retrieved: 2026-05-14

## Type Definitions
- **entry_index_type**: uint32_t (entry indexing within ZIM files)
- **cluster_index_type**: uint32_t (cluster indexing)
- **blob_index_type**: uint32_t (blob indexing within clusters)
- **size_type**: uint64_t (sizes of entries, archives, clusters)
- **offset_type**: uint64_t (file offsets)

## Compression Enumeration
- **None** (value: 1)
- **Zstd** (value: 5)
- Intermediate values correspond to compression methods no longer supported (LZMA was value 4, now only read-supported)

## Header Fields (in order)

| Field | Type | Purpose |
|-------|------|---------|
| Magic Number | uint32_t | ZIM file identifier (0x44D495A) |
| Major Version | uint16_t | Format version (major) -- currently 5 or 6 |
| Minor Version | uint16_t | Format version (minor) -- 0 or 1 (new namespace scheme) |
| UUID | Uuid (16 bytes) | Unique file identifier |
| Article Count | entry_index_type (uint32_t) | Number of articles |
| Title Index Position | offset_type (uint64_t) | Byte offset to title listing |
| Path Pointer Position | offset_type (uint64_t) | Byte offset to path pointers |
| MIME List Position | offset_type (uint64_t) | Byte offset to MIME types |
| Cluster Count | cluster_index_type (uint32_t) | Number of data clusters |
| Cluster Pointer Position | offset_type (uint64_t) | Byte offset to cluster directory |
| Main Page | entry_index_type (uint32_t) | Entry index of main page |
| Layout Page | entry_index_type (uint32_t) | Entry index of layout page |
| Checksum Position | offset_type (uint64_t) | Byte offset to MD5 checksum (16 bytes) |

## Integrity Check Types
CHECKSUM, DIRENT_PTRS, DIRENT_ORDER, TITLE_INDEX, CLUSTER_PTRS, CLUSTERS_OFFSETS, DIRENT_MIMETYPES

## Namespace Scheme
- Old scheme (minor version 0): entries use namespace prefixes (A/ for articles, I/ for images, M/ for metadata, X/ for search index)
- New scheme (minor version >= 1): entries use C/ namespace exclusively; metadata accessed via archive.metadata_keys

## Content Entry Structure (from first search result)
Fixed header: 2-byte MIME type index, 1-byte unused parameter length (must be 0), 1-byte namespace identifier, 4-byte unused revision (must be 0), 4-byte cluster number, 4-byte blob number = 16 bytes before variable-length strings (URL and title, null-terminated).

## Redirect Entry Structure
Similar to content entry but instead of cluster/blob numbers, contains a 4-byte redirect target index.

## Cluster Format
- First byte indicates compression type (0=default/uncompressed, 1=none, 4=LZMA, 5=Zstd)
- Followed by a list of 4-byte (version 5) or 8-byte (version 6 extended) offsets delineating blob boundaries
- Number of blobs = first_offset / offset_size
- Typical cluster size: ~1-2 MiB

## Full-Text Search Index
Stored as Xapian databases within the ZIM file's X namespace (old scheme) or embedded in the archive. Uses BM25 ranking algorithm with term frequency, document length, and inverse document frequency.

## Constants
- MimeHtmlTemplate: "text/x-zim-htmltemplate"

## Direct Data Access
ItemDataDirectAccessInfo structure enables direct data access via filename and offset for uncompressed items, bypassing cluster decompression.
