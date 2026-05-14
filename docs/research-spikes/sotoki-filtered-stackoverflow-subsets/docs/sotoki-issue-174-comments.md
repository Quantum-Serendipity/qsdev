---
source: https://github.com/openzim/sotoki/issues/174
retrieved: 2026-05-14
type: github-issue-comments
---

# Sotoki Issue #174 - Stack Overflow Takes Too Long to Complete

## Issue Summary
Opened: September 14, 2020
Status: Closed
Assigned to: @rgaudin

Main problem: "stackoverflow's duration is not manageable" - need to investigate bottlenecks.

## Comments (chronological)

### @satyamtg (2020-09-14)
"I think in the meanwhile, we can launch a nopic version to see if we can get that to complete in a reasonable time. That may give some clues on whether we shall investigate first into the image download/optimization stuff or somewhere else."

### @dattaz (2020-09-14)
"BTRFS was required because, if I remember, because we hit EXT4 limit of number of file by directory (which we also bypass, by splitting folder with 26 folder of first letter of name)"

### @kevinmcmurtrie (2020-10-23)
"I was able to tune ZFS to handle its 20 million questions files but sotoki was still slow. It was building a ZIM file at about 3 GB per day and using memory even faster."

**OOM Kill Evidence:**
```
Memory cgroup out of memory: Killed process 2906950 (sotoki)
total-vm:33841352kB (~32GB virtual memory)
anon-rss:33477636kB (~32GB resident memory)
```

The sotoki process consumed ~32GB of RAM before being OOM-killed. The process had 8,369,409 pages RSS (about 32GB).

### @kelson42 (2020-10-23) - Kiwix lead
"I believe the problem and the solution are quite clear here. We face an architectural weakness (known and workaround in the past with BTRFS) and this can now be removed by avoiding to write temporary data to the fs by using the full power of python-libzim. At this stage we need a time estimation and a volunteer to work on it."

## Key Findings
- Building SO ZIM: ~3 GB/day build rate
- Memory usage: 32GB+ before OOM kill
- 20 million question files on filesystem
- EXT4 inode limits were a known problem (workaround: BTRFS)
- Proposed solution: use python-libzim to avoid writing temp files to filesystem
- Resource-constrained: needed volunteer developer time
