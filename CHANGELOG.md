# Changelog

All notable changes to qsdev are recorded in this file. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Project-root detection now stops at the git repository toplevel. A submodule
  or nested repository is its own project boundary: a `.qsdev.yaml`, `.devinit/`
  or `.qsdev/` in a directory above a nested `.git` no longer resolves from
  inside that repository, so its commands, logs and catalog overlays no longer
  act on the outer project. Projects whose markers sit at the repository
  toplevel, or in a subdirectory between the working directory and the
  toplevel (monorepos), resolve as before. If you relied on an outer project's
  configuration governing a nested repository, run `qsdev init` inside it.

### Security

- A stray or planted qsdev marker in a parent directory (for example a shared
  or temporary directory) can no longer capture root resolution, session logs
  or configuration for a git repository created beneath it (U01-01, XS-N5).
  Directories that are not inside a git repository are not yet covered.
