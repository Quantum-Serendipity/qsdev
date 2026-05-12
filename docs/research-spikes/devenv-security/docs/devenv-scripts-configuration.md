# devenv.sh Scripts Configuration
- **Source**: https://devenv.sh/scripts/
- **Retrieved**: 2026-05-12

## Script Definition

Scripts are defined in `devenv.nix` using the `scripts.<name>.exec` attribute. The basic structure allows specifying executable code as a string:

```nix
scripts.silly-example.exec = ''
  curl "https://httpbin.org/get?$1" | jq '.args'
'';
```

Scripts become available as commands when entering the development environment via `devenv shell`.

## Execution Model

Scripts execute with access to packages declared in the environment. Three approaches are supported:

1. **Global packages**: Tools added to `packages` are available in PATH during script execution
2. **Runtime-specific packages**: The `scripts.<name>.packages` attribute provides dependencies only when that script runs
3. **Direct path reference**: Packages can be interpolated directly (e.g., `${pkgs.curl}/bin/curl`)

Scripts support argument forwarding using the standard shell pattern `"$@"` to pass command-line arguments through.

## Language Support

Scripts can execute in various languages by specifying a `package` and `binary` attribute. The documentation shows examples using Python and Nu Shell, allowing developers to write scripts in their preferred language rather than just shell.

## Environment Interaction

Scripts integrate with the declarative environment -- they depend on packages defined within `devenv.nix`. The environment provides isolation, ensuring consistent tooling across developers without requiring global system dependencies.

## Security Model

The documentation contains no explicit information about sandboxing or security restrictions. Scripts appear to execute with standard shell permissions within the development environment context, inheriting access to all configured packages and environment variables.
