---
source: https://raw.githubusercontent.com/openzim/zimfarm/main/README.md
retrieved: 2026-05-14
type: github-readme
---

# ZIM Farm Overview

## Architecture Description

ZIM Farm operates as "a semi-decentralised software solution to build ZIM files efficiently" through several integrated components:

**Central Systems:**
- **Backend**: A central database and API managing recipes (ZIM metadata) and tasks, deciding when files need recreation
- **Frontend**: Web interface at farm.openzim.org for creating, editing recipes and monitoring task progress

**Worker Infrastructure:**
Workers comprise multiple components working together:
- **Manager**: Low-resource container declaring available resources and receiving task assignments
- **Task-Worker**: Spawned per assigned task to monitor scrapers and manage uploads
- **Uploader**: Handles individual ZIM file and log uploads via SFTP or SCP
- **DNSCache**: dnsmasq server ensuring stable DNS resolution during task execution

**Supporting Components:**
- **Receiver**: Jailed SSH server accepting logs and ZIM files, routing them to download.kiwix.org
- **Scrapers**: Independent tools (like mwoffliner for MediaWiki) converting content to ZIM format

## How Recipes and Tasks Work

Recipes represent metadata for ZIM production. The backend automatically generates tasks based on recipe schedules, assigning them to available workers. This decentralized approach allows distributed processing across multiple worker nodes.

## Getting Started

Documentation guides are available for:
- Recipe managers (Editors Guide)
- Infrastructure implementers (Integrators Guide)
- New workers (Worker README)
- Contributors (CONTRIBUTING.md)

**Contact**: contact+zimfarm@kiwix.org
