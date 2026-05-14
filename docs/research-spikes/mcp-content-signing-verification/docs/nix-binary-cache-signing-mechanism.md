# Nix Binary Cache Signing Mechanism

- **Sources**:
  - https://wiki.nixos.org/wiki/Binary_Cache
  - https://www.tweag.io/blog/2019-11-21-untrusted-ci/
  - https://nix.dev/manual/nix/2.28/store/derivation/outputs/content-address.html
  - https://nix.dev/manual/nix/2.18/command-ref/new-cli/nix3-key-generate-secret
  - https://docs.tvix.dev/rust/nix_compat/narinfo/index.html (narinfo format reference)
- **Retrieved**: 2026-05-14

## Key Generation

Ed25519 keypairs are generated via:
```
nix-store --generate-binary-cache-key [domain] [private-key-file] [public-key-file]
```

Both secret and public keys are represented as `key-name:base64-encoded-ed25519-key-data`.

## Narinfo Signature Format

The fingerprint that gets signed has the format:
```
1;<StorePath>;<NarHash>;<NarSize>;<refs>
```

This fingerprint is signed with an Ed25519 key. In narinfo files, signatures appear as:
```
Sig: binarycache.example.com:EmAANryZ1FFHGmz5P+HXLSDbc0KckkBEAkHsht7gEIOUXZk9yhhZSBV+eSX9Kj+db/b36qmYmffgiOZbAe21Ag==
```

Format: `[key-name]:[base64-encoded-signature]`.

The signature covers the SHA-256 hash of the narinfo file contents (up to but not including the Sig line itself).

## Trust Configuration

Users configure trust through two nix.conf settings:
- **substituters**: specify which binary cache URLs to use
- **trusted-public-keys**: list the public keys for those caches

Example:
```
substituters = https://cache.nixos.org https://example.org
trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= example.org:My56...Q==%
```

Warning from NixOS wiki: "When adding a third-party binary cache you now trust all packages being served from that cache."

## Security Properties

The binary cache signing prevents two attack vectors:
1. **MITM on cache downloads**: narinfo signatures are verified against trusted-public-keys before substitution
2. **Unauthorized store manipulation**: only packages signed with configured private keys are accepted

## Signing Existing Packages

Already-built packages can be signed retroactively:
```
nix store sign --all --key-file [key-path]
```

## Limitations

- Trust is per-cache, not per-package. Trusting a cache's key means trusting everything it serves.
- The `trusted` setting on a store can override signature requirements entirely.
- CA derivation trust infrastructure is still incomplete: caches "do not actually implement the endpoints to share" realizations.
