# TypeScript / JavaScript Conventions

## Type safety
- Enable `strict: true` in `tsconfig.json`. Never disable strictness flags
  in production code.
- Use `unknown` instead of `any` for values of uncertain type. Narrow with
  type guards before use.
- Add explicit return types on all exported functions and methods.
- Prefer `interface` for object shapes; use `type` for unions, intersections,
  and mapped types.

## Variables and constants
- Prefer `const` over `let`. Never use `var`.
- Use `as const` assertions for literal objects and arrays that should not
  be widened.
- Avoid magic numbers and strings. Extract to named constants.

## Module structure
- Avoid barrel exports (`index.ts` re-exporting everything). They increase
  bundle size and create circular dependency risks.
- Use named exports over default exports for consistent import names.
- Keep modules focused: one primary responsibility per file.

## Error handling
- Use typed error classes extending `Error` for domain-specific failures.
- Always handle promise rejections. Prefer `async/await` with `try/catch`
  over raw `.then().catch()` chains.
- Validate external input at system boundaries using runtime validation
  (e.g., Zod, io-ts, or manual checks).

## Formatting and linting
- Use the project's configured formatter (Prettier, Biome, or dprint).
- Run ESLint / Biome lint before committing.
- Keep line length under 100 characters where practical.

## Testing
- Use `describe` / `it` blocks with descriptive names.
- Prefer `toEqual` for value comparison, `toBe` for identity.
- Mock external modules at the module boundary, not deep internals.
