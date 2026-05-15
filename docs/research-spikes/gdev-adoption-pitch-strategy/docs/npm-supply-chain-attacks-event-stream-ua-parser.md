<!-- Source: https://www.rescana.com/post/in-depth-analysis-supply-chain-poisoning-of-popular-npm-packages-exploiting-event-stream-ua-parser/ -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: This source focuses on technical analysis; specific financial cost figures for event-stream and ua-parser-js are not provided. Supplementary data from other sources used in research. -->

# npm Supply Chain Attack Analysis: event-stream and ua-parser-js

## Attack Mechanism

Both packages were compromised through **code poisoning during update cycles**. The attacks involved:

- Injecting malicious backdoors and command execution payloads into legitimate updates
- Using "advanced code obfuscation and modified digital signatures to evade detection"
- Exploiting "cryptographic weaknesses and simulating authentic package version sequencing"
- Embedding modifications that "seamlessly integrate into development workflows"

## Affected Versions

**event-stream:** versions 3.3.3 through 3.3.4, where the attack exploited historical trust through "subtle tampering"

**ua-parser-js:** versions 0.7.21 through 0.7.23, exhibiting "apparent deviation in parsing logic that suggests malicious intervention"

## Threat Actors

APT34 ("OilRig") and Wizard Spider -- both targeting "finance, government, telecommunications, technology, healthcare, and critical infrastructure"

## Lessons Learned

Organizations should implement rigorous code auditing, cryptographic verification, continuous network monitoring, developer security training, and adopt zero-trust approaches within development lifecycles.

## Supplementary Context (from search results)

**event-stream (2018):** The attack involved a long dwell time and surgical targeting of a specific application (Copay bitcoin wallet), with the attacker hiding an extra dependency called flatmap-stream. The attacker social-engineered maintainer access by volunteering to help maintain the package.

**ua-parser-js (October 2021):** The ua-parser-js package had over 7 million downloads per week. The threat actor hijacked the author's NPM account and published three malicious versions. The malware dropped a DLL that steals credentials from over 100+ popular Windows applications. The compromise lasted approximately four hours before detection. The package is used by major companies including Microsoft, Google, Amazon, and Facebook.
