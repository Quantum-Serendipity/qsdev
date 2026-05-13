# GitHub-Hosted Runners Reference
> Source: https://docs.github.com/en/actions/reference/runners/github-hosted-runners
> Retrieved: 2026-05-12

## Public Repositories (Free & Unlimited)

**Linux Runners:**
- `ubuntu-slim`: 1 CPU, 5GB RAM, 14GB SSD, x64 architecture
- `ubuntu-latest`, `ubuntu-24.04`, `ubuntu-22.04`: 4 CPUs, 16GB RAM, 14GB SSD, x64
- `ubuntu-24.04-arm`, `ubuntu-22.04-arm`: 4 CPUs, 16GB RAM, 14GB SSD, ARM64

**Windows Runners:**
- `windows-latest`, `windows-2025`, `windows-2025-vs2026`, `windows-2022`: 4 CPUs, 16GB RAM, 14GB SSD, x64
- `windows-11-arm`: 4 CPUs, 16GB RAM, 14GB SSD, ARM64

**macOS Runners:**
- `macos-15-intel`, `macos-26-intel`: 4 CPUs, 14GB RAM, 14GB SSD, Intel architecture
- `macos-latest`, `macos-14`, `macos-15`, `macos-26`: 3 CPUs (M1), 7GB RAM, 14GB SSD, ARM64

## Private Repositories (Paid, Uses Account Minutes)

**Linux Runners:**
- `ubuntu-slim`: 1 CPU, 5GB RAM, 14GB SSD, x64
- `ubuntu-latest`, `ubuntu-24.04`, `ubuntu-22.04`: 2 CPUs, 8GB RAM, 14GB SSD, x64
- `ubuntu-24.04-arm`, `ubuntu-22.04-arm`: 2 CPUs, 8GB RAM, 14GB SSD, ARM64

**Windows Runners:**
- `windows-latest`, `windows-2025`, `windows-2022`: 2 CPUs, 8GB RAM, 14GB SSD, x64
- `windows-11-arm`: 2 CPUs, 8GB RAM, 14GB SSD, ARM64

**macOS Runners:**
- `macos-15-intel`, `macos-26-intel`: 4 CPUs, 14GB RAM, 14GB SSD, Intel
- `macos-latest`, `macos-14`, `macos-15`, `macos-26`: 3 CPUs (M1), 7GB RAM, 14GB SSD, ARM64

## Concurrency Limits by Account Type

| Plan | Max Concurrent Jobs | Max Concurrent macOS |
|------|-------------------|----------------------|
| Free | 20 | 5 |
| Pro | 40 | 5 |
| Team | 60 (standard) / 1,000 (larger) | 5 |
| Enterprise | 500 (standard) / 1,000 (larger) | 50 |

## Job & Workflow Limits

- GitHub-hosted runners: 6 hours max per job
- Self-hosted runners: 5 days max per job
- Workflow runs: 35 days max
- Job matrix: max 256 jobs per workflow run
- Workflow re-runs: max 50 times

## Storage Limits

| Plan | Artifact Storage | Cache Storage | Monthly Minutes |
|------|------------------|---------------|-----------------|
| Free | 500 MB | 10 GB | 2,000 |
| Pro | 1 GB | 10 GB | 3,000 |
| Team | 2 GB | 10 GB | 3,000 |
| Enterprise | 50 GB | 10 GB | 50,000 |
