# Nix Configuration Settings Reference (Hardening-Relevant Subset)
- **Source**: https://nix.dev/manual/nix/2.28/command-ref/conf-file
- **Retrieved**: 2026-05-12

## connect-timeout
**Type:** Integer (seconds)
**Default:** `0`
**Description:** "The timeout (in seconds) for establishing connections in the binary cache substituter."
**Notes:** Corresponds to curl's `--connect-timeout` option; 0 means no limit.

## download-attempts
**Type:** Integer
**Default:** `5`
**Description:** "How often Nix will attempt to download a file before giving up."

## filter-syscalls
**Type:** Boolean
**Default:** `true`
**Description:** "Whether to prevent certain dangerous system calls, such as creation of setuid/setgid files or adding ACLs or extended attributes."
**Notes:** Only disable if aware of security implications.

## sandbox
**Type:** Boolean or `relaxed`
**Default:** `true` on Linux, `false` elsewhere
**Description:** Builds isolated from filesystem; "they're isolated from the normal file system hierarchy and will only see their dependencies in the Nix store."
**Notes:** Requires root on Linux/macOS; `relaxed` allows fixed-output derivations outside sandbox.

## sandbox-fallback
**Type:** Boolean
**Default:** `true`
**Description:** "Whether to disable sandboxing when the kernel doesn't allow it."

## sandbox-paths
**Type:** List of paths
**Default:** _empty_
**Description:** "A list of paths bind-mounted into Nix sandbox environments."
**Syntax:** `target=source` mounts source at different location; `?` suffix makes source optional.
**Deprecated alias:** `build-chroot-dirs`, `build-sandbox-paths`

## restrict-eval
**Type:** Boolean
**Default:** `false`
**Description:** "The Nix evaluator will not allow access to any files outside of builtins.nixPath, or to URIs outside of allowed-uris."

## allowed-uris
**Type:** List of URI prefixes
**Default:** _empty_
**Description:** "A list of URI prefixes to which access is allowed in restricted evaluation mode."
**Access logic:** Granted when URI equals prefix, is subpath, or shares scheme with colon-terminated prefix.

## trusted-users
**Type:** Whitespace-separated list
**Default:** `root`
**Description:** Users with additional rights connecting to Nix daemon, including "the ability to specify additional substituters."
**Group syntax:** Prefix with `@` (e.g., `@wheel`)
**Warning:** Essentially grants root-equivalent access.
**System-only:** Must be set in system `nix.conf`.

## allowed-users
**Type:** Whitespace-separated list
**Default:** `*`
**Description:** "These users are allowed to connect to the Nix daemon."
**Group syntax:** Prefix with `@`; `*` allows all users
**Note:** Trusted users always connect regardless.

## trusted-substituters
**Type:** Whitespace-separated Nix store URLs
**Default:** _empty_
**Description:** "These are not used by default, but users of the Nix daemon can enable them by specifying substituters."
**Access control:** Unprivileged users can only specify URLs listed here.
**Deprecated alias:** `trusted-binary-caches`

## trusted-public-keys
**Type:** Whitespace-separated public keys
**Default:** `cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`
**Description:** Keys for verifying store object signatures from substituters.
**Acceptance:** At least one condition must be met: signed with trusted key, `require-sigs` disabled, store URL marked `trusted=true`, or content-addressed object.
**Deprecated alias:** `binary-cache-public-keys`

## require-sigs
**Type:** Boolean
**Default:** `true`
**Description:** "Any non-content-addressed path added or copied to the Nix store must have a signature by a trusted key."
**Note:** Content-addressed paths unaffected by this setting.

## substituters
**Type:** Whitespace-separated store URLs
**Default:** `https://cache.nixos.org/`
**Description:** "Additional stores from which Nix can obtain store objects instead of building them."
**Priority:** Lower values indicate higher priority (default cache has priority 40).
**Access requirement:** User must be in `trusted-users` or substituter in `trusted-substituters`.
**List append:** Use `extra-substituters` prefix.
**Deprecated alias:** `binary-caches`

## extra-substituters
**Type:** Whitespace-separated store URLs
**Default:** _empty_
**Description:** Appends to `substituters` setting rather than replacing it.
**Usage:** Allows configuration composition without overwriting defaults.

## extra-trusted-public-keys
**Type:** Whitespace-separated public keys
**Default:** _empty_
**Description:** Appends to `trusted-public-keys` rather than replacing.
**Usage:** Enables adding keys alongside defaults.

## extra-trusted-substituters
**Type:** Whitespace-separated store URLs
**Default:** _empty_
**Description:** Appends to `trusted-substituters`.

## max-substitution-jobs
**Type:** Integer
**Default:** `16`
**Description:** "The maximum number of substitution jobs that Nix will try to run in parallel."
**Constraint:** Minimum value is `1`; lower values interpreted as `1`.
**Deprecated alias:** `substitution-max-jobs`

## download-buffer-size
**Type:** Integer (bytes)
**Default:** `67108864` (64 MiB)
**Description:** "The size of Nix's internal download buffer in bytes during curl transfers."
**Note:** Insufficient processing speed may cause download stalls.
