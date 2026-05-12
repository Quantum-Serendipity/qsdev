<!-- Source: https://docs.renovatebot.com/merge-confidence/ -->
<!-- Retrieved: 2026-05-12 -->

# Merge Confidence: Complete Technical Overview

## Core Functionality

Merge Confidence analyzes dependency updates to "prevent updates which break in production" by examining test and release adoption data from Mend Renovate App users.

## Data Metrics

The system evaluates four key metrics displayed as pull request badges:

1. **Age**: How long the package has existed
2. **Adoption**: The percentage of this package's users (within Renovate) which are using this release
3. **Passing**: The percentage of updates which have passing tests for this package
4. **Confidence**: Overall confidence level for the update

## Confidence Levels

- **Low**: Indicates probable breaking changes, often expected in major version updates
- **Neutral**: Insufficient data to determine risk level
- **High**: Strong confidence in safety based on combined Age, Adoption, and Passing metrics
- **Very High**: Applied only to months-old updates with high adoption or test pass rates

## Supported Languages

Coverage includes: Go, JavaScript, Java, Python, .NET, PHP, and Ruby via datasources like npm, maven, pypi, nuget, packagist, and rubygems.

## Enabling the Feature

**Mend App**: Badges are enabled automatically

**Self-hosted**: Add `"mergeConfidence:all-badges"` to the `extends` array, or use `"mergeConfidence:age-confidence-badges"` for limited badges

**Disabling**: Add preset to `ignorePresets` array

## Intelligent Workflows

Paying Mend customers and OSS projects access "Merge Confidence Workflows" enabling conditional automation like automerging "Very High confidence updates" or delaying PRs until reaching High confidence.

## Technical Safeguards

npm packages require minimum three-day age before High confidence status due to unpublish policies. Adoption and Passing percentages are weighted toward organizations and high-reliability projects rather than raw counts.
