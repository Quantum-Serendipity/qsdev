# devenv.sh WireMock Service Configuration

- **Source URL**: https://devenv.sh/services/wiremock/
- **Retrieval Date**: 2026-05-14

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| services.wiremock.enable | boolean | false | Whether to enable WireMock |
| services.wiremock.package | package | pkgs.wiremock | WireMock package |
| services.wiremock.disableBanner | boolean | false | Disable banner logo |
| services.wiremock.mappings | JSON | [] | Request/response mock mappings |
| services.wiremock.port | uint16 | 8080 | HTTP server port |
| services.wiremock.verbose | boolean | false | Verbose logging |

## Notes

- JSON-based mapping configuration for API mocking
- Simple setup, minimal options
