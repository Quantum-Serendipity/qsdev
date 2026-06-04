# OpenGrep Modern Framework Taint Rules -- Known Limitations

This document describes known limitations and compatibility constraints for the
modern framework taint rules in `rules/core/modern-frameworks/`.

## 1. SvelteKit `.svelte` file parsing

OpenGrep/Semgrep does not include a Svelte Tree-sitter grammar. `.svelte` files
cannot be parsed as a structured language. XSS rules targeting the `{@html}`
directive use `languages: [generic]` with regex-based audit mode. These rules
flag all `{@html}` usage without taint tracking. Server-side `.ts` files
(`+page.server.ts`, `+server.ts`, `hooks.server.ts`) are fully scannable with
taint analysis.

## 2. Cross-middleware taint tracking not supported

Taint does not propagate across middleware boundaries in any framework. Middleware
functions are separate function bodies registered via callback. Sources must be
defined at point-of-use inside handler functions, not at middleware entry points.
This applies to Express middleware, Gin middleware (`c.Next()`), NestJS guards
and interceptors, and SvelteKit hooks.

## 3. DI container opacity (NestJS, FastAPI)

Dependency injection wires services at runtime. The taint engine cannot see
through `constructor(private readonly userService: UserService)` (NestJS) or
`Depends(get_current_user)` (FastAPI). All handler parameters are treated as
tainted to compensate. This may produce false positives on infrastructure
dependencies (database sessions, config objects). Use `# nosemgrep` or
`// nosemgrep` to suppress known-safe parameters.

## 4. Gin `c.Set()`/`c.Get()` context propagation not tracked

Gin's per-request key-value store (`c.Set("key", value)` in middleware,
`c.Get("key")` in handler) is a cross-function data flow boundary. Taint stored
via `c.Set()` does not automatically propagate to `c.Get()` retrievals in
downstream handlers. The `--taint-intrafile` flag partially mitigates this when
middleware and handler are in the same file.

## 5. Pydantic/NestJS ValidationPipe is NOT a sanitizer

Pydantic model validation (`@IsString()`, `constr(max_length=N)`) and NestJS
`ValidationPipe` with `class-validator` validate shape and type, not content
safety. Only numeric type coercion genuinely sanitizes:

- **Safe (sanitizer)**: `int`, `float`, `bool`, `ParseIntPipe`, `ParseUUIDPipe`,
  `ParseBoolPipe`, `ParseEnumPipe`, `conint()`, `confloat()`
- **Not safe**: `str`, `@IsString()`, `@IsNotEmpty()`, `@MaxLength()`,
  `constr(max_length=N)`, `@IsEmail()` (partially)

Rules use `taint_assume_safe_numbers: true` and `taint_assume_safe_booleans: true`
to handle numeric coercion rather than explicit Pydantic/pipe sanitizer patterns.

## 6. `Prisma.raw()` indistinguishable from safe tagged template

The safe pattern `prisma.$queryRaw\`SELECT * FROM users WHERE id = ${id}\`` and
the dangerous pattern `prisma.$queryRawUnsafe(\`SELECT * FROM users WHERE id = ${id}\`)`
look similar in source code but have fundamentally different security properties.
The tagged template version parameterizes; the string version does not. Rules use
`pattern-not` exclusions to avoid flagging tagged template usage, but edge cases
exist where the distinction is ambiguous (e.g., variables holding pre-built SQL
fragments).

## 7. Cross-file taint unavailable

OpenGrep's `--taint-intrafile` flag enables cross-function tracking within a
single file. Cross-file taint tracking requires Semgrep Pro's `interfile: true`
mode. For the open-source engine, `--taint-intrafile` provides approximately 90%
coverage. Common cross-file gaps include:

- Server Component to Client Component data flow (Next.js)
- Controller to Service method calls (NestJS)
- Hook data (`event.locals`) to load function (SvelteKit)
- Middleware to handler data flow (all frameworks)
- Data access layer functions in separate files (all frameworks)

Workaround: Rules include broad source patterns (e.g., service method string
parameters as secondary sources) to partially compensate for cross-file gaps
when used with `--taint-intrafile`.
