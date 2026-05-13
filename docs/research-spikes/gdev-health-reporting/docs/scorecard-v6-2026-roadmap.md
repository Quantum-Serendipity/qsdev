<!-- Source: https://github.com/ossf/scorecard/pull/4952 -->
<!-- Retrieved: 2026-05-12 -->

# Scorecard v6: 2026 Roadmap Summary

## Core Vision
Scorecard is evolving from a scoring tool into an "open source security evidence engine." The proposal reframes the project's mission around producing trusted, structured security evidence for the ecosystem rather than numeric scores alone.

## OSPS Baseline Conformance
The primary 2026 initiative adds conformance evaluation against the OSPS Baseline as the first proof-of-concept for this new architecture. Rather than replacing existing checks, conformance operates as a parallel evaluation layer -- both scoring (0-10) and compliance labels coexist in single-run output.

## Output Format Expansion
Phase 1 deliverables include multiple interoperable formats:
- Enriched JSON (enhanced from current format)
- in-toto attestations with new `scorecard.dev/evidence/v0.1` predicate
- Gemara SDK formats for multi-tool consumption
- OSCAL Assessment Results for standards alignment

The existing `scorecard.dev/result/v0.1` predicate for check-based scoring is preserved as an additive change.

## Evaluation Model Changes
Rather than PASS/FAIL/UNKNOWN replacing scores, the proposal establishes a three-tier architecture where probe evidence feeds into parallel evaluation layers -- one producing check scores, another producing conformance labels (PASS/FAIL/UNKNOWN/NOT_OBSERVABLE).

## Phased Delivery
**Phase 1** focuses on OSPS Baseline Level 1 conformance with CLI and GitHub Action support, plus probe catalog extraction. **Phase 2** addresses release integrity and Level 2 controls. **Phase 3** tackles enforcement detection and attestation mechanisms -- explicitly excluding policy enforcement itself.

## Ecosystem Integration
Design emphasizes interoperability: Scorecard detects security signals but doesn't enforce policies, positioning it alongside complementary tools like Darnit, AMPEL, Minder, and Allstar for a complete security posture framework.
