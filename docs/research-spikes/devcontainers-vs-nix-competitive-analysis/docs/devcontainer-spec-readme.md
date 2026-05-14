---
source: https://raw.githubusercontent.com/devcontainers/spec/main/README.md
retrieved: 2026-03-20
type: documentation
---

# Dev Containers Specification README

## Core Purpose
The Development Container Specification enables using containers as complete development environments. As stated: "A development container allows you to use a container as a full-featured development environment."

## Key Format
The primary specification format is `devcontainer.json`, described as "a structured JSON with Comments (jsonc) metadata format that tools can use to store any needed configuration required to develop inside of local or cloud-based containerized coding."

## Specification Structure
The specification documentation is organized in the docs/specs folder (https://github.com/devcontainers/spec/tree/main/docs/specs), with active proposals maintained separately in the proposals folder (https://github.com/devcontainers/spec/tree/main/proposals).

## Implementation & Tools
- Open-source CLI reference implementation available
- Integrates with Docker Compose and single-container options
- GitHub Action and Azure DevOps Task available through devcontainers/ci (https://github.com/devcontainers/ci)

## Contributing
The project welcomes contributions through the How to Contribute (contributing.md) document, issue reports, or via their community Slack channel (https://aka.ms/dev-container-community).

## Licensing
The specification is licensed under Creative Commons Attribution 4.0 License (International), with copyright held by Microsoft Corporation.

## Governance Note
While the spec is open (CC BY 4.0), copyright is held by Microsoft Corporation. The spec lives on GitHub under the devcontainers org. Contributions are welcome but Microsoft controls the repository and final decisions. The reference CLI implementation is MIT-licensed and also under Microsoft copyright.
