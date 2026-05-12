# Pwning the Entire Nix Ecosystem
- **Source**: https://ptrpa.ws/nixpkgs-actions-abuse
- **Retrieved**: 2026-05-12

## Vulnerability Overview

Security researchers discovered critical flaws in nixpkgs' GitHub Actions workflows that could have enabled complete ecosystem compromise through supply chain attack.

## Technical Vulnerabilities

**EditorConfig Workflow Flaw:**
The workflow processed changed file lists using xargs, which has documented security limitations. Attackers could craft filenames as command-line arguments (e.g., --help) to manipulate tool behavior, potentially achieving arbitrary code execution through editorconfig-checker exploitation.

**CODEOWNERS Validator - Local File Inclusion:**
This posed greater risk. The workflow checked out untrusted PR code into a working directory, then executed a validator on the OWNERS file. Researchers could replace the OWNERS file with a symbolic link targeting system files like "/home/runner/runners/2.320.0/.credentials".

When validation failed, error messages revealed file contents -- including GitHub Actions credentials with read/write repository access.

## Attack Surface

The pull_request_target trigger granted dangerous privileges: these workflows received secrets and write permissions even from fork-originated PRs. Unlike standard pull_request triggers, this configuration enabled attackers to bypass authorization checks entirely.

## Remediation

Maintainers immediately:
- Disabled vulnerable workflows
- Separated untrusted data from privileged operations
- Renamed fixed workflows (addressing historical branch vulnerabilities)
- Enforced minimum necessary permissions

## Security Implications

Discovery within one day demonstrated how rapidly supply chain attacks could compromise a major ecosystem. The incident highlighted critical GitHub Actions misconfigurations and the necessity of careful credential handling in CI/CD pipelines.
