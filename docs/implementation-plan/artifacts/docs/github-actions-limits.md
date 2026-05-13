# GitHub Actions Limits
> Source: https://docs.github.com/en/actions/reference/limits
> Retrieved: 2026-05-12

## Concurrency Limits

| Plan | Max Concurrent Jobs | Max Concurrent macOS | Max Concurrent GPU |
|------|-------------------|----------------------|-------------------|
| Free | 20 | 5 | N/A |
| Pro | 40 | 5 | N/A |
| Team | 60 (standard) / 1,000 (larger) | 5 | 100 |
| Enterprise | 500 (standard) / 1,000 (larger) | 50 | 100 |

The maximum concurrent macOS jobs is shared across standard and larger runners.

## Execution Limits

- GitHub-hosted runners: 6 hours per job
- Self-hosted runners: 5 days per job
- Workflow runs: 35 days max
- Job queue time (self-hosted): 24 hours before cancellation
- Environment approvals: up to 30 days

## Matrix & Re-run

- Job matrix: max 256 jobs per workflow run
- Workflow re-runs: max 50 times

## Storage

| Plan | Artifact Storage | Cache Storage | Monthly Minutes |
|------|------------------|---------------|-----------------|
| Free | 500 MB | 10 GB | 2,000 |
| Pro | 1 GB | 10 GB | 3,000 |
| Team | 2 GB | 10 GB | 3,000 |
| Enterprise | 50 GB | 10 GB | 50,000 |

## API Rate Limits

- GITHUB_TOKEN: 1,000 requests/hour/repo
- Enterprise: 15,000 requests/hour/repo

## 2026 Pricing Changes

- Jan 1, 2026: Runner prices reduced by up to 39%
- Mar 1, 2026: New $0.002/min cloud platform charge for self-hosted runner usage
- Public repo standard runners remain free
