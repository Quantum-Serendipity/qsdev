<!-- Source: https://copier.readthedocs.io/en/stable/updating/ -->
<!-- Retrieved: 2026-05-12 -->

# Copier's Project Update Mechanism

## Core Update Process

Copier manages template-based project updates through a sophisticated three-way merge approach. When you run `copier update`, the tool performs these sequential steps:

1. **Template Regeneration**: A fresh project is generated from the current template version using stored answers
2. **Diff Calculation**: The system compares the fresh generation against your actual project to identify your customizations
3. **Pre-Migration Application**: Custom migration scripts run before template updates
4. **Interactive Update**: Users answer template questions (defaulting to previous responses), and the latest template applies
5. **Diff Reapplication**: Previously identified customizations are reapplied to preserve user modifications
6. **Post-Migration Execution**: Final migration scripts execute

## Answer Tracking via .copier-answers.yml

The `.copier-answers.yml` file is critical for update functionality. As the documentation states, you should "Never update `.copier-answers.yml` manually" because doing so will "trick Copier, making it believe that those modified answers produced the current subproject." This file enables Copier to understand what answers generated your current project, which is essential for the smart diff algorithm.

## Conflict Resolution Strategies

When updates create conflicts, two resolution modes exist:

- **Inline conflicts** (default): Uses markers similar to git merge conflicts
- **Rejection files**: Creates `.rej` files containing unresolved diffs

The documentation recommends adding pre-commit hooks to prevent accidentally committing unresolved conflicts.

## Configuration Drift Management

Copier automatically excludes template files deleted in your project from future updates. However, paths matching `skip_if_exists` remain protected during updates.

For severely broken updates, `copier recopy` discards the smart algorithm and regenerates the project while preserving your answers.
