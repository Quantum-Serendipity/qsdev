---
name: add-tests
description: Generate tests for uncovered code. Follows existing test patterns in the codebase.
disable-model-invocation: true
allowed-tools: Bash(*) Read Write Edit Grep Glob
arguments: [target]
argument-hint: "path/to/module"
---

# Add Tests

Generate comprehensive tests for uncovered code, following existing test patterns in the codebase.

## Step 1: Understand Test Patterns

Before writing any tests, analyze the existing test infrastructure:

1. **Framework**: Identify the test framework (e.g., `testing`, `pytest`, `jest`, `vitest`).
2. **Naming convention**: How are test files and functions named?
3. **Directory structure**: Are tests co-located or in a separate `__tests__`/`test/` directory?
4. **Mock patterns**: What mocking libraries or patterns are used?
5. **Fixtures**: Are there shared test fixtures, helpers, or setup functions?
6. **Assertions**: What assertion style is preferred?

## Step 2: Analyze Target

Examine the target module/file:

1. **Public functions**: List all exported/public functions and methods.
2. **Input types**: Document parameter types, including edge-case values.
3. **Error conditions**: Identify all error return paths and failure modes.
4. **Side effects**: Note file I/O, network calls, database access, or global state mutations.
5. **Dependencies**: List external dependencies that need mocking.

## Step 3: Generate Tests

For each public function, generate tests covering:

1. **Happy path**: Normal operation with valid inputs.
2. **Edge cases**: Empty inputs, boundary values, nil/null, zero values.
3. **Error paths**: Invalid inputs, dependency failures, timeout scenarios.
4. **Type safety**: Ensure type constraints are exercised.

Follow the existing test style exactly. Match naming conventions, assertion patterns, and file organization.

## Step 4: Verify

1. Run the new tests and confirm they pass.
2. Run the full test suite to ensure no regressions.
3. Report results including any failures or issues discovered.
