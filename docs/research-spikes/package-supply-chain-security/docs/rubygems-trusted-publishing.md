# Announcing Trusted Publishing on RubyGems.org
- **Source**: https://blog.rubygems.org/2023/12/14/trusted-publishing.html
- **Retrieved**: 2026-05-12

## How It Works

Trusted Publishing uses OpenID Connect (OIDC) to enable secure automated gem publishing. The mechanism "allows obtaining short-lived API tokens in an automated environment (such as CI) without having to store long-lived API tokens or username/password credentials."

## OIDC Integration

The system exchanges short-lived identity tokens between a trusted third-party service and RubyGems.org through the OIDC protocol. This eliminates the need for developers to manage persistent credentials.

## Supported CI Providers

Currently, GitHub Actions is the only supported platform. The documentation indicates future expansion: "Support for other trusted publishing platforms" is listed as a next step.

## Setup Process

Configuration requires four simple steps:
1. Repository owner name
2. Repository name
3. GitHub Actions workflow filename
4. Optional GitHub Environment specification

Once configured, developers use a "short, simple, and copy/pastable workflow" to automate publishing.

## Gem Attestations and Sigstore

The announcement mentions planned future enhancements: "A comprehensive GitHub Actions workflow that handles building the gem, generating provenance, signing it using sigstore, pushing it." However, these features were not yet implemented at announcement time.

## Key Security Benefits

- Short-lived tokens replacing long-lived API credentials
- Transparent, auditable releases from trusted environments
- No manual credential storage required by developers

## Current Status (2026)

RubyGems has since added `--attestation` support in rubygems >= 3.6.0 and is tracking attestation adoption at https://segiddins.github.io/are-we-attested-yet/. Rails and other major gems have begun releasing with attestations from GitHub Actions.
