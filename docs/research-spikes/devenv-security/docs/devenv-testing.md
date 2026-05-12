# devenv Testing
- **Source**: https://devenv.sh/tests/
- **Retrieved**: 2026-05-12

## Core Concept
Testing in devenv ensures your development environment functions as intended. The `devenv test` command builds your environment and executes tests defined in the `enterTest` attribute.

## How enterTest Works

**Execution Model:**
- `enterTest` is a shell script block that runs after the environment is built
- By default, it detects and executes a `.test.sh` file if present
- Tests run within the configured shell environment with all packages and variables available

## Basic Test Definition

```nix
{
  packages = [ pkgs.ncdu ];
  enterTest = ''
    ncdu --version | grep "ncdu 2.2"
  '';
}
```

## Testing with Processes

If your configuration includes processes, they automatically start before tests run and stop afterward:

```nix
{
  services.nginx = {
    enable = true;
    httpConfig = ''
      server {
        listen 8080;
        location / { return 200 "Hello, world!"; }
      }
    '';
  };
  
  enterTest = ''
    wait_for_port 8080
    curl -s localhost:8080 | grep "Hello, world!"
  '';
}
```

## Environment Modification During Testing

Conditional modification using `config.devenv.isTesting`:

```nix
processes = {
  backend.exec = "cargo watch";
} // lib.optionalAttrs (!config.devenv.isTesting) {
  frontend.exec = "parcel serve";
};
```

## Available Test Functions

**`wait_for_port <port> <timeout>`** -- Pauses until a specified port becomes accessible.

## Alternative: Tasks for Complex Tests

For sophisticated test setups requiring dependencies and better parallelization, documentation recommends using tasks with the `before` attribute instead of `enterTest`.
