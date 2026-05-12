# Sigstore Cosign Signing Overview
- **Source**: https://docs.sigstore.dev/cosign/signing/overview/
- **Retrieved**: 2026-05-12

## Core Architecture

Sigstore implements "keyless" signing that associates identities rather than persistent keys with artifact signatures. The system comprises three main components:

- **Fulcio**: Certificate authority issuing short-lived certificates
- **Rekor**: Transparency log recording signing events
- **Cosign**: Client tool orchestrating the signing process

## OIDC Identity Providers

Sigstore currently supports three OAuth identity issuers:
- Microsoft
- Google
- GitHub

Additionally, the system can "automatically detect and produce identity tokens" in environments like Google Cloud Platform and GitHub Actions.

## The Signing Flow

**Step 1: Key Generation & Identity Verification**
An ephemeral public/private keypair is generated in-memory. Cosign retrieves an OIDC identity token, which Fulcio verifies and uses to issue a short-lived certificate binding the identity to the public key.

**Step 2: Signing & Transparency Logging**
The artifact is signed with the ephemeral private key. The signing event is recorded in Rekor with a timestamp, creating an "auditable record." The private key is then destroyed.

**Step 3: Verification**
Verifiers compare the signature, certificate, and artifact against the timestamped Rekor entry. Valid matches confirm the signing event occurred.

## Root of Trust

Sigstore distributes its "root of trust, which includes Fulcio's root CA certificate and Rekor's public key" through The Update Framework (TUF), providing protection against various attack vectors.

## Supported Environments

- **Interactive**: Standard OAuth flow with browser redirect
- **Non-interactive**: Device flow (link printed to stdout)
- **Automated**: Direct `--identity-token` flag or `SIGSTORE_ID_TOKEN` environment variable
- **GCP**: Automatic metadata server detection
- **Custom Infrastructure**: User-operated Fulcio, Rekor, and TSA instances
