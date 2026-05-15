<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/plugins/coding-agents-plugin/src/event.rs -->
<!-- Retrieved: 2026-05-15 -->
<!-- Extraction: Path canonicalization and field extraction logic -->

# Prempti Plugin Path Canonicalization Logic

## 1. File Path Extraction from tool_input JSON

```rust
fn extract_raw_file_path(tool_name: &str, tool_input: &serde_json::Value) -> String {
    if !matches!(tool_name, "Write" | "Edit" | "Read") {
        return String::new();
    }
    tool_input
        .get("file_path")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string()
}
```

Extracts `file_path` only from Write, Edit, and Read tools; returns empty string otherwise.

## 2. Real File Path Computation

```rust
fn resolve_file_path(file_path: &str, resolved_cwd: &str) -> String {
    if file_path.is_empty() {
        return String::new();
    }
    let path = Path::new(file_path);
    let abs = if path.is_absolute() {
        PathBuf::from(file_path)
    } else {
        let mut p = PathBuf::from(resolved_cwd);
        p.push(file_path);
        p
    };
    // Try filesystem canonicalization first.
    if let Ok(resolved) = std::fs::canonicalize(&abs) {
        return normalize_separators(resolved.to_string_lossy().into_owned());
    }
    // Fallback: lexical normalization.
    normalize_separators(normalize_path(&abs).to_string_lossy().into_owned())
}
```

Two-tier approach: `std::fs::canonicalize()` resolves symlinks; falls back to lexical normalization when the path doesn't exist yet (common for Write operations to new files).

## 3. Real CWD Computation

```rust
fn resolve_path(raw: &str) -> String {
    if raw.is_empty() {
        return String::new();
    }
    if let Ok(resolved) = std::fs::canonicalize(raw) {
        return normalize_separators(resolved.to_string_lossy().into_owned());
    }
    normalize_separators(normalize_path(Path::new(raw)).to_string_lossy().into_owned())
}
```

Same two-tier approach for working directory.

## 4. Tool Input Command Extraction

```rust
fn extract_command(tool_name: &str, tool_input: &serde_json::Value) -> String {
    if tool_name != "Bash" {
        return String::new();
    }
    tool_input
        .get("command")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string()
}
```

## 5. Lexical Normalization (fallback when filesystem canonicalization fails)

```rust
fn normalize_path(path: &Path) -> PathBuf {
    let mut result = PathBuf::new();
    for component in path.components() {
        match component {
            Component::ParentDir => {
                if result.file_name().is_some() {
                    result.pop();
                }
            }
            Component::CurDir => {}
            other => result.push(other),
        }
    }
    result
}
```

## 6. Windows Separator Normalization

```rust
fn normalize_separators(path: String) -> String {
    #[cfg(windows)]
    {
        let stripped = path.strip_prefix(r"\\?\").unwrap_or(&path).to_string();
        stripped.replace('\\', "/")
    }
    #[cfg(not(windows))]
    { path }
}
```

## Key Design Insight

The two-tier canonicalization (filesystem first, lexical fallback) is critical because:
- Write tool targets files that don't exist yet — `canonicalize()` will fail
- The lexical fallback still resolves `../` traversal but won't resolve symlinks
- Rules should use `real_file_path` for enforcement (catches symlink tricks)
- Rules should use `file_path` for audit (preserves what the agent declared)
