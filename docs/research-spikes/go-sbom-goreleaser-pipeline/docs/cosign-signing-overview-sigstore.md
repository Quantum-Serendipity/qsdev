<!-- Source: https://docs.sigstore.dev/cosign/signing/overview/ -->
<!-- Retrieved: 2026-05-15 -->

# Cosign Signing Overview

## Keyless Signing Fundamentals

Cosign implements identity-based signing by default, which associates identities rather than cryptographic keys with artifact signatures. Fulcio issues short-lived certificates binding an ephemeral key to an OpenID Connect identity. Signing events are logged in Rekor, a signature transparency log, providing an auditable record of when a signature was created.

## Root of Trust

Sigstore's trust foundation relies on The Update Framework (TUF) to distribute Fulcio's root CA certificate and Rekor's public key. TUF provides protocols to protect against various types of attacks in the software update supply chain.

## Identity Token Support

Cosign supports two OAuth authentication flows:

1. **Standard OAuth flow** - Interactive browser-based authentication
2. **Device flow** - Used in non-interactive environments where a link is printed for browser-based completion

The platform currently supports identity providers including Microsoft, Google, and GitHub. Users can also supply tokens directly using the `--identity-token` flag, provided the token's `audiences` claim contains `sigstore`.

## Signing Process

The three-phase signing workflow operates as follows:

**Phase 1: Identity Verification and Certification**
- An ephemeral public/private keypair is generated in memory
- An identity token is obtained from the configured provider
- Sigstore's certificate authority validates the identity token and issues a short-lived certificate binding the identity to the public key
- The private key is destroyed after use; the certificate expires shortly thereafter

**Phase 2: Transparency Log Entry**
A Sigstore client creates a timestamped object containing the artifact hash, public key, and signature. The Rekor transparency log records this entry, creating an immutable transparency log that documents the signing event with a timestamp.

**Phase 3: Verification**
Consumers verify signatures by comparing the timestamped transparency log entry against the signature components. Successful matching confirms the expected creator's certified identity signed the specific artifact.

## GCP Integration

From a Google Compute Engine VM, service account identity is automatically detected:

```bash
$ cosign sign gcr.io/user-vmtest2/demo
```

From outside GCP, you can impersonate a service account:

```bash
$ cosign sign --identity-token=$(
    gcloud auth print-identity-token \
        --audiences=sigstore \
        --include-email \
        --impersonate-service-account my-sa@my-project.iam.gserviceaccount.com) \
    gcr.io/user-vmtest2/demo
```

## Custom Infrastructure Configuration

For self-hosted Sigstore deployments, create a signing configuration file:

```bash
$ cosign signing-config create \
    --fulcio="url=https://fulcio.example.com,api-version=1,start-time=2024-01-01T00:00:00Z,operator=example.com" \
    --rekor="url=https://rekor.example.com,api-version=2,start-time=2024-01-01T00:00:00Z,operator=example.com" \
    --tsa="url=https://tsa.example.com,api-version=1,start-time=2024-01-01T00:00:00Z,operator=example.com" \
    --output-file custom.signingconfig.json
```

Then sign artifacts using this configuration:

```bash
$ cosign sign --signing-config custom.signingconfig.json \
    ghcr.io/jdoe/somerepo/testcosign
```

## Key Advantages

The keyless approach eliminates traditional key management overhead since developers typically possess trusted identities through existing platforms rather than managing cryptographic keys independently.
