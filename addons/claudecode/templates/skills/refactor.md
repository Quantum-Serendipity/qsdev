# Refactor

Refactor code for improved clarity, performance, and maintainability while
preserving existing behavior.

## Pre-refactor checklist

1. Run the full test suite and confirm all tests pass. Record the results as
   the baseline. Do not proceed if tests are failing.
2. If test coverage is insufficient for the code being refactored, generate
   additional tests first to lock in current behavior.
3. Identify the scope: which files, functions, or types are being refactored.

## Code smell detection

Scan the target code for these common problems:

### Structural issues
- **Long functions**: Functions longer than ~40 lines likely do too much.
  Apply extract-method to break them into focused helpers.
- **Deep nesting**: More than 3 levels of indentation. Use early returns
  (guard clauses) to flatten control flow.
- **Duplication**: Repeated code blocks or near-identical logic. Extract
  shared behavior into a common function or type.

### Naming and clarity
- **Ambiguous names**: Variables like `data`, `result`, `tmp`, `x` that
  convey no intent. Rename to describe the value's purpose.
- **Boolean parameters**: `func process(data []byte, true, false)` is
  unreadable. Replace with named options, enums, or separate functions.
- **Magic numbers**: Numeric literals without context. Extract to named
  constants with explanatory names.

### Design issues
- **God objects**: Types with many unrelated methods. Split into focused
  types with clear responsibilities.
- **Feature envy**: A function that accesses another type's fields more
  than its own. Move the logic to the type that owns the data.
- **Primitive obsession**: Using strings or ints where a domain type would
  add clarity and type safety.

## Refactoring techniques

Apply these transformations as appropriate:

1. **Extract method**: Move a coherent block of code into a named function.
2. **Rename**: Change variable, function, or type names to match their purpose.
3. **Simplify conditionals**: Replace complex boolean expressions with named
   predicates. Convert if/else chains to switch statements or lookup tables.
4. **Introduce parameter object**: When a function takes many related parameters,
   group them into a struct or type.
5. **Remove dead code**: Delete unreachable code, unused variables, and
   commented-out blocks.
6. **Extract interface**: When a concrete type is used for testing or
   polymorphism, define an interface at the usage site.

## Post-refactor verification

1. Run the full test suite again. All tests must still pass.
2. Run the linter and fix any new warnings.
3. Verify that no public API signatures changed unless intentionally modified.
4. Produce a summary of changes: what was refactored, why, and how behavior
   is preserved.

## Principles

- **One thing at a time**: Each refactoring step should be a single, reversible
  transformation. Do not combine rename + extract + restructure in one step.
- **Tests are the safety net**: If a refactoring breaks tests, revert and
  take a smaller step.
- **No behavior changes**: Refactoring must not alter observable behavior.
  If behavior changes are needed, do them in a separate step with explicit tests.
- **Leave it better**: Every touched file should be cleaner after the refactor.
