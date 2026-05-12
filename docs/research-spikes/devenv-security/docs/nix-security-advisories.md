# NixOS/Nix Security Advisories
- **Source**: https://github.com/NixOS/nix/security/advisories
- **Retrieved**: 2026-05-12

## Advisory List (as of May 2026)

1. **Absolute path traversal when unpacking archives to disk**
   - ID: GHSA-gr92-w2r5-qw5p
   - Severity: Moderate
   - Published: May 4, 2026
   - Description: Risk of directory traversal during archive extraction operations.

2. **Coroutine stack-to-heap overflow via unbounded recursion in NAR directory parser**
   - ID: GHSA-vh5x-56v6-4368
   - Severity: High
   - Published: May 4, 2026
   - Description: Memory overflow caused by excessive recursion in NAR parsing.

3. **Sandbox escape: file write via symlink at FOD `.tmp` copy destination**
   - ID: GHSA-g3g9-5vj6-r3gj
   - Severity: Critical
   - Published: Apr 7, 2026
   - Description: Attackers could escape sandbox restrictions using symlink manipulation in fixed-output derivations.

4. **Privilege dropping to build user broke for macOS**
   - ID: GHSA-qc7j-jgf3-qmhg
   - Severity: High
   - Published: Jul 12, 2025
   - Description: macOS-specific failure in user privilege reduction mechanisms.

5. **Credential leak when credentials are used with `<nix/fetchurl.nix>`**
   - ID: GHSA-6fjr-mq49-mm2c
   - Severity: Moderate
   - Published: Sep 26, 2024
   - Description: Authentication credentials exposed during URL fetch operations.

6. **Unsafe NAR unpacking**
   - ID: GHSA-h4vv-h3jq-v493
   - Severity: Critical
   - Published: Sep 10, 2024
   - Description: Insecure handling of NAR archive extraction.

7. **Sandbox escape**
   - ID: GHSA-q82p-44mg-mgh5
   - Severity: Low
   - Published: Jun 27, 2024
   - Description: Method to bypass sandbox isolation restrictions.

8. **Corruption of fixed-output derivations**
   - ID: GHSA-2ffj-w4mj-pg37
   - Severity: Moderate
   - Published: Mar 7, 2024
   - Description: Integrity issues affecting fixed-output build derivations.

9. **macOS sandbox escape via built-in builders**
   - ID: GHSA-wf4c-57rh-9pjg
   - Severity: Low
   - Published: Oct 31, 2024
   - Description: macOS sandbox bypass through default builder mechanisms.

## Pattern Analysis

The advisories reveal recurring themes:
- **Sandbox escapes** (3 advisories): The sandbox is not a perfect security boundary; symlinks and fixed-output derivations are recurring attack vectors
- **NAR handling** (2 advisories): The archive format parsing has been a source of both memory safety and path traversal bugs
- **Platform-specific issues** (2 advisories): macOS sandbox and privilege dropping have different failure modes than Linux
- **Credential handling** (1 advisory): fetchurl can leak credentials
- **FOD integrity** (1 advisory): Fixed-output derivation corruption undermines hash verification
