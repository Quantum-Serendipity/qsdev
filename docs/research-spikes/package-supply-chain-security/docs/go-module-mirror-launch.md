# Go Module Mirror and Checksum Database Launch
- **Source**: https://go.dev/blog/module-mirror-launch
- **Retrieved**: 2026-05-12

## Overview
On August 29, 2019, the Go team launched three production-ready services for Go 1.13 module users:
- **Module Mirror**: proxy.golang.org
- **Checksum Database**: sum.golang.org
- **Module Index**: index.golang.org

## Module Mirror (proxy.golang.org)

### How It Works
The module mirror is a special proxy that caches module metadata and source code, allowing faster downloads and protection from disappearing dependencies.

### Benefits
- Fetches only needed module metadata/source code
- Reduces latency by avoiding full repository history downloads
- Caches source code even when original locations disappear
- Speaks an API better suited to `go` command needs

### Default Behavior
- Automatic for Go 1.13+ users
- For earlier Go versions, manually enable: `export GOPROXY=https://proxy.golang.org`

## Checksum Database (sum.golang.org)

### How It Works
A global source of `go.sum` hash lines using a Transparent Log (Merkle tree) architecture backed by Trillian.

### Verification Process
The `go` command verifies two types of proofs:
1. **Inclusion proofs**: Confirms a record exists in the log
2. **Consistency proofs**: Confirms the tree hasn't been tampered with

### Security Guarantee
Even module authors cannot change version bits without detection. A proxy or origin server cannot serve wrong code without getting caught.

## Configuration & Opt-Out

### Non-Public Modules
Configure environment variables to exclude private modules from the proxy and checksum database.

### Privacy
Details at proxy.golang.org/privacy

## Module Index (index.golang.org)
A public feed of new module versions available through proxy.golang.org.
