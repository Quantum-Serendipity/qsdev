<!-- Source: Multiple sources (web search aggregation) -->
<!-- Retrieved: 2026-05-15 -->
<!-- Key sources:
  - https://www.microsoft.com/en-us/security/blog/2026/03/24/detecting-investigating-defending-against-trivy-supply-chain-compromise/
  - https://www.paloaltonetworks.com/blog/cloud-security/trivy-supply-chain-attack/
  - https://www.aquasec.com/blog/trivy-supply-chain-attack-what-you-need-to-know/
  - https://www.crowdstrike.com/en-us/blog/from-scanner-to-stealer-inside-the-trivy-action-supply-chain-compromise/
  - https://github.com/aquasecurity/trivy/security/advisories/GHSA-69fq-xp46-6x23
-->

# Trivy Supply Chain Compromise - March 2026

## Overview

On March 19, 2026, a threat actor known as TeamPCP compromised Aqua Security's Trivy vulnerability scanner -- the most widely adopted open-source scanner in the cloud-native ecosystem.

## What Was Compromised

The attack simultaneously compromised:
- The core scanner binary (malicious v0.69.4 released)
- The trivy-action GitHub Action (76 of 77 version tags force-pushed)
- The setup-trivy GitHub Action (all 7 tags hijacked)

## Attack Mechanism

The payload was injected into entrypoint.sh and executed before the legitimate Trivy scan. Pipelines appeared to work normally while the stealer ran silently underneath. Stolen NPM tokens were used to download legitimate packages, inject a malicious preinstall hook, bump the patch version, and republish -- turning each compromise into a new supply chain vector.

## Root Cause

Attackers exploited a misconfiguration in Trivy's GitHub Actions environment in late February 2026, extracting a privileged access token. Subsequent rotation was not fully comprehensive, allowing residual access.

## Second Wave

Three days later on March 22, the attacker pushed additional malicious Docker Hub images (v0.69.5, v0.69.6, and latest), extending the exposure by another ~10 hours.

## Impact on SBOM Ecosystem

Multiple sources now discourage using Trivy in CI/CD pipelines. The incident is widely cited as a reason to prefer alternative tools like Syft for SBOM generation in automated pipelines.
