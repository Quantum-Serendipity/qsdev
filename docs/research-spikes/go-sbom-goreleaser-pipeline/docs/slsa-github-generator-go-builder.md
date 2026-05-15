<!-- Source: https://github.com/slsa-framework/slsa-github-generator -->
<!-- Retrieved: 2026-05-15 -->

# SLSA GitHub Generator

## Purpose and Capabilities

The SLSA GitHub Generator is a toolset enabling projects to generate non-forgeable provenance using GitHub Actions. It helps achieve "SLSA Build Level 3 and above" compliance by automating secure build processes that protect against supply chain tampering.

## What is SLSA?

Supply-chain Levels for Software Artifacts (SLSA) is a security framework, a checklist of standards and controls to prevent tampering across software supply chains. It defines incrementally adoptable security levels.

## Supported Builders

The project provides ecosystem-specific builders:

| Ecosystem | Builder | Status |
|-----------|---------|--------|
| Go | Go Builder | Stable since v1.0.0 |
| Node.js | Node.js Builder | Beta since v1.6.0 |
| Maven | Maven Builder | Beta since v1.9.0 |
| Gradle | Gradle Builder | Beta since v1.9.0 |
| Bazel | Bazel Builder | Work in Progress |
| Docker | Container Builder | Beta since v1.7.0 |
| Generic Files | Generic Generator | Stable since v1.2.0 |

## Key Features

**Provenance Generation**: The tools produce tamper-proof metadata documenting how artifacts were created, including source code, build system, and build steps.

**Builder Requirements**: MUST be referenced by tag (vX.Y.Z format) for slsa-verifier to verify the trusted builder's reusable workflow.

## Go Builder

The Go builder specifically enables building and generating provenance for Go projects, meeting SLSA Level 3 requirements through isolation and non-forgeable attestation.

## Verification

Projects use the separate "slsa-framework/slsa-verifier" tool to verify generated provenance, confirming artifacts were created through the secure builder workflow.

## Notable Adopters

Over 30 major projects use this generator, including Flask, Kubernetes components, Docker tools, and Google projects.
