---
source: https://blog.thestaticturtle.fr/fixing-a-dev-worst-nightmare/
retrieved: 2026-05-14
type: blog-post
---

# TheStaticTurtle's Offline Stack Overflow Setup

## Tools Used
- ZIM file format (LZMA2 compression)
- Kiwix reader
- Docker and Docker Compose
- Proxmox cluster
- NAS storage (CIFS/SMB)
- Custom browser extension for automatic redirects

## Resource Requirements
- ZIM file size: 161 GB (September 2021 dump, 21,958,765 articles)
- Download time: ~2 days via torrent
- Needed CIFS support and Docker nesting enabled

## Problems Encountered

1. **Outdated content**: The Kiwix wiki's Feb 2019 dump was stale for tech topics. Resolved by sourcing a Sept 2021 version from Reddit's DataHoarder community.

2. **Network/storage**: Required SMB credentials, /etc/fstab entries, manual uid/gid permissions.

3. **Availability detection**: Custom browser extension checks for broken pages and redirects to local Kiwix instance.

## Key Insight
- Used the FULL SO ZIM (161GB), no tag filtering
- Sourced from community rather than official Kiwix downloads
- Self-hosted via Docker on home infrastructure
