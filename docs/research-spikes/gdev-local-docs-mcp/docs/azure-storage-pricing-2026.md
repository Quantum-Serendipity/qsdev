<!-- Source: https://www.nops.io/blog/azure-storage-pricing/ -->
<!-- Retrieved: 2026-05-14 -->

# Azure Storage Pricing 2026

## Blob Storage (LRS, per GB/month)

| Tier | Storage Cost | Retrieval Cost | Min Retention |
|------|-------------|----------------|---------------|
| Hot | $0.018/GB/month | Free | None |
| Cool | $0.010/GB/month | $0.01/GB | 30 days |
| Cold | $0.0045/GB/month | $0.03/GB | 90 days |
| Archive | $0.00099/GB/month | $0.02/GB | 180 days |

## Azure Files Pricing (per GiB/month)

| Tier | Model | Cost |
|------|-------|------|
| Premium (SSD) | Provisioned | $0.181/GiB/month |
| Standard HDD v2 | Provisioned | $0.0088/GiB/month |
| Transaction Optimized | Pay-as-you-go | $0.060/GiB/month |
| Hot | Pay-as-you-go | $0.0276/GiB/month |
| Cool | Pay-as-you-go | $0.015/GiB/month |

## Transaction Costs (per 10,000 operations)

- Premium: $0.0273
- Archive read: $5.50 (1,000x Hot tier rate)
- Write operations range from $0.0273 (Premium) to $0.288 (Cold)

## Egress & Data Transfer

- First 10 TB/month: $0.087/GB
- Note: First 100 GB/month may be free (check current free tier)

## Redundancy Multipliers (vs LRS)

- GRS: ~2x
- RA-GRS: ~2.5x
- RA-GZRS: ~2.7x

## Cost Optimization Tips

- Lifecycle management policies automate tier transitions
- Right-size redundancy by workload criticality
- Consolidate small objects before tiering (128 KiB minimum billable size effective July 2026)
- Monitor transaction charges separately
- Use 1-3 year reserved capacity for 20-48% savings
- Audit and eliminate orphaned/unattached resources
