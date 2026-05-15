<!-- Source: https://github.com/jojo2357/kiwix-zim-updater -->
<!-- Retrieved: 2026-05-14 -->

# kiwix-zim-updater: Automatic ZIM Library Maintenance

## Core Function
Automatically maintains a local ZIM library by detecting and downloading newer versions from download.kiwix.org. "I wanted an easy way to ensure my ZIM library was kept updated without actually needing to check every ZIM individually."

## Update Detection Method
Parses ZIM filenames in local directory, compares against available files on download.kiwix.org using standardized naming convention. Checks "each ZIM against what is on the `download.kiwix.org` website via the file name Year-Month part."

Relies on Kiwix's standardized filename format — "This script is only for ZIM(s) hosted by `download.kiwix.org` due to the file naming standard they use." Renamed or third-party ZIMs won't be processed.

## ZIM Filename Convention
Standard format: `<site>_<language>_<type>_<year>-<month>.zim`
Example: `unix.stackexchange.com_en_all_2026-02.zim`

The year-month suffix is the version indicator. Newer months = newer version.

## API/Directory Queried
Queries download.kiwix.org's file listings. No formal API — works by parsing publicly available file metadata from the website directory listings.

## Version Comparison Logic
Extracts date information from standard Kiwix filenames. When newer versions found, downloads replacements and optionally verifies checksums before replacing old files.

## Implications for gdev
- ZIM files follow a predictable naming convention suitable for automation
- download.kiwix.org directory listings can be parsed for update detection
- Checksum verification is available for downloaded files
- `qsdev outdated` could implement similar filename-based version comparison
- `qsdev update` could automate the download-verify-replace cycle
