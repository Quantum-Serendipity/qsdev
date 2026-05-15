# Go Dependency Submission Action (actions/go-dependency-submission)

- **Source URL**: https://github.com/actions/go-dependency-submission/blob/main/README.md
- **Retrieved**: 2026-05-15

---

## Purpose
This GitHub Action analyzes Go project dependencies and submits them to GitHub's Dependency Submission API, enabling vulnerability tracking through Dependabot.

## Key Parameters

**Required:**
- `go-mod-path`: Path to the go.mod file for your build target

**Optional:**
- `go-build-target`: Path to a Go file containing a `main()` function. When omitted, the action collects dependencies across all build targets.

## How It Works

The action calculates dependencies for a Go build-target (a Go file with a `main` function) and submits the list to the Dependency Submission API. Dependencies then appear in your repository's dependency graph with Dependabot alerts.

## Basic Workflow Example

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
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: ">=1.18.0"
      - uses: actions/go-dependency-submission@v2
        with:
          go-mod-path: go-example/go.mod
          go-build-target: go-example/cmd/octocat.go
```

## Private Module Handling

The action fails when `go.mod` references private modules. Two authentication approaches are supported:

**HTTPS with Personal Access Token:**
```yaml
- name: Authenticate with GitHub
  run: git config --global url.https://${{ secrets.GH_ACCESS_TOKEN }}@github.com/.insteadOf https://github.com/
```

Configure environment variables:
```yaml
env:
  GOPROXY: 'https://proxy.golang.org,direct'
  GOPRIVATE: 'github.com/foo/*'
```

**SSH Authentication:** Deploy keys or SSH keys can substitute for token-based approaches.

## Environment Variables for Private Modules

- `GONOPROXY`: Bypass module proxy for specific modules
- `GOSUMDB`: Set to `off` to skip checksum verification
- `GOPROXY`: Set to `direct` to bypass proxies entirely
