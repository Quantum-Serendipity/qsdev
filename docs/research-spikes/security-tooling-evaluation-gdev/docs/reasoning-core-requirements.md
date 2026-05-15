# reasoning-core requirements.txt

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/requirements.txt
- **Retrieved**: 2026-05-15

---

```
# Hybrid Reasoning Core — runtime + dev deps
# Pinned conservatively. macOS arm64 + Python 3.11+.

# --- HTTP & MCP ---
fastapi>=0.110,<1.0
uvicorn[standard]>=0.29,<1.0
httpx>=0.27,<1.0
pydantic>=2.6,<3.0
mcp[cli]>=1.0,<2.0

# --- Tree-sitter (Python, JavaScript, TypeScript, C#, SQL) ---
tree-sitter>=0.25,<0.27
tree-sitter-python>=0.23
tree-sitter-javascript>=0.23
tree-sitter-typescript>=0.23
tree-sitter-c-sharp>=0.23
tree-sitter-sql>=0.3
tree-sitter-markdown>=0.3
tree-sitter-json>=0.23
tree-sitter-yaml>=0.7
tree-sitter-css>=0.20
tree-sitter-html>=0.20
tree-sitter-dockerfile>=0.2

# --- Real Mamba SSM backbone ---
transformers>=4.39,<5.0
torch>=2.2,<3.0
accelerate>=0.27
safetensors>=0.4
sentencepiece>=0.1.99
einops>=0.7

# --- Cross-platform file locking ---
portalocker>=2.7

# --- Rule engine ---
PyYAML>=6.0

# --- Test harness ---
pytest>=8.0,<9.0
pytest-asyncio>=0.23
pytest-timeout>=2.2
```
