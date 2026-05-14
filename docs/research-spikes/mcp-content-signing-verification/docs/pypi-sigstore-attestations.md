<!-- Source: https://blog.sigstore.dev/pypi-attestations-ga/ -->
<!-- Retrieved: 2026-05-14 -->

# PyPI's Sigstore-powered Attestations Overview

## Core Functionality
The implementation enables developers to upload Sigstore-based attestations to PyPI, following the PEP 740 standard. The feature reached general availability in November 2024.

## How It Works
The system leverages existing infrastructure: "if a project uses Trusted Publishing and the canonical GitHub Action then they'll produce attestations by default, with no changes required." Both mechanisms rely on OpenID Connect foundations, enabling keyless signing without manual key management.

## Key Technical Architecture
The approach utilizes shared building blocks between Trusted Publishing and Sigstore's OpenID Connect implementation. This means maintainers benefit from "Sigstore's key transparency and auditability properties" without managing cryptographic keys themselves.

## Adoption Metrics
Early adoption has been notable: over 20,000 attestations uploaded to PyPI, with approximately 5% of the top 360 projects already publishing attested versions. The blog references a tracking tool called "Are we PEP 740 yet?" for monitoring ecosystem progress.

## Notable Gap
The provided content doesn't specify technical details about verification processes, included metadata, or lessons learned — those details reportedly appear in the linked PyPI and Trail of Bits blogs rather than this summary announcement.
