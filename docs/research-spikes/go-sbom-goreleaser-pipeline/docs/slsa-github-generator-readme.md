# SLSA GitHub Generator README

- **Source URL**: https://github.com/slsa-framework/slsa-github-generator/blob/main/README.md
- **Retrieved**: 2026-05-15

---

## Overview

The slsa-github-generator provides free tools to generate and verify SLSA Build Level 3 provenance for native GitHub projects using GitHub Actions. It enables developers to build software securely while protecting against supply chain attacks.

## What is SLSA?

Supply-chain Levels for Software Artifacts defines an incrementally adoptable framework with compliance levels to prevent tampering, improve integrity, and secure packages and infrastructure.

## Provenance Concept

Provenance represents metadata about software artifact creation, including source code, build systems, build steps, and initiation details. It enables users to determine the authenticity and trustworthiness of software artifacts.

## Supported Builders

| Ecosystem | Builder | Status |
|-----------|---------|--------|
| Go | Go Builder | Stable since v1.0.0 |
| Node.js | Node.js Builder | Beta since v1.6.0 |
| Maven | Maven builder | Beta since v1.9.0 |
| Gradle | Gradle builder | Beta since v1.9.0 |
| Bazel | Bazel builder | Work in Progress |
| Docker | Container-based Builder | Beta since v1.7.0 |

## Go Builder Details

The Go Builder specifically builds and generates provenance for Go projects and has maintained stable status since initial release.

## SLSA Level Achievement

These tools meet provenance generation and isolation strength requirements for SLSA Build level 3 and above. However, the workflows alone don't address provenance distribution or verification.

## Referencing Requirements

GitHub Actions MUST be referenced by tag for slsa-verifier compatibility, specifically as `@vX.Y.Z` format rather than shorter tag variations.

## Verification

Users verify provenance using the separate slsa-verifier project (https://github.com/slsa-framework/slsa-verifier) with CLI tools supporting multiple artifact types.

## Limitations

The framework does not address provenance distribution or verification independently -- these require complementary tools. Known issues include TUF remote mirror errors in versions up to v1.9.0 (fixed in v1.10.0).
