# AWS Credential Management Tools Comparison

- **Sources**: https://github.com/99designs/aws-vault, https://github.com/Versent/saml2aws, https://docs.leapp.cloud/, https://news.ycombinator.com/item?id=29090858
- **Retrieved**: 2026-05-14

## aws-vault (99designs)
- Securely stores and accesses AWS credentials in development environments
- Credentials never touch disk in plaintext; stored in OS keychain
- Supports SSO, MFA caching, role chaining
- Backend options: macOS Keychain, Pass, File, KWallet
- Linux: typically uses `pass` or `file` backend
- Nixpkgs: `aws-vault` v7.7.10

## saml2aws (Versent)
- CLI tool for SAML-based login and retrieval of AWS temporary credentials
- Supports 15+ IdP types: Okta, ADFS, Azure AD, OneLogin, Ping, KeyCloak, Google Apps
- Automates SAML login flow for enterprise IdPs
- Nixpkgs: `saml2aws` v2.36.19

## aws-sso-cli
- Enhanced AWS SSO CLI experience
- Can log into same SSO instance as two different SSO users simultaneously
- Better profile management and credential caching than stock `aws sso`
- Nixpkgs: `aws-sso-cli` v2.1.0

## aws-sso-util
- Smooths out rough edges of AWS SSO
- Designed to address pain points in standard AWS SSO workflow
- Largely superseded by aws-sso-cli for most use cases

## Leapp
- Visual credential manager (Electron GUI app)
- Rotates credentials every 20 minutes via STS
- Never writes long-term credentials to ~/.aws/credentials
- NOT in nixpkgs (Electron app)
- Less suitable for CLI-first devenv integration

## Key Differences
- **aws-vault**: Best general-purpose choice, works with IAM keys + SSO + MFA
- **saml2aws**: Required when client uses enterprise SAML IdP (Okta, ADFS, etc.)
- **aws-sso-cli**: Best for pure AWS SSO environments
- **Leapp**: GUI-first, not ideal for devenv.sh integration
