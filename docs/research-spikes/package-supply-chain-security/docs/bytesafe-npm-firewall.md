# Bytesafe: npm Registry Supply Chain Security & Package Firewall

- **Source URLs**:
  - https://docs.bytesafe.dev/dependency-firewall/firewall-registry/
  - https://bytesafe.dev/supply-chain-security/
  - https://github.com/bitfront-se/bytesafe-ce
- **Retrieved**: 2026-05-12

## Overview

Bytesafe is a security platform that protects organizations from open source software supply chain attacks. Provides repositories for npm, Maven, NuGet, and PyPI package managers.

## Firewall Registry Features

A firewall registry centralizes security policies, delegates responsibilities, and simplifies administration. Policies are automatically enforced before packages reach downstream registries. Acts as a secure proxy to public open source package registries.

## Security Policies

- **Vulnerability Scanning**: Enabled by default for new firewall registries. Quarantines packages surpassing security threshold levels.
- **License Compliance**: Enabled by default for new firewall registries.
- **Block Install Scripts**: Quarantines all npm packages with pre- and post-install scripts.
- **Dependency Confusion Protection**: Configurable to prevent substitution attacks.

## Community Edition

Bytesafe CE is available as an open source community edition on GitHub, providing basic supply chain security features.

## Supported Ecosystems

- npm (primary focus)
- Maven
- NuGet
- PyPI

## Limitations

- Primarily focused on npm ecosystem
- Less mature than Artifactory/Nexus for enterprise use
- Smaller community and ecosystem support
