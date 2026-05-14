# Example CLAUDE.md: Context Engineering Intro (coleam00)
- **Source**: https://raw.githubusercontent.com/coleam00/context-engineering-intro/main/CLAUDE.md
- **Retrieved**: 2026-03-27
- **Significance**: Popular "vibe coding" style CLAUDE.md (~100 lines) that references PLANNING.md and TASK.md external files.

---

### Project Awareness & Context
- **Always read `PLANNING.md`** at the start of a new conversation to understand architecture, goals, style, constraints.
- **Check `TASK.md`** before starting a new task. If not listed, add it.
- **Use consistent naming conventions, file structure, and architecture patterns** as described in PLANNING.md.
- **Use venv_linux** whenever executing Python commands.

### Code Structure & Modularity
- **Never create a file longer than 500 lines of code.** Refactor by splitting.
- **Organize code into clearly separated modules**, grouped by feature or responsibility.
  - `agent.py` - Main agent definition
  - `tools.py` - Tool functions
  - `prompts.py` - System prompts
- **Use clear, consistent imports** (prefer relative imports within packages).
- **Use python_dotenv and load_env()** for environment variables.

### Testing & Reliability
- **Always create Pytest unit tests for new features**.
- **After updating any logic**, check whether existing unit tests need updating.
- Tests in `/tests` folder mirroring main app structure.
  - 1 test for expected use
  - 1 edge case
  - 1 failure case

### Task Completion
- **Mark completed tasks in `TASK.md`** immediately after finishing.
- Add new sub-tasks to TASK.md under "Discovered During Work" section.

### Style & Conventions
- Python primary language, follow PEP8, type hints, black formatting
- Pydantic for data validation, FastAPI for APIs
- Google-style docstrings for every function

### Documentation & Explainability
- **Update `README.md`** when features added
- **Comment non-obvious code** with `# Reason:` comments explaining why

### AI Behavior Rules
- **Never assume missing context. Ask questions if uncertain.**
- **Never hallucinate libraries or functions**
- **Always confirm file paths and module names** exist
- **Never delete or overwrite existing code** unless explicitly instructed

## Notable Characteristics
- ~100 lines
- References external files (PLANNING.md, TASK.md) for state management
- AI-specific behavior rules ("never hallucinate", "ask if uncertain")
- File length limits (500 lines max)
- Test coverage requirements (expected, edge case, failure case)
- Task tracking integrated into workflow
