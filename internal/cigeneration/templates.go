// Package cigeneration produces CI/CD workflow files (GitHub Actions, GitLab CI)
// from a registry of step contributors. Each contributor provides CI steps for
// specific security scanning tools, and the generator assembles them into
// platform-specific workflow files.
//
// Workflow files are built using string builders rather than yaml.Marshal or
// text/template to maintain precise control over the output format, which is
// critical for GitHub Actions expressions and YAML formatting.
package cigeneration
