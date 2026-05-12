# Python Conventions

## Type annotations
- Add type hints on all public function parameters and return types.
- Use `from __future__ import annotations` for modern annotation syntax.
- Run a type checker (`mypy` or `pyright`) in CI.

## Data modeling
- Use `dataclasses` for simple data containers.
- Use Pydantic `BaseModel` when validation or serialization is needed.
- Prefer immutable data: use `frozen=True` on dataclasses where possible.

## Resource management
- Use context managers (`with` statements) for files, database connections,
  locks, and any resource requiring cleanup.
- Prefer `pathlib.Path` over `os.path` for filesystem operations.
- Use `contextlib.contextmanager` to create custom context managers.

## Code style
- Format with `ruff format` (or `black`). Run `ruff check .` for linting.
- Follow PEP 8 naming: `snake_case` for functions and variables,
  `PascalCase` for classes, `UPPER_CASE` for module-level constants.
- Keep functions under ~30 lines. Extract helpers for complex logic.
- Use f-strings for string formatting. Avoid `%` formatting and `.format()`.

## Imports
- Group imports: stdlib, third-party, local (separated by blank lines).
- Use absolute imports. Avoid wildcard imports (`from module import *`).
- Sort imports with `isort` or Ruff's import sorting.

## Testing
- Use `pytest` as the test runner.
- Use `@pytest.mark.parametrize` for data-driven tests.
- Use fixtures for setup and teardown instead of `setUp`/`tearDown` methods.
- Mock external dependencies with `unittest.mock.patch` or `pytest-mock`.
- Name test files `test_<module>.py` and test functions `test_<behavior>`.

## Dependencies
- Pin dependencies in `requirements.txt` or `pyproject.toml`.
- Use virtual environments (managed by devenv) for isolation.
- Run `pip-audit` or `safety check` to scan for known vulnerabilities.
