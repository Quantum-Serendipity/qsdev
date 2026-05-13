<!-- Source: https://docs.renovatebot.com/config-presets/ -->
<!-- Retrieved: 2026-05-12 -->

# Renovate Shared Presets

## Core Concept
Renovate allows developers to create and share reusable configuration presets that can be extended across multiple repositories. As the documentation states, "To use a preset put it in an `extends` array within your Renovate config."

## Hosting Options

### Repository-Based Hosting
Presets are primarily "repo-hosted, and you can have one or more presets hosted per repository." The system supports multiple platforms:

- **GitHub**: Reference format `github>owner/name`
- **GitLab**: Reference format `gitlab>owner/name`
- **Gitea**: Reference format `gitea>owner/name`
- **Forgejo**: Reference format `forgejo>owner/name`
- **Self-hosted Git**: Reference format `local>owner/name`
- **HTTP servers**: Full URL specification with query parameters

### HTTP Server Alternative
For unsupported platforms, presets can be fetched directly via HTTP URLs, supporting parameters similarly to other methods.

## File Naming Conventions

When omitting a filename (e.g., `github>abc/foo`), Renovate searches for `default.json`. Named presets require explicit specification: `github>abc/foo:xyz` loads `xyz.json`. The documentation notes that "We've deprecated using a `renovate.json` file for the default _preset_ file name" -- developers should rename such files to `default.json`.

## Versioning with Git Tags

Presets support semantic versioning through Git tags: `github>abc/foo#1.2.3` pins to that specific release, while omitting the tag uses the default branch.

## Inheritance and Override Pattern

Presets support nesting through the `extends` array. Later entries can override earlier ones. A typical configuration looks like:

```json
{
  "extends": ["config:recommended", "schedule:nonOfficeHours"]
}
```

## Parameterized Presets

Presets can accept parameters using `{{arg0}}`, `{{arg1}}` syntax, enabling flexible reuse without duplication:

```json
{
  "extends": [":labels(dependencies,devops)", ":assignee(renovate-tests)"]
}
```

The entire parameter string is available as `{{args}}`.

## Organization-Level Presets

Renovate automatically discovers organization-wide defaults by checking for a `renovate-config` repository with `default.json` in the parent org/group. This suggested preset overrides any `onboardingConfig` settings during onboarding.

## Local vs. Hosted Distinction

Local presets determine the platform automatically based on Renovate's current context, making them ideal for self-hosted scenarios where public presets aren't accessible. The syntax supports both implicit (`owner/name`) and explicit (`local>owner/name`) forms.

## npm-Based Presets (Deprecated)

The documentation includes a deprecation notice: npm-hosted presets are "deprecated, we recommend you do not follow these instructions and instead use a `local` preset."

## Format Requirements

Presets must use "JSON or JSON5 formats, other formats are not supported." Renovate supports JSONC syntax with comments within preset files.
