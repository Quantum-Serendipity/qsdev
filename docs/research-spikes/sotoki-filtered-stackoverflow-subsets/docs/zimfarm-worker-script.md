---
source: https://raw.githubusercontent.com/openzim/zimfarm/main/workers/contrib/zimfarm.sh
retrieved: 2026-05-14
type: source-code-extraction
---

# Zimfarm Worker Manager Script Configuration

## Default Hardware Resource Declarations

- **Memory**: 2G (ZIMFARM_MAX_RAM) -- this is the DEFAULT, workers can declare more
- **Disk**: 10G (ZIMFARM_DISK)
- **CPU cores**: 3 (ZIMFARM_CPU)
- **Task CPU allocation**: ZIMFARM_TASK_CPUS and ZIMFARM_TASK_CPUSET (optional pinning)

## Worker Registration

- Web API endpoint: https://api.farm.openzim.org/v2
- Polling interval: 180 seconds
- Worker identification: ZIMFARM_WORKER_NAME

## Resource Declaration

Resources passed as environment variables to Docker container:
```
--env ZIMFARM_MEMORY=$ZIMFARM_MAX_RAM
--env ZIMFARM_DISK=$ZIMFARM_DISK
--env ZIMFARM_CPUS=$ZIMFARM_CPU
```

The manager container handles downstream task assignment based on declared resources.

## Platform-Specific Limits

Maximum concurrent tasks can be set per platform:
- wikimedia, youtube, wikihow, ifixit, devdocs, ted

## Key Insight for SO Builds

The default 2G RAM / 10G disk is woefully insufficient for SO builds which need:
- 80GB+ RAM (sort alone needs 32GB+, issue #394 allocated 80GB container)
- Hundreds of GB disk (75GB output ZIM + temp files + XML dumps)
- Days of CPU time
- Sustained network access for 4M+ image downloads

SO builds run on dedicated high-memory machines (e.g., "kathrin" with 172GB RAM).
