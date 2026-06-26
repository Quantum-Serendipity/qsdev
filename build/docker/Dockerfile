# syntax=docker/dockerfile:1
#
# Universal qsdev MCP server — distroless container image (Phase 32, Unit 32.10).
#
# Build from the REPOSITORY ROOT (the build context must be the repo root so the
# vendor tree and cmd/qsdev are visible):
#
#   docker build -f build/docker/Dockerfile -t qsdev-mcp-gateway .
#
# Key build detail: this repo carries a go.work that `use`s ../gdev, which does
# not exist inside the container. gdev is VENDORED (vendor/fastcat.org/go/gdev)
# and pinned via a replace directive, so the build IGNORES the workspace and uses
# the vendor tree: GOWORK=off + -mod=vendor. The .dockerignore deliberately
# excludes go.work/go.work.sum from the context for the same reason.

# ---- builder stage --------------------------------------------------------
# Module is Go 1.26.4; use the matching toolchain (the spec's 1.23 is stale).
# Pinned by digest for reproducible, supply-chain-resistant builds (the tag is
# kept for readability). Re-resolve with: docker buildx imagetools inspect golang:1.26-alpine
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

WORKDIR /src

# Ignore the workspace and build strictly from the committed vendor tree.
ENV GOWORK=off
ENV GOFLAGS=-mod=vendor
ENV CGO_ENABLED=0
ENV GOOS=linux

# Copy the whole context (kept small by .dockerignore) and build the static,
# stripped binary. -ldflags='-s -w' drops the symbol table and DWARF info.
COPY . .
RUN go build -ldflags='-s -w' -o /qsdev ./cmd/qsdev

# ---- runtime stage --------------------------------------------------------
# distroless/static is the minimal base for a CGO-free static Go binary; the
# :nonroot variant runs as an unprivileged user (uid 65532) by default. Pinned by
# digest (tag kept for readability); re-resolve with:
#   docker buildx imagetools inspect gcr.io/distroless/static-debian12:nonroot
FROM gcr.io/distroless/static-debian12:nonroot@sha256:d093aa3e30dbadd3efe1310db061a14da60299baff8450a17fe0ccc514a16639

COPY --from=builder /qsdev /qsdev

# The universal MCP server is the entrypoint; deployment mode is selected via
# QSDEV_DEPLOY_MODE / --deploy-mode and transport via --transport.
ENTRYPOINT ["/qsdev", "mcp", "serve"]
