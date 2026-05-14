# Mountpoint for Amazon S3 vs Fusion File System
- **Source**: https://seqera.io/blog/mountpoint-for-amazon-s3-vs-fusion-file-system/
- **Retrieved**: 2026-05-14

## Performance Characteristics
Mountpoint delivers ~6-8x better performance than s3fs-fuse when reading and writing sequential files, according to FIO benchmarks. However, dramatic performance gaps emerge with large files: random reads of 100 GB files with Fusion were up to ~1,300 times faster than AWS Mountpoint.

## Supported Operations
Mountpoint doesn't implement a full POSIX interface, but it supports most common file operations. It supports sequential and random reads, sequential (append only) writes.

## Caching Behavior
Unlike Fusion, Mountpoint has limited client-side caching. Small files perform comparably because most requests are served directly from the kernel page cache, but Mountpoint lacks aggressive local disk caching for large files.

## IAM Integration
Users configure credentials through the ~/.aws/config file or appropriate AWS environment variables before mounting buckets.

## Installation & Deployment
Mountpoint is user-installable and requires moderate configuration effort.

## Scope
Mountpoint is designed exclusively for Amazon S3 only, whereas Fusion supports multiple cloud storage backends.
