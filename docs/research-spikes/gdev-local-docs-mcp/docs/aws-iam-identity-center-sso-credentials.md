# AWS IAM Identity Center SSO Credential Provider
- **Source**: https://docs.aws.amazon.com/sdkref/latest/guide/feature-sso-credentials.html
- **Retrieved**: 2026-05-14

## Overview

IAM Identity Center credential provider uses AWS IAM Identity Center to get SSO access to AWS services. After enabling IAM Identity Center, you define a profile in your shared AWS config file to connect to the IAM Identity Center access portal. When a user successfully authenticates, the portal returns short-term credentials for the IAM role associated with that user.

## Two Configuration Modes

1. **(Recommended) SSO token provider configuration** — Extended session durations with automatic token refresh
2. **Legacy non-refreshable configuration** — Fixed eight-hour session, no auto-refresh

## SSO Token Provider Config File Format

```ini
[profile dev]
sso_session = my-sso
sso_account_id = 111122223333
sso_role_name = SampleRole

[sso-session my-sso]
sso_region = us-east-1
sso_start_url = https://my-sso-portal.awsapps.com/start
sso_registration_scopes = sso:account:access
```

Required settings:
- sso_account_id and sso_role_name in the profile section
- sso_region, sso_start_url, sso_registration_scopes in the sso-session section

Multiple profiles can share the same sso-session.

## Token Caching

Authentication token is cached to disk under ~/.aws/sso/cache/ with a filename based on the session name.

## SDK Support

Supported by all major AWS SDKs: Python (boto3), JavaScript 3.x, Go V2, Java 2.x, .NET, Ruby, PHP, C++, Kotlin, Swift, PowerShell. Rust SDK supports only legacy non-refreshable configuration.

## Login Flow

1. Developer runs `aws sso login --profile dev`
2. Browser opens for SSO authentication
3. Token cached to ~/.aws/sso/cache/
4. SDKs automatically use cached token to get temporary credentials
5. Token auto-refreshes (with SSO token provider config)

## Credential Resolution

When using SSO credentials, the SDK:
1. Reads the sso-session config from ~/.aws/config
2. Uses cached SSO token from ~/.aws/sso/cache/
3. Calls sso:GetRoleCredentials to exchange SSO token for temporary AWS credentials
4. Returns standard accessKeyId, secretAccessKey, sessionToken
