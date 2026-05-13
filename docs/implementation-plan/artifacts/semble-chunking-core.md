# Semble - Tree-Sitter Chunking Implementation

- **Source**: https://raw.githubusercontent.com/MinishLab/semble/main/src/semble/chunking/core.py
- **Retrieved**: 2026-05-12

```python
from __future__ import annotations

from dataclasses import dataclass
from functools import cache
from logging import getLogger

from tree_sitter import Node, Parser
from tree_sitter_language_pack import SupportedLanguage, get_parser, manifest_languages

logger = getLogger(__name__)

_TREE_SITTER_LANGUAGES: frozenset[str] = frozenset(manifest_languages())


def is_supported_language(language: str) -> bool:
    return language in _TREE_SITTER_LANGUAGES


@dataclass
class ChunkBoundary:
    start: int
    end: int


@cache
def _cached_get_parser(language: SupportedLanguage) -> Parser:
    return get_parser(language)


def _merge_adjacent_chunks(chunks, desired_length):
    """Merge adjacent chunks up to the desired length."""
    merged = []
    current_start = chunks[0].start
    current_end = chunks[0].end
    current_length = current_end - current_start

    for group in chunks[1:]:
        start, end = group.start, group.end
        length = end - start
        if current_length + length > desired_length:
            merged.append(ChunkBoundary(start=current_start, end=current_end))
            current_start = start
            current_end = end
            current_length = length
            continue
        current_end = end
        current_length += length

    merged.append(ChunkBoundary(start=current_start, end=current_end))
    return merged


def _merge_node_inner(node, desired_length):
    """Recursively merge and split nodes."""
    if not node.children:
        return [ChunkBoundary(node.start_byte, node.end_byte)]

    groups = []
    children = node.children
    index = 0

    while index < len(children):
        child = children[index]
        start = child.start_byte
        end = child.end_byte
        length = child.end_byte - child.start_byte
        index += 1
        if length > desired_length:
            groups.extend(_merge_node_inner(child, desired_length))
            continue
        while index < len(children):
            child = children[index]
            child_length = child.end_byte - child.start_byte
            if length + child_length > desired_length:
                break
            end = child.end_byte
            length += child_length
            index += 1
        groups.append(ChunkBoundary(start, end))

    return groups


def chunk(text, language, desired_length):
    """Chunk source code using tree-sitter AST."""
    if not text.strip():
        return []
    as_bytes = text.encode("utf-8")
    parser = _cached_get_parser(language)
    root = parser.parse(as_bytes).root_node
    chunks = []
    for chunk_boundary in _merge_node(root, desired_length):
        start_char = len(as_bytes[: chunk_boundary.start].decode("utf-8"))
        end_char = len(as_bytes[: chunk_boundary.end].decode("utf-8"))
        chunks.append(ChunkBoundary(start=start_char, end=end_char))
    return chunks
```
