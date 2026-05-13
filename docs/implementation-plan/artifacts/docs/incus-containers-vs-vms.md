# Incus System Containers vs VMs
> Source: https://linuxcontainers.org/incus/docs/main/explanation/containers_and_vms/
> Retrieved: 2026-05-12

## System Containers

- Use host kernel with namespaces and cgroups
- Consume fewer resources than VMs
- Limited to Linux-based OSes
- Share kernel across instances
- Simulate full OS (libraries, apps, databases)
- Multiple user spaces with per-user process isolation

## Virtual Machines

- Dedicated kernels
- Can host non-Linux OSes
- Require hardware virtualization
- More resources but stronger isolation

## Key Distinction from Docker

Incus system containers simulate full OS environments. Docker packages single processes.
System containers enable multiple user spaces with process isolation -- outside Docker's design scope.

## Limitations

- System containers depend on host kernel
- Cannot run incompatible OS (e.g., Windows, macOS)
- Systemd support depends on host configuration
