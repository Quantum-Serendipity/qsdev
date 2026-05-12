# SecretSpec Integration with devenv
- **Source**: https://devenv.sh/integrations/secretspec/
- **Retrieved**: 2026-05-12

## Overview
SecretSpec is a tool that "separates secret declaration from secret provisioning." Developers define required secrets in a `secretspec.toml` file, allowing different environments to source secrets from their preferred secure providers.

## Configuration Methods

### CLI Flags (devenv 2.0+)
```bash
devenv --secretspec-provider dotenv --secretspec-profile dev shell
```

Environment variables also work:
```bash
SECRETSPEC_PROVIDER=dotenv SECRETSPEC_PROFILE=dev devenv shell
```

### Via devenv.yaml
```yaml
secretspec:
  enable: true
  provider: keyring
  profile: default
```

CLI flags take precedence over YAML configuration.

## Supported Providers
- Keyring
- dotenv
- env
- 1Password
- LastPass

## Accessing Secrets in devenv.nix
```nix
{ config, ... }:
{
  env.DATABASE_URL = config.secretspec.secrets.DATABASE_URL or "";
}
```

## Recommended Approach (Runtime Loading)
The documentation strongly recommends using the Rust SDK and loading secrets at runtime:
```bash
secretspec run -- npm start
```

This approach "keeps secrets out of your shell environment" and "reduces exposure of sensitive data."

## Security Characteristics
- Secrets remain excluded from the shell environment when using runtime loading
- Supports easy secret rotation
- Follows the principle of least privilege when using runtime loading
