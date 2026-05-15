<!-- Source: https://raw.githubusercontent.com/dandavison/delta/main/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Full raw content retrieved via GitHub API. -->

<p align="center">
  <img width=400px src="https://user-images.githubusercontent.com/52205/147996902-9829bd3f-cd33-466e-833e-49a6f3ebd623.png" alt="image" />
</p>
<p align="center">
  Badges: CI, Coverage Status, Gitter
</p>

## Get Started

Install it (the package is called "git-delta" in most package managers, but the executable is just `delta`) and add this to your `~/.gitconfig`:

```gitconfig
[core]
    pager = delta

[interactive]
    diffFilter = delta --color-only

[delta]
    navigate = true  # use n and N to move between diff sections
    dark = true      # or light = true, or omit for auto-detection

[merge]
    conflictStyle = zdiff3
```

Or run:

```sh
git config --global core.pager delta
git config --global interactive.diffFilter 'delta --color-only'
git config --global delta.navigate true
git config --global delta.dark true
git config --global merge.conflictStyle zdiff3
```

Delta has many features and is very customizable; please see `delta -h` (short help) or `delta --help` (full manual), or the online user manual.

## Features

- Language syntax highlighting with the same themes as bat
- Word-level diff highlighting using a Levenshtein edit inference algorithm
- Side-by-side view with line-wrapping
- Line numbering
- `n` and `N` keybindings to move between files in large diffs, and between diffs in `log -p` views (`--navigate`)
- Improved merge conflict display
- Improved `git blame` display (syntax highlighting; `--hyperlinks` formats commits as links to hosting providers)
- Syntax-highlights grep output from `rg`, `git grep`, `grep`, etc
- Support for Git's `--color-moved` feature
- Code can be copied directly from the diff (`-/+` markers removed by default)
- `diff-highlight` and `diff-so-fancy` emulation modes
- Commit hashes as terminal hyperlinks to hosting provider page
- Stylable box/line decorations for commit, file and hunk header sections
- Style strings for >20 stylable elements, using same color/style language as git
- Handles traditional unified diff output in addition to git output
- Automatic detection of light/dark terminal background

## A syntax-highlighting pager for git, diff, and grep output

Code evolves, and we all spend time studying diffs. Delta aims to make this both efficient and enjoyable: it allows you to make extensive changes to the layout and styling of diffs, as well as allowing you to stay arbitrarily close to the default git/diff output.

Multiple screenshots showing:
- delta with `line-numbers` activated
- delta with `side-by-side` and `line-numbers` activated
- git show with Dracula theme vs GitHub theme

### Syntax-highlighting themes

All syntax-highlighting color themes available with bat are available with delta.
Screenshots of `delta --show-syntax-themes --dark` and `--light`.

### Side-by-side view

```gitconfig
[delta]
    side-by-side = true
```

Screenshots showing side-by-side with automatic line wrapping.

### Line numbers

```gitconfig
[delta]
    line-numbers = true
```

### Merge conflicts

Screenshot of improved merge conflict display.

### Git blame

Screenshot of syntax-highlighted git blame.

### Ripgrep, git grep

Screenshot of syntax-highlighted grep output.

### Installation and usage

Please see the user manual and `delta --help`.

### Maintainers

- @dandavison
- @th1000s
