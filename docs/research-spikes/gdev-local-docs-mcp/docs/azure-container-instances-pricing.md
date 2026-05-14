<!-- Source: https://www.pump.co/blog/azure-container-instances-pricing/ -->
<!-- Retrieved: 2026-05-14 -->

# Azure Container Instances Pricing

## Per-Second Rates (Linux, Pay-As-You-Go)

- **vCPU**: $0.0000135/vCPU-s (~$29.57/month per vCPU if always-on)
- **Memory**: $0.0000015/GB-s (~$3.25/month per GB if always-on)

## Windows Container Premium
Additional $0.000012 per vCPU second surcharge for Windows licensing.

## Savings Plans
- 1-year commitment: ~27% savings
- 3-year commitment: ~52% savings

## Spot Container Pricing
Up to 70% discount compared to regular-priority pricing (subject to preemption).

## Monthly Cost Examples

**Scenario 1**: 1 vCPU, 1 GB container running 5 minutes daily = ~$0.135/month
**Scenario 2**: 2 vCPU, 2.2 GB container running 50 times daily for 150 seconds = ~$6.808/month

## Always-On Cost Estimate
For a 1 vCPU, 2 GB container running 24/7:
- vCPU: ~$29.57/month
- Memory: ~$6.50/month
- **Total: ~$36.07/month**

For a 2 vCPU, 4 GB container running 24/7:
- vCPU: ~$59.14/month
- Memory: ~$13.00/month
- **Total: ~$72.14/month**

## Comparison
Azure Container Apps or AKS recommended for long-running services. ACI better for burst/ephemeral workloads. A container can run ~20 hours/day before matching VM cost.
