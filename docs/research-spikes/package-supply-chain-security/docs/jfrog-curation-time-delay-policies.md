# How to Use Curation Time-Delay Policies to Block Package Hijacks — JFrog Academy

- **Source URL**: https://academy.jfrog.com/how-to-use-curation-time-delay-policies-to-block-package-hijacks
- **Retrieved**: 2026-05-12

## Core Function

JFrog Curation's immature package policy implements a protective mechanism by "blocking new packages under a set age (such as 14 days)." This addresses the critical "attacker's window" — the period between a malicious package's release and its detection, which can extend up to 14 days.

## How It Works

The policy operates proactively rather than reactively. When developers request a blocked package that hasn't reached the minimum age threshold, the system "seamlessly substitutes it with a safe, older version" instead of denying access entirely.

## Security Approach

The strategy shifts organizations from "reactive scanning to the power of proactive prevention" by eliminating the window attackers exploit. The course references the September 2025 NPM supply chain crisis where 26 packages were compromised, including the `chalk` package converted to cryptocurrency-stealing malware, illustrating the real-world threat this policy addresses.

## Additional Defense Layer

While time-delay policies provide preventive protection, JFrog Xray serves as complementary infrastructure, functioning as "continuous security radar" to identify and remediate malicious components that bypass initial defenses.
