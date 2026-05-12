# Nix Configuration Security-Relevant Options
- **Source**: https://nix.dev/manual/nix/2.28/command-ref/conf-file
- **Retrieved**: 2026-05-12

## sandbox
**Description:** "builds will be performed in a _sandboxed environment_, i.e., they're isolated from the normal file system hierarchy"

Isolation prevents undeclared dependencies. Supports `true`, `false`, or `relaxed` mode. Requires Nix running as root.

**Default:** `true` on Linux, `false` on other platforms

**Security Implications:** Essential for preventing builds from accessing unexpected system resources.

---

## sandbox-fallback
**Description:** "Whether to disable sandboxing when the kernel doesn't allow it."

**Default:** `true`

**Security Implications:** When enabled, allows builds to proceed without isolation if sandboxing fails, potentially reducing security guarantees.

---

## sandbox-paths
**Description:** "A list of paths bind-mounted into Nix sandbox environments. You can use the syntax `target=source` to mount a path in a different location in the sandbox"

Supports optional sources (suffix with `?`). If source is in Nix store, its closure is added.

**Default:** Empty or provides `/bin/sh` as bash bind-mount

**Security Implications:** Expands sandbox access surface; should be minimized to necessary paths only.

---

## restrict-eval
**Description:** "the Nix evaluator will not allow access to any files outside of [`builtins.nixPath`], or to URIs outside of [`allowed-uris`]"

**Default:** `false`

**Security Implications:** Critical for preventing evaluation from accessing arbitrary filesystem and network resources.

---

## pure-eval
**Description:** Ensures results depend only on explicitly declared inputs. Restricts file system and network access to cryptographically hashed files; disables impure constants including `builtins.currentSystem`, `builtins.currentTime`, `builtins.nixPath`, `builtins.storePath`.

**Default:** `false`

**Security Implications:** Prevents external state from influencing evaluation results, eliminating time-based and environment-based side channels.

---

## allowed-users
**Description:** "A list user names, separated by whitespace. These users are allowed to connect to the Nix daemon."

Supports groups via `@` prefix. Special value `*` allows all users.

**Default:** `*`

**Security Implications:** Controls who can access daemon functionality. Trusted users can always connect regardless.

---

## trusted-users
**Description:** "These users will have additional rights when connecting to the Nix daemon, such as the ability to specify additional substituters"

Groups supported via `@` prefix (e.g., `@wheel`).

**Default:** `root`

**Security Implications:** "Adding a user to `trusted-users` is essentially equivalent to giving that user root access to the system." Critical access control mechanism.

---

## trusted-public-keys
**Description:** "A whitespace-separated list of public keys." Required to accept store objects from substituters unless `require-sigs=false`, store URL has `trusted=true`, or object is content-addressed.

**Default:** `cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`

**Security Implications:** Validates authenticity of substituted store objects; compromised keys allow malicious substitution.

---

## trusted-substituters
**Description:** "These are not used by default, but users of the Nix daemon can enable them by specifying substituters."

Unprivileged users can only use substituters from this list.

**Default:** Empty

**Security Implications:** Restricts unprivileged users from adding arbitrary substituters, preventing supply chain attacks.

---

## require-sigs
**Description:** "any non-content-addressed path added or copied to the Nix store must have a signature by a trusted key"

Content-addressed paths are inherently trustworthy regardless.

**Default:** `true`

**Security Implications:** Enforces cryptographic verification; disabling trusts all non-content-addressed paths unconditionally.

---

## substituters
**Description:** "A list of URLs of Nix stores to be used as substituters." Tried based on priority values (lower=higher priority).

Users must be in `trusted-users` list or substituter in `trusted-substituters` to use non-default substituters.

**Default:** `https://cache.nixos.org/`

**Security Implications:** Substituters provide binary packages; untrusted ones enable package injection attacks.

---

## secret-key-files
**Description:** "A whitespace-separated list of files containing secret (private) keys. These are used to sign locally-built paths."

Generated via `nix-store --generate-binary-cache-key`.

**Default:** Empty

**Security Implications:** Private keys must be protected; compromise allows forging signatures for malicious packages.

---

## allowed-uris
**Description:** "A list of URI prefixes to which access is allowed in restricted evaluation mode." Grants access when URI equals prefix, is subpath of prefix, or prefix is scheme with colon and URI matches scheme.

**Default:** Empty

**Security Implications:** Whitelist-based network access control; prevents arbitrary network fetches during evaluation.

---

## flake-registry
**Description:** "Path or URI of the global flake registry. When empty, disables the global flake registry."

**Default:** `https://channels.nixos.org/flake-registry.json`

**Security Implications:** Registry provides flake input resolution; malicious registry enables flake spoofing.

---

## sandbox-dev-shm-size
**Description:** "This option determines the maximum size of the `tmpfs` filesystem mounted on `/dev/shm` in Linux sandboxes."

Format follows `mount(8)` tmpfs size option.

**Default:** `50%`

**Security Implications:** Limits temporary storage within sandboxes, preventing resource exhaustion attacks.

---

## max-build-log-size
**Description:** "This option defines the maximum number of bytes that a builder can write to its stdout/stderr."

Builder killed upon exceeding limit.

**Default:** `0` (no limit)

**Security Implications:** Unbounded logs enable denial-of-service; capped logs prevent resource exhaustion.

---

## allowed-impure-host-deps
**Description:** "Which prefixes to allow derivations to ask for access to (primarily for Darwin)."

**Default:** Empty

**Security Implications:** Specifies exceptions to sandbox isolation; should be minimized to required system resources only.
