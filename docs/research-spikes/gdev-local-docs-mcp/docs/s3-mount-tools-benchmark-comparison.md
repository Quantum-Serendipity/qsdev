# Comparative Analysis of Mountpoint for S3, S3FS and Goofys
- **Source**: https://medium.com/@maksym.lutskyi/a-comparative-analysis-of-mountpoint-for-s3-s3fs-and-goofys-9a097a25
- **Retrieved**: 2026-05-14

## Performance Testing Overview

Evaluated three tools using FIO utility and JuiceFS bench on EC2 m5.4xlarge instances. Tests included sequential/random read-write operations on 4GB and 16GB files, plus small file operations (100 files x 128KB).

Tool versions tested:
- Goofys: 0.24
- S3FS: 1.93
- Mount-s3: 1.0.0

## Key Performance Findings

Throughput Rankings:
1. Goofys - Highest sequential performance due to minimal metadata overhead and weak POSIX compliance, but limited functionality
2. Mountpoint for S3 - Performance comparable to Goofys with better enterprise features
3. S3FS - Slowest throughput due to metadata transfer via headers and caching overhead

Notable Limitations:
- S3FS: Cache exhaustion caused operational failure during 16GB parallel tests; handler pool depletion during multipart uploads degraded performance
- Mountpoint: Unable to append/modify files; incompatible with fstab; cannot delete files by default
- Goofys: Supports only sequential writes; lacks file metadata, symlinks, hardlinks

## Workload Recommendations

| Use Case | Best Choice | Rationale |
|----------|------------|-----------|
| Read-heavy analytics, ML training | Mountpoint for S3 | Enterprise-ready, high throughput at scale |
| Legacy app compatibility | S3FS | Strongest POSIX support despite slower speeds |
| Resource-constrained environments | Goofys | Lightweight, simple setup |

Critical consideration: All three tools work best for straightforward read-write workflows, not as general-purpose filesystems. Random writes require expensive full object rewrites.
