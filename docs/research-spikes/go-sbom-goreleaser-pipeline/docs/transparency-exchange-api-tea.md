# The Transparency Exchange API (TEA): Standardizing SBOM Discovery

- **Source**: https://sbomify.com/2026/03/01/why-were-bullish-on-tea/
- **Retrieved**: 2026-05-15

## Core Concept

The Transparency Exchange API represents a fundamental shift in how supply chain transparency artifacts are shared. Rather than relying on manual processes, TEA enables "automated discovery and exchange of SBOMs, VEX, and other" artifacts through standardized mechanisms.

## How TEA Works

At TEA's foundation is the **Transparency Exchange Identifier (TEI)**, a URN scheme built on DNS architecture. This design allows vendors to wrap existing product identifiers like EAN/UPC codes, PURLs, or CPEs within the TEI framework.

The discovery mechanism operates elegantly: given a TEI, a client automatically resolves it through DNS, discovers the API endpoint via the IETF `.well-known` namespace, and retrieves transparency artifacts -- all without manual intervention. As the article explains, "no portal logins. No email requests. No hunting through vendor websites."

## Supported Artifact Types

TEA supports far more than just SBOMs. The specification accommodates various xBOM formats (hardware, AI/ML, SaaS variants), vulnerability information (VDR and VEX), attestations, and product lifecycle events through Common Lifecycle Enumeration (CLE).

Critically, TEA maintains **format agnosticism** -- supporting both SPDX and CycloneDX SBOMs equally, alongside OpenVEX and CycloneDX VEX formats.

## Standardization Timeline

TEA development occurs within **ECMA TC54 Task Group 1**, the same committee that standardized CycloneDX (ECMA-424) and Package-URL (ECMA-427). The standard is currently in Beta 2, with consumer API implementation ready. The roadmap includes ISO submission for international recognition.

## The Problem It Solves

Current SBOM sharing involves "email chains, NDAs, portal logins, manual downloads" -- a fundamentally broken workflow that regulatory requirements now demand fix.
