<!-- Source: https://learn.microsoft.com/en-us/azure/storage/files/storage-files-netapp-comparison -->
<!-- Retrieved: 2026-05-14 -->

# Azure Files vs Azure NetApp Files Comparison

## Key Differences

| Category | Azure Files | Azure NetApp Files |
|----------|-------------|-------------------|
| Optimized for | Random access workloads | High-performance, low-latency workloads |
| Protocols (SSD) | SMB 2.1/3.0/3.1.1, NFSv4.1, REST | NFSv3, NFSv4.1, SMB, Dual protocol |
| Protocols (HDD) | SMB 2.1/3.0/3.1.1, REST | N/A (all SSD) |
| Min Share Size | 32 GiB | 50 GiB (1 TiB min pool) |
| Max Share Size | 256 TiB | 100 TiB (regular), 2 PiB (large) |
| Max IOPS | SSD: 102,400; HDD: 50,000 | Ultra/Premium: up to 450k |
| Max Throughput | SSD: 10,340 MiB/s; HDD: 5,120 MiB/s | Ultra: 4.5-12.5 GiB/s |
| Max File Size | 4 TiB | 16 TiB |
| Latency | Single-ms min (2-3ms for small IO) | Sub-ms (<1ms for random IO) |

## Identity-Based Authentication

**Azure Files:**
- SMB: AD DS, Entra Domain Services, Entra Kerberos
- NFS: NOT supported (network level auth only)

**Azure NetApp Files:**
- SMB: AD DS, Entra Domain Services
- NFS: ADDS/LDAP integration with NFS extended groups

## Key Insight for Documentation Hosting
Azure Files NFS does NOT support identity-based auth — only network-level authentication.
Azure Files SMB supports Entra Kerberos but has boot-time mount limitations on Linux.
For the documentation hosting use case, Azure Files with SMB + Entra Kerberos or storage account key is the most practical option.
