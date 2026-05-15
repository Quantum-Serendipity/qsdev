<!-- Source: https://raw.githubusercontent.com/pnpm/pnpm/main/README.md -->
<!-- Retrieved: 2026-05-15 -->

[Simplified Chinese](https://pnpm.io/zh/) |
[Japanese](https://pnpm.io/ja/) |
[Korean](https://pnpm.io/ko/) |
[Italian](https://pnpm.io/it/) |
[Portuguese](https://pnpm.io/pt/)

<picture>
  <source media="(prefers-color-scheme: light)" srcset="https://i.imgur.com/qlW1eEG.png">
  <source media="(prefers-color-scheme: dark)"  srcset="https://i.imgur.com/qlW1eEG.png">
  <img src="https://i.imgur.com/qlW1eEG.png" alt="pnpm">
</picture>

Fast, disk space efficient package manager:

* **Fast.** Up to 2x faster than the alternatives (see [benchmark](#benchmark)).
* **Efficient.** Files inside `node_modules` are linked from a single content-addressable storage.
* **[Great for monorepos](https://pnpm.io/workspaces).**
* **Strict.** A package can access only dependencies that are specified in its `package.json`.
* **Deterministic.** Has a lockfile called `pnpm-lock.yaml`.
* **Works as a Node.js version manager.** See [pnpm runtime](https://pnpm.io/11.x/cli/runtime).
* **Works everywhere.** Supports Windows, Linux, and macOS.
* **Battle-tested.** Used in production by teams of [all sizes](https://pnpm.io/workspaces#usage-examples) since 2016.
* [See the full feature comparison with npm and Yarn](https://pnpm.io/feature-comparison).

To quote the [Rush](https://rushjs.io/) team:

> Microsoft uses pnpm in Rush repos with hundreds of projects and hundreds of PRs per day, and we've found it to be very fast and reliable.

[![npm version](https://img.shields.io/npm/v/pnpm.svg?label=latest)](https://github.com/pnpm/pnpm/releases/latest)
[![OpenCollective](https://opencollective.com/pnpm/backers/badge.svg)](https://opencollective.com/pnpm)
[![OpenCollective](https://opencollective.com/pnpm/sponsors/badge.svg)](https://opencollective.com/pnpm)
[![X Follow](https://img.shields.io/twitter/follow/pnpmjs.svg?style=social&label=Follow)](https://x.com/intent/follow?screen_name=pnpmjs&region=follow_link)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

## Platinum Sponsors

Bit

## Gold Sponsors

Sanity, Discord, Vite, SerpApi, CodeRabbit, Stackblitz, Workleap, Nx

## Silver Sponsors

Replit, Cybozu, devowl.io, u|screen, Leniolabs_, Depot, Cerbos, Time.now

## Background

pnpm uses a content-addressable filesystem to store all files from all module directories on a disk. When using npm, if you have 100 projects using lodash, you will have 100 copies of lodash on disk. With pnpm, lodash will be stored in a content-addressable storage, so:

1. If you depend on different versions of lodash, only the files that differ are added to the store.
2. All the files are saved in a single place on the disk. When packages are installed, their files are linked from that single place consuming no additional disk space. Linking is performed using either hard-links or reflinks (copy-on-write).

As a result, you save gigabytes of space on your disk and you have a lot faster installations!

## Getting Started

- [Installation](https://pnpm.io/installation)
- [Usage](https://pnpm.io/pnpm-cli)
- [Frequently Asked Questions](https://pnpm.io/faq)
- [X](https://x.com/pnpmjs)
- [Bluesky](https://bsky.app/profile/pnpm.io)
- [Discord](https://r.pnpm.io/chat)

## Benchmark

pnpm is up to 2x faster than npm and Yarn classic. See all benchmarks [here](https://r.pnpm.io/benchmarks).

Benchmarks on an app with lots of dependencies:

![](https://pnpm.io/img/benchmarks/alotta-files.svg)

## License

[MIT](https://github.com/pnpm/pnpm/blob/main/LICENSE)
