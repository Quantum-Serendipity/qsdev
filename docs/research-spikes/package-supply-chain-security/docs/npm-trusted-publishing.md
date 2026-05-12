# npm Trusted Publishing
- **Source**: https://docs.npmjs.com/trusted-publishers/
- **Retrieved**: 2026-05-12

## What It Is

Trusted publishing enables package distribution through CI/CD workflows using OpenID Connect (OIDC) authentication, eliminating reliance on long-lived npm tokens. It "implements the trusted publishers industry standard specified by the Open Source Security Foundation (OpenSSF)."

## How OIDC-Based Publishing Works

The mechanism establishes a trust relationship between npm and your CI/CD provider. When properly configured, npm accepts publishes from authorized workflows in addition to traditional authentication. The npm CLI automatically detects OIDC environments and leverages them before reverting to token-based methods. Each publish operation uses short-lived, cryptographically-signed tokens specific to your workflow that cannot be extracted or reused.

## Supported CI/CD Providers

Three platforms currently support trusted publishing:

- **GitHub Actions** (GitHub-hosted runners only)
- **GitLab CI/CD** (GitLab.com shared runners)
- **CircleCI** (CircleCI cloud)

Self-hosted runners remain unsupported but are planned for future releases.

## Configuration Process

### Step 1: Register on npmjs.com
Navigate to your package settings and access the Trusted Publisher section. Select your CI/CD provider and supply required information (repository name, workflow filename, organization/user details depending on your platform).

### Step 2: Update Your Workflow
Add necessary OIDC permissions to your CI/CD configuration:
- **GitHub Actions**: Include `id-token: write` permission
- **GitLab CI/CD**: Configure `id_tokens` with audience `"npm:registry.npmjs.org"`
- **CircleCI**: Set `NPM_ID_TOKEN` environment variable with OIDC token

## Relationship to Provenance

Automatic provenance generation occurs when publishing through GitHub Actions or GitLab CI/CD using trusted publishers. This happens "by default — you don't need to add the `--provenance` flag." Provenance is unavailable for CircleCI and private repositories.

## Security Advantages Over Access Tokens

Traditional tokens present several vulnerabilities that trusted publishing eliminates:
- Tokens risk exposure in CI logs or configuration files
- They demand manual rotation and ongoing management
- Compromised tokens grant persistent access until revoked
- They typically possess broader permissions than necessary

By contrast, trusted publishing provides workflow-specific, temporary credentials that the system manages automatically.

## Recommended Security Practice

After enabling trusted publishers, restrict traditional token publishing by requiring two-factor authentication while disallowing tokens. This configuration preserves OIDC functionality while eliminating token-based attack vectors.
