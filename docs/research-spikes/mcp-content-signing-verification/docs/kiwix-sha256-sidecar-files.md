# Kiwix SHA-256 Sidecar Checksum Files

- **Source URLs**: 
  - https://github.com/jojo2357/kiwix-zim-updater (kiwix-zim-updater.sh)
  - https://github.com/kiwix/kiwix-build/issues/237
  - Web search results for kiwix SHA-256 verification
- **Retrieved**: 2026-05-14

## URL Pattern

SHA-256 checksum files are available at:
```
https://download.kiwix.org/zim/<category>/<filename>.zim.sha256
```

Example:
```
https://download.kiwix.org/zim/wikipedia/wikipedia_en_all_maxi_2026-02.zim.sha256
```

## File Format

Standard sha256sum format — 64-character hex hash followed by the filename:
```
<64-char-hex-hash>  <filename.zim>
```

The kiwix-zim-updater script extracts the hash with:
```bash
ExpectedHash=$(grep -ioP "^[0-9a-f]{64}" <"$OldZIMPath.sha256")
```

## Verification

Standard sha256sum verification:
```bash
cd "$ZIMPath" && sha256sum --status -c "$NewZIM.sha256"
```

## Discovery

- Available through browse.library.kiwix.org alongside download links
- Also available in Kiwix catalog metadata as `<hash type="sha-256">` XML elements
- The .sha256 files are NOT listed in the directory index at download.kiwix.org/zim/ — they are served by the backend but not shown in directory listings

## History

- Issue kiwix/kiwix-build#237 (2018): User requested checksum info be more visible
- Resolution: Added to FAQ and library browser interface
- Note: These are transit-integrity checksums, not cryptographic signatures — they verify the download matches what the server has, but do not prove who created the content
