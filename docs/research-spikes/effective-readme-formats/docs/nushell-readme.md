<!-- Source: https://raw.githubusercontent.com/nushell/nushell/main/README.md -->
<!-- Retrieved: 2026-05-15 -->

# Nushell
[![Crates.io](https://img.shields.io/crates/v/nu.svg)](https://crates.io/crates/nu)
[![Build Status](https://img.shields.io/github/actions/workflow/status/nushell/nushell/ci.yml?branch=main)](https://github.com/nushell/nushell/actions)
[![Nightly Build](https://github.com/nushell/nushell/actions/workflows/nightly-build.yml/badge.svg)](https://github.com/nushell/nushell/actions/workflows/nightly-build.yml)
[![Discord](https://img.shields.io/discord/601130461678272522.svg?logo=discord)](https://discord.gg/NtAbbGn)
[![The Changelog #363](https://img.shields.io/badge/The%20Changelog-%23363-61c192.svg)](https://changelog.com/podcast/363)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/m/nushell/nushell)](https://github.com/nushell/nushell/graphs/commit-activity)
[![GitHub contributors](https://img.shields.io/github/contributors/nushell/nushell)](https://github.com/nushell/nushell/graphs/contributors)

A new type of shell.

![Example of nushell](assets/nushell-autocomplete6.gif "Example of nushell")

## Table of Contents

- [Status](#status)
- [Learning About Nu](#learning-about-nu)
- [Installation](#installation)
- [Configuration](#configuration)
- [Philosophy](#philosophy)
- [Goals](#goals)
- [Officially Supported By](#officially-supported-by)
- [Contributing](#contributing)
- [License](#license)

## Status

This project has reached a minimum-viable-product level of quality. Many people use it as their daily driver, but it may be unstable for some commands. Nu's design is subject to change as it matures.

## Learning About Nu

The [Nushell book](https://www.nushell.sh/book/) is the primary source of Nushell documentation. You can find [a full list of Nu commands in the book](https://www.nushell.sh/commands/), and we have many examples of using Nu in our [cookbook](https://www.nushell.sh/cookbook/).

We're also active on [Discord](https://discord.gg/NtAbbGn); come and chat with us!

## Installation

To quickly install Nu:

```bash
# Linux and macOS
brew install nushell
# Windows
winget install nushell
```

To use `Nu` in GitHub Action, check [setup-nu](https://github.com/marketplace/actions/setup-nu) for more detail.

Detailed installation instructions can be found in the [installation chapter of the book](https://www.nushell.sh/book/installation.html). Nu is available via many package managers:

[![Packaging status](https://repology.org/badge/vertical-allrepos/nushell.svg?columns=3)](https://repology.org/project/nushell/versions)

## Configuration

The default configurations can be found at [sample_config](crates/nu-utils/src/default_files) which are the configuration files one gets when they startup Nushell for the first time.

To see where *config.nu* is located on your system simply type this command:

```rust
$nu.config-path
```

## Philosophy

Nu draws inspiration from projects like PowerShell, functional programming languages, and modern CLI tools. Rather than thinking of files and data as raw streams of text, Nu looks at each input as something with structure. For example, when you list the contents of a directory what you get back is a table of rows, where each row represents an item in that directory. These values can be piped through a series of steps, in a series of commands called a 'pipeline'.

### Pipelines

In Unix, it's common to pipe between commands to split up a sophisticated command over multiple steps. Nu takes this a step further and builds heavily on the idea of _pipelines_. Commands are separated by the pipe symbol (`|`) to denote a pipeline flowing left to right.

```shell
ls | where type == "dir" | table
```

### Opening files

Nu can load file and URL contents as raw text or structured data (if it recognizes the format).

```shell
open Cargo.toml
open Cargo.toml | get package
open Cargo.toml | get package.version
```

### Plugins

Nu supports plugins that offer additional functionality to the shell and follow the same structured data model that built-in commands use.

## Goals

- First and foremost, Nu is cross-platform.
- Nu ensures compatibility with existing platform-specific executables.
- Nu's workflow and tools should have the usability expected of modern software in 2022 (and beyond).
- Nu views data as either structured or unstructured. It is a structured shell like PowerShell.
- Finally, Nu views data functionally. Rather than using mutation, pipelines act as a means to load, change, and save data without mutable state.

## Officially Supported By

zoxide, starship, oh-my-posh, Couchbase Shell, virtualenv, atuin, clap, Dorothy, Direnv, x-cmd, vfox, Windmill

## Contributing

See [Contributing](CONTRIBUTING.md) for details.

<a href="https://github.com/nushell/nushell/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=nushell/nushell&max=750&columns=20" />
</a>

## License

The project is made available under the MIT license.
