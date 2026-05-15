<!-- Source: https://docs.sigstore.dev/logging/overview/ -->
<!-- Retrieved: 2026-05-15 -->

# Rekor Transparency Log Overview

## Core Purpose

Rekor is described as providing "an immutable, tamper-resistant ledger of metadata generated within a software project's supply chain." It enables organizations to "record signed metadata to an immutable record" that other parties can query for trust decisions.

## Key Capabilities

The system functions as a "restful API-based server for validation, and a transparency log for storage." Users can leverage a CLI application to:
- Create and authenticate entries
- Query the log for inclusion verification
- Confirm log integrity
- Retrieve entries by public key or artifact

## Transparency and Auditing

Rekor operates on a verifiable data structure foundation, allowing auditors to monitor whether the ledger remains append-only without mutation or deletion. Verifiers can also monitor the log for their identities.

Two auditing approaches are mentioned:
1. **Rekor Monitor** - A GitHub Actions-based tool for consistency checks
2. **Omniwitness** - Created by Trillian's team for log auditing

## Availability

A public instance runs at rekor.sigstore.dev with a stated 99.5% availability SLO and dedicated oncall monitoring.

The system is intentionally designed as "extendable to working with different manifest schemas and PKI tooling" and can operate independently from Sigstore's broader infrastructure.
