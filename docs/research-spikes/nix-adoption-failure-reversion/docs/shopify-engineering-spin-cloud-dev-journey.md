<!-- Source: https://shopify.engineering/shopifys-cloud-development-journey -->
<!-- Retrieved: 2026-03-20 -->

# The Journey to Cloud Development: How Shopify Went All-in on Spin

## Timeline
- **Pre-2020**: Used homegrown `dev` tool with `xhyve` virtual machine called `railgun`
- **Early 2020**: Started porting `dev` to Linux but paused for other priorities
- **Fall 2020**: Released first Kubernetes pod-based iteration to early adopters
- **2020-2021**: Developed cloud-native iteration with individual pods per repository
- **January 2022**: Completed migration from "Spin Legacy" to Isospin
- **June 9, 2022**: Published this retrospective article

## Core Problems They Faced

**Resource Constraints**: As projects grew more complex with multiple repositories and services, laptops became overwhelmed. Developers faced "spinning fans and spooling swapfiles" with code-build-test cycles extending from hours to weeks.

**Tophatting Friction**: Code review validation required developers to replicate exact environments. Managing database state across multiple integrations created complexity — developers had to unwind migrations before switching contexts, then regenerate their setup afterward.

**Inter-repository Complexity**: Teams needed features spanning multiple repositories with multiple feature branches active simultaneously. Some resorted to running pseudo-staging versions in the cloud.

## Why Cloud Development

The Environments team recognized that "development environments needed to be able to scale just as well as production applications." Local laptops provided an inelastic resource constraint that prevented scaling with growing architectural complexity.

## Evolution of Solutions

**Experiment 1: GCE VMs**
Created a command allowing developers to provision Google Compute Engine instances. Surprisingly, teams with self-contained, well-documented repositories adopted this widely — showing organic demand for cloud development.

**Experiment 2: Kubernetes Pods (Spin)**
Early iterations used pods with Docker Compose services. However, developers found the dual-context approach (editing in host container, running in application container) created "ceremony" and cognitive load.

**Final Solution: Isospin**
Represented a philosophical shift toward simplification. Rather than forcing cloud infrastructure understanding onto developers, the team created "a laptop in the cloud" using Linux, systemd orchestration, and automatic dependency inference.

## Key Realization from Leadership

CEO Tobi Lutke advised the team that developers shouldn't need to understand infrastructure implementation details. Just as developers use Rails without understanding Ruby internals, development environments should abstract away orchestration complexity.

## What Worked Right

- **Early adoption signals**: GCE VM experiment showed genuine developer demand
- **Iterative approach**: Observing real usage patterns informed each redesign
- **Abstraction layers**: Isospin simplified configuration compared to raw Docker/Kubernetes
- **Dependency inference**: Automatically detecting project needs reduced configuration overhead

## Architecture Details (Isospin)

Systemd-based orchestration with CGroup partitioning replaced Docker Compose complexity. Repositories defined their own configurations; systemd units automatically triggered cloning, configuration discovery, and service startup.

The team maintained their core skill: "writing CLI tools to accelerate developers' code-test loop" rather than managing Kubernetes orchestration.
