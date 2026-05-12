<!-- Source: https://docs.npmjs.com/trusted-publishers/ -->
<!-- Retrieved: 2026-05-12 -->

# npm Trusted Publishing: Complete Guide

## Overview

Trusted publishing enables npm package distribution directly from CI/CD workflows using OpenID Connect (OIDC) authentication, eliminating reliance on long-lived tokens. As stated in the documentation: "This feature implements the trusted publishers industry standard specified by the Open Source Security Foundation (OpenSSF)."

**Requirements:** npm CLI 11.5.1+ and Node 22.14.0+

## How It Works

The system establishes a trust relationship between npm and your CI/CD provider through OIDC. When configured, npm accepts publishes from authorized workflows using short-lived, cryptographically-signed tokens specific to each workflow. The npm CLI automatically detects OIDC environments and uses them before falling back to traditional tokens.

## Supported CI/CD Providers

- **GitHub Actions** (GitHub-hosted runners)
- **GitLab CI/CD** (GitLab.com shared runners)
- **CircleCI** (CircleCI cloud)

Self-hosted runners are not currently supported but planned for future releases.

## Configuration Process

### Step 1: Register Trusted Publisher

Navigate to your package settings on npmjs.com and locate the "Trusted Publisher" section. Select your CI/CD provider and configure provider-specific fields:

**GitHub Actions:**
- Organization/username
- Repository name
- Workflow filename (e.g., `publish.yml`)
- Optional environment name

**GitLab CI/CD:**
- Namespace (username or group)
- Project name
- CI file path (e.g., `.gitlab-ci.yml`)
- Optional environment name

**CircleCI:**
- Organization ID (UUID)
- Project ID (UUID)
- Pipeline definition ID (UUID)
- VCS origin URL
- Optional context IDs

### Step 2: Workflow Configuration

**GitHub Actions** requires: "the `id-token: write` permission, which allows GitHub Actions to generate OIDC tokens."

Example workflow structure:
```yaml
permissions:
  id-token: write
  contents: read
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-node@v6
      - run: npm ci
      - run: npm publish
```

**GitLab CI/CD** requires ID token configuration with audience set to `"npm:registry.npmjs.org"`:
```yaml
id_tokens:
  NPM_ID_TOKEN:
    aud: "npm:registry.npmjs.org"
```

**CircleCI** uses environment variables:
```bash
export NPM_ID_TOKEN=$(circleci run oidc get --claims '{"aud": "npm:registry.npmjs.org"}')
npm publish
```

## Automatic Provenance Generation

When publishing via trusted publishing from GitHub Actions or GitLab CI/CD, npm automatically generates provenance attestations. This occurs by default without requiring the `--provenance` flag, provided the package is public and the repository is public.

**Conditions for automatic generation:**
- Publishing via OIDC
- Public repository source
- Public package

CircleCI does not currently support provenance generation.

To disable provenance, set `provenance=false` via environment variable, `.npmrc`, or `package.json`:
```json
{
  "publishConfig": {
    "provenance": false
  }
}
```

## Security Recommendations

**Restrict Token Access:** After enabling trusted publishers, navigate to package Settings > Publishing access and select "Require two-factor authentication and disallow tokens."

This approach eliminates risks associated with long-lived credentials. The documentation emphasizes: "Trusted publishers use short-lived, scoped credentials generated on-demand during CI/CD workflows, eliminating the need for long-lived tokens."

**Private Dependencies:** Use read-only granular access tokens for installing dependencies, reserving OIDC authentication for the publish operation exclusively.

**Additional Measures:**
- Deploy approval requirements using GitHub environments
- Enable tag protection rules
- Regularly audit configurations
- Remove unused publish tokens

## Managing Configurations

Each package supports only one trusted publisher at a time. Modifications and deletions occur through package settings on npmjs.com. Switching providers requires editing the existing configuration.

## Current Limitations

- Cloud-hosted runners only
- One trusted publisher per package
- OIDC limited to publish operations
- `npm whoami` doesn't reflect OIDC authentication
- No provenance support for private repositories or CircleCI
