<!-- Source: https://arxiv.org/html/2511.20313v1 -->
<!-- Retrieved: 2026-05-15 -->

# A Reality Check on SBOM-based Vulnerability Management: An Empirical Study and A Path Forward

## False Positive Rate Crisis

The research reveals a staggering 97.5% false positive rate across vulnerability scanners. Testing four repositories (one per language), the study found that of 81 reported vulnerabilities, only 2 were confirmed as genuine threats.

## Tested Scanners

Two primary vulnerability scanners were evaluated:
- **Grype** (paired with Syft SBOM generator)
- **Trivy** (used for both SBOM generation and scanning)

## Root Cause of False Positives

The primary culprit is unreachable code. "Flagging of vulnerabilities within unreachable code" causes the overwhelming false alarm rate. Package-level matching identifies vulnerable library versions present in dependencies without analyzing whether those vulnerable functions are actually invoked by the application.

## Accuracy Improvements

Function call analysis successfully eliminated 63.3% of false positives, reducing required manual inspection from 81 to 31 alerts. This coverage-based filtering demonstrates significant potential for refinement.

## SBOM Generation Recommendations

Lock files trump project files. The study found that using lock files from strong package managers (Poetry for Python, Cargo for Rust, Bundler for Ruby, Composer for PHP) produces 100% accuracy and consistency between Syft and Trivy across all languages tested.

## Go-Specific Finding

The research notes that existing SBOM generators accept go.sum files "only for early Golang versions," suggesting compatibility limitations with contemporary Go projects. This makes go.mod the primary artifact for Go SBOM generation.

## Key Takeaway

SBOM-based vulnerability scanning without reachability analysis produces an unacceptably high false positive rate. Tools like govulncheck (for Go) that perform call graph analysis are essential complements to SBOM-based scanning.
