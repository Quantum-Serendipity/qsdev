<!-- Source: https://central.sonatype.org/publish/requirements/ -->
<!-- Retrieved: 2026-05-12 -->

# Maven Central Publishing Requirements

## Core Deployment Requirements

**Javadoc and Sources**: Projects must provide corresponding `-sources.jar` and `-javadoc.jar` files for every JAR deployed. Placeholder JARs with README files are acceptable if comprehensive documentation cannot be provided.

**File Checksums**: All deployed files require:
- `.md5` and `.sha1` checksums (mandatory)
- `.sha256` and `.sha512` checksums (optional but supported)

**GPG/PGP Signatures**: Each file needs an accompanying `.asc` signature file. The documentation notes that "signature files don't need checksum files, nor do checksum files need signature files."

## POM Metadata Requirements

**Coordinates (GAV)**:
- `groupId`: reverse domain name format for namespace
- `artifactId`: unique component identifier
- `version`: cannot end in `-SNAPSHOT`

**Project Information**:
- Name, description, and URL are mandatory
- Dependencies should be included to enable transitive dependency resolution

**License Declaration**: At least one license must be specified with name and URL.

**Developer Information**: Required developer details including name, email, organization, and organization URL.

**SCM Details**: Connection information (read-only and read-write variants) and web front-end URL for source control systems. The requirement states "the URL itself does not need to be public" and can reference private repositories.

## Quality Standards

The requirements exist to "ensure a minimum level of quality" and allow "consumers of your components to automatically access to Javadoc and sources for browsing as well as for display and navigation."

## Namespace Verification

To combat typosquatting and namespace hijacking, Sonatype requires all publishers to prove ownership of their `groupId` namespace, typically by verifying control of an associated domain name. To publish under `com.mycompany`, you must prove you own `mycompany.com`.

For domain verification, Sonatype provides a unique verification code that must be added as a TXT record to the domain's DNS settings. Publishers can also use `io.github.<username>` as a domain if they don't own a custom domain.
