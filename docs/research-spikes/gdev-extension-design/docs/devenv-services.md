# devenv.sh Services
- **Source**: https://devenv.sh/services/
- **Retrieved**: 2026-05-12

## Core Concept

Services function as "a higher-level abstraction over processes." While processes enable running arbitrary commands, services provide pre-configured interfaces for established software like databases.

## PostgreSQL Example

The documentation includes this configuration example:

```nix
{ pkgs, ... }:
{
  services.postgres = {
    enable = true;
    package = pkgs.postgresql_15;
    initialDatabases = [{ name = "mydb"; }];
    extensions = extensions: [
      extensions.postgis
      extensions.timescaledb
    ];
    settings.shared_preload_libraries = "timescaledb";
    initialScript = "CREATE EXTENSION IF NOT EXISTS timescaledb;";
  };
}
```

## Available Services

The documentation lists 45+ supported services, including:
- Databases: PostgreSQL, MySQL, MongoDB, Cassandra, CouchDB, Elasticsearch
- Caching: Redis, Memcached
- Message queues: Kafka, RabbitMQ, NATS
- Web servers: Nginx, Caddy
- Additional tools: Vault, Keycloak, Prometheus, and many others

## Operation Details

Services launch with `devenv up` and operate in foreground mode by default. Service state persists in `$DEVENV_STATE`. Configuration changes require deleting the service directory before restarting. Background execution is possible using the `-d` flag: `devenv up -d`
