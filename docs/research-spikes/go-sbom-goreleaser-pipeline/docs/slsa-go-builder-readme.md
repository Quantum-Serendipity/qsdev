# SLSA Go Builder Documentation

- **Source URL**: https://raw.githubusercontent.com/slsa-framework/slsa-github-generator/main/internal/builders/go/README.md
- **Retrieved**: 2026-05-15

---

## Overview
The SLSA3+ provenance generation system for Go projects, implemented through a GitHub Actions reusable workflow that creates cryptographic attestations of build processes.

## Key Requirements

**Builder Reference**: The builder must be referenced using semantic versioning tags in the format `@vX.Y.Z`. Using shortened tags like `@vX.Y` or hash references will cause build failures.

**Private Repository Handling**: Private repositories require explicit opt-in via the `private-repository: true` flag, as all builds publish entries to the public Rekor transparency log at https://rekor.sigstore.dev/, potentially exposing repository names.

## Supported Triggers

The following GitHub trigger events are fully tested:
- `schedule` events
- `push` events (including new tags)
- `release` events
- Manual execution via `workflow_dispatch`

Most other triggers work in practice, with `pull_request` being the notable exception.

## Configuration Structure

Projects define a `.slsa-goreleaser.yml` configuration file specifying:
- Environment variables (optional)
- Compiler flags (optional)
- Target OS and architecture
- Binary output naming with template variables
- Dynamic ldflags with environment substitution

## Template Variables

The configuration supports these substitution variables:
- `{{ .Os }}` and `{{ .Arch }}` for OS/architecture
- `{{ .Version }}`, `{{ .Tag }}`, `{{ .FullCommit }}`, `{{ .ShortCommit }}`
- `{{ .CommitDate }}`, `{{ .Major }}`, `{{ .Minor }}`, `{{ .Patch }}`

## Multi-Platform Building

Using GitHub Actions matrix strategy, developers can build for multiple OS and architecture combinations by specifying platform-specific configuration files referenced dynamically through matrix variables.

## Workflow Inputs

| Input | Required | Default | Purpose |
|-------|----------|---------|---------|
| `config-file` | No | `.github/workflows/slsa-goreleaser.yml` | Builder configuration location |
| `go-version` or `go-version-file` | No | -- | Go environment setup |
| `evaluated-envs` | No | Empty | Dynamically-generated environment variables |
| `upload-assets` | No | True on tags | Release artifact uploads |
| `private-repository` | No | False | Transparency log participation |

## Workflow Outputs

- `go-binary-name`: Generated binary artifact identifier
- `go-provenance-name`: Signed provenance filename (intoto.jsonl format)

## Provenance Structure

Generated provenance follows the in-toto Statement v0.1 specification, containing:
- Subject identifying the built artifact with SHA256 digest
- Builder identity and build type specifications
- Configuration source details with version control references
- Execution environment variables capturing build context
- Build steps with command sequences, environment settings, and working directories

## Known Issues

**TUF Key Error (v1.2.x)**: Workflows may fail with TUF repository key validation errors. Workaround involves setting `compile-builder: true`.

**Artifact Compatibility**: Provenance downloads require `actions/download-artifact@v3`, not compatible with v4 due to breaking API changes.

## Migration Path

Projects using GoReleaser can transition gradually by creating separate configuration files for SLSA-tracked builds while maintaining existing GoReleaser configurations.
