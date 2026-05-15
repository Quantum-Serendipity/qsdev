<!-- Source: https://slsa.dev/spec/v1.1/faq -->
<!-- Retrieved: 2026-05-15 -->

# SLSA Frequently Asked Questions (Spec v1.1)

## Key Questions and Answers

### Why SLSA Isn't Transitive
SLSA Build levels evaluate individual artifact trustworthiness independently, not requiring dependencies to meet identical levels. This approach enables "parallel progress and prioritization based on risk" rather than forcing backward work through the entire supply chain.

### Reproducible Builds
The FAQ distinguishes between "reproducible" (bit-for-bit identical outputs) and "verified reproducible" (corroborated by independent platforms). While verified reproducible builds can satisfy SLSA requirements, they don't address "source, dependency, or distribution threats" and aren't always practical for closed-source projects.

### SLSA and in-toto Relationship
in-toto provides an "unopinionated layer to express information pertaining to a software supply chain," while SLSA functions as the "opinionated layer specifying exactly what information must be captured" for specific assurance levels. SLSA Provenance uses in-toto attestations as its vehicle.

### SLSA vs. SBOM Comparison
SBOMs focus on "understanding software to evaluate risk through known vulnerabilities and license compliance," while SLSA Provenance emphasizes "trustworthiness of the build process." The documents operate at different abstraction levels:
- **SBOMs** provide fine-grained component data (what's inside the artifact)
- **SLSA Provenance** coarsely describes build parameters (how the artifact was built)

They are complementary, not competing.

### Self-Hosted Runners
Requirements apply to "the transitive closure of the systems which are responsible for informing the provenance generated," whether the platform, runner, or both generates provenance data.
