# Semble - Dense Embedding Backend

- **Source**: https://raw.githubusercontent.com/MinishLab/semble/main/src/semble/index/dense.py
- **Retrieved**: 2026-05-12

```python
import numpy as np
import numpy.typing as npt
from huggingface_hub.utils.tqdm import disable_progress_bars
from model2vec import StaticModel
from vicinity.backends.basic import CosineBasicBackend
from vicinity.datatypes import QueryResult
from vicinity.utils import normalize

from semble.types import Chunk, Encoder

_DEFAULT_MODEL_NAME = "minishlab/potion-code-16M"


def load_model(model_path: str | None = None) -> Encoder:
    """Return the current model, loading the default if none was provided."""
    if model_path is None:
        model_path = _DEFAULT_MODEL_NAME
    disable_progress_bars()
    try:
        model = StaticModel.from_pretrained(model_path)
    finally:
        disable_progress_bars()
    return model


def embed_chunks(model: Encoder, chunks: list[Chunk]) -> npt.NDArray[np.float32]:
    """Embed chunks using the configured model."""
    if not chunks:
        return np.empty((0, 256), dtype=np.float32)
    return np.array(model.encode([c.content for c in chunks]), dtype=np.float32)


class SelectableBasicBackend(CosineBasicBackend):
    """Extended cosine backend with selector-based filtering."""

    def _selector_dist(self, x, selector):
        x_norm = normalize(x)
        sim = x_norm.dot(self._vectors[selector].T)
        return 1 - sim

    def query(self, vectors, k, selector=None):
        if k < 1:
            raise ValueError(f"k should be >= 1, is now {k}")
        out = []
        num_vectors = len(self.vectors)
        effective_k = min(k, num_vectors)
        if selector is not None:
            effective_k = min(effective_k, len(selector))
        for index in range(0, len(vectors), 1024):
            batch = vectors[index : index + 1024]
            if selector is not None:
                distances = self._selector_dist(batch, selector)
            else:
                distances = self._dist(batch)
            indices = np.argpartition(distances, kth=effective_k - 1, axis=1)[:, :effective_k]
            sorted_indices = np.take_along_axis(
                indices, np.argsort(np.take_along_axis(distances, indices, axis=1)), axis=1
            )
            sorted_distances = np.take_along_axis(distances, sorted_indices, axis=1)
            if selector is not None:
                sorted_indices = selector[sorted_indices]
            out.extend(zip(sorted_indices, sorted_distances))
        return out
```

## Key Details

- Default model: `minishlab/potion-code-16M` (16M parameter static embedding model from HuggingFace)
- Embedding dimension: 256 (implied by the empty array shape)
- Uses cosine distance for similarity (converted to similarity score in search.py via `1.0 - distance`)
- Model is downloaded from HuggingFace Hub on first use, then cached locally
- No GPU required - model2vec StaticModel is CPU-only by design
