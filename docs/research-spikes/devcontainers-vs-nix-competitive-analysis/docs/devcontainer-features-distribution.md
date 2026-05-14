---
source: https://raw.githubusercontent.com/devcontainers/spec/main/docs/specs/devcontainer-features-distribution.md
retrieved: 2026-03-20
type: specification
---

# Dev Container Features Distribution Specification

## Overview
The specification enables community members to author and distribute Dev Container Features through self-service publishing mechanisms, whether publicly or privately, with standardized versioning and integrity validation.

## Versioning
Features follow semantic versioning as defined by semver.org. The `version` property in `devcontainer-feature.json` determines republication eligibility. Publishing tools will not republish identical versions but must republish major and minor version updates per semver guidelines.

## Packaging Format
Features are distributed as compressed tarballs named `devcontainer-feature-<id>.tgz`, containing the entire Feature sub-directory including:
- `devcontainer-feature.json` (metadata file)
- `install.sh` (installation entrypoint)
- Any additional supporting files

A `devcontainer-collection.json` auto-generated metadata file aggregates all Features in a collection, containing sourceInformation and a Features array.

## OCI Registry Distribution
The primary distribution mechanism uses OCI registries implementing the OCI Artifact Distribution Specification. Features follow this naming convention: `<registry>/<namespace>/<id>[:version]`.

Key characteristics:
- Custom media types: `application/vnd.devcontainers` and `application/vnd.devcontainers.layer.v1+tar`
- Version tags include major, minor, patch, and `latest` tags
- "Namespace" represents the globally identifiable collection identifier (typically `owner/repo`)
- The `devcontainer-collection.json` is pushed with only namespace and tagged as `latest`
- Manifests include a `dev.containers.metadata` annotation containing the escaped JSON Feature metadata

## Alternative Distribution Methods

**Direct Tarball References**: Features can be referenced via HTTPS URIs pointing directly to `.tgz` files, with the filename requirement matching the standard naming convention.

**Local References**: Features may be locally referenced relative to a project's `devcontainer.json` using unix-style paths (e.g., `./myFeature`). Requirements include:
- Project must contain a `.devcontainer/` folder
- Local Feature must reside in a `.devcontainer/` subfolder
- Subfolder name must match the Feature's `id`
- Cannot use absolute paths
- Must contain `devcontainer-feature.json` and `install.sh`

## Feature Identifier Normalization
"Feature identifiers should be provided lowercase by an author. If not, an implementing packaging tool should normalize the identifier into lowercase for any internal representation."
