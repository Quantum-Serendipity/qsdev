# Java/Kotlin Conventions

- Follow standard Java naming: `PascalCase` for classes, `camelCase` for methods and variables, `UPPER_SNAKE_CASE` for constants.
- Use constructor injection for dependencies. Avoid field injection.
- Prefer immutable data classes. In Kotlin use `data class`; in Java use records or final fields.
- Handle exceptions at the appropriate layer. Do not catch `Exception` broadly — catch specific types.
- Write unit tests with JUnit 5. Use `@ParameterizedTest` for data-driven tests.
- Use `Optional` for nullable return types in Java. In Kotlin, use nullable types with `?`.
- Close resources with try-with-resources (Java) or `.use {}` (Kotlin).
- Keep Maven/Gradle dependencies locked. Do not add dependencies via raw CLI commands.
- Prefer `final` variables in Java and `val` in Kotlin to reduce mutability.
- Run `mvn verify` or `gradle check` before committing.
- Use SLF4J for logging. Never use `System.out.println` in production code.
