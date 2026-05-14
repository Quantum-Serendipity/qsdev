# Three AWS Tools for Easier Consultant Work

- **Source URL**: https://wolkencode.de/posts/3-aws-tools-for-easier-consultant-work/
- **Retrieved**: 2026-05-14

## Tool #1: aws-vault

Securely manages and accesses AWS credentials without storing access keys in plain text.

**Usage patterns:**
- Store credentials: `aws-vault add YOUR-PROFILE-NAME`
- Console login: `aws-vault login YOUR-PROFILE-NAME`
- Execute commands: `aws-vault exec YOUR-PROFILE-NAME -- SHELL COMMAND`
- Open subshell: `aws-vault exec YOUR-PROFILE-NAME`
- View profiles: `aws-vault list`

Multi-account: Requires separate profile setup for each account.

## Tool #2: aws-sso-cli

Manages IAM Identity Center (AWS SSO) logins from terminal, supporting multiple AWS Organizations.

- Define multiple personalized profiles
- Interactive role and account selection
- Override default organization: `aws-sso -S YOUR-ORGANIZATION-PROFILE-NAME`
- Detects SSO profiles from `~/.aws/config`

Multi-account: Supports simultaneous management of multiple organizations.

## Tool #3: aws-nuke

Deletes all resources in an AWS account with optional filtering. Solves infrastructure cleanup when ClickOps mixed with IaC leaves orphaned resources.

## Bonus: spaceship-prompt

Terminal theme with runtime environment icons and customization.
