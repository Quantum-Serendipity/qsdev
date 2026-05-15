<!-- Source: https://raw.githubusercontent.com/astral-sh/ruff/main/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Very long README. Key sections preserved; the full "Who's Using Ruff" list is extensive. -->

# Ruff

[![Ruff](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/astral-sh/ruff/main/assets/badge/v2.json)](https://github.com/astral-sh/ruff)
[![image](https://img.shields.io/pypi/v/ruff.svg)](https://pypi.python.org/pypi/ruff)
[![image](https://img.shields.io/pypi/l/ruff.svg)](https://github.com/astral-sh/ruff/blob/main/LICENSE)
[![image](https://img.shields.io/pypi/pyversions/ruff.svg)](https://pypi.python.org/pypi/ruff)
[![Actions status](https://github.com/astral-sh/ruff/workflows/CI/badge.svg)](https://github.com/astral-sh/ruff/actions)
[![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?logo=discord&logoColor=white)](https://discord.com/invite/astral-sh)

[**Docs**](https://docs.astral.sh/ruff/) | [**Playground**](https://play.ruff.rs/)

An extremely fast Python linter and code formatter, written in Rust.

<p align="center">
  <picture align="center">
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/1309177/232603514-c95e9b0f-6b31-43de-9a80-9e844173fd6a.svg">
    <source media="(prefers-color-scheme: light)" srcset="https://user-images.githubusercontent.com/1309177/232603516-4fb4892d-585c-4b20-b810-3db9161831e4.svg">
    <img alt="Shows a bar chart with benchmark results." src="https://user-images.githubusercontent.com/1309177/232603516-4fb4892d-585c-4b20-b810-3db9161831e4.svg">
  </picture>
</p>

<p align="center">
  <i>Linting the CPython codebase from scratch.</i>
</p>

- 10-100x faster than existing linters (like Flake8) and formatters (like Black)
- Installable via `pip`
- `pyproject.toml` support
- Python 3.14 compatibility
- Drop-in parity with Flake8, isort, and Black
- Built-in caching, to avoid re-analyzing unchanged files
- Fix support, for automatic error correction (e.g., automatically remove unused imports)
- Over 900 built-in rules, with native re-implementations of popular Flake8 plugins
- First-party editor integrations for VS Code and more
- Monorepo-friendly, with hierarchical and cascading configuration

Ruff aims to be orders of magnitude faster than alternative tools while integrating more functionality behind a single, common interface.

Ruff can be used to replace Flake8 (plus dozens of plugins), Black, isort, pydocstyle, pyupgrade, autoflake, and more, all while executing tens or hundreds of times faster than any individual tool.

Ruff is extremely actively developed and used in major open-source projects like:
- Apache Airflow
- Apache Superset
- FastAPI
- Hugging Face
- Pandas
- SciPy

Ruff is backed by [Astral](https://astral.sh), the creators of [uv](https://github.com/astral-sh/uv) and [ty](https://github.com/astral-sh/ty).

## Testimonials

**Sebastian Ramirez**, creator of FastAPI:
> Ruff is so fast that sometimes I add an intentional bug in the code just to confirm it's actually running and checking the code.

**Nick Schrock**, founder of Elementl, co-creator of GraphQL:
> Why is Ruff a gamechanger? Primarily because it is nearly 1000x faster. Literally. Not a typo. On our largest module (dagster itself, 250k LOC) pylint takes about 2.5 minutes, parallelized across 4 cores on my M1. Running ruff against our entire codebase takes .4 seconds.

**Bryan Van de Ven**, co-creator of Bokeh, original author of Conda:
> Ruff is ~150-200x faster than flake8 on my machine, scanning the whole repo takes ~0.2s instead of ~20s. This is an enormous quality of life improvement for local dev.

**Timothy Crosley**, creator of isort:
> Just switched my first project to Ruff. Only one downside so far: it's so fast I couldn't believe it was working till I intentionally introduced some errors.

## Getting Started

### Installation

Invoke Ruff directly with `uvx`:
```shell
uvx ruff check   # Lint all files in the current directory.
uvx ruff format  # Format all files in the current directory.
```

Or install Ruff with `uv` (recommended), `pip`, or `pipx`:
```shell
uv tool install ruff@latest
pip install ruff
pipx install ruff
```

Standalone installers:
```shell
# On macOS and Linux.
curl -LsSf https://astral.sh/ruff/install.sh | sh
# On Windows.
powershell -c "irm https://astral.sh/ruff/install.ps1 | iex"
```

### Usage

```shell
ruff check                          # Lint all files in the current directory
ruff format                         # Format all files in the current directory
```

Also usable as a pre-commit hook, VS Code extension, and GitHub Action.

### Configuration

Ruff can be configured through `pyproject.toml`, `ruff.toml`, or `.ruff.toml` file.

## Rules

Ruff supports over 900 lint rules, many inspired by popular tools like Flake8, isort, pyupgrade, and others. All rules are re-implemented in Rust as first-party features.

## Who's Using Ruff?

Used by Apache Airflow, FastAPI, Hugging Face, Pandas, PyTorch, SciPy, Home Assistant, Microsoft, Mozilla Firefox, and many more.

## License

MIT License

<div align="center">
  <a target="_blank" href="https://astral.sh" style="background:none">
    <img src="https://raw.githubusercontent.com/astral-sh/ruff/main/assets/svg/Astral.svg" alt="Made by Astral">
  </a>
</div>
