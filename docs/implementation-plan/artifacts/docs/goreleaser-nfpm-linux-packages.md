<!-- Source: https://goreleaser.com/customization/nfpm/ -->
<!-- Retrieved: 2026-05-12 -->

# GoReleaser nFPM Configuration - Linux Packages

nFPM generates .deb, .rpm, .apk, .ipk, and Archlinux packages.

## Configuration Example

```yaml
nfpms:
  - id: foo
    package_name: foo
    file_name_template: "{{ .ConventionalFileName }}"
    vendor: MyOrg
    homepage: https://example.com
    maintainer: Maintainer <user@example.com>
    description: Package description
    license: MIT
    formats:
      - deb
      - rpm
      - apk
      - archlinux
    dependencies:
      - git
    contents:
      - src: path/to/foo
        dst: /usr/bin/foo
      - src: path/to/foo.conf
        dst: /etc/foo.conf
        type: config
    scripts:
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"
```

## Format-Specific Overrides

- RPM: compression, summary, group, pre/post-transaction scripts
- DEB: data compression, lintian overrides, triggers, predepends
- APK: pre/post-upgrade scripts, signature config
- Archlinux: pkgbase, packager identification
