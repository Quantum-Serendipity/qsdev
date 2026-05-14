# GCP Application Default Credentials & Workload Identity

- **Sources**: https://docs.cloud.google.com/docs/authentication/application-default-credentials, https://oneuptime.com/blog/post/2026-02-17-how-to-set-up-application-default-credentials-for-local-development-on-gcp/view
- **Retrieved**: 2026-05-14

## Application Default Credentials (ADC)

ADC provides a standard way to authenticate locally when you don't have built-in GCP infrastructure authentication.

### Setup for Local Development
Run `gcloud auth application-default login` -- opens browser, credential file saved locally.

### ADC Credential Discovery Chain
1. `GOOGLE_APPLICATION_CREDENTIALS` environment variable
2. User credentials set up by `gcloud auth application-default login`
3. Attached service account (on GCE/GKE/Cloud Run/Cloud Functions)
4. Compute metadata service

### Service Account Impersonation
`gcloud auth application-default login --impersonate-service-account=SA_EMAIL`
Preferred over downloading service account key files.

## Workload Identity Federation
- External workloads authenticating to GCP without service account keys
- Uses OIDC/SAML tokens from external IdPs
- Config generated via: `gcloud iam workload-identity-pools create-cred-config`
- Supports AWS, Azure, and generic OIDC providers

## Best Practices
- Never use service account key files if avoidable
- Use `gcloud auth application-default login` for local development
- Use attached service account on Cloud Run and Cloud Functions
- Use Workload Identity on GKE
- Use Workload Identity Federation for external workloads

## Verification
`gcloud auth application-default print-access-token` -- if this returns a token, ADC is set up.
