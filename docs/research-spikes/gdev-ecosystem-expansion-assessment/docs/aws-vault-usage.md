# AWS-Vault Usage Documentation

- **Source**: https://github.com/99designs/aws-vault/blob/master/USAGE.md
- **Retrieved**: 2026-05-14

## Core Functionality

AWS-Vault is a credential management tool that securely stores and manages AWS credentials. Four primary use patterns:

1. **Direct Executor Model**: aws-vault runs commands with temporary credentials generated on-demand.
2. **Master Credentials Vault**: Integrates with AWS SDK's `credential_process` to supply base credentials for role assumption.
3. **MFA Session Caching**: Caches authenticated sessions across multiple profiles, eliminating repeated MFA prompts.
4. **Alternative Source Caching**: Works with SSO, web identity tokens, and external credential processes.

## Managing Credentials

### Adding & Removing
Use `aws-vault add <profile>` to store credentials and `aws-vault remove <profile>` for deletion. The tool prompts for Access Key ID and Secret Access Key during addition.

### Rotation
The `aws-vault rotate <profile>` command regularly updates access keys. Required IAM permissions include "iam:CreateAccessKey," "iam:DeleteAccessKey," and "iam:GetUser" scoped to the user's own resources.

### Multiple Profiles
Support for parallel AWS accounts enables managing unrelated environments (work/home) with distinct credential sets.

## Configuration Methods

### AWS Config File
Profile configuration supports standard AWS settings plus custom aws-vault options:

- **`include_profile`**: Horizontally imports settings from another profile ("mixin" or "parent" style)
- **`session_tags`/`transitive_session_tags`**: Attach tags to AssumeRole operations
- **`source_identity`**: Sets identity monitoring parameters
- **`mfa_process`**: Specifies an external command generating MFA tokens

### Environment Variables
Control defaults via:
- `AWS_VAULT_BACKEND`, `AWS_VAULT_KEYCHAIN_NAME`, `AWS_VAULT_PROMPT`
- `AWS_SESSION_TOKEN_TTL`, `AWS_ASSUME_ROLE_TTL` (session durations)
- `AWS_MFA_SERIAL`, `AWS_ROLE_ARN` (profile overrides)

Session duration variables expect units (e.g., "12h" or "43200s").

## Session Management

### Execution
`aws-vault exec <profile> -- <command>` runs commands with temporary credentials. Using `--` preserves shell autocompletion for the target command.

### Console Access
`aws-vault login <profile>` opens AWS Console in a browser with authenticated session.

### Session Control
- `--no-session`: Skips GetSessionToken, exposing master credentials (security tradeoff)
- `aws-vault clear [profile]`: Removes cached sessions before expiration
- `--duration`: Sets credential lifetime (limited to 1 hour for role chaining)

### Server Modes
- **EC2 Metadata Server** (`--ec2-server`): Requires root/administrator privileges; mimics EC2 instance credentials
- **ECS Metadata Server** (`--ecs-server`): Ephemeral port with authorization token; supports discrete processes

## MFA Integration

Configure MFA by specifying `mfa_serial` (the ARN of the MFA device) in profiles. AWS-Vault caches GetSessionToken results, reusing authenticated sessions across profiles with matching MFA serials.

**Configuration gotcha**: Version 5+ requires explicit `mfa_serial` per profile (matches AWS CLI behavior). Use `include_profile` or `[default]` section to reduce repetition.

### External MFA Tools
- **YubiKey OATH-TOTP**: Configure via `ykman oath accounts add` and invoke with `--prompt ykman` or set `mfa_process`
- **Pass/1Password**: `mfa_process = pass otp my_aws_mfa` or similar commands

## Single Sign-On & Web Identity

### SSO Configuration
Profiles using IAM Identity Center specify:
- `sso_start_url`, `sso_region`, `sso_account_id`, `sso_role_name`
- Alternative: `sso_session` for shared session configuration

### Web Identity Federation
Assume roles via OpenID Connect using:
- `web_identity_token_file`: Static token path
- `web_identity_token_process`: Command generating dynamic tokens

## credential_process Integration

AWS-Vault can both invoke and be invoked by `credential_process`:

**As provider**: `credential_process = aws-vault export --format=json home` supplies credentials to AWS SDK

**As consumer**: When executing profiles with `credential_process` defined, aws-vault executes the command and caches results.

For role assumption via SDK, provide master credentials: "aws-vault export --no-session --format=json <profile>"

## Backends & Storage

Configurable backends (set via `--backend` flag or `AWS_VAULT_BACKEND`):
- **Keychain (macOS)**: Lock timeout configurable via Keychain Access app
- **Other options**: Pass, File, LastPass (system-dependent availability)

## Advanced Use Cases

### Desktop Applications
Combine `--server` with application launch for auto-refreshing credentials: "aws-vault exec --server jonsmith -- open -W -a Lens"

### Docker Integration
ECS metadata server exposes `/role-arn/YOUR_ROLE_ARN` endpoint. Containers use `AWS_CONTAINER_CREDENTIALS_RELATIVE_URI` or `AWS_CONTAINER_CREDENTIALS_FULL_URI` without embedded keys.

## Limitations

Temporary credentials restrict certain STS/IAM operations (GetUser, CreateAccessKey). Workarounds: configure MFA for the IAM user or use `--no-session` to expose master credentials.
