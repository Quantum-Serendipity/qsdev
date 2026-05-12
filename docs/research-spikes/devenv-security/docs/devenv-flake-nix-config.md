# nixConfig Settings in devenv's flake.nix
- **Source**: https://github.com/cachix/devenv/blob/main/flake.nix
- **Retrieved**: 2026-05-12

## nixConfig Section

```nix
nixConfig = {
  extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=";
  extra-substituters = "https://devenv.cachix.org https://cachix.cachix.org";
};
```

## Summary of Security-Relevant Settings

**Cache Substituters:** Devenv configures two binary cache sources -- its own cachix repository and the general Cachix service -- to fetch pre-built packages rather than compiling locally.

**Trusted Public Keys:** Two cryptographic keys are registered to verify cache authenticity, ensuring downloaded binaries haven't been tampered with.

**Not Present:** The configuration doesn't explicitly modify sandbox settings or other advanced Nix parameters. These defaults inherit system-wide Nix configuration unless overridden elsewhere in the flake or locally.
