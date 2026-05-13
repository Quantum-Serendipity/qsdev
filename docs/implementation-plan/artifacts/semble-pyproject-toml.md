# Semble - pyproject.toml

- **Source**: https://raw.githubusercontent.com/MinishLab/semble/main/pyproject.toml
- **Retrieved**: 2026-05-12

```toml
[build-system]
requires = ["setuptools>=64", "setuptools_scm>=8"]
build-backend = "setuptools.build_meta"

[project]
name = "semble"
description = "Fast and Accurate Code Search for Agents"
authors = [{name = "Thomas van Dongen", email = "thomas123@live.nl"}, { name = "Stéphan Tulkens", email = "stephantul@gmail.com"}]
readme = { file = "README.md", content-type = "text/markdown" }
dynamic = ["version"]
license = { file = "LICENSE" }
requires-python = ">=3.10"
keywords = ["code-search", "hybrid-search", "semantic-search", "mcp", "agent", "rag", "embeddings"]
classifiers = [
    "Development Status :: 4 - Beta",
    "Intended Audience :: Developers",
    "Intended Audience :: Science/Research",
    "Topic :: Scientific/Engineering :: Artificial Intelligence",
    "Topic :: Software Development :: Libraries",
    "License :: OSI Approved :: MIT License",
    "Programming Language :: Python :: 3 :: Only",
    "Programming Language :: Python :: 3.10",
    "Programming Language :: Python :: 3.11",
    "Programming Language :: Python :: 3.12",
    "Programming Language :: Python :: 3.13",
    "Natural Language :: English",
]
dependencies = [
    "model2vec>=0.4.0",
    "vicinity>=0.4.4",
    "numpy>=1.24.0",
    "bm25s>=0.2.0",
    "pathspec>=0.12",
    "tree-sitter>=0.25",
    "tree-sitter-language-pack>=1.0,<1.8.0,!=1.6.3"
]

[project.optional-dependencies]
mcp = [
    "mcp>=1.0,<2.0",
    "watchfiles>=0.21",
]
benchmark = [
    "sentence-transformers>=3.0",
    "numpy>=1.24.0",
    "einops>=0.8.2",
    "matplotlib>=3.7",
    "tiktoken>=0.7",
    "openai>=1.50",
]
dev = [
    "pytest>=8.0",
    "pytest-cov>=5.0",
    "ruff>=0.9.0",
    "mypy>=1.0",
    "pydoclint>=0.5.3",
    "pre-commit>=3.0",
]

[project.urls]
"Homepage" = "https://github.com/MinishLab/semble"
"Bug Reports" = "https://github.com/MinishLab/semble/issues"
"Source" = "https://github.com/MinishLab/semble"

[project.scripts]
semble = "semble.cli:main"

[tool.setuptools]
package-dir = {"" = "src"}
license-files = ["LICENSE*"]

[tool.setuptools.packages.find]
where = ["src"]
include = ["semble*"]

[tool.setuptools.package-data]
semble = ["py.typed", "agents/*.md"]

[tool.setuptools_scm]

[tool.setuptools.dynamic]
version = {attr = "semble.version.__version__"}
```
