# NuGet: Manage Package Trust Boundaries
- **Source**: https://learn.microsoft.com/en-us/nuget/consume-packages/installing-signed-packages
- **Retrieved**: 2026-05-12

## Default Behavior

Signed packages don't require any specific action to be installed; however, if the content has been modified since it was signed, the installation is blocked with error NU3008.

**Warning**: Packages signed with untrusted certificates are considered as unsigned and are installed without any warnings or errors like any other unsigned package.

## Configure Package Signature Requirements

Requires NuGet 4.9.0+ and Visual Studio version 15.9 and later on Windows.

### signatureValidationMode = require

```cmd
nuget.exe config -set signatureValidationMode=require
```

```xml
<config>
  <add key="signatureValidationMode" value="require" />
</config>
```

This mode will verify that all packages are signed by any of the certificates trusted in the `nuget.config` file.

### Trust Package Author

```cmd
nuget.exe trusted-signers Add -Name MyCompanyCert -CertificateFingerprint CE40881FF5F0AD3E58965DA20A9F571EF1651A56933748E1BF1C99E537C4E039 -FingerprintAlgorithm SHA256
```

```xml
<trustedSigners>
  <author name="MyCompanyCert">
    <certificate fingerprint="CE40881..." hashAlgorithm="SHA256" allowUntrustedRoot="false" />
  </author>
</trustedSigners>
```

### Trust All Packages from a Repository

```xml
<trustedSigners>
  <repository name="nuget.org" serviceIndex="https://api.nuget.org/v3/index.json">
    <certificate fingerprint="0E5F38F57DC1BCC806D8494F4F90FBCEDD988B..." hashAlgorithm="SHA256" allowUntrustedRoot="false" />
  </repository>
</trustedSigners>
```

### Trust Package Owners

Repository signatures include additional metadata to determine the owners of the package at the time of submission. You can restrict packages from a repository based on a list of owners:

```xml
<trustedSigners>
  <repository name="nuget.org" serviceIndex="https://api.nuget.org/v3/index.json">
    <certificate fingerprint="0E5F38F57DC1BCC806D8494F4F90FBCEDD988B..." hashAlgorithm="SHA256" allowUntrustedRoot="false" />
    <owners>microsoft;nuget</owners>
  </repository>
</trustedSigners>
```

### Untrusted Root Certificates

Use the `allowUntrustedRoot` attribute to enable verification using certificates that do not chain to a trusted root.

### Sync Repository Certificates

Use `nuget.exe trusted-signers sync` to update certificates when repositories rotate them.
