<!-- Source: https://github.com/kubernetes-sigs/bom -->
<!-- Retrieved: 2026-05-15 -->

# Kubernetes bom Tool - SPDX SBOM Generator

## Overview
bom was created as part of the project to create an SBOM for the Kubernetes project. SIG Release developed a set of libraries to produce fully compliant SPDX SBOMs.

## Key Details
- Generates SBOMs conformant to SPDX version 2.3
- Supports tag-value and JSON formats
- Built-in license classifier recognizing 400+ SPDX catalog licenses
- Can generate SBOMs from directories, container images, single files
- Processes golang dependencies natively
- Written in Go, available as both CLI and Go library

## Kubernetes Release Integration
The intention is to ensure the quality and integrity of artifacts produced on each release cut by adding a Bill of Materials (BOM). The BOM is published in SPDX format and includes integrity and licensing information.

## Key Takeaway
Kubernetes -- the largest Go project in existence -- chose SPDX as their SBOM format and built dedicated tooling (bom) for it.
