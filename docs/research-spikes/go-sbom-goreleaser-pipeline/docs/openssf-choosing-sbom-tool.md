<!-- Source: https://openssf.org/blog/2025/06/05/choosing-an-sbom-generation-tool/ -->
<!-- Retrieved: 2026-05-15 -->

# OpenSSF Guidance on SBOM Generation Tools (June 2025)

## Selection Criteria

The OpenSSF recommends evaluating tools based on language support, dependency accuracy (including transitive dependencies), integration with CI/CD pipelines, output format support, and fit within existing workflows.

## Recommended Tools by Category

**Single-Language Applications:**
- **Node.js**: "npm-sbom command is a great choice" and "can produce SBOMs in either of the two most popular specifications: CycloneDX and SPDX"
- **CycloneDX ecosystem tools**: Available for Java (Maven/Gradle), Node.js, Python, and Golang, offering "thorough inventory of all components"
- **General approach**: Language-specific tools "tend to produce the highest-quality SBOMs"

**Multi-Language Applications:**
- **cdxgen** (CycloneDX/OWASP): "Official SBOM generation tool" with broad language support and transitive dependency analysis for certain ecosystems
- **syft** (Anchore): User-friendly with good CI/CD integration, though it "sometimes miss[es] dependencies found by other tools"
- **Tern**: Container-focused, generating "layer-by-layer view," but time-consuming and limited for non-containerized projects

## Format Recommendations

Two primary standards recommended: **CycloneDX** and **SPDX**

## Key Accuracy Consideration

"Imperfect SBOMs are better than no SBOMs." Tools can be enriched post-generation with vulnerability and dependency data.

## Go-Specific Guidance

CycloneDX provides a dedicated Go module tool supporting transitive dependency analysis.
