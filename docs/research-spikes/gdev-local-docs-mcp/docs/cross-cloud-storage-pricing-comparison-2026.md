# Cloud & AI Storage Pricing Comparison 2026: AWS, Azure, GCP, OCI
- **Source**: https://www.finout.io/blog/cloud-storage-pricing-comparison
- **Retrieved**: 2026-05-14

## Per-GB Storage Rates by Tier

| Provider | Hot/Standard | Infrequent Access | Cold | Archive |
|----------|-------------|------------------|------|---------|
| AWS S3 | $0.023/GB | $0.0125/GB (Standard-IA) | $0.004/GB (Glacier Instant) | $0.00099/GB (Deep Archive) |
| Azure Blob | $0.018/GB (Hot, LRS) | $0.010/GB (Cool) | $0.0045/GB | $0.00099/GB |
| GCP Cloud Storage | $0.020/GB (Regional) | $0.010/GB (Nearline) | $0.004/GB (Coldline) | $0.0024/GB (Multi-region) |
| Oracle OCI | $0.0255/GB | $0.015/GB | — | $0.0026/GB |

## Egress Charges

- AWS: $0.09/GB for first 10 TB/month
- Oracle OCI: $0.0085/GB beyond 10 TB/month free allowance (~10x cheaper than AWS)
- Azure & GCP: Vary by region and replication method

## Request Costs

- AWS S3: PUT/COPY/POST/LIST at $0.005 per 1,000 requests; GET at $0.0004 per 1,000
- Azure: Per-operation charges vary
- GCP: Class A and Class B operation charges

## Key Insight

Real-world spend is two to five times higher than raw storage rates when accounting for retrieval fees, egress, and operations.
