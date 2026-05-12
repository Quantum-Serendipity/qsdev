# Nix's Trust Model for Binary Caching Untrusted Builds
- **Source**: https://www.tweag.io/blog/2019-11-21-untrusted-ci/
- **Retrieved**: 2026-05-12

## The Security Challenge

The article addresses a fundamental tension: sharing CI cache artifacts with developers accelerates workflows, but allowing untrusted contributors to populate a shared cache creates attack vectors. An untrusted contributor could either manipulate local store contents before uploading or sign artifacts themselves, potentially introducing compromised binaries disguised as legitimate build outputs.

## Multi-User Nix: Decoupled, Privileged Building

The solution leverages multi-user Nix architecture, which separates build recipe assembly (nix-instantiate), actual compilation, and store persistence across different user contexts. This design means untrusted contributors cannot directly manipulate the Nix store. Instead, they submit build recipes to a privileged daemon that executes builds as unprivileged users in sandboxed environments, with output hashes computed by the daemon itself. This prevents store poisoning through privilege escalation exploits or direct manipulation.

## Post-Build Hooks: Automated Trusted Signing

Since Nix 2.3, "post-build hooks" enable the daemon to execute scripts automatically after derivation builds complete. The configuration:

```
post-build-hook = /etc/nix/upload-to-cache.sh
```

runs signing and uploading operations in the daemon's root context, shielding cryptographic keys from untrusted contributors. "The Nix daemon can be configured to execute a script after building a derivation," ensuring signatures remain under privileged control.

## Security Properties

**Signature Verification Flow:**
- Build artifacts enter the store only through the privileged daemon's validated process
- Signing occurs exclusively in root-controlled contexts
- Even if cache manipulation occurs, broken signatures cause substitution failures, triggering local rebuilds

## Additional Benefits

Beyond security, post-build hooks eliminate manual cache management across pipelines, automatically cache all intermediate artifacts regardless of build success, and simplify distributed development workflows by removing explicit dependency tracking requirements.
