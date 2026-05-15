<!-- Source: https://github.com/modu-ai/moai-adk/issues/342 -->
<!-- Retrieved: 2026-05-15 -->

# macOS Unicode NFD/NFC Path Mismatch Issue

## The Problem

The moai-adk security check fails on macOS systems when project directories contain non-ASCII characters. Users encounter this error despite files being in the correct location:

"Path traversal detected: file is outside project directory"

## Root Cause Analysis

macOS filesystems (HFS+/APFS) store filenames using Unicode NFD (decomposed form), while Claude Code transmits paths in the same format via stdin JSON. The moai binary's validation logic performs raw string comparison without Unicode normalization:

- **NFD representation** (decomposed): Korean `코딩` = `코딩` (5 codepoints)
- **NFC representation** (composed): Korean `코딩` = `코딩` (2 codepoints)

These encode identical characters but represent different byte sequences, causing `strings.HasPrefix()` comparisons to fail.

## Affected Scenarios

**Impacted environments:**
- macOS 24.6.0 (darwin/arm64)
- moai-adk version 2.0.3
- Any project path containing: Korean characters (`~/코딩/`), Japanese, Chinese, accented Latin characters

**Operations blocked:** Write, Edit, and Bash commands through hook system

## Workaround Implementation

The issue reporter applied Python-based Unicode normalization to all 7 hook wrapper scripts, normalizing environment variables and stdin JSON from NFD to NFC before invoking moai. This incurred approximately 50ms latency per hook invocation due to Python startup overhead.

## Recommended Solution

Implement path normalization in the Go binary using the standard library:

```go
import "golang.org/x/text/unicode/norm"

func normalizePath(p string) string {
    return norm.NFC.String(p)
}
```

Normalize all paths to NFC before security validation comparisons.
