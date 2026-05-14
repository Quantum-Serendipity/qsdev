---
source: multiple web searches
retrieved: 2026-03-20
type: web-search-synthesis
note: Content synthesized from multiple web search results
sources:
  - https://code.visualstudio.com/remote/advancedcontainers/improve-performance
  - https://github.com/microsoft/vscode-remote-release/issues/2174
  - https://arxiv.org/html/2602.15214
  - https://www.docker.com/blog/dockers-developer-innovation-unveiling-performance-milestones/
  - https://endjin.com/blog/2025/07/supercharge-dev-containers-on-windows
  - https://orbstack.dev/blog/fast-filesystem
---

# Dev Container Performance and Resource Overhead

## Docker Desktop Overhead

Docker Desktop's hypervisor layer introduces a 2.69x startup penalty and 9.5x higher CPU throttling variance compared to native Linux. Container startup is dominated by runtime overhead rather than image size, with only 2.5% startup variation across images ranging from 5 MB to 155 MB on SSD.

### Recent Improvements
Since Docker Desktop 4.23, Docker reduced startup time by 75%. More specifically, startup times decreased from 20.257 seconds (with 4.12) to just 10.799 seconds (with 4.23), representing a 47% performance boost.

## File I/O Performance

File I/O is a critical bottleneck. Building Redis from within a container with source code on the local host took 7 minutes 25 seconds with Docker Desktop 4.11, but with Docker Desktop 4.23 now takes only 2 minutes 6 seconds — a 71% reduction in build time.

## Volume Mount Performance on macOS

On macOS and Windows, bind mounts are not as fast as using the container's filesystem directly. Since macOS and Windows run containers in a VM, bind mounts incur a lot of overhead when crossing the VM boundary. There is a much bigger overhead on macOS in keeping the file system consistent — which leads to performance degradation.

### Solutions
- Docker "named volumes" act like the container's filesystem but survive container rebuilds, ideal for storing package folders like node_modules or output folders like build where write performance is critical
- Repository Containers use isolated, local Docker volumes instead of binding to the local filesystem
- On Windows with WSL, source code within the WSL filesystem uses native bind mounts with significantly faster performance

## Memory and CPU on macOS (Apple Silicon)

The unified memory architecture means Docker's memory allocation directly competes with other system resources. Docker uses 2GB of RAM by default. Docker Desktop will synchronize many file system events and actions between host and container, which is particularly CPU intensive.

Key issues on Apple Silicon:
- Memory overcommitment doesn't work efficiently
- Swapping behavior is aggressive
- Memory compression is ineffective

## Alternatives to Docker Desktop

- **OrbStack** (macOS): Claims truly fast container filesystems, dynamic memory management
- **Colima** (macOS): Lightweight Docker Desktop alternative
- **Podman**: Daemonless, rootless alternative that works with devcontainer.json
