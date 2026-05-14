# ZIM File Format Specification

- **Source URL**: https://docs.fileformat.com/compression/zim/
- **Retrieved**: 2026-05-14

## Header Structure

The ZIM file begins with a 80-byte header using little-endian integers:

| Field | Type | Offset | Size (bytes) | Purpose |
|-------|------|--------|--------------|---------|
| magicNumber | uint32 | 0 | 4 | Format identifier: 72173914 (0x44D495A) |
| majorVersion | uint16 | 4 | 2 | Major version (5 or 6) |
| minorVersion | uint16 | 6 | 2 | Minor version |
| uuid | uint128 | 8 | 16 | Unique file identifier |
| articleCount | uint32 | 24 | 4 | Total article quantity |
| clusterCount | uint32 | 28 | 4 | Total cluster quantity |
| urlPtrPos | uint64 | 32 | 8 | Directory pointer list (URL-ordered) location |
| titlePtrPos | uint64 | 40 | 8 | Directory pointer list (title-ordered) location |
| clusterPtrPos | uint64 | 48 | 8 | Cluster pointer list location |
| mimeListPos | uint64 | 56 | 8 | MIME type list position (also header size) |
| mainPage | uint32 | 64 | 4 | Main page reference or 0xffffffff |
| layoutPage | uint32 | 68 | 4 | Layout page reference or 0xffffffff |
| checksumPos | uint64 | 72 | 8 | MD5 checksum position (16 bytes before EOF) |

## File Layout Sections

The overall structure follows this sequence:

1. **Header** (80 bytes, offset 0)
2. **MIME Type List** (offset from mimeListPos)
3. **Directory Entries** (URL and title-ordered pointer lists)
4. **Cluster Pointers**
5. **Clusters** (compressed content blocks)
6. **MD5 Checksum** (16 bytes before end of file)

## Checksum Details

The checksum resides at the position indicated by checksumPos and consists of 16 bytes containing an MD5 hash. The checksum covers the entire file *excluding* the checksum field itself. That is, it is an MD5 of bytes [0, checksumPos).

## Compression

ZIM files use Zstd compression (current). Older files used LZMA2.

## Version History

The format supports majorVersion values of 5 or 6, with corresponding minorVersion fields indicating incremental updates within each major release.
