---
paths: ["**/*.java", "**/pom.xml", "**/build.gradle", "**/build.gradle.kts"]
---

# Java LSP (jdtls)

Use the `LSP` tool, not Grep, to navigate Java symbols here. `jdtls`
(Eclipse JDT Language Server) is configured for this project.

- `goToDefinition` / `findReferences` / `workspaceSymbol` resolve symbols across
  packages and dependencies.
- `goToImplementation` navigates from interfaces and abstract methods to their
  concrete implementations.
- Null analysis (`java.compile.nullAnalysis.mode = "automatic"`) flags potential
  null dereferences in diagnostics after edits.
- `hover` shows resolved generic types and Javadoc.

Caveat: jdtls has a slow cold start (roughly 5-10 seconds while it indexes the
workspace). The first navigation request after opening the project may pause;
subsequent ones are fast. Reach for Grep only for string literals, build-file
keys, and resources jdtls does not index.
