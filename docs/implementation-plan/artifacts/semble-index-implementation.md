# Semble - Index Implementation (SembleIndex)

- **Source**: https://raw.githubusercontent.com/MinishLab/semble/main/src/semble/index/index.py
- **Retrieved**: 2026-05-12

```python
from __future__ import annotations

import os
import subprocess
import tempfile
from collections import defaultdict
from pathlib import Path

import numpy as np
import numpy.typing as npt
from bm25s import BM25

from semble.index.create import create_index_from_path
from semble.index.dense import SelectableBasicBackend, load_model
from semble.search import search_bm25, search_hybrid, search_semantic
from semble.stats import save_search_stats
from semble.types import CallType, Chunk, Encoder, IndexStats, SearchMode, SearchResult

_GIT_CLONE_TIMEOUT = int(os.environ.get("SEMBLE_CLONE_TIMEOUT", 60))


class SembleIndex:
    """Fast local code index with hybrid search."""

    def __init__(
        self,
        model: Encoder,
        bm25_index: BM25,
        semantic_index: SelectableBasicBackend,
        chunks: list[Chunk],
        root: Path | None = None,
    ) -> None:
        self.model: Encoder = model
        self.chunks: list[Chunk] = chunks
        self._bm25_index: BM25 = bm25_index
        self._semantic_index: SelectableBasicBackend = semantic_index
        self._root: Path | None = root
        self._file_sizes: dict[str, int] = self._compute_file_sizes(root) if root else {}
        self._file_mapping, self._language_mapping = self._populate_mapping()

    def _populate_mapping(self) -> tuple[dict[str, list[int]], dict[str, list[int]]]:
        language_to_id = defaultdict(list)
        file_to_id = defaultdict(list)
        for i, chunk in enumerate(self.chunks):
            language = chunk.language
            if language:
                language_to_id[language].append(i)
            file_to_id[chunk.file_path].append(i)
        return dict(file_to_id), dict(language_to_id)

    def _compute_file_sizes(self, root: Path) -> dict[str, int]:
        sizes: dict[str, int] = {}
        for chunk in self.chunks:
            if chunk.file_path in sizes:
                continue
            try:
                sizes[chunk.file_path] = len((root / chunk.file_path).read_text(encoding="utf-8", errors="replace"))
            except OSError:
                pass
        return sizes

    @property
    def stats(self) -> IndexStats:
        language_counts: dict[str, int] = defaultdict(int)
        for chunk in self.chunks:
            if chunk.language:
                language_counts[chunk.language] += 1
        return IndexStats(
            indexed_files=len(self._file_mapping),
            total_chunks=len(self.chunks),
            languages=dict(language_counts),
        )

    @classmethod
    def from_path(cls, path, model=None, extensions=None, ignore=None, include_text_files=False) -> SembleIndex:
        """Create and index a SembleIndex from a directory."""
        model = model or load_model()
        path = Path(path)
        if not path.exists():
            raise FileNotFoundError(f"Path does not exist: {path}")
        if not path.is_dir():
            raise NotADirectoryError(f"Path is not a directory: {path}")
        path = path.resolve()
        bm25, vicinity, chunks = create_index_from_path(
            path, model=model, extensions=extensions, ignore=ignore,
            include_text_files=include_text_files, display_root=path,
        )
        return SembleIndex(model, bm25, vicinity, chunks, root=path)

    @classmethod
    def from_git(cls, url, ref=None, model=None, extensions=None, ignore=None, include_text_files=False) -> SembleIndex:
        """Clone a git repository and index it."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            cmd = ["git", "clone", "--depth", "1", *(["--branch", ref] if ref else []), "--", url, tmp_dir]
            try:
                result = subprocess.run(
                    cmd, capture_output=True, text=True, stdin=subprocess.DEVNULL, timeout=_GIT_CLONE_TIMEOUT
                )
            except FileNotFoundError:
                raise RuntimeError("git is not installed or not on PATH") from None
            except subprocess.TimeoutExpired:
                raise RuntimeError(f"git clone timed out for {url!r} (limit: {_GIT_CLONE_TIMEOUT} s)") from None
            if result.returncode != 0:
                raise RuntimeError(f"git clone failed for {url!r}:\n{result.stderr.strip()}")
            model = model or load_model()
            resolved_path = Path(tmp_dir).resolve()
            bm25, vicinity, chunks = create_index_from_path(
                resolved_path, model=model, extensions=extensions, ignore=ignore,
                include_text_files=include_text_files, display_root=resolved_path,
            )
            return SembleIndex(model, bm25, vicinity, chunks, root=resolved_path)

    def search(self, query, top_k=10, mode=SearchMode.HYBRID, alpha=None,
               filter_languages=None, filter_paths=None) -> list[SearchResult]:
        """Search the index and return the top-k most relevant chunks."""
        if not self.chunks or not query.strip():
            return []
        selector = self._get_selector_vector(filter_languages, filter_paths)
        if mode == SearchMode.BM25:
            results = search_bm25(query, self._bm25_index, self.chunks, top_k, selector=selector)
        elif mode == SearchMode.SEMANTIC:
            results = search_semantic(query, self.model, self._semantic_index, self.chunks, top_k, selector=selector)
        elif mode == SearchMode.HYBRID:
            results = search_hybrid(
                query, self.model, self._semantic_index, self._bm25_index, self.chunks, top_k,
                alpha=alpha, selector=selector
            )
        else:
            raise ValueError(f"Unknown search mode: {mode!r}")
        save_search_stats(results, CallType.SEARCH, self._file_sizes)
        return results

    def find_related(self, source, *, top_k=5) -> list[SearchResult]:
        """Return chunks semantically similar to the given chunk or search result."""
        target = source.chunk if isinstance(source, SearchResult) else source
        selector = self._get_selector_vector(filter_languages=[target.language]) if target.language else None
        results = search_semantic(target.content, self.model, self._semantic_index, self.chunks, top_k + 1, selector)
        results = [r for r in results if r.chunk != target][:top_k]
        save_search_stats(results, CallType.FIND_RELATED, self._file_sizes)
        return results

    def _get_selector_vector(self, filter_languages=None, filter_paths=None):
        selector = []
        for language in filter_languages or []:
            selector.extend(self._language_mapping.get(language, []))
        for filename in filter_paths or []:
            selector.extend(self._file_mapping.get(filename, []))
        return np.unique(selector) if selector else None
```
