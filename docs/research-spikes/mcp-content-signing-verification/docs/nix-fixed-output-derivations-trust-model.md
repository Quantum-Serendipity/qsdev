# Nix Fixed-Output Derivations: Trust Model

- **Sources**:
  - https://nix.dev/manual/nix/2.28/store/derivation/outputs/content-address.html
  - https://nix.dev/manual/nix/2.22/language/advanced-attributes.html
  - https://bmcgee.ie/posts/2023/02/nix-what-are-fixed-output-derivations-and-why-use-them/ (403 on fetch, title confirms topic)
  - https://discourse.nixos.org/t/how-do-content-addressed-derivations-work-in-terms-of-trust/54718
- **Retrieved**: 2026-05-14

## How Fixed-Output Derivations Work

Fixed-output derivations are derivations where the hash of the output must be specified in advance. Unlike normal derivations (which are sandboxed with no network access), fixed-output derivations are granted network access as a compromise -- but in return they must declare the expected content hash.

When the build completes, Nix computes the cryptographic hash of the output and compares it to the declared hash. A mismatch causes the build to fail.

The relevant attributes are:
- `outputHash`: the expected hash value
- `outputHashAlgo`: the hash algorithm (sha256, sha512, etc.)
- `outputHashMode`: "flat" (hash the file directly) or "recursive" (hash the NAR serialization)

## The Trust Anchor

The hash IS the trust anchor. The security property is:

> "Regardless of what the builder does during the build, it cannot influence downstream builds in unanticipated ways because all information it passed downstream flows through the outputs whose content-addresses are fixed."

This is "carefully controlled impurity" -- the builder can do anything (download from anywhere, run arbitrary code), but the output must match the declared hash or the build fails.

## Content-Addressable Store Model

In a content-addressed store, the store path is derived from the content itself. Identical outputs receive the same store path regardless of origin. This means:
- The same file downloaded from different URLs gets the same store path
- Changing a URL doesn't trigger cascading rebuilds if the content is unchanged
- The path encodes what the content IS, not where it came from

## SRI Hash Format

Modern Nix uses Subresource Integrity (SRI) format: `sha256-<base64-encoded-hash>`. This is the same format used by web browsers for script integrity verification.

## Trust Model Gaps

1. **Who verified the initial hash?** The hash was recorded by someone (a nixpkgs maintainer, an automated update bot). The hash is only as trustworthy as the process that recorded it.

2. **Hash updates**: When upstream releases a new version and the hash changes, the new hash is typically recorded by running a build with `lib.fakeHash`, getting the real hash from the error, and updating. This TOFU (Trust On First Use) workflow means the person updating trusts whatever content the URL served at that moment.

3. **Compromised nixpkgs**: If the git commit that records the hash is compromised, the hash itself is wrong. All downstream consumers will faithfully verify against the wrong hash.

4. **No provenance**: The hash proves the content matches what was recorded, but says nothing about WHO produced the content or whether it was the legitimate author.

## CA Derivations (Experimental)

Content-addressed derivations extend the model to allow outputs to be content-addressed without pre-specifying their hash (floating CA). This requires the `ca-derivations` experimental feature. The trust model for CA derivations is still being developed -- current caches don't fully implement the required endpoints for sharing "realizations" (the mapping from derivation to output).
