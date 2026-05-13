# Semble - Types and Data Model

- **Source**: https://raw.githubusercontent.com/MinishLab/semble/main/src/semble/types.py
- **Retrieved**: 2026-05-12

```python
from collections.abc import Sequence
from dataclasses import dataclass, field
from enum import Enum
from typing import Protocol, TypeAlias

import numpy as np
import numpy.typing as npt

EmbeddingMatrix: TypeAlias = npt.NDArray[np.float32]


class SearchMode(str, Enum):
    HYBRID = "hybrid"
    SEMANTIC = "semantic"
    BM25 = "bm25"


class CallType(str, Enum):
    SEARCH = "search"
    FIND_RELATED = "find_related"


class Encoder(Protocol):
    def encode(self, texts: Sequence[str], /) -> EmbeddingMatrix: ...


@dataclass(frozen=True, slots=True)
class Chunk:
    content: str
    file_path: str
    start_line: int
    end_line: int
    language: str | None = None

    @property
    def location(self) -> str:
        return f"{self.file_path}:{self.start_line}-{self.end_line}"


@dataclass(frozen=True, slots=True)
class SearchResult:
    chunk: Chunk
    score: float
    source: SearchMode


@dataclass(frozen=True, slots=True)
class IndexStats:
    indexed_files: int = 0
    total_chunks: int = 0
    languages: dict[str, int] = field(default_factory=dict)
```
