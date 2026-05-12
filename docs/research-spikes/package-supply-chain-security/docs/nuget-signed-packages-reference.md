# NuGet Signed Packages Reference
- **Source**: https://learn.microsoft.com/en-us/nuget/reference/signed-packages-reference
- **Retrieved**: 2026-05-12

## Overview

NuGet 4.6.0+ and Visual Studio 2017 version 15.6 and later support signed packages. NuGet packages can include a digital signature that provides protection against tampered content. This signature is produced from an X.509 certificate that also adds authenticity proofs to the actual origin of the package.

## Signature Types

- **Author signature**: Guarantees that the package has not been modified since the author signed the package, no matter from which repository or what transport method the package is delivered. Author-signed packages provide an extra authentication mechanism to the nuget.org publishing pipeline because the signing certificate must be registered ahead of time.
- **Repository signature**: Provides an integrity guarantee for **all** packages in a repository whether they are author signed or not, even if those packages are obtained from a different location than the original repository where they were signed.

## Certificate Requirements

Package signing requires a code signing certificate:
- Valid for the `id-kp-codeSigning` purpose (RFC 5280 section 4.2.1.12)
- RSA public key length of 2048 bits or higher

## Timestamp Requirements

Signed packages should include an RFC 3161 timestamp to ensure signature validity beyond the package signing certificate's validity period. The timestamp certificate must be valid for the `id-kp-timeStamping` purpose with RSA public key length of 2048 bits or higher.

## Signature Requirements on NuGet.org

nuget.org has additional requirements for accepting a signed package:
- The primary signature must be an author signature
- The primary signature must have a single valid timestamp
- X.509 certificates for both author signature and timestamp signature:
  - Must have RSA public key 2048 bits or greater
  - Must be within validity period per current UTC time
  - Must chain to a trusted root authority trusted by default on Windows
  - Must be valid for its purpose (code signing / timestamping)
  - Must not be revoked at signing time

## Important Notes

- Author signing packages is only supported by nuget.exe on Windows at this time
- All packages uploaded to nuget.org are automatically repository signed
- Verification via `dotnet nuget verify` or `nuget verify` commands
- Trust configuration via `trusted-signers` command in nuget.config
