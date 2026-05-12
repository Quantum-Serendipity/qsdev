<!-- Source: https://google.github.io/osv-scanner/ -->
<!-- Source: https://security.googleblog.com/2025/03/announcing-osv-scanner-v2-vulnerability.html -->
<!-- Retrieved: 2026-05-12 -->

# OSV-Scanner Overview

## Core Purpose
OSV-Scanner is a vulnerability detection tool that finds existing vulnerabilities affecting your project's dependencies by connecting to the OSV database.

## How It Works
The tool provides an officially supported frontend to the OSV database that matches a project's dependencies against known vulnerabilities. It leverages an open-source, distributed database model where each advisory comes from an open and authoritative source.

## Usage Modes
1. **CLI Tool**: Can be executed directly in terminals or integrated into CI/CD pipelines
2. **Go Library**: Available as an importable package for Go application integration

## Scanning Capabilities
- Container image scanning (layer-aware for Debian, Ubuntu, Alpine)
- Project source code scanning
- License scanning
- Offline mode operations
- Guided remediation (recommends minimum set of upgrades)

## Supported Languages
C/C++, Dart, Elixir, Go, Java, Javascript, PHP, Python, R, Ruby, Rust

## Package Managers
npm, pip, yarn, maven, go modules, cargo, gem, composer, nuget and others

## Key Features
- **Guided Remediation**: Analyzes dependency graph and recommends minimum set of upgrades needed to resolve vulnerabilities, ranked by dependency depth, severity, and ROI
- **OSV-Reporter**: Reporting functionality
- **Deprecated Package Detection**: Flags outdated packages
- **GitHub Actions Integration**: Native CI/CD workflow support via reusable workflows
- **Interactive HTML Reports**: Visual reporting in V2

## Database Advantages
The system reduces false positives by using machine-readable OSV format that precisely maps onto a developer's list of packages, resulting in fewer, more actionable vulnerability notifications compared to CVE-only databases.

## V2 Enhancements (March 2025)
- Container image scanning with layer awareness
- Guided upgrade recommendations
- Interactive HTML reports
- Transformed from basic dependency checker into full remediation tool
