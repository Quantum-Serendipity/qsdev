# Cross-Platform GitHub Action
> Source: https://github.com/cross-platform-actions/action
> Retrieved: 2026-05-12

## Supported Operating Systems

- **OpenBSD** (6.8-7.8): x86-64 and ARM64
- **FreeBSD** (12.2-15.0): x86-64 and ARM64
- **NetBSD** (9.2-10.1): x86-64 and ARM64
- **DragonFly BSD** (6.4.2): x86-64 only
- **MidnightBSD** (4.0.4): x86-64 only
- **Haiku** (r1beta5): x86-64 only
- **OmniOS** (r151056-r151058): x86-64 only

## Technical Implementation

Uses QEMU hypervisor. VM images built with Packer. Communication via SSH with rsync file sync.

## Performance

Aims for fast boot: pre-built resources, no compression, async operations, pre-provisioned setup.

## Limitations

- Only runs on Linux, macOS, and FreeBSD runners
- Linux itself not supported (use Docker for that)
- Haiku is single-user only

## Workflow Config

```yaml
- uses: cross-platform-actions/action@v1.0.0
  with:
    operating_system: freebsd
    version: '15.0'
    architecture: x86-64
    memory: 6G
    cpu_count: 2
```
