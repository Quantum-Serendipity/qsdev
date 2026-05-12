# Building Reproducible Docker Images with Locked Dependencies

- **Source**: https://oneuptime.com/blog/post/2026-02-08-how-to-build-reproducible-docker-images-with-locked-dependencies/view
- **Retrieved**: 2026-05-12

## Overview

Creating consistent Docker images requires locking dependencies across three key areas: base images, application dependencies, and system packages. This ensures identical results regardless of when or where builds occur.

## Base Image Pinning

Rather than using mutable tags like `node:20-alpine`, pin to specific digests:

**Bad approach:** `FROM node:20-alpine`

**Better approach:** `FROM node:20-alpine@sha256:a1b2c3d4e5f6...`

Find current digests using:
```bash
docker buildx imagetools inspect node:20-alpine --format '{{.Manifest.Digest}}'
```

## Language-Specific Lock File Strategies

### Node.js
```dockerfile
FROM node:20-alpine@sha256:a1b2c3d4...
COPY package.json package-lock.json ./
RUN npm ci --production
```

### Python
```dockerfile
FROM python:3.12-slim@sha256:e5f6a1b2...
RUN pip install --no-cache-dir --require-hashes -r requirements.txt
```

Generate hashes via: `pip-compile --generate-hashes requirements.in`

### Go
```dockerfile
COPY go.mod go.sum ./
RUN go mod download -x
RUN go mod verify
```

### Rust
```dockerfile
RUN cargo build --release --locked
```

## System Package Pinning

Pin OS packages to specific versions:

```dockerfile
RUN apt-get install -y --no-install-recommends \
    curl=7.88.1-10+deb12u5 \
    ca-certificates=20230311
```

## Build Context Control

Use `.dockerignore` to exclude non-essential files that might change between builds.

## Verification Process

Build an image twice and compare digests to confirm reproducibility:

```bash
docker build --no-cache -t myapp:build1 .
docker build --no-cache -t myapp:build2 .
# Compare resulting image IDs
```
