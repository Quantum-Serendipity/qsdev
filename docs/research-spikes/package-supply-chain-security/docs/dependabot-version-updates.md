<!-- Source: https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/about-dependabot-version-updates -->
<!-- Retrieved: 2026-05-12 -->

# Dependabot Version Updates: Technical Overview

## Core Functionality

Dependabot automates dependency maintenance by raising pull requests to keep packages current. The system operates through two mechanisms: "Dependabot security updates are automated pull requests that help you update dependencies with known vulnerabilities," while version updates maintain packages regardless of vulnerability status.

## Configuration & Detection

Version updates require a `dependabot.yml` configuration file checked into the repository. This file specifies the location of the manifest, or of other package definition files, stored in your repository. Dependabot identifies outdated packages by analyzing semantic versioning (semver) against available releases.

## Update Mechanisms

For standard dependencies, Dependabot raises a pull request to update the manifest to the latest version. For vendored dependencies (cached in repositories rather than referenced externally), it raises a pull request to replace the outdated dependency with the new version directly.

## Supported Scope

Dependabot handles references to actions in a repository's workflow.yml file and reusable workflows used inside workflows. It supports all major package ecosystems.

## Notable Limitations

The system includes automatic deactivation: When maintainers of a repository stop interacting with Dependabot pull requests, Dependabot temporarily pauses its updates.

## Security Updates

When Dependabot security updates are enabled for a repository, Dependabot will automatically try to open pull requests to resolve every open Dependabot alert that has an available patch.

## Grouping Updates

Use the groups option with the applies-to: security-updates key to create sets of dependencies (per package manager), so that Dependabot opens a single pull request to update multiple dependencies at the same time. You can define groups by package name (the patterns and exclude-patterns keys), dependency type (dependency-type key), and SemVer (the update-types key).

## Auto-Triage Rules

You can use Dependabot auto-triage rules to manage your alerts at scale, so you can auto-dismiss or snooze alerts, and specify which alerts you want Dependabot to open pull requests for.
