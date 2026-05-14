# AWS ECS Fargate Pricing 2026
- **Source**: https://leanopstech.com/blog/aws-ecs-fargate-pricing-2026/
- **Retrieved**: 2026-05-14

## Base Compute Rates (us-east-1, Per-Second Billing, 1-Minute Minimum)

- vCPU: $0.04048 per hour
- Memory (GB): $0.004445 per hour
- Ephemeral storage (above 20GB): $0.000111 per GB-hour

## Sample Monthly Costs (24/7 Operations, 730 hrs)

| Configuration | Hourly | Monthly | With 50% Savings Plan |
|---|---|---|---|
| 0.5 vCPU / 1GB | $0.025 | $17.87 | $8.94 |
| 1 vCPU / 2GB | $0.049 | $35.74 | $17.87 |
| 4 vCPU / 8GB | $0.197 | $143.75 | $71.88 |
| 16 vCPU / 32GB | $0.789 | $576.21 | $288.11 |

## Spot Pricing

Fargate Spot offers 40-70% discount versus on-demand. Variable pricing: $0.012-0.024/vCPU-hour and $0.001-0.003/GB-hour. Tasks receive 2-minute interruption warning.

## Savings Plans

Compute Savings Plans provide 50% discounts on Fargate workloads with annual commitments.

## ARM/Graviton

Switching to ARM reduces costs by 20% compared to x86 pricing with no other changes required.
