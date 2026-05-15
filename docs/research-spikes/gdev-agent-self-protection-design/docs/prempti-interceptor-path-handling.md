<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/hooks/claude-code/src/main.rs -->
<!-- Retrieved: 2026-05-15 -->
<!-- Extraction: Path canonicalization architecture -->

# Prempti Interceptor Path Handling

The interceptor does NOT perform path canonicalization. It is a thin passthrough that:

1. Reads hook JSON from stdin
2. Extracts only `tool_use_id` for correlation
3. Forwards the entire raw JSON to the plugin broker via Unix socket
4. Maps the broker's verdict back to Claude Code's hook response format

```rust
#[derive(Deserialize)]
struct HookInputMinimal {
    #[serde(default)]
    tool_use_id: String,
}
```

All path canonicalization, working directory resolution, and tool input field extraction
happen in the **plugin broker** (Rust Falco plugin), not in the interceptor.

The plugin broker is responsible for:
- Extracting `tool.file_path` (raw) from the tool input JSON
- Computing `tool.real_file_path` (canonicalized, symlinks resolved)
- Extracting `agent.cwd` (raw) and computing `agent.real_cwd` (canonicalized)
- Extracting `tool.input_command` from Bash tool inputs

Rules use `real_*` fields for policy enforcement and raw fields for audit logging.
