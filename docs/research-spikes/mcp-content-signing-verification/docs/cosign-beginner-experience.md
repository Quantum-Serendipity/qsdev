<!-- Source: https://code.mendhak.com/understanding-sigstore-cosign-as-a-beginner/ -->
<!-- Retrieved: 2026-05-14 -->

# Cosign Experience: Non-Container Artifact Signing Analysis

## What Worked Well

**Signing Process**: The initial signing workflow proved straightforward. As the author noted, "Signing a text file was easy, using the sign-blob subcommand." The keyless approach through OIDC providers (GitHub, Google, Microsoft) eliminated manual key management overhead.

**Automated CI/CD Integration**: The author identified this as the strongest use case. GitHub Actions integration with built-in OIDC token support meant signing happened without user interaction, and importantly, "This workflow _is_ private, because the identifier is the Github Action URL."

## Major Pain Points

**Verification Complexity**: The verification experience proved unexpectedly difficult. Multiple issues emerged:
- Missing the `--new-bundle-format` flag blocked Python releases verification
- Determining correct certificate identity and OIDC issuer values required extensive research
- Different ecosystems (npm, GitHub, Python) required separate tools or hidden verification methods

**Security-UX Tradeoff**: While identity/issuer parameters existed "specifically to mitigate a security risk," the author observed users would resort to regex workarounds like `'.*'` patterns, echoing problematic patterns from certificate validation disabling.

**Privacy Concerns**: Email addresses from identity providers appeared in transparency logs. The author noted discomfort with this exposure and acknowledged "there aren't any convenient solutions" currently available.

## Blob vs. Container Signing

**Blob Signing**: Cleaner implementation with simpler transparency logs and straightforward verification (when parameters were discovered).

**Container Signing**: Created practical issues — the system generated numerous digest-based layer tags that cluttered registries. The author felt "put off" as users couldn't "control which tags are available for download."

## Key Management Challenges

Local key signing worked but felt redundant: "this isn't too far off from just using `openssl` to sign artifacts." The inability to import ed25519 keys and link to GitHub-hosted key URLs limited sophisticated workflows the author envisioned.

## Adoption Barriers

The author identified fragmentation as critical: individual ecosystems hiding verification details behind proprietary tools rather than standardized Cosign commands, poor metadata discoverability, and incomplete package distribution (notably, no official Ubuntu repository despite CI/CD systems predominantly running on Ubuntu).
