<!-- Source: https://raw.githubusercontent.com/oven-sh/bun/main/README.md -->
<!-- Retrieved: 2026-05-15 -->

<p align="center">
  <a href="https://bun.com"><img src="https://github.com/user-attachments/assets/50282090-adfd-4ddb-9e27-c30753c6b161" alt="Logo" height=170></a>
</p>
<h1 align="center">Bun</h1>

<p align="center">
<a href="https://bun.com/discord" target="_blank"><img height=20 src="https://img.shields.io/discord/876711213126520882" /></a>
<img src="https://img.shields.io/github/stars/oven-sh/bun" alt="stars">
<a href="https://twitter.com/jarredsumner/status/1542824445810642946"><img src="https://img.shields.io/static/v1?label=speed&message=fast&color=success" alt="Bun speed" /></a>
</p>

<div align="center">
  <a href="https://bun.com/docs">Documentation</a>
  <span>&nbsp;&nbsp;*&nbsp;&nbsp;</span>
  <a href="https://bun.com/discord">Discord</a>
  <span>&nbsp;&nbsp;*&nbsp;&nbsp;</span>
  <a href="https://github.com/oven-sh/bun/issues/new">Issues</a>
  <span>&nbsp;&nbsp;*&nbsp;&nbsp;</span>
  <a href="https://github.com/oven-sh/bun/issues/159">Roadmap</a>
  <br />
</div>

### [Read the docs ->](https://bun.com/docs)

## What is Bun?

Bun is an all-in-one toolkit for JavaScript and TypeScript apps. It ships as a single executable called `bun`.

At its core is the _Bun runtime_, a fast JavaScript runtime designed as **a drop-in replacement for Node.js**. It's written in Zig and powered by JavaScriptCore under the hood, dramatically reducing startup times and memory usage.

```bash
bun run index.tsx             # TS and JSX supported out-of-the-box
```

The `bun` command-line tool also implements a test runner, script runner, and Node.js-compatible package manager. Instead of 1,000 node_modules for development, you only need `bun`. Bun's built-in tools are significantly faster than existing options and usable in existing Node.js projects with little to no changes.

```bash
bun test                      # run tests
bun run start                 # run the `start` script in `package.json`
bun install <pkg>             # install a package
bunx cowsay 'Hello, world!'   # execute a package
```

## Install

Bun supports Linux (x64 & arm64), macOS (x64 & Apple Silicon), and Windows (x64 & arm64).

> **Linux users** -- Kernel version 5.6 or higher is strongly recommended, but the minimum is 5.1.

```sh
# with install script (recommended)
curl -fsSL https://bun.com/install | bash

# on windows
powershell -c "irm bun.sh/install.ps1 | iex"

# with npm
npm install -g bun

# with Homebrew
brew tap oven-sh/bun
brew install bun

# with Docker
docker pull oven/bun
docker run --rm --init --ulimit memlock=-1:-1 oven/bun
```

### Upgrade

To upgrade to the latest version of Bun, run:

```sh
bun upgrade
```

## Quick links

Extensive categorized list of links to documentation covering:
- Intro (What is Bun, Installation, Quickstart, TypeScript)
- Templating (bun init, bun create)
- Runtime (bun run, File types, JSX, Environment variables, Bun APIs, Web APIs, Node.js compatibility, Plugins, Watch mode, Module resolution, Auto-install, bunfig.toml, Debugger, REPL, $ Shell)
- Package manager (bun install, bun add, bun remove, bun update, bun link, bun pm, bun outdated, bun publish, bun patch, bun why, bun audit, bun info, Global cache, Global store, Isolated installs, Workspaces, Catalogs, Lifecycle scripts, Filter, Lockfile, Scopes and registries, Overrides and resolutions, Security scanner API, .npmrc)
- Bundler (Bun.build, Loaders, Plugins, Macros, vs esbuild, Single-file executable, CSS, HTML & static sites, HMR, Full-stack with HTML imports, Standalone HTML, Bytecode caching, Minifier)
- Test runner (bun test, Writing tests, Lifecycle hooks, Mocks, Snapshots, Dates and times, DOM testing, Code coverage, Configuration, Discovery, Reporters, Runtime Behavior)
- Package runner (bunx)
- API (HTTP server, HTTP routing, WebSockets, Workers, Binary data, Streams, File I/O, Archive, SQLite, PostgreSQL, Redis, S3 Client, FileSystemRouter, TCP sockets, UDP sockets, Globals, Child processes, Cron, WebView, Transpiler, Hashing, Colors, Console, FFI, C Compiler, HTMLRewriter, Cookies, CSRF, Secrets, YAML, TOML, JSON5, JSONL, Markdown, Image processing, Utils, Node-API, Glob, Semver, DNS, fetch API extensions)

## Guides

100+ categorized guides covering Deployment, Binary, Ecosystem, HTMLRewriter, HTTP, Install, Process, Read file, Runtime, Streams, Test, Util, WebSocket, Write file.

## Contributing

Refer to the [Project > Contributing](https://bun.com/docs/project/contributing) guide to start contributing to Bun.

## License

Refer to the [Project > License](https://bun.com/docs/project/license) page for information about Bun's licensing.
