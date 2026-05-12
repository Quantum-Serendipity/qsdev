# SLSA Specification v1.2 — About
- **Source**: https://slsa.dev/spec/v1.2/about
- **Retrieved**: 2026-05-12

## Overview

Supply-chain Levels for Software Artifacts, or SLSA ('salsa'), is a set of incrementally adoptable guidelines for supply chain security.

## Track Structure

The framework organizes around "tracks and levels" with the Build Track currently spanning Levels 1-3. Both Build and Source tracks exist.

## Core Purpose

SLSA offers protection against code modification, unexpected artifact uploads, and build platform threats by providing "guidelines and tamper-resistant evidence for securing each step."

## Levels (from search results and spec overview)

- **Level 0**: No provenance exists. No verifiable evidence of how artifacts were built.
- **Level 1**: The build platform must automatically generate provenance describing how the artifact was built. Provenance must be available for distribution to consumers. Documentation of the build process.
- **Level 2**: Adds hosted builds and digital signatures to prevent tampering. Build service is authenticated, provenance is signed.
- **Level 3**: Enforces platform isolation and confidentiality controls. Hardened build platform. Builds run in isolated environments, preventing one build from influencing another.

## Trade-offs

Higher levels provide better guarantees against supply chain threats but come at higher implementation costs. Lower SLSA levels are designed to be easier to adopt but with only modest security guarantees.

## Related spec pages

- Build Track details: `/spec/v1.2/build-track-basics`
- Threats & mitigations: `/spec/v1.2/threats`
- Provenance formats: `/spec/v1.2/provenance` and `/spec/v1.2/build-provenance`
- Source Track requirements: `/spec/v1.2/source-requirements`
