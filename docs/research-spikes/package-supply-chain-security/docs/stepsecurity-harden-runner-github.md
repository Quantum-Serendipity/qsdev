<!-- Source: https://github.com/step-security/harden-runner -->
<!-- Retrieved: 2026-05-12 -->

# StepSecurity Harden-Runner: Technical Overview

## Core Function
Harden-Runner operates as a CI/CD security agent that works like an EDR for GitHub Actions runners. It monitors network egress, file integrity, and process activity on those runners, detecting threats in real-time.

## What It Monitors

Three primary dimensions:
1. **Network Egress** - All outbound network connections
2. **File Integrity** - Source code modifications and file writes during builds
3. **Process Activity** - Execution and arguments of processes during workflows

Each event correlates to the specific workflow step, job, and run that generated it.

## Operating Modes

**Audit Mode** - Observes and reports activity without enforcement; generates security baselines from historical workflow data

**Block Mode** - Actively prevents unauthorized outbound connections using allowlisted domains; enforces policies based on established baselines

## Technical Architecture

### GitHub-Hosted Runners
```yaml
- name: Harden Runner
  uses: step-security/harden-runner@v2.17.0
  with:
    egress-policy: audit
```

### Self-Hosted Runners
The agent deploys as infrastructure component (VM image inclusion or service installation) without workflow modifications needed.

### Kubernetes/ARC
Deploys as a DaemonSet for Actions Runner Controller environments.

## Pricing Structure

**Community (Free):**
- Automated baseline creation from past workflow behavior
- Anomaly detection for outbound calls
- Domain allowlist blocking
- Source code modification detection
- Public repository support only

**Enterprise (Paid):**
- Private repository support
- Self-hosted runner monitoring
- GitHub Checks integration
- File write visibility with process correlation
- Process name/argument tracking
- Minimum GITHUB_TOKEN permission recommendations

## Configuration
Users configure via action parameters. The system automatically builds behavioral baselines without manual rule creation, then flags deviations as potential threats.

## Real-World Detection Success
- Compromised axios npm package dropping remote access trojans
- Trivy v0.69.4 malicious release
- tj-actions/changed-files supply chain attack (CVE-2025-30066)
- NX build system compromise
- Anomalous api.ipify.org traffic across multiple customers

## Deployment Scale
Secures over 25 million CI/CD workflow runs weekly across 11,000+ projects, including usage by Microsoft, Google, CISA, Kubernetes, and AWS.

## Limitations
- Windows/macOS support: audit mode only
- Block mode only available on Linux runners
- GitHub Actions specific (not directly portable to other CI systems)
