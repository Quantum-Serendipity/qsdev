# How Cargo-Vet Works
- **Source**: https://mozilla.github.io/cargo-vet/how-it-works.html
- **Retrieved**: 2026-05-12

## Core Audit Model

Cargo-vet establishes a decentralized system for verifying third-party Rust dependencies. The fundamental principle emphasizes minimizing friction: "the driving principle behind cargo-vet is to minimize friction and make it as easy as possible to do the right thing."

The audit process verifies that new code has received approval from trusted organizations before integration. When developers attempt adding dependencies, the system "analyzes the updated build graph to verify that the new code has been audited by a trusted organization."

## Exemptions System

New projects begin without requiring retroactive audits of existing dependencies. During initialization, current third-party code is "automatically added to the exemptions list," enabling teams to adopt cargo-vet without addressing legacy backlog immediately. This approach facilitates "tackling the backlog incrementally from an approved state."

## Audit Criteria and Storage

The system supports customizable audit criteria for different security requirements. Significantly, "audits are stored in-tree," meaning developers maintain audit records directly within their repositories rather than external systems. This design choice allows audit submissions to accompany code changes naturally.

## Importing and Sharing Audits

Organizations can leverage shared work through imports. The mechanism operates by "pointing directly to the audit files in external repositories," with a registry serving as an index for well-known organizations' audits. Crucially, "imports used to vet the dependency graph are always fetched directly from the relevant organization, and only after explicitly adding that organization to the trusted set."

## Verification Process

Cargo-vet operates as a linter within continuous integration pipelines, automatically refusing patches lacking appropriate audits. The system assists developers by scanning registries for pre-existing audits from established organizations, offering options to "add that organization to the project's trusted imports."

## Practical Implementation

Organizations can implement differential audits, examining code changes rather than complete packages. The system supports "custom audit criteria, configurable policies for different subtrees in the build graph, and filtering out platform-specific code," though these remain optional for basic usage.
