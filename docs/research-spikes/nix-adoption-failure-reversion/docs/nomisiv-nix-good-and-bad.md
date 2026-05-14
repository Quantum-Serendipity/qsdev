# Nix, the Good and the Bad
- **Source URL**: https://nomisiv.com/blog/nix-good-and-bad
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Simon Gutgesell

## Duration
Not specified.

## Use Cases
System configuration management, development environment setup, package management across multiple computers.

## Complaints and Pain Points

### Build Times
"when evaluating all the systems I have declared in my flake can take up to 10 minutes before nix is done 'thinking' and starts to actually build stuff."

### Error Messages
Error outputs often contain "stack traces filling up half your terminal buffer, filled with references to internal functions in nixpkgs that you didn't write."

### Dynamically Linked Libraries
Nix's all-or-nothing approach means precompiled binaries typically fail because "the libraries aren't in the usual places, and if the binary isn't specifically linked to the right path, it's just going to crash."

### Documentation
Multiple sources exist but are fragmented and sometimes incomplete, with primary manuals being "single pages, making them painfully slow to load."

## Resolution
Neither explicitly left nor departed. Maintains ambivalent acceptance: "Nix is pretty bad, but it's the best that there is."

## Nuanced Takes
Appreciates declarative configuration for its transparency and reproducibility while acknowledging that current alternatives (Guix, Docker) have greater limitations.
