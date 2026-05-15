<!-- Source: https://goreleaser.com/pro/ -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser Pro-Exclusive Features (Complete List)

The following features are ONLY available in GoReleaser Pro (paid, closed-source):

1. **macOS Installers (`.pkg`)** - Create macOS installers
2. **Windows Installers (`.exe`) with NSIS** - Create Windows installers with NSIS
3. **Smart SemVer Tag Sorting** - Smart SemVer tag sorting
4. **NPM Registry Publishing** - Publish to NPM registries
5. **Native macOS Code Signing** - Native sign and notarize macOS App Bundles, Disk Images, and Installers
6. **AI-Enhanced Changelog** - Use AI to improve/format your release notes
7. **Conditional Artifact Filtering** - Further filter artifacts with `if` statements
8. **macOS App Bundles (`.app`)** - Create macOS App Bundles
9. **CloudSmith Integration** - Easily create alpine, apt, and yum repositories with CloudSmith
10. **Global Configuration Defaults** - Have global defaults for homepage, description, etc
11. **Pre-Publish Hooks** - Run hooks before publishing artifacts
12. **Cross-Platform Publishing** - Cross publish (e.g. releases to GitLab, pushes Homebrew Tap to GitHub)
13. **DockerHub Description Management** - Keep DockerHub image descriptions up to date
14. **macOS Disk Images (`.dmg`)** - Create macOS disk images
15. **Windows MSI Installers with Wix** - Create Windows installers with Wix
16. **Single-Target Builds** - Use `goreleaser release --single-target`
17. **Template Entire Files** - Template entire files and add them to releases
18. **Artifacts Template Variable** - Use `.Artifacts` template variable
19. **Build Splitting/Merging** - Split and merge builds to speed up releases
20. **Advanced Changelog Options** - More changelog options: Filter commits by path & subgroups
21. **Archive Hooks** - Custom before and after hooks for archives
22. **Release Preparation** - Prepare a release with `goreleaser release --prepare`
23. **Changelog Preview** - Preview your next release's changelog with `goreleaser changelog`
24. **Nightly Builds** - Continuously release nightly builds
25. **Prebuilt Binary Import** - Import pre-built binaries with prebuilt builder
26. **Podman Support** - Rootless build Docker images and manifests with Podman
27. **GemFury Integration** - Easily create apt, yum, and alpine repositories with gemfury.io
28. **Configuration Includes** - Reuse configuration files with include keyword
29. **Global After Hooks** - Run commands after release with global after hooks
30. **Monorepo Support** - Use GoReleaser within your monorepo
31. **Custom Template Variables** - Create custom template variables

## SBOM-Relevant Pro Features

From the SBOM docs, these artifact types for the `sboms:` block are Pro-only:
- `installer` - Generate SBOMs for MSI, NSIS, macOS pkg installers
- `diskimage` - Generate SBOMs for macOS DMG disk images

## Key Finding
**Core SBOM generation is NOT a Pro feature.** The `sboms:` configuration block with `archive`, `binary`, `source`, `package`, and `any` artifact types are all available in the free/OSS version. Only the `installer` and `diskimage` artifact types (which are themselves Pro-only artifact types) require Pro.
