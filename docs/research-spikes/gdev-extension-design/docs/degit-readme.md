<!-- Source: https://github.com/Rich-Harris/degit -->
<!-- Retrieved: 2026-05-12 -->

# Degit: Project Scaffolding Tool

## Overview

Degit is a straightforward utility for duplicating git repositories. It "makes copies of git repositories" by downloading the latest commit as a tar file rather than cloning the entire git history, making it significantly faster than traditional git operations.

## Key Features

**Speed & Efficiency**: The tool downloads only the necessary files without the `.git` folder, avoiding the confusion of inheriting a template's repository history.

**Caching Support**: "If you already have a `.tar.gz` file for a specific commit, you don't need to fetch it again," enabling offline usage.

**Multi-Platform Support**: Works with GitHub, GitLab, BitBucket, and Sourcehut repositories.

## Installation & Basic Usage

Install globally via npm, then use simple commands like `degit user/repo` to clone to the current directory.

**Specifying versions**: Use hash syntax to target branches, tags, or commits:
- `degit user/repo#dev` (branch)
- `degit user/repo#v1.2.3` (tag)
- `degit user/repo#1234abcd` (commit)

## Advanced Options

Users can specify destination folders, extract subdirectories, configure HTTPS proxying, and use `--mode=git` for private repositories (slower but necessary for SSH access).

## Actions System

Degit supports post-cloning manipulation through `degit.json` configuration files:

- **Clone**: Merge another repository's contents while preserving the current directory
- **Remove**: Delete specified files after cloning

## JavaScript API

The tool offers programmatic access through Node.js, allowing developers to integrate degit into custom workflows with configurable caching, force operations, and verbose logging options.
