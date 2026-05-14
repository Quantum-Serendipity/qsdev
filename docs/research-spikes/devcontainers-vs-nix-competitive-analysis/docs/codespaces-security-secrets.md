# GitHub Codespaces Security and Secrets
- **Source**: https://docs.github.com/en/codespaces/reference/security-in-github-codespaces
- **Retrieved**: 2026-03-20
- **Note**: Content synthesized from web search results (WebFetch unavailable)

## Security Architecture

### VM Isolation
- Each codespace runs on its own dedicated, newly-built VM
- Two codespaces are never co-located on the same VM
- Every time a codespace is restarted, it's deployed to a new VM with latest security updates
- Each codespace has its own isolated virtual network

### Network Security
- Firewalls block all incoming connections from the internet
- Codespaces cannot communicate with each other on internal networks
- Outbound connections to the internet are allowed (not restricted by default)
- Connections use TLS-encrypted tunnel provided by GitHub Codespaces service
- Only the creator of a codespace can connect to it

### Authentication
- All connections authenticated with GitHub
- GITHUB_TOKEN automatically provisioned with each codespace
- Token gets a new value with automatic expiry period on each create/restart
- Token scope: read/write access to the source repository by default
- Additional repository access can be authorized via devcontainer.json `customizations.codespaces.repositories`

## Secrets System

### Three Levels of Secrets

1. **User/Account Secrets**
   - Set in personal GitHub Codespaces settings
   - Available as environment variables in all codespaces (or scoped to specific repos)
   - Encrypted with libsodium sealed box before reaching GitHub
   - Decrypted only when used in a codespace

2. **Repository Secrets**
   - Set by repo admins in repository settings
   - Available to all codespaces created for that repository
   - Useful for shared credentials (API keys, database URLs)

3. **Organization Secrets**
   - Set by org admins
   - Can be shared across multiple repositories
   - Reduces need to duplicate secrets across repos
   - Scoped to specific repos or all repos in the org

### Precedence
If secrets with the same name exist at multiple levels, the lowest-level secret wins:
Repository > Organization > User

### Naming Rules
- Alphanumeric characters and underscores only
- Cannot start with `GITHUB_` prefix
- Cannot start with a number

### Recommended Secrets
- Repos can specify "recommended secrets" in devcontainer.json
- Users are prompted to set these when creating a codespace
- Improves onboarding experience

## Known Security Risks

### Supply Chain Concerns
- devcontainer.json configuration is trusted and executed automatically
- Malicious devcontainer configs can execute arbitrary commands
- Risk: attackers could exfiltrate GitHub tokens and secrets via crafted configs
- Mitigation: review devcontainer configs before creating codespaces from forks

### Secret Access
- Any user who can create a codespace for a repo can access that repo's secrets
- If a user shouldn't have access to certain secrets, they shouldn't have codespace access
- Organization admins should carefully scope secret access

### Port Forwarding
- Public ports expose the codespace's services to the internet
- Organizations can restrict port visibility via policy

## Data Residency (Enterprise)

- **GitHub Enterprise Cloud with data residency**: Codespaces available in public preview as of January 2026
- Enterprise or organization-owned codespaces required (user-owned not supported)
- All sensitive data remains within the selected region
- Addresses compliance requirements for regulated industries

## Additional Sources

- https://docs.github.com/en/codespaces/managing-your-codespaces/managing-your-account-specific-secrets-for-github-codespaces
- https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization/managing-development-environment-secrets-for-your-repository-or-organization
- https://www.legitsecurity.com/blog/github-codespaces-security-best-practices
- https://github.blog/changelog/2026-01-29-codespaces-is-now-in-public-preview-for-github-enterprise-with-data-residency/
