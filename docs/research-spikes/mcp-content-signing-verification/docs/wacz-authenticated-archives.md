# WACZ Authenticated Web Archives

- **Source URL**: https://dispatch.starlinglab.org/p/authenticated-web-archives-wacz-files
- **Retrieved**: 2026-05-14

## Signing Architecture

Two-stage signing process:

1. **Browsertrix Signing**: Initial signature applied during web capture
2. **Starling Integrity Pipeline Signing**: Secondary signature applied during archival

## Cryptographic Mechanisms

**Hash-based Verification**: The `datapackage.json` file contains a cryptographic hash of the data from each page, creating a fingerprint of the captured content.

**Signature Method**: Authentication uses X.509 SSL certificates used to verify domains, providing identity verification that the signing entity is legitimate.

## Integrity Verification Chain

The `datapackage-digest.json` file stores a cryptographic signature created from the `datapackage.json` file, enabling detection of unauthorized modifications to archived data.

## Supporting Infrastructure

**Starling Integrity Pipeline Components**:
- Integrity preprocessor: Prepares files from web crawlers
- Integrity backend: Configurable authentication and preservation across distributed storage and on various blockchains

## Key Tools & Developers

- **Webrecorder**: Created the WACZ specification
- **Starling Lab**: Maintains the authentication pipeline
- **ReplayWeb.page**: Primary viewing/verification tool

The system emphasizes preservation through cryptographic identity binding.
