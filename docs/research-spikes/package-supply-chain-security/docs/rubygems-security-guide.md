# RubyGems Security Guide
- **Source**: https://guides.rubygems.org/security/
- **Retrieved**: 2026-05-12

## Gem Signing with `gem cert`

RubyGems supports cryptographic gem signing since version 0.8.11. The process involves:

1. **Key Generation**: Developers use `gem cert --build your@email.com` to create a self-signed certificate pair
2. **Configuration**: The public key is added to the gem repository, and paths are configured in the gemspec
3. **Installation**: Users can verify signed gems during installation

## Security Policies

Installation with trust policies using the `-P` flag:

- **HighSecurity**: "All dependent gems must be signed and verified"
- **MediumSecurity**: "All signed dependent gems must be verified"

Bundler uses `--trust-policy` instead of `-P` for the same policies.

## Building and Signing Gems

Developers should:
- Create self-signed certificates in `~/.ssh`
- Store public certificates in a `certs/` directory within the repository
- Configure signing keys in gemspec files
- Test installation with security policies before release

## Verification Methods

Users can verify gem integrity through:
- **Signature verification** during installation with trust policies
- **Checksum validation** using SHA512: `gem fetch gemname -v version` followed by manual hash verification

## System Limitations

The current signing system "is not widely used" and "requires a number of manual steps on the part of the developer." There exists "no well-established chain of trust for gem signing keys."

## Future Improvements

Discussion continues regarding alternative models like X509 and OpenPGP through community channels, with goals to make signing "easy for authors and transparent for users."
