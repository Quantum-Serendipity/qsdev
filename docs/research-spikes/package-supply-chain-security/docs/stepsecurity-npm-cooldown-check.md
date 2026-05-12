# Introducing the NPM Package Cooldown Check — StepSecurity

- **Source URL**: https://www.stepsecurity.io/blog/introducing-the-npm-package-cooldown-check
- **Retrieved**: 2026-05-12

## Overview

StepSecurity introduced the NPM Package Cooldown Check, a GitHub pull request verification tool designed to automatically block dependencies released within a configurable timeframe, typically 2 days (48 hours).

## How It Works

The check operates as an automated status gate in GitHub workflows. When a PR introduces or updates an npm package version, the system verifies the release date. If the package version was published within the cooldown period, the PR fails and displays clear feedback about which dependency is too recent and when the check will automatically pass.

## Core Purpose

The rationale stems from observed patterns in supply chain compromises. As StepSecurity notes, "Most malicious package releases are discovered within the first 24 hours of being published." By enforcing a brief waiting period before adoption, teams create a protective buffer while the security community can identify and flag emerging threats.

## Real-World Examples

StepSecurity cited specific incidents including the NX compromise, eslint vulnerabilities, and the "is" package compromise — cases where malicious code spread rapidly among developers who immediately upgraded to newly released versions.

## Key Features

- **Configurable Window**: Organizations can adjust the cooldown period based on risk tolerance
- **Automatic Resolution**: Once the period elapses, checks pass automatically without manual intervention
- **Emergency Override**: StepSecurity org admins can approve critical security patches that require immediate deployment
- **Integration**: Seamlessly operates within existing GitHub PR workflows

## Broader Ecosystem Protection

The Cooldown Check complements additional security controls, including checks for compromised packages, PWN Request vulnerabilities in GitHub Actions, and script injection flaws in CI/CD pipelines.
