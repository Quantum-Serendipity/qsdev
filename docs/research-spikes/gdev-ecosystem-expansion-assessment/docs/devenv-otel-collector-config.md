# devenv.sh OpenTelemetry Collector Service Configuration

- **Source URL**: https://devenv.sh/services/opentelemetry-collector/
- **Retrieval Date**: 2026-05-14

## Available Configuration Options

### services.opentelemetry-collector.enable
- Type: boolean
- Default: `false`
- Description: Activates the opentelemetry-collector service

### services.opentelemetry-collector.package
- Type: package
- Default: `pkgs.opentelemetry-collector-contrib`
- Description: Specifies which OpenTelemetry Collector package distribution to use

### services.opentelemetry-collector.configFile
- Type: null or absolute path
- Default: `null`
- Example: `pkgs.writeTextFile { name = "otel-config.yaml"; text = "..."; }`
- Description: Override the configuration file used by OpenTelemetry Collector instead of auto-generating from settings. Note: when overriding, ensure the `health_check` extension is enabled, or disable the readiness probe via `processes.opentelemetry-collector.ready = lib.mkForce null;`

### services.opentelemetry-collector.settings
- Type: open submodule of YAML 1.1 value
- Default: `defaultSettings`
- Description: Contains the Collector configuration per the official OpenTelemetry documentation at https://opentelemetry.io/docs/collector/configuration/
