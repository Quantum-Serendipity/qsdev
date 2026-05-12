<!-- Source: https://devenv.sh/scripts/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv Scripts Documentation

## Script Definition Basics

Scripts in devenv are defined within the `devenv.nix` configuration file using the `scripts` attribute. A minimal example involves specifying an `exec` block containing shell commands:

```nix
scripts.silly-example.exec = ''
  curl "https://httpbin.org/get?$1" | jq '.args'
'';
```

Once defined, scripts become available as executable commands when entering the devenv shell environment.

## Environment Access & Package Integration

Scripts can access packages declared in the configuration. Developers should "rely on `packages` executables being available" within scripts. This works because packages added to the environment are automatically accessible in the script's PATH.

### Runtime-Specific Packages

For tools needed only by particular scripts, the `packages` attribute allows script-level package specification:

```nix
scripts.analyze-json = {
  exec = ''curl "https://httpbin.org/get?$1" | jq '.args''';
  packages = [ pkgs.curl pkgs.jq ];
  description = "Fetch and analyze JSON";
};
```

This approach prevents "polluting the global development environment" while ensuring necessary tools are accessible.

## Advanced Script Features

**Argument Forwarding**: Scripts support passing arguments using `"$@"` syntax, enabling flexible command-line interfaces.

**Multiple Language Support**: Scripts can execute code in various languages by specifying a `package` and optional `binary` attribute.

**External Files**: Script content can be loaded from external files rather than inline definitions.

**Direct Path References**: Package binaries can be directly interpolated as paths (`${pkgs.curl}/bin/curl`), pinning specific versions without relying on PATH variables.

## Shell Integration

Scripts can be listed during shell initialization using the `enterShell` hook, displaying descriptions to help developers discover available utilities.
