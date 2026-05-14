# Client Isolation & Credential Switching Patterns

- **Source**: Multiple web searches and articles
- **Retrieval Date**: 2026-05-14

## AWS Multi-Account Credential Management

### aws-vault (99designs)
- Stores AWS credentials in system keychain, not disk
- Supports MFA, STS AssumeRole, SSO
- `aws-vault exec profile -- command` pattern
- Session caching across profiles sharing source credentials
- https://github.com/99designs/aws-vault

### aws-vault-switch
- Go CLI that enhances aws-vault for seamless profile switching
- Switch profiles without exiting current shell
- https://blog.kobebigs.com/aws-vault-switch-easily-switch-between-aws-profiles

### Granted (Common Fate)
- CLI for multi-account cloud access
- Opens multiple AWS console tabs simultaneously via browser containers/profiles
- Firefox Multi-Account Containers or Chrome profiles
- Secure SSO token storage in system keychain
- Session expiration notifications
- Supports SSO, IAM roles, federated login
- https://docs.commonfate.io/granted/

## Git Multi-Identity Management

### Conditional Includes (.gitconfig)
```ini
[includeIf "gitdir:~/work/client-a/"]
    path = ~/.gitconfig-client-a
[includeIf "gitdir:~/work/client-b/"]
    path = ~/.gitconfig-client-b
```
Each config file sets user.name, user.email, signing key.

### Git Credential Manager (GCM)
- `credential.useHttpPath` stores credentials per repository URL, not per hostname
- Supports credential namespacing for multi-tenant scenarios
- Service principal auth: `{tenantId}/{clientId}` format for Azure

## Multi-Tenant Developer Patterns

1. **Directory-based isolation**: ~/work/client-a/, ~/work/client-b/ with per-directory git config
2. **AWS profile-per-client**: Named profiles in ~/.aws/config with SSO or IAM role per client
3. **VPN switching**: Each client may require separate VPN connection
4. **Separate SSH keys per client**: SSH config Host blocks mapping to different keys
5. **Environment variable sets**: Different sets of env vars (API keys, endpoints) per client

## Key Insight for gdev

The patterns above are all manually configured today. A "client profile" that bundles:
- AWS profile name
- Git user.name/email/signing key
- SSH key path
- VPN config identifier
- Time tracking project/workspace
- Environment variables

...would be genuinely novel and valuable for consulting engineers. No existing tool does this holistically.
