# JFrog Curation: Preventive Supply Chain Security for Open-Source Packages

- **Source URL**: https://www.securityscientist.net/blog/12-questions-and-answers-about-jfrog-curation-jfrog/
- **Retrieved**: 2026-05-12

## Core Function
JFrog Curation operates as a preventive gatekeeper that "blocks malicious, vulnerable, or non-compliant open-source packages at the point of consumption — before they enter your software development lifecycle." Unlike reactive scanning tools, it intercepts requests at the earliest possible moment.

## How It Integrates with Artifactory

Curation sits at the **remote repository proxy layer** within JFrog Artifactory. When developers request packages via `npm install` or `pip install`, the request flows through Artifactory's proxy before package delivery. Curation analyzes metadata at this interception point and either blocks or allows the request—no separate scanning infrastructure required.

## Supported Package Ecosystems
- npm (JavaScript)
- PyPI (Python)
- Maven (Java)
- Go modules
- Additional ecosystems expanding

## Policy Types

**Security Policies:**
- CVE blocking (configurable CVSS thresholds, commonly ≥7.0)
- Malicious package detection via threat intelligence
- Known exploited vulnerabilities

**Operational Risk Policies:**
- Unmaintained package blocking
- New packages lacking community trust signals
- Packages without security/disclosure processes

**Compliance Policies:**
- Open-source license restrictions (copyleft, restrictive terms)

Policies apply per repository, enabling tiered enforcement across different projects.

## Malicious Package Detection

JFrog's Security Research team maintains a database of 1,500+ documented malicious packages, identifying:
- Typosquatting attempts
- Dependency confusion payloads
- Post-publication code injections

## Developer Experience

When packages are blocked, Curation provides **suggested compliant alternatives** rather than just denying access. Organizations can deploy in observe-only mode initially to calibrate policies before enforcement.
