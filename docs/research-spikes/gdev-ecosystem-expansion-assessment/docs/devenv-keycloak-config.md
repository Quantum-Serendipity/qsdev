# devenv.sh Keycloak Service Configuration

- **Source URL**: https://devenv.sh/services/keycloak/
- **Retrieval Date**: 2026-05-14

## Core Options

- `services.keycloak.enable` — boolean, default `false`
- `services.keycloak.package` — package, default `pkgs.keycloak`
- `services.keycloak.database.type` — "dev-mem" or "dev-file", default "dev-file"
- `services.keycloak.initialAdminPassword` — string, default "admin"
- `services.keycloak.plugins` — list of paths (plugin jars)

## Realm Configuration

- `services.keycloak.realms` — attribute set of realm configs
  - `realms.<name>.path` — relative path to import/export JSON
  - `realms.<name>.import` — boolean, default `true`
  - `realms.<name>.export` — boolean, default `false`

## Process & Script Options

- `services.keycloak.processes.exportRealms` — boolean, default `true`
- `services.keycloak.scripts.exportRealm` — boolean, default `true`

## Settings (conf/keycloak.conf)

- `services.keycloak.settings.hostname` — default "localhost"
- `services.keycloak.settings.http-host` — default "::"
- `services.keycloak.settings.http-port` — default 8080
- `services.keycloak.settings.http-relative-path` — default "/"
- `services.keycloak.settings.https-port` — default 34429
- Supports `_secret` attribute for secret data references

## SSL/TLS

- `services.keycloak.sslCertificate` — PEM certificate path
- `services.keycloak.sslCertificateKey` — PEM private key path

## Notes

- Very mature devenv integration with realm import/export
- Plugin support for custom extensions
- dev-file and dev-mem database modes (no external DB needed for dev)
- Comprehensive settings passthrough to keycloak.conf
