# OpenTelemetry Collector Service - devenv

- **Source URL**: https://devenv.sh/services/opentelemetry-collector/
- **Retrieval Date**: 2026-05-14

## Overview

The OpenTelemetry Collector service integration enables observability data collection within devenv environments.

## Configuration Options

### Enable the Service
**Property:** `services.opentelemetry-collector.enable`
- **Type:** Boolean
- **Default:** `false`
- **Usage:** Set to `true` to activate the collector

### Package Selection
**Property:** `services.opentelemetry-collector.package`
- **Type:** Package
- **Default:** `pkgs.opentelemetry-collector-contrib`
- Allows you to specify an alternative OpenTelemetry Collector distribution

### Custom Configuration File
**Property:** `services.opentelemetry-collector.configFile`
- **Type:** Null or absolute path
- **Default:** `null`
- When set, overrides auto-generated configuration
- **Important Note:** "If overriding, enable the `health_check` extension to allow the readiness probe to check whether the Collector is ready" or disable the readiness probe by setting `processes.opentelemetry-collector.ready = lib.mkForce null;`
- **Example:** `pkgs.writeTextFile { name = "otel-config.yaml"; text = "..."; }`

### Settings Configuration
**Property:** `services.opentelemetry-collector.settings`
- **Type:** Open YAML 1.1 submodule
- **Default:** `defaultSettings`
- Provides declarative configuration management
- Reference the official documentation at https://opentelemetry.io/docs/collector/configuration/ for detailed configuration guidance

## Key Integration Points

The service integrates with devenv's process management system, allowing health checks and readiness probes to monitor collector status during development workflows.
