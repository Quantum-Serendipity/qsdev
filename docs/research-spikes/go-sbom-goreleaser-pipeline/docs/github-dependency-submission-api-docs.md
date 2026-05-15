# GitHub Dependency Submission API Documentation

- **Source URL**: https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/using-the-dependency-submission-api
- **Retrieved**: 2026-05-15

---

## Overview
The dependency submission API enables projects to submit dependencies to GitHub's dependency graph, particularly for dependencies not captured through static analysis. This is especially useful for build-time dependency resolution.

## Core Functionality
The API accepts dependency data "in a GitHub Actions workflow to submit dependencies for your project when your project is built." It translates project dependencies into a snapshot format and submits them via API calls.

## Supported Ecosystems & Pre-Built Actions

GitHub provides pre-made actions for streamlined integration:

| Ecosystem | Action |
|-----------|--------|
| Go | Go Dependency Submission |
| Gradle | Gradle Dependency Submission |
| Maven | Maven Dependency Tree Dependency Submission |
| Mill | Mill Dependency Submission |
| Mix (Elixir) | Mix Dependency Submission |
| Scala | Sbt Dependency Submission |
| Multi-ecosystem | Component Detection (supports Vcpkg, Conan, Conda, Crates, NuGet) |

## Go Workflow Example

```yaml
name: Go Dependency Submission
on:
  push:
    branches:
      - main

permissions:
  contents: write

env:
  GOPROXY: ''
  GOPRIVATE: ''
  
jobs:
  go-action-detection:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v5
        with:
          go-version: ">=1.18.0"
      - uses: actions/go-dependency-submission@v2
        with:
          go-mod-path: go-example/go.mod
          go-build-target: go-example/cmd/octocat.go
```

## Custom Action Development

For tailored implementations, teams can create custom actions by:

1. Generating a dependency list for the project
2. Converting dependencies to the snapshot format specified in REST API documentation
3. Submitting the formatted data to the API

GitHub provides the Dependency Submission Toolkit -- a TypeScript library designed to assist developers building custom GitHub Actions for this purpose.

## SBOM Integration

Projects can submit Software Bills of Materials in SPDX and CycloneDX formats. The snapshot format aligns closely with these standard SBOM formats, allowing conversion between formats. Two recommended tools include the SPDX Dependency Submission Action and Anchore SBOM Action.

## SBOM Submission Example

```yaml
name: SBOM upload
on:
  workflow_dispatch:
  push:
    branches: ["main"]

jobs:
  SBOM-upload:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: write

    steps:
    - uses: actions/checkout@v6
    - name: Generate SBOM
      run: |
        curl -Lo $RUNNER_TEMP/sbom-tool https://github.com/microsoft/sbom-tool/releases/latest/download/sbom-tool-linux-x64
        chmod +x $RUNNER_TEMP/sbom-tool
        $RUNNER_TEMP/sbom-tool generate -b . -bc . -pn $ -pv 1.0.0 -ps OwnerName -nsb https://sbom.mycompany.com -V Verbose
    - uses: actions/upload-artifact@v4
      with:
        name: sbom
        path: _manifest/spdx_2.2
    - uses: advanced-security/spdx-dependency-submission-action@v0.2.0
      with:
        filePath: "_manifest/spdx_2.2/"
```

## Key Requirements

- Repository write permissions are required for API submission
- Dependencies must be formatted according to the snapshot specification
- The API integrates seamlessly with GitHub Actions workflows
