<!-- Source: https://nix.dev/manual/nix/2.18/language/advanced-attributes -->
<!-- Retrieved: 2026-05-12 -->

# Security-Relevant Advanced Attributes in Nix Derivations

## Environment Variable Control

**impureEnvVars**: Permits specific environment variables from the calling user's environment to reach the builder. This is "only allowed in fixed-output derivations" since impurities are acceptable when output hashes are pre-known. Example use case: `fetchurl` passes proxy configuration variables.

## Data Passing Security

**passAsFile**: Routes attribute values through temporary files rather than environment variables, circumventing OS-imposed environment size limits. For each attribute listed, Nix creates an `xPath` environment variable pointing to the file containing that attribute's value.

## Dependency Validation

**allowedReferences**: Restricts direct runtime dependencies to a specified list. Setting this to an empty list enforces zero dependencies, useful for validating that generated boot artifacts contain no accidental store references.

**allowedRequisites**: Similar to `allowedReferences` but validates the entire recursive closure, ensuring all transitive dependencies conform to the allowlist.

**disallowedReferences**: Blacklists specific direct dependencies, preventing the output from referencing named derivations.

**disallowedRequisites**: Blacklists packages from the entire recursive closure, blocking both direct and transitive dependencies on specified derivations.
