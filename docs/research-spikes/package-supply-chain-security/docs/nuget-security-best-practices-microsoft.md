<!-- Source: https://learn.microsoft.com/en-us/nuget/concepts/security-best-practices -->
<!-- Retrieved: 2026-05-12 -->

# NuGet Security Best Practices (Microsoft Learn)

## Dependencies and Supply Chain

Open Source is everywhere. The term software supply chain refers to everything that goes into your software and where it comes from -- the dependencies and properties of your dependencies.

## Key Security Features

### Packages with Known Vulnerabilities (NuGetAudit)

.NET 8 and Visual Studio 17.8 added NuGetAudit, which warns about direct packages with known vulnerabilities during restore. .NET 9 and Visual Studio 17.12 changed the default to warn about transitive packages as well.

### Lock Files

Lock files store the hash of your package's content. If the content hash of a package you want to install matches with the lock file, it will ensure package repeatability.

### Package Source Mapping

Package Source Mapping allows you to centrally declare which source each package in your solution should restore from in your nuget.config file. This is the primary defense against dependency confusion attacks.

### Client Trust Policies

There are policies you can opt into in which you require the packages you use to be signed. This allows you to trust a package author, as long as it is author signed, or trust a package if it is owned by a specific user or account that is repository signed by NuGet.org.

### Author Package Signing

Author signing allows a package author to stamp their identity on a package and for a consumer to verify it came from them. This protects against content tampering and serves as a single source of truth about the origin and authenticity of the package.

### Two-Factor Authentication (2FA)

Every account on nuget.org has 2FA enabled. This adds an extra layer of security.

### Package ID Prefix Reservation

To protect the identity of packages, you can reserve a package ID prefix with your respective namespace to associate a matching owner. Reserved prefixes prevent others from publishing packages under reserved namespaces.

### Reproducible Builds

Reproducible builds create binaries that are byte-for-byte identical each time you build, containing source code links and compiler metadata that enable package consumers to recreate the binary directly and validate that the build environment has not been compromised.

### NuGet Configuration Best Practices

Add a nuget.config file in the root of your project repository with `clear` elements to ensure no user or machine specific configuration is applied. Use package sources that you trust. When using multiple public and private NuGet source feeds, a package can be downloaded from any of the feeds -- use Package Source Mapping to control this.

### Build Agent Security

Build agents (CI agents) that are not reset to an initial state after every build have multiple risks. Directories should be configured to a directory that the CI agent cleans after every build.

### GitHub Secret Scanning

GitHub scans repositories for NuGet API keys to prevent fraudulent uses of secrets that were accidentally committed.
