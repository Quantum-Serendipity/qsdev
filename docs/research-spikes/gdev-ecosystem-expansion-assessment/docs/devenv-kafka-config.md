# devenv.sh Kafka Service Configuration

- **Source URL**: https://devenv.sh/services/kafka/
- **Retrieval Date**: 2026-05-14

## Core Service Options

- `services.kafka.enable` — boolean, default `false`
- `services.kafka.package` — package, default `pkgs.apacheKafka`
- `services.kafka.jre` — JRE package
- `services.kafka.jvmOptions` — list of JVM args

## Mode Configuration

- `services.kafka.defaultMode` — "zookeeper" or "kraft", default "kraft"
  - KRaft mode requires no extra configuration
  - Zookeeper mode requires additional setup
- `services.kafka.formatLogDirs` — boolean, default `true`
- `services.kafka.formatLogDirsIgnoreFormatted` — boolean, default `true`

## Broker Settings

- `services.kafka.settings."broker.id"` — null or integer
- `services.kafka.settings.listeners` — list of string, default `["PLAINTEXT://localhost:9092"]`
- `services.kafka.settings."log.dirs"` — list of paths

## Kafka Connect

- `services.kafka.connect.enable` — boolean, default `false`
- `services.kafka.connect.initialConnectors` — list of connector configs
- `services.kafka.connect.settings."bootstrap.servers"` — default `["localhost:9092"]`
- `services.kafka.connect.settings."key.converter"` — default JsonConverter
- `services.kafka.connect.settings."value.converter"` — default JsonConverter
- `services.kafka.connect.settings.listeners` — Connect REST API listeners
- `services.kafka.connect.settings."plugin.path"` — plugin directories

## Notes

- KRaft mode is default — no Zookeeper dependency needed
- Kafka Connect support is built-in for connector pipelines
- Comprehensive log4j property configuration available
