<!-- Source: https://learn.microsoft.com/en-us/azure/storage/blobs/blobfuse2-what-is -->
<!-- Source: https://github.com/Azure/azure-storage-fuse/wiki/How-Blobfuse2-Works -->
<!-- Retrieved: 2026-05-14 -->

# BlobFuse2 Overview

BlobFuse is an open-source virtual file system driver (MIT licensed) that enables seamless integration of Azure Blob Storage with Linux environments. Uses libfuse (fuse3) library to interface with the Linux FUSE kernel module. Translates file system operations into Azure Blob REST API calls.

## Operating Modes

### File Cache (Caching Mode)
- Downloads the **entire file** from Azure Blob Storage into a local cache directory before making it available to the application
- All subsequent reads and writes operate on this local cache until the file is evicted or invalidated
- When you create or modify a file, closing the file handle triggers upload to the storage container
- Works well for workloads that **repeatedly access files** or work with datasets that fit on the local disk
- Can preload entire containers or subdirectories to local cache at mount time

### Block Cache (Streaming Mode)
- Streams data in chunks (blocks) and serves it as it downloads
- Designed for workloads involving large files (AI/ML training, genomic sequencing, HPC)
- Azure Storage enforces maximum of 50,000 blocks per blob — requires appropriate block-size configuration for TiB-scale files
- Concurrent writes on identical files lack data consistency checks
- Simultaneous reads during writes cannot guarantee current data retrieval
- Data persists only upon close, sync, or flush operations

## Supported Operations
mkdir, opendir, readdir, rmdir, open, read, create, write, close, unlink, truncate, stat, rename, chmod (HNS accounts only)

## Key Capabilities
- Mount Azure Blob Storage container or Azure Data Lake Storage file system on Linux
- Supports both flat namespaces and hierarchical namespaces
- Local file caching to improve subsequent access times
- Health monitor for mount activities and resource usage
- Blob filter to restrict which blobs a mount can see

## Important Notes
- BlobFuse does NOT guarantee full POSIX compliance (translates requests into Blob REST APIs)
- Rename operations are atomic in POSIX but NOT in BlobFuse
- BlobFuse v1 support ends in September 2026
- Validated with PyTorch and Ray distributed ML frameworks

## Caching Details
- File, metadata, and attribute caching supported
- Configure cache location, size, and retention policies
- Can preload data by downloading entire containers/subdirectories at mount time

## NixOS Compatibility
No specific NixOS support mentioned. BlobFuse2 is available for various Linux distributions. Would need to be packaged via Nix or built from source (Go-based, MIT license).
