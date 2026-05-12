# The 2026 State of Package Registry Provenance: Who Is Signing What?
- **Source**: https://zenn.dev/sqer/articles/e4df3d397f5651?locale=en
- **Retrieved**: 2026-05-12

## SLSA Levels Achieved

**npm & PyPI**: Both achieve **L3** (full signer identity + transparency logs via Sigstore)

**GitHub Releases**: **L3** with language-agnostic support

**Maven Central**: **L3-capable** but with limited adoption (Sigstore opt-in; PGP mandatory)

**crates.io, Go, NuGet**: **L1** (checksums only; no registry-level provenance)

## Adoption Statistics

| Ecosystem | Signing Coverage | Key Infrastructure |
|-----------|------------------|-------------------|
| **npm** | ~7% | Provenance automatically granted by default via Trusted Publishing |
| **PyPI** | ~17% (132,360+ packages) | Sigstore; Trusted Publishers GA |
| **Maven Central** | <1% Sigstore | PGP universal; Sigstore added January 2025 |
| **crates.io** | 0% | Trusted Publishing GA; Sigstore RFC pending |
| **Go** | 0% registry-level | Checksum transparency only (sum.golang.org) |

## Consumer Enforcement Capabilities

**npm/PyPI**: Allows verification of Sigstore attestation via CLI; dedicated attestation APIs enable automated verification

**GitHub Releases**: Native verification through GitHub Actions and `gh attestation verify`

**crates.io/Go**: Lock file checksums provide integrity but no identity verification

**Maven Central**: PGP key IDs available; no clean identity chain for Sigstore bundles

## Key Finding

"Sigstore (Fulcio + Rekor) is becoming the universal provenance layer. Ecosystems that adopt it immediately achieve L3. The remaining difference is not capability, but adoption rate."
