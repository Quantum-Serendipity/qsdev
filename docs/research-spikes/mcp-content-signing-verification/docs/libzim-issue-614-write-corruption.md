# libzim Issue #614 — Write Corruption Integrity Gap

- **Source URL**: https://github.com/openzim/libzim/issues/614
- **Retrieved**: 2026-05-14
- **Note**: Content fetched via WebFetch; may be summarized

## Issue Status
**Open** — Created August 22, 2021; Assigned to Milestone 10.0.0

## Core Problem

The reporter describes an integrity vulnerability in ZIM file creation. The current process involves three sequential steps: content writing, header writing, and then re-reading the entire file to generate checksums. This approach creates an "integrity gap."

## Specific Corruption Scenario

The critical vulnerability: "any data corruption occurring during the initial writing of the ZIM file to the disk can not be detected by the checksum." Since checksums are calculated *after* all data has already been written, corrupted data written during the initial disk I/O operations cannot be detected by post-hoc verification.

## Performance and Efficiency Concerns

Beyond integrity issues, the re-reading step is described as inefficient, effectively doubling I/O operations for large files. The reporter notes this "nearly doubles the time it takes to create a ZIM file" in their use case.

## Proposed Solution Direction

The issue suggests generating checksums incrementally "whenever a chunk is written to the disk" rather than through post-write verification. However, the reporter acknowledges technical barriers related to non-linear header writing patterns.

## Participants
**IMayBeABitShy** opened the issue; no assignees or active discussion participants documented.
