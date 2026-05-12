<!-- Source: https://rubycentral.org/news/securing-rubys-future-how-ruby-central-is-strengthening-security/ -->
<!-- Retrieved: 2026-05-12 -->

# Ruby Central's Security Strengthening Efforts

## Overview
Ruby Central has implemented a comprehensive security approach for the Ruby ecosystem, which processes billions of monthly downloads across over 180,000 gems on RubyGems.org.

## Key Security Initiatives

**AWS-Funded Sigstore Integration**
AWS funds Samuel Giddins' full-time security position, with focus on integrating Sigstore into RubyGems, Bundler, and RubyGems.org. This technology enables developers to "securely sign and verify packages without relying on long-lived" signing keys, preventing tampering and ensuring software integrity.

**Vulnerability Scanning**
Mend.io operates continuous automated gem scanning to detect vulnerabilities in newly updated packages. Additionally, the RubyGems team performs manual malware reviews of published and updated gems to identify malicious code.

**Bundler Lockfile Checksums**
Developed over two years with Sovereign Tech Agency (STA) funding, this feature ensures development packages match production environments, preventing supply chain attacks.

**Trusted Publishing**
Built on OpenID Connect technology, this mechanism restricts gem publication to verified users only, reducing unauthorized or malicious publication risks.

**MFA Reinforcement**
Following CVE-2024-21654's discovery in December 2023, Ruby Central overhauled multi-factor authentication systems. Updates included stronger prompts aligned with "OWASP security guidelines" and tightened enforcement requiring additional authentication for sensitive actions.

**Infrastructure Security**
STA funding enabled Kubernetes platform upgrades and OpenSearch cluster improvements. Implementation of Datadog Cloud Security Management provides real-time vulnerability monitoring.

## Security Audit Results
Trail of Bits' Alpha-Omega Project audit identified 33 findings, including 7 medium-severity and 1 high-severity issue, with recommendations addressing infrastructure security, code quality, and access controls.

## Future Direction
Ruby Central plans to unveil supply chain transparency features allowing users to verify gem builders and detect tampering, alongside continued operational resilience investments.
