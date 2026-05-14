# Browser Use Monorepo CLAUDE.md (pirate/ef7b8923)
- **Source**: https://gist.github.com/pirate/ef7b8923de3993dd7d96dbbb9c096501
- **Retrieved**: 2026-03-27
- **Significance**: Real-world monorepo CLAUDE.md from the Browser Use AI browser automation project. ~300 lines, comprehensive.

---

## Repository Overview

Monorepo containing multiple projects in the Browser Use ecosystem — AI browser automation framework. Main components:
- **browser-use**: Core Python library for browser automation with LLM agents
- **cloud**: Full-stack cloud platform (FastAPI backend + Next.js frontend)
- **bubus**: Mini event bus library
- **web-ui**: Gradio-based web interface (marked "less important, ignore unless directed")
- **workflow-use**: Chrome extension (marked "less important, ignore unless directed")

## Development Commands

### Python Projects (browser-use, bubus, cloud/backend)
All use `uv`: `uv sync --dev --all-extras`, `pytest tests/`, `uv run pre-commit run --all-files`

### JavaScript/TypeScript Projects (cloud/frontend, workflow-use/ui)
Cloud frontend uses yarn, workflow-use uses npm.

## Architecture Guidelines

### Python Code Style
- async/await throughout, threadsafe and async-optimized
- Tabs for indentation in browser-use and cloud (not spaces), spaces in bubus
- Modern Python 3.12+ typing: `str | int` not `Union[str, int]`
- Pydantic v2 with strict validation, ConfigDict, model_validator()
- Logging in separate `_log_...()` private methods
- Big subcomponents in `service.py` files, types in `views.py` or `models.py`
- Runtime assertions for critical invariants
- Use bubus event bus for inter-component communication involving writes to shared state

### Testing Strategy
1. Write failing tests first
2. Use real objects instead of mocks for everything except LLM
3. Use pytest-httpserver for HTML, never live URLs
4. No need for @pytest.mark.asyncio (auto mode)
5. All tests for a component in single test file
6. Always run with timeout to prevent hanging
7. Final pass to remove duplicated test logic

### Making Changes
1. Check existing tests/, examples/, and docs/
2. If big changes, create proposal comparing approaches
3. Write failing test for new functionality
4. Implement minimal code to pass test
5. Run full test suite
6. Don't ignore warnings or skip tests
7. Update docs/ and examples/

### Working with the Monorepo
- Each sub-project is its own git repository
- Each has own dependencies and build process
- Cross-package changes affect dependent packages
- Don't update web-ui and workflow-use after library changes unless directed

## Notable Characteristics
- ~300 lines, highly structured
- Explicit component priority ("less important, ignore unless directed")
- Project-specific idioms (tabs vs spaces per project)
- Cross-package dependency awareness
- Test philosophy embedded (TDD, real objects not mocks)
- Pydantic v2 specific patterns
- Monorepo navigation guidance
