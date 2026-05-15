<!-- Source: https://anchore.com/sbom/grype-support-cyclonedx-spdx/ -->
<!-- Retrieved: 2026-05-15 -->

# Grype's CycloneDX and SPDX Support

## Supported Formats

Grype added support for both CycloneDX and SPDX standards alongside its native Syft lossless SBOM format. "Grype is the first open source vulnerability scanner that supports both SPDX and CycloneDX at the time of writing this." (March 2022)

## Key Capability

The tool enables scanning SBOMs directly for vulnerabilities rather than requiring initial file identification. This approach is "incredibly fast" because it "skip[s] over that identification step by using an SBOM."

## Important Limitations (at time of writing)

The article explicitly warned that support was "very new" at publication. The authors acknowledged: "There are going to be bugs and difficulties scanning SPDX and CycloneDX SBOMs." They encouraged community contributions through bug reports and pull requests.

Note: This article is from March 2022. Grype's SBOM scanning has matured significantly since then.
