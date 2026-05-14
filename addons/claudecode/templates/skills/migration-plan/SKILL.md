---
name: migration-plan
description: Generate a phased migration plan for framework, library, or architecture changes.
disable-model-invocation: true
allowed-tools: Read Grep Glob Bash(git *) Bash(find *) Bash(wc *)
arguments: [migration-description]
argument-hint: "migrate from Express to Fastify"
---

# Migration Plan

Generate a phased migration plan for framework, library, or architecture changes.

## Step 1: Scope Assessment

1. **Files affected**: Find all files that reference the component being migrated.
2. **Dependency chains**: Map transitive dependencies that may be affected.
3. **Risk profile**: Assess complexity based on file count, test coverage, and integration depth.
4. **Current usage patterns**: Catalog how the component is used across the codebase.

## Step 2: Interview User

Ask the user about:

1. **Timeline**: What is the target completion date?
2. **Coexistence**: Can old and new coexist during migration? For how long?
3. **Constraints**: Any files, modules, or features that cannot be migrated?
4. **Rollback requirements**: What rollback strategy is needed?

## Step 3: Generate Plan

Create the migration plan at `docs/migrations/{description}.md` with:

### Overview
- Migration scope and rationale
- Estimated effort and timeline
- Risk assessment summary

### Phase 1: Preparation
- Set up the new framework/library alongside the old
- Create adapter layers for coexistence
- Establish migration validation criteria
- Dependencies, risks, verification steps, and rollback procedure

### Phase 2: Incremental Migration
- Order of migration (least risky to most risky)
- For each component:
  - Files to change
  - Expected behavior changes
  - Test updates required
- Dependencies, risks, verification steps, and rollback procedure

### Phase 3: Cleanup
- Remove old framework/library dependencies
- Remove adapter layers and compatibility shims
- Update documentation
- Dependencies, risks, verification steps, and rollback procedure

Each phase includes:
- **Dependencies**: What must be completed first
- **Risks**: What could go wrong
- **Verification**: How to confirm success
- **Rollback**: How to revert if needed
