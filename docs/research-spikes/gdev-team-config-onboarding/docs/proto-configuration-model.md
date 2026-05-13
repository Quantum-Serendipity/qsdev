<!-- Source: https://moonrepo.dev/docs/proto/config -->
<!-- Retrieved: 2026-05-12 -->

# Proto Configuration Model

## File Format & Locations

Proto uses **TOML-based `.prototools` files** across three locations:

- **Local**: `./.prototools` (current directory)
- **Global**: `~/.proto/.prototools` (proto-specific user config)
- **User**: `~/.prototools` (general user home directory)

The documentation suggests storing "project specific versions of tools" in local configuration and "default/fallback versions" globally.

## Configuration Inheritance & Resolution Modes

Proto supports four resolution modes controlling which files are loaded and merged:

1. **`global`**: Only loads `~/.proto/.prototools`
2. **`local`**: Only loads `./.prototools` in the current directory
3. **`upwards`**: Traverses from current directory upward to the filesystem root or home directory, loading all `.prototools` files encountered
4. **`upwards-global`/`all`**: Combines upwards traversal with the global config file appended last

Configuration files are "deeply merge[d]" with the current directory taking highest precedence. Different commands default to different modes; `activate`, `install`, `outdated`, and `status` use `upwards`, while others default to `upwards-global`.

## Environment-Specific Configuration

When `PROTO_ENV` is set, proto loads environment-aware files like `.prototools.production` or `.prototools.development`. These take precedence over base `.prototools` files within their directory level, following the same merge pattern as standard resolution.

## Version Pinning for Teams

Tools are pinned at the file level using simple key-value pairs:

```toml
node = "16.16.0"
npm = "9"
go = "~1.20"
rust = "stable"
```

Versions support "fully-qualified version, a partial version, a range or requirement, or an alias." Proto itself can be pinned: `proto = "0.38.0"`.

The `pin-latest` setting automatically persists resolved "latest" versions to specified locations (`global`, `local`, or `user`) after installation.

## Project vs. User-Level Configuration

| Configuration Type | Recommended Location |
|---|---|
| Project-specific versions | Local (`./.prototools`) |
| Project-specific settings | Local |
| Shared/developer settings | User (`~/.prototools`) |
| Default tool versions | Global (`~/.proto/.prototools`) |

## Notable Features

- **Tool aliases**: Custom labels mapping to versions via `[tools.*.aliases]`
- **Tool-specific settings**: Configured under `[tools.*]` and passed to WASM plugins
- **Environment variables**: Per-directory via `[env]`, with override capabilities at tool and backend levels
- **Dotenv support**: `[env]` sections support `file` fields pointing to `.env` files
