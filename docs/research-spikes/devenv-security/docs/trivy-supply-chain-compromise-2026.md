<!-- Source: https://github.com/aquasecurity/trivy/security/advisories/GHSA-69fq-xp46-6x23 -->
<!-- Retrieved: 2026-05-12 -->

# Trivy Supply Chain Compromise (March 2026)

## What Happened
On March 19-22, 2026, attackers exploited compromised credentials to inject malicious code across multiple Trivy ecosystem components. The threat actors published poisoned releases, hijacked version tags, and distributed credential-stealing malware designed to exfiltrate secrets from CI/CD pipelines.

## Affected Components & Versions

**Trivy Binary/Images:**
- v0.69.4 (March 19, ~3-hour window)
- v0.69.5 and v0.69.6 Docker images (March 22-23, ~10 hours)

**GitHub Actions:**
- `aquasecurity/trivy-action`: versions prior to 0.35.0
- `aquasecurity/setup-trivy`: versions prior to 0.2.6

**Safe Versions:**
- Trivy binary: v0.69.2 or v0.69.3
- trivy-action: v0.35.0
- setup-trivy: v0.2.6

## Attack Timeline

| Component | Compromise Start | Resolution | Duration |
|-----------|------------------|-----------|----------|
| trivy v0.69.4 | March 19, 18:22 UTC | ~21:42 UTC | ~3 hours |
| trivy-action tags | March 19, ~17:43 UTC | March 20, ~05:40 UTC | ~12 hours |
| setup-trivy tags | March 19, ~17:43 UTC | ~21:44 UTC | ~4 hours |
| Docker images v0.69.5/0.69.6 | March 22, 15:43 UTC | March 23, ~01:40 UTC | ~10 hours |

## Attack Mechanics

**Memory Extraction:** Dumps Runner.Worker process memory via /proc/<pid>/mem to extract secrets
**Credential Harvesting:** Scanned 50+ filesystem paths targeting SSH keys, cloud credentials (AWS/GCP/Azure), Kubernetes tokens, Docker configs, environment files, and cryptocurrency wallets
**Data Exfiltration:** Encrypted collected data using hybrid encryption (AES-256-CBC with RSA-4096) and transmitted to attacker infrastructure

## Root Cause
Credential rotation following an earlier February 2026 incident was non-atomic — not all secrets revoked simultaneously. This allowed attackers to exfiltrate newly-rotated credentials during the rotation window, maintaining persistent access.

## Remediation
1. Update to patched versions immediately
2. Rotate all potentially-exposed secrets
3. Audit artifact downloads from March 19-20, 2026
4. Pin GitHub Actions to immutable SHA hashes rather than version tags
5. Verify installations using sigstore signatures and cosign

## Lessons Learned
- Non-atomic credential rotation created extended exposure windows
- GitHub's immutable releases feature (enabled March 3) protected v0.69.3
- Attackers compromised multiple distribution channels simultaneously
- Detection required ~3-4 hours despite high-visibility repositories
- Pin actions to SHA hashes, not mutable version tags
