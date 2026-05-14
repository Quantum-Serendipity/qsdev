# Zimcheck — Integrity and Checksum Verification

- **Source URL**: https://raw.githubusercontent.com/openzim/zim-tools/main/src/zimcheck/zimcheck.cpp
- **Retrieved**: 2026-05-14

## Integrity Checks Performed

### 1. Low-Level Integrity Check (Primary)
```cpp
if(enabled_tests.isEnabled(TestType::INTEGRITY)) {
    should_run_full_test = test_integrity(filename, error);
}
```

This foundational check validates basic ZIM file structure using libzim's IntegrityCheck enum (CHECKSUM, DIRENT_PTRS, DIRENT_ORDER, TITLE_INDEX, CLUSTER_PTRS, CLUSTERS_OFFSETS, DIRENT_MIMETYPES). If it fails, subsequent tests are skipped to avoid false positives from corrupted data.

### 2. Internal Checksum Verification (Secondary)
```cpp
if(enabled_tests.isEnabled(TestType::CHECKSUM)) {
    if ( enabled_tests.isEnabled(TestType::INTEGRITY) ) {
        error.infoMsg("[INFO] Avoiding redundant checksum test...");
    } else {
        test_checksum(archive, error);
    }
}
```

The checksum test is automatically skipped if integrity checks already ran (since integrity includes checksum verification), preventing duplicate work.

## Additional Checks

Beyond integrity/checksum, zimcheck also validates:
- URL correctness
- Redundant content detection
- Empty content detection
- MIME type consistency
- Article link validation

## Usage in Production

zimcheck is systematically run in the Zimfarm (Kiwix's build farm) after every ZIM file is created, serving as the primary quality gate before files are published to download.kiwix.org.
