# DevEnv SecretSpec Integration
- **Source**: https://devenv.sh/integrations/secretspec/
- **Retrieved**: 2026-05-12

## Core Concept

SecretSpec enables developers to "separate secret declaration from secret provisioning." Rather than embedding secrets directly in devenv configurations, it allows teams to define required secrets in a `secretspec.toml` file while letting each environment (development, CI, production) supply them from preferred secure sources.

## Recommended Approach

The documentation emphasizes runtime loading as the best practice. Instead of loading secrets into devenv's environment, developers should:

- Utilize the Rust SDK within application code to retrieve secrets
- Execute applications with `secretspec run -- [command]` to inject secrets only where needed

This strategy offers several advantages: "Keeps secrets out of your shell environment," "Reduces exposure of sensitive data," and "Makes secret rotation easier."

## Configuration Methods

### CLI Flags (DevEnv 2.0+)

Users can override provider and profile directly:
```
devenv --secretspec-provider dotenv --secretspec-profile dev shell
```

Environment variables also work:
```
SECRETSPEC_PROVIDER=dotenv SECRETSPEC_PROFILE=dev devenv shell
```

### DevEnv.yaml Configuration

```yaml
secretspec:
  enable: true
  provider: keyring
  profile: default
```

Supported providers include "keyring, dotenv, env, 1password, lastpass."

## Accessing Secrets in DevEnv.nix

Once configured, secrets become accessible through `config.secretspec.secrets`:

```nix
env.DATABASE_URL = config.secretspec.secrets.DATABASE_URL or "";
```

CLI flags take precedence over YAML configuration values.
