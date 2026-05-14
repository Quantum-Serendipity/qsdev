# devenv.sh MinIO Service Configuration

- **Source URL**: https://devenv.sh/services/minio/
- **Retrieval Date**: 2026-05-14

## Available Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `services.minio.enable` | boolean | `false` | Whether to enable MinIO Object Storage |
| `services.minio.package` | package | `pkgs.minio` | MinIO application to deploy |
| `services.minio.accessKey` | string | `"minioadmin"` | Access key (5-20 chars) |
| `services.minio.secretKey` | string | `"minioadmin"` | Secret key (8-40 chars) |
| `services.minio.listenAddress` | string | `"127.0.0.1:9000"` | Server IP and port |
| `services.minio.consoleAddress` | string | `"127.0.0.1:9001"` | Web UI IP and port |
| `services.minio.region` | string | `"us-east-1"` | Server region |
| `services.minio.browser` | boolean | `true` | Enable web UI access |
| `services.minio.buckets` | list of string | `[ ]` | Buckets to ensure exist on startup |
| `services.minio.clientPackage` | package | `pkgs.minio-client` | MinIO client package |
| `services.minio.clientConfig` | null or JSON | (configured) | Client configuration as Nix attrs |
| `services.minio.afterStart` | string | `""` | Bash code to execute after minio is running |

## Notes

- Includes both server and client (mc) packages
- Auto-creates buckets on startup via `buckets` option
- afterStart hook allows bucket permission setup, e.g.: `mc anonymous set download local/mybucket`
