# Verifying Attestations Offline

- **Source URL**: https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/verifying-attestations-offline
- **Retrieved**: 2026-05-15

---

## Overview
How to verify artifact attestations without internet connectivity using GitHub CLI.

## Step 1: Obtain the Attestation Bundle

Download the attestation bundle using an internet-connected machine:

```bash
gh attestation download PATH/TO/YOUR/BUILD/ARTIFACT-BINARY -R ORGANIZATION_NAME/REPOSITORY_NAME
```

Output: `"Wrote attestations to file sha256:ae57936def59bc4c75edd3a837d89bcefc6d3a5e31d55a6fa7a71624f92c3c3b.jsonl"`

## Step 2: Retrieve Trusted Roots

Fetch the key material for verification:

```bash
gh attestation trusted-root > trusted_root.jsonl
```

The system supports the Sigstore public good instance for public repositories and GitHub's Sigstore instance for private repositories through a single command.

### Managing Trusted Roots in Air-Gapped Environments

Create a fresh `trusted_root.jsonl` file whenever you introduce new signed material into your isolated environment. While the key material itself lacks an expiration date, Sigstore typically rotates its cryptographic keys several times annually. Materials signed before your trusted root file was generated will verify successfully, but you won't detect key revocations occurring after the file's creation.

## Step 3: Execute Offline Verification

Transfer these items to your offline environment:
- GitHub CLI
- Artifact binary
- Bundle file
- Trusted root file

Run the verification command:

```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY \
  -R ORGANIZATION_NAME/REPOSITORY_NAME \
  --bundle sha256:ae57936def59bc4c75edd3a837d89bcefc6d3a5e31d55a6fa7a71624f92c3c3b.jsonl \
  --custom-trusted-root trusted_root.jsonl
```
