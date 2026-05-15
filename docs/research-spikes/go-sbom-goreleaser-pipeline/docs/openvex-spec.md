<!-- Source: https://github.com/openvex/spec -->
<!-- Retrieved: 2026-05-15 -->

# OpenVEX Specification Overview

## Document Format

OpenVEX documents are minimal JSON-LD files structured with metadata and vulnerability statements. A typical document includes:
- Context reference to the OpenVEX namespace
- Document metadata (author, timestamp, version)
- Array of VEX statements referencing vulnerabilities and affected products

The specification uses Package URLs (purl) as its preferred software identifier, enabling lightweight, embeddable records suitable for integration with Sigstore and in-toto attestations.

## Status Categories

OpenVEX defines four status labels:
- **not_affected**: The product is not affected by the vulnerability
- **affected**: The product is affected
- **fixed**: The vulnerability has been fixed
- **under_investigation**: Assessment is in progress

## SBOM Relationship

OpenVEX maintains deliberate independence from specific SBOM formats. As stated: "VEX metadata should be kept separate from the SBOM." The format works with both SPDX and CycloneDX SBOMs while remaining agnostic to any single format.

## Producer and Consumer Implementation

**Producers** create VEX documents using available tooling, primarily the `vexctl` command-line interface, which enables document creation, merging, and attestation.

**Consumers** validate documents against the provided JSON Schema and use the go-vex library to parse and process VEX metadata from various implementations.

## Supporting Tools

- **vexctl**: CLI tool for creating and managing VEX documents
- **go-vex**: Go library for generating, transforming, and consuming OpenVEX files
- JSON Schema validation tool support

## Current Status

The specification remains in draft form (v0.2.0 as of August 2023), with community governance and a target for 1.0 release pending broader implementation adoption.
