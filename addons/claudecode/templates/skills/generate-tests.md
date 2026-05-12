# Generate Tests

Generate comprehensive test suites for existing code. Tests should be
thorough, maintainable, and follow the conventions of the target language.

## Analysis phase

1. Identify the file or package to test. Read the source code carefully.
2. Catalog every exported/public function and method, noting:
   - Input parameters and their types
   - Return values and error conditions
   - Side effects (file I/O, network calls, database access)
   - Dependencies that will need mocking or stubbing
3. Identify edge cases: nil/null inputs, empty collections, boundary values,
   concurrent access, large inputs, invalid types.

## Test structure by language

### Go
- Use table-driven tests with `[]struct{ name string; ... }` slices.
- Name test functions `TestFunctionName_Scenario`.
- Use `t.Run(tt.name, ...)` for subtests.
- Use `t.Helper()` in test helper functions.
- Use `t.Parallel()` where tests are independent.
- Prefer `testify/assert` or standard `if got != want` comparisons.
- Mock external dependencies with interfaces; use constructor injection.

### TypeScript / JavaScript
- Use `describe` / `it` blocks with clear descriptions.
- Group by function or class under test.
- Use `beforeEach` for shared setup, avoid `beforeAll` for mutable state.
- Mock external modules with `jest.mock()` or `vi.mock()`.
- Use `expect(...).toEqual(...)` for value comparison.
- Prefer `async/await` over raw promises in tests.

### Python
- Use `pytest` with `@pytest.mark.parametrize` for data-driven tests.
- Name test functions `test_function_name_scenario`.
- Use fixtures for shared setup and teardown.
- Mock external dependencies with `unittest.mock.patch` or `pytest-mock`.
- Use `pytest.raises` for exception testing.
- Group related tests in classes when it improves organization.

### Rust
- Place tests in a `#[cfg(test)] mod tests` block within the same file.
- Use `#[test]` attribute on each test function.
- Use `assert_eq!`, `assert_ne!`, and `assert!` macros.
- Test error variants with pattern matching.
- Use `#[should_panic(expected = "...")]` for panic tests.

## Coverage requirements

For each function under test, generate:

1. **Happy path**: Normal inputs producing expected outputs.
2. **Edge cases**: Empty inputs, zero values, maximum values, Unicode strings.
3. **Error paths**: Invalid inputs, missing resources, permission failures.
4. **Boundary conditions**: Off-by-one, integer overflow, timeout thresholds.

## Principles

- Each test must be independent. No test should depend on the execution order
  or side effects of another test.
- Mock all external dependencies (databases, HTTP clients, file systems).
  Tests must pass without network access or special infrastructure.
- Use descriptive test names that explain the scenario and expected outcome.
- Keep assertions focused: one logical assertion per test case.
- Do not test implementation details; test observable behavior.
- Generated tests must compile and pass on the first run.
