---
source: Multiple web searches (pixi.prefix.dev, prefix.dev/blog)
retrieved: 2026-03-20
---

# Pixi pixi.toml Configuration Reference

## Configuration Structure

Pixi uses pixi.toml (or can integrate with pyproject.toml for Python projects).

## Full Example

```toml
[project]
name = "my_project"
requires-python = ">=3.9"
dependencies = [
    "numpy",
    "pandas",
    "matplotlib",
    "ruff",
]

[tool.pixi.workspace]
channels = ["conda-forge"]
platforms = ["linux-64", "osx-arm64", "osx-64", "win-64"]

[tool.pixi.dependencies]
compilers = "*"
cmake = "*"

[tool.pixi.tasks]
start = "python my_project/main.py"
lint = "ruff lint"

[tool.pixi.system-requirements]
cuda = "11.0"

[tool.pixi.feature.test.dependencies]
pytest = "*"

[tool.pixi.feature.test.tasks]
test = "pytest"

[tool.pixi.environments]
test = ["test"]
```

## Advanced Tasks

Tasks can be configured with:
- Commands as lists with documentation
- Task descriptions
- Dependencies on other tasks
- Environment variables
- Cross-platform file operations
- Default environments for tasks

```toml
[tasks]
configure = { cmd = ["cmake", "-G", "Ninja", "-S", ".", "-B", ".build"] }
say-hello = { cmd = ["echo", "hello world"], description = "Greet the world." }
build = { cmd = ["ninja", "-C", ".build"], depends-on = ["configure"] }
run = "python main.py $PIXI_PROJECT_ROOT"
test = { cmd = "pytest", default-environment = "test" }
```

## Features and Environments

Features can include: dependencies, pypi-dependencies, system-requirements, activation information, platforms, channels, and feature-specific tasks.

```toml
[tool.pixi.environments]
default = {features = [], solve-group = "default"}
test = {features = ["test"], solve-group = "default"}
docs = {features = ["docs"], solve-group = "default"}
dev = {features = ["dev"], solve-group = "default"}
```

## Mixed Conda and PyPI Dependencies

```toml
[dependencies]
click = "*"  # Conda dependency

[pypi-dependencies]
flask = "*"  # PyPI dependency
```

## Performance vs Conda

- About 3x faster than micromamba
- Over 10x faster than conda when resolving and installing environments from scratch
- Written in Rust, built atop the rattler library
- Natively supports lockfiles and cross-platform tasks

## Key Differentiators from Conda

1. Project-focused environments tied to projects rather than global system state
2. Built-in lockfiles for automatic dependency locking
3. Task management with cross-platform task runner
4. Multi-environment support
5. Speed through Rust implementation
6. Universal compatibility working with both conda and PyPI packages
