---
name: upgrade-dep
description: Upgrade a dependency with changelog research, breaking change analysis, and verification.
disable-model-invocation: true
allowed-tools: Bash(*) Read Write Edit Grep Glob
arguments: [package, target-version]
argument-hint: "lodash 4.18.0"
---

# Upgrade Dependency

Upgrade a dependency with changelog research, breaking change analysis, and full verification.

## Step 1: Current State

1. **Find current version**: Locate the dependency in manifest files (package.json, go.mod, requirements.txt, Cargo.toml, etc.).
2. **List importing files**: Find all files that import/use this dependency.
3. **Run tests for baseline**: Execute the test suite and record pass/fail state before any changes.

## Step 2: Research Breaking Changes

1. Review the changelog or release notes between the current and target versions.
2. Identify breaking changes, deprecations, and migration guides.
3. Check for known issues or regressions in the target version.
4. Note any transitive dependency changes that may affect the project.

## Step 3: Plan

Present the upgrade plan to the user:

1. **Breaking changes found**: List each with affected files.
2. **Migration steps**: Ordered list of code changes required.
3. **Risk assessment**: Low/medium/high based on scope of changes.
4. **Estimated files affected**: Count of files needing modification.

Wait for user approval before proceeding.

## Step 4: Execute

1. Update the dependency version in the manifest file.
2. Fix each breaking change identified in the plan.
3. Update lock files as needed.
4. Build the project to check for compilation errors.
5. Run the test suite.

## Step 5: Verify

1. Run the full test suite and confirm all tests pass.
2. Run the linter and confirm no new warnings.
3. Check for deprecation warnings in build output.
4. Report the final status.
