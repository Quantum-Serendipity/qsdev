# devenv with Nix Flakes Integration
- **Source**: https://devenv.sh/guides/using-with-flakes/
- **Retrieved**: 2026-05-12

## nixConfig Settings

The documentation provides a specific example of `nixConfig` options for security:

```nix
nixConfig = {
  extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=";
  extra-substituters = "https://devenv.cachix.org";
};
```

These settings allow the Nix evaluator to trust binary caches from devenv's cachix infrastructure, enabling pre-built dependencies rather than compilation from source.

## Security Model Changes

The documentation highlights a critical distinction regarding purity in evaluation:

> "Flakes use 'pure evaluation' by default, which prevents devenv from figuring out the environment its running in"

This purity constraint limits devenv's ability to determine runtime context. To work around this, users must either invoke flakes with the `--no-pure-eval` flag or manually specify absolute paths -- both approaches compromise the security benefits of pure evaluation.

## Practical Limitation

The guide notes that running processes during testing isn't supported with flakes, and `devenv test` doesn't start processes. This architectural constraint means certain development workflows cannot leverage the full feature set when using the flakes integration.

The documentation suggests most projects benefit from the dedicated devenv CLI instead, which avoids these security model trade-offs.
