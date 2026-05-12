# SecretSpec: Declarative Secrets Management (Announcement)
- **Source**: https://devenv.sh/blog/2025/07/21/announcing-secretspec-declarative-secrets-management/
- **Retrieved**: 2026-05-12

## Problem Statement

SecretSpec addresses critical gaps in secret management workflows:

- **Disconnected applications** - Apps lack clear contracts specifying required secrets
- **Parser ambiguity** - `.env` files have unclear behavior with comments, multiline values, and special characters
- **Integration challenges** - Password managers require manual workarounds
- **Vendor lock-in** - Custom parsing logic complicates provider switching
- **Security risks** - Plain text `.env` files are vulnerable to accidental commits

The core tension: "Don't you feel some anxiety given we've normalized committing encrypted secrets to git repos?"

## Architecture: Three-Concern Separation

SecretSpec decouples secret management into distinct layers:

1. **Declaration (WHAT)** - Application specifies needed secrets in `secretspec.toml`
2. **Requirements (HOW)** - Defines attributes like required status, defaults, validation, environment
3. **Sourcing (WHERE)** - Environment determines provider without code changes

## Supported Providers

- Keyring (system keychain)
- OnePassword
- .env files (dotenv)
- Environment variables
- LastPass
- AWS Secrets Manager (production)

## devenv Integration

Configuration in `devenv.yaml`:
```yaml
secretspec:
  enable: true
```

Enables `secretspec.toml` parsing and environment variable injection into devenv shells.

## Configuration Format

The `secretspec.toml` structure uses profiles:

```toml
[profiles.development]
DATABASE_URL = { default = "postgresql://localhost/myapp_dev" }
STRIPE_API_KEY = { description = "Stripe API key (test mode)" }
```

Profiles inherit and override from a default base configuration.

## Security Model

- Avoiding single master key distributions
- Enabling granular trust models per environment
- Simplified rotation without team-wide key management
- Secrets kept out of shell environment when using runtime loading

## Future Roadmap (as of announcement)

Planned features include secret rotation during runtime, secret generation, and mixing multiple providers simultaneously.
