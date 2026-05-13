# GitHub Actions Runner Pricing (2026)
> Source: https://docs.github.com/en/billing/reference/actions-runner-pricing
> Retrieved: 2026-05-12

## Standard Runners

| Runner Type | Rate |
|---|---|
| Linux 1-core (x64) | $0.002/min |
| Linux 2-core (x64) | $0.006/min |
| Linux 2-core (arm64) | $0.005/min |
| Windows 2-core (x64) | $0.010/min |
| Windows 2-core (arm64) | $0.010/min |
| macOS 3-4 core | $0.062/min |

## Larger Runners (x64)

- Linux: $0.006/min (2-core) to $0.252/min (96-core)
- Windows: $0.022/min (4-core) to $0.552/min (96-core)
- macOS 12-core: $0.077/min

## Larger Runners (arm64)

- Linux: $0.005/min (2-core) to $0.098/min (64-core)
- Windows: $0.008-$0.194/min
- macOS 5-core (M2 Pro): $0.102/min

## GPU Runners

- Linux 4-core GPU: $0.052/min
- Windows 4-core GPU: $0.102/min

## Key Notes

- Included free minutes cannot be used for larger runners
- Larger runners are NOT free for public repositories
- Free tier: 2,000 min (Free), 3,000 min (Pro/Team), 50,000 min (Enterprise)
- Free minutes use multipliers: Linux 1x, Windows 2x, macOS 10x
- Self-hosted: $0.002/min cloud platform charge (since March 2026)
