# devenv.sh Prometheus Service Configuration

- **Source URL**: https://devenv.sh/services/prometheus/
- **Retrieval Date**: 2026-05-14

## Available Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `services.prometheus.enable` | boolean | `false` | Whether to enable Prometheus monitoring system |
| `services.prometheus.package` | package | `pkgs.prometheus` | Selects which Prometheus package to use |
| `services.prometheus.port` | 16-bit unsigned integer | `9090` | Port for Prometheus web interface |
| `services.prometheus.storage.path` | string | `${config.devenv.state}/prometheus` | Path where Prometheus will store its database |
| `services.prometheus.storage.retentionTime` | string | `"15d"` | How long to retain data |
| `services.prometheus.globalConfig` | attribute set | `{evaluation_interval = "1m"; scrape_interval = "1m"; scrape_timeout = "10s";}` | Global Prometheus configuration settings |
| `services.prometheus.scrapeConfigs` | list of attribute sets | `[ ]` | List of scrape configurations |
| `services.prometheus.ruleFiles` | list of strings | `[ ]` | List of rule files to load |
| `services.prometheus.alerting` | null or attribute set | `null` | Alerting configuration |
| `services.prometheus.remoteRead` | list of attribute sets | `[ ]` | Remote read configurations |
| `services.prometheus.remoteWrite` | list of attribute sets | `[ ]` | Remote write configurations |
| `services.prometheus.extraArgs` | string | `""` | Additional arguments to pass to Prometheus |
| `services.prometheus.advanced.storage` | attribute set | `{ }` | Storage configuration settings |
| `services.prometheus.advanced.tsdb` | attribute set | `{ }` | TSDB configuration settings |
| `services.prometheus.experimentalFeatures.enableExemplars` | boolean | `false` | Enable exemplar storage |
| `services.prometheus.experimentalFeatures.enableOTLP` | boolean | `false` | Enable OTLP receiver |
| `services.prometheus.experimentalFeatures.enableTracing` | boolean | `false` | Enable tracing |
