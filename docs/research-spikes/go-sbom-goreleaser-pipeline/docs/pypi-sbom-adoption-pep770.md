# SBOM Adoption on PyPI Is at 1.58% - PEP 770 Analysis

- **Source**: https://sbomify.com/2026/03/12/pypi-sbom-analysis/
- **Retrieved**: 2026-05-15

## Overview

Sbomify conducted a comprehensive analysis of SBOM adoption across Python's package ecosystem, examining over 15,000 top packages on PyPI.

## The PyPI-TEA Bridge

The analysis was enabled by PyPI-TEA, an open-source tool that serves as "a bridge between PyPI and the Transparency Exchange API." This utility fetches wheel files and checks the `.dist-info/sboms/` directory as specified by PEP 770, returning SBOM links and hashes in TEA format.

## PEP 770 and Distribution Standards

PEP 770 establishes a standardized location for SBOMs within Python packages. Rather than scattering SBOMs across various directories, the standard designates `.dist-info/sboms/` as the canonical location, making package ecosystems discoverable and uniform.

## Adoption Statistics

| Metric | Result |
|--------|--------|
| Packages with SBOMs | 238 of 15,021 (1.58%) |
| Wheels with SBOMs | 5,679 of 67,151 (8.46%) |
| SPDX implementations | 0 |
| Invalid SBOMs | 37 (2.5%) |

## Key Observations

**Format Dominance**: "Every SBOM across all 15,021 packages is CycloneDX JSON." No SPDX SBOMs were discovered.

**Version Stagnation**: The majority of implementations use older CycloneDX versions (1.4 and 1.5), with only one CycloneDX 1.6 SBOM identified.

**Quality Issues**: All 37 invalid SBOMs traced to a single bug in cargo-cyclonedx.

**Discrepancy Pattern**: Wheel-level adoption (8.46%) significantly exceeds package-level adoption (1.58%), reflecting that packages producing SBOMs typically publish multiple platform-specific wheels.

## Call for Improvement

Implementing PEP 770 SBOMs requires minimal effort — less than 10 lines of configuration in CI pipelines.
