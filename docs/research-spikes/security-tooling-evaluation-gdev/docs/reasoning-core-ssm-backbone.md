# reasoning-core ssm_backbone.py (Multi-Backend Embedder Loader)

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/src/ssm_backbone.py
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Core Purpose
The module handles the lifecycle of embedding models that power architectural-impact and coherence scoring, with deferred heavyweight imports to maintain startup efficiency.

## Available Backends
The system supports five configurable embedders via the `RC_EMBEDDER` environment variable:

1. **codestral-mamba** (default): Mistral's code-pretrained Mamba-2 model with 256K context capability
2. **mamba-130m**: Legacy fallback using state-spaces implementation
3. **bge-code**: BAAI's code-specialized transformer (~4GB)
4. **unixcoder-base**: Microsoft's code transformer baseline
5. **random-mamba**: Randomly-initialized Mamba-2 for falsifiability testing

## Security Architecture
The implementation emphasizes supply-chain hardening through:

- **Allowlist validation**: Only registered checkpoints can load via `_ALLOWED_CHECKPOINTS`
- **SHA pinning**: Commit SHAs are enforced; mutable refs like "main" are rejected
- **Environment overrides**: Operators can pin versions via `RC_<REPO_SLUG>_REVISION` (SHA format only)
- **Defense-in-depth checks**: Multiple validation layers prevent arbitrary repository loading

## Key Functions

**`load_backbone()`**: Primary entry point returning cached or newly-loaded (model, tokenizer) tuple

**`embed(text)`**: Generates 1D pooled embeddings using backend-specific strategies (mean-pooling for SSMs, CLS-token for transformers)

**`ast_to_tokens()`**: Linearizes Abstract Syntax Trees with source code into deterministic token strings

**`get_handle()`**: Returns the loaded backbone handle on-demand

## State Management
Uses thread-safe singleton pattern with `_LOAD_LOCK` to ensure single-instance model loading across concurrent access attempts.
