# Gradle Dependency Locking Documentation

- **Source**: https://docs.gradle.org/current/userguide/dependency_locking.html
- **Retrieved**: 2026-05-12

## Overview

Dependency locking is a mechanism that ensures reproducible builds by recording resolved dependency versions in lock files. This prevents unexpected version changes from dynamic version selectors like `1.+` or `[1.0,2.0)`.

## Key Benefits

- **Stability with Flexibility**: Teams can use dynamic versions during development while maintaining locked versions for releases
- **Preventing Cascading Failures**: Eliminates reliance on `-SNAPSHOT` dependencies that may introduce breaking changes
- **Build Cache Optimization**: Ensures stable task inputs required for effective caching

## Enabling Dependency Locking

### For Specific Configurations

```kotlin
configurations {
    compileClasspath {
        resolutionStrategy.activateDependencyLocking()
    }
}
```

### For All Configurations

```kotlin
dependencyLocking {
    lockAllConfigurations()
}
```

## Generating Lock Files

Run: `./gradlew dependencies --write-locks`

This creates or updates `gradle.lockfile` at the project root. "Gradle won't write the lock state to disk if the build fails, preventing the persistence of potentially invalid states."

## Lock File Format

**Location**: `gradle.lockfile` in project directory; `buildscript-gradle.lockfile` for build scripts

**Structure**: Each line contains `group:artifact:version=configuration1,configuration2`

Example entry: `org.springframework:spring-beans:5.0.5.RELEASE=compileClasspath,runtimeClasspath`

Features:
- Alphabetically ordered (aids version control diffs)
- Empty configurations listed as `empty=`
- Should be committed to source control

## Lock File Enforcement Behavior

During resolution, locked versions are "enforced as if they were declared with `strictly()`":

- If declared version is **lower** than locked: silently upgrades
- If declared version is **higher** than locked: build fails

## Lock Modes

Three modes control validation strictness:

1. **Default**: Validates entries match and no extras exist
2. **Strict**: Fails if locked configurations lack lock state
3. **Lenient**: Pins dynamic versions but allows dependency additions/removals and transitive shifts

Configure: `dependencyLocking { lockMode = LockMode.STRICT }`

## Selective Lock Updates

Update specific modules without regenerating all locks:

```
./gradlew dependencies --update-locks org.apache.commons:commons-lang3,org.slf4j:slf4j-api
```

Wildcards supported: `org.apache.commons:*`, `*:guava`

## Important Limitations

- "Dependency locking does not currently apply to source dependencies"
- Should **not** be used with changing versions (e.g., `-SNAPSHOT`)
- Gradle lockfiles do NOT include checksums (a significant gap vs other ecosystems)
