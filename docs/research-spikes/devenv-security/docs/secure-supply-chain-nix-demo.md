<!-- Source: https://github.com/applicative-systems/secure-supply-chain -->
<!-- Retrieved: 2026-05-12 -->

# Secure Software Supply Chain Demonstration with Nix

## Overview

This repository showcases a method for proving software integrity through reproducible builds. The project creates a minimal NixOS system image containing realistic demo applications while enabling complete offline verification of all source components.

## Key Security Mechanisms

**Source Closure Creation**: The system captures all required source tarballs, Nix expressions, and bootstrap tools through a dedicated script (`source-closure.sh`). This creates an exportable closure file suitable for auditing and offline reconstruction.

**Offline Rebuild Verification**: Organizations can import the closure into isolated systems and rebuild the entire image without accessing binary caches or network resources. This guarantees that the claimed sources genuinely produced the final artifact.

**Complete Dependency Tracking**: The approach includes "all application sources and toolchains (e.g., compilers and their compilers)" ensuring visibility across the entire build stack, not just direct dependencies.

## Demonstration Components

The example system includes:
- A C++ application listening on TCP that writes data to PostgreSQL
- A Rust service exposing database content via HTTP

## Novel Approach

Rather than relying on binary signatures or cache validation, this method emphasizes **demonstrable reproducibility**. Users can:

1. Export all sources in compressed format for third-party audits
2. Rebuild on disconnected systems to prove integrity
3. Use containerized environments (Docker without networking) for quick verification

The accompanying article at nixcademy.com provides deeper technical context on implementing this verification workflow.

**Primary Use Cases**: Regulatory compliance, high-security environments, and supply chain transparency for development teams and compliance officers.
