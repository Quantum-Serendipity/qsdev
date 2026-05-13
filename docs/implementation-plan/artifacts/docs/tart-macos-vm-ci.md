# Tart - macOS VM Virtualization for CI
> Source: https://github.com/cirruslabs/tart
> Retrieved: 2026-05-12

## Overview

Tart is a virtualization toolset to build, run, and manage macOS and Linux virtual machines on Apple Silicon. Built by CI engineers for automation needs.

## Technical Details

- Uses Apple's native Virtualization.Framework for near-native performance
- Apple Silicon only (no Intel support)
- Latest release: v2.32.1 (April 12, 2026) - actively maintained
- 177 total releases
- Codebase: Swift (90.6%), Go (6.7%), Python (1.7%)

## Key Features

- Push/Pull VMs from OCI-compatible container registries
- Automated VM creation via Tart Packer Plugin
- CLI tools (tart clone, tart run)
- CI/CD system integration
- Orchard Orchestration for cluster management

## Relationship to Anka

Same underlying technology as Anka 3.0 (both use Virtualization.Framework). No real difference in performance or features. Anka additionally supports Intel macOS environments.

## CI Integration

Powers Cirrus Runners service. Claims 2-3x better performance for a fraction of the price compared to GitHub Actions.

## Available Images

- macOS-Tahoe-base and other macOS versions via container registry
- Linux VM support added in 2025 via Tart+Vetu stack
