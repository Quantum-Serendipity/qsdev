# devenv.sh Vault Service Configuration

- **Source URL**: https://devenv.sh/services/vault/
- **Retrieval Date**: 2026-05-14

## Available Configuration Options

- `services.vault.enable` — boolean, default `false`
- `services.vault.package` — package, default `pkgs.vault-bin`
- `services.vault.address` — string, default `"127.0.0.1:8200"`
- `services.vault.disableClustering` — boolean, default `true`
- `services.vault.disableMlock` — boolean, default `true`
- `services.vault.ui` — boolean, default `true`

## Notes

- Runs in dev mode by default (suitable for local development)
- UI enabled by default at the configured address
- Clustering and mlock disabled for dev simplicity
- Defined in `src/modules/services/vault.nix`
