---
source: https://hub.kiwix.org/weblog/2020/8/a-short-history-of-farming/
retrieved: 2026-05-14
type: blog-post
---

# The Evolution of Kiwix's ZIM Build Infrastructure

## From Manual to Automated Processing

In Kiwix's early days, creating new ZIM file versions required manual intervention. A team member would execute commands to retrieve website data, compress it, and manage downloads and uploads -- a tedious process that became impractical as the project scaled.

## The Zimfarm Solution

To address this challenge, Kiwix developed an automated system called the Zimfarm. The system now operates continuously across multiple machines.

## Current Infrastructure Specifications

The Zimfarm has grown substantially:
- Executes over 4,000 automated build recipes
- Supports more than 100 languages
- Runs on "a half-dozen servers, 24/7"
- Updates most files monthly
- Distributes completed files to a central library, then to mirrors worldwide

## Hardware Requirements for Contributors

Those wishing to donate computing resources need:
- Minimum 2GB RAM and 3 processor cores
- Docker CE installation
- Fast bidirectional internet connectivity
- Linux/macOS operating system
- Synchronized system clock
