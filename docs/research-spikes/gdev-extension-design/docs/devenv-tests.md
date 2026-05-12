# devenv.sh Testing
- **Source**: https://devenv.sh/tests/
- **Retrieved**: 2026-05-12

## Overview

Testing in devenv ensures development environments function as intended. The `devenv test` command builds the environment and executes tests defined in the `enterTest` attribute.

## Basic Test Structure

A minimal test configuration:

```nix
{ pkgs, ... }: {
  packages = [ pkgs.ncdu ];
  
  enterTest = ''
    ncdu --version | grep "ncdu 2.2"
  '';
}
```

Running this produces output indicating successful test execution with timing information.

## Process Integration

When processes are defined in the environment, they automatically start before tests run and stop afterward. Example with nginx:

```nix
{ pkgs, ... }: {
  services.nginx = {
    enable = true;
    httpConfig = ''
      server {
        listen 8080;
        location / {
          return 200 "Hello, world!";
        }
      }
    '';
  };
  
  enterTest = ''
    wait_for_port 8080
    curl -s localhost:8080 | grep "Hello, world!"
  '';
}
```

## Conditional Environment Changes

As of version 1.0.6, environments can be modified during testing using configuration flags:

```nix
{ pkgs, lib, config, ... }: {
  processes = {
    backend.exec = "cargo watch";
  } // lib.optionalAttrs (!config.devenv.isTesting) {
    frontend.exec = "parcel serve";
  };
}
```

This pattern allows disabling certain processes when tests run.

## Available Test Functions

- `wait_for_port <port> <timeout>`: Waits for port availability

## Alternative: Tasks

For complex test scenarios, using tasks with the `before` attribute provides better dependency management and parallelization than `enterTest`.
