# Rclone Mount Technical Overview
- **Source**: https://rclone.org/commands/rclone_mount/
- **Retrieved**: 2026-05-14

## Supported Backends

Rclone mount supports Linux, FreeBSD, macOS and Windows systems. 50+ storage backends including S3, Google Drive, Dropbox, Azure Blob Storage, OneDrive, and many others through its VFS layer abstraction.

## VFS Cache Modes

Four cache strategies with increasing disk usage:

**Off (Default):** Read directly from remote, write directly to remote without caching. Cannot handle files opened for both reading and writing simultaneously.

**Minimal:** Similar to off, but files opened for read AND write will be buffered to disk.

**Writes:** Read-only files access remotes directly; write operations buffer to disk. Should support all normal file system operations.

**Full:** All reads and writes buffer to disk. Files in cache are sparse files and rclone tracks which bits have been downloaded.

## Random Read Behavior

Critical limitation: Without --vfs-cache-mode, this can only write files sequentially, it can only seek when reading. Full caching enables true random access patterns.

## Cache Configuration Options

- --cache-dir: Storage location for cached data
- --vfs-cache-max-size: Maximum cache disk space
- --vfs-cache-max-age: Eviction time (default 1 hour)
- --vfs-cache-poll-interval: Cleanup frequency (default 1 minute)
- --vfs-write-back: Upload delay after file closure (default 5 seconds)
- --buffer-size: Memory buffer per open file
- --vfs-read-ahead: Disk buffering beyond memory buffer

## Performance Characteristics

Chunked reading with configurable parallelization. With --vfs-read-chunk-streams > 0, rclone reads multiple chunks concurrently, improving throughput on high-latency backends. S3 and Swift benefit from --no-modtime flag.

## FUSE Implementation

On Windows: WinFsp. On macOS: macFUSE or FUSE-T. On Linux: standard FUSE with fusermount.

## I/O Limitations for Large Files

Large file random access requires explicit caching. Files can't be opened for both read AND write in off mode. Sparse file support is essential for full mode — FAT/exFAT do not support sparse files, causing severe performance degradation.
