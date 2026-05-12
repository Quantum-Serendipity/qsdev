<!-- Source: https://github.com/Yelp/detect-secrets -->
<!-- Retrieved: 2026-05-12 -->

# detect-secrets: Enterprise Secret Detection

## Core Functionality

Python module designed to identify hardcoded secrets in code repositories. Emphasizes an enterprise-friendly workflow that prevents new secrets from entering a codebase while acknowledging existing repositories may contain legacy secrets.

## Baseline Approach

Operates on a baseline system — a snapshot of currently identified secrets. Rather than scanning entire git history repeatedly, it uses periodic diff analysis against the baseline. This strategy allows teams to:

> "accepting that there may currently be secrets hiding in your large repository (this is what we refer to as a baseline), but preventing this issue from getting any larger"

## Detection Methods

Three complementary strategies:

1. **Regex-based plugins** — Pattern matching for well-structured secrets (API keys, tokens, credentials), with optional network verification
2. **Entropy detection** — Identifying high-entropy strings using Base64 and hexadecimal analysis with configurable thresholds
3. **Keyword detection** — Flagging variable names commonly associated with hardcoded secrets

27+ built-in detectors for services like AWS, GitHub, SendGrid, Stripe, and others.

## Usage Integration

- `detect-secrets scan` — Creates/updates baselines
- `detect-secrets-hook` — Pre-commit integration to block new secrets
- `detect-secrets audit` — Interactive analysis and labeling of baseline findings

Pre-commit integration prevents staged commits containing unregistered secrets.

## Filtering & Customization

Configuration options include regex-based exclusion rules (`--exclude-lines`, `--exclude-files`, `--exclude-secrets`), inline allowlisting via code comments, and custom plugin/filter development.

## Limitations

"This is not meant to be a sure-fire solution to prevent secrets from entering the codebase. Only proper developer education can truly do that."

## Nix/devenv Integration

No mentions of Nix or devenv integration. Available in nixpkgs as `pkgs.detect-secrets`.
