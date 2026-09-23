# syntax=docker/dockerfile:1
#
# Universal qsdev MCP server — distroless container image (Phase 32, Unit 32.10).
#
# Build from the REPOSITORY ROOT (the build context must be the repo root so the
# vendor tree and cmd/qsdev are visible):
#
#   docker build -f build/docker/Dockerfile -t qsdev-mcp-gateway \
#     --build-arg VERSION="$(cat VERSION)" --build-arg COMMIT="$(git rev-parse --short HEAD)" .
#
# Key build detail: this repo carries a go.work that `use`s ../gdev, which does
# not exist inside the container. gdev is VENDORED (vendor/fastcat.org/go/gdev)
# and pinned via a replace directive, so the build IGNORES the workspace and uses
# the vendor tree: GOWORK=off + -mod=vendor. The .dockerignore deliberately
# excludes go.work/go.work.sum from the context for the same reason.

# ---- builder stage --------------------------------------------------------
# The image sets GOTOOLCHAIN=local, so its Go must be at least go.mod's `go`
# line (the security pin from nix/go-overlay.nix); CI builds this image, so a
# stale pin fails there. Pinned by digest for reproducible, supply-chain-
# resistant builds (the tag is kept for readability; dependabot bumps both).
# Re-resolve with: docker buildx imagetools inspect golang:<version>-alpine
FROM golang:1.26.8-alpine@sha256:8ac98ca534ac3f51e1f420a1dd2c15e74c75cfa0f23f3ad27eb5d7236c349a0c AS builder

WORKDIR /src

# Ignore the workspace and build strictly from the committed vendor tree.
ENV GOWORK=off
ENV GOFLAGS=-mod=vendor
ENV CGO_ENABLED=0
ENV GOOS=linux

# Version stamping, matching .goreleaser.yaml's ldflags.
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Copy the whole context (kept small by .dockerignore) and build the static,
# stripped, reproducible binary. -ldflags='-s -w' drops the symbol table and
# DWARF info; -trimpath drops build-machine paths.
COPY . .
RUN go build -trimpath -o /qsdev \
      -ldflags="-s -w \
        -X github.com/Quantum-Serendipity/qsdev/internal/version.version=${VERSION} \
        -X github.com/Quantum-Serendipity/qsdev/internal/version.commit=${COMMIT} \
        -X github.com/Quantum-Serendipity/qsdev/internal/version.date=${DATE} \
        -X github.com/Quantum-Serendipity/qsdev/internal/version.builtBy=docker" \
      ./cmd/qsdev

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
