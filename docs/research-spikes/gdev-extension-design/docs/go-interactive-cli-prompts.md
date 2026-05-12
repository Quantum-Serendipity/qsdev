<!-- Source: https://dev.to/tidalcloud/interactive-cli-prompts-in-go-3bj9 -->
<!-- Retrieved: 2026-05-12 -->

# Interactive CLI Prompts in Go

## Overview

The article from Tidal Migrations demonstrates how to implement interactive command-line prompts in Go, offering both manual implementations and library-based solutions.

## Manual Implementation Examples

### Text Input Prompt
The basic approach uses `bufio.NewReader` to read from stdin until encountering a newline character. The function loops until receiving non-empty input, then returns the trimmed string.

### Password Input Prompt
This implementation leverages `golang.org/x/term.ReadPassword()` to hide user input while typing, preventing visible password display on screen.

### Yes/No Prompt
An infinite loop persists until users provide valid responses ("y," "yes," "n," or "no"). The function accepts a boolean default parameter to handle empty inputs gracefully.

### Interactive Checkboxes
The article demonstrates using the `survey` package for multi-select functionality with a simple API that accepts a label and options list.

## Featured Libraries

### Survey
**Status:** No longer actively maintained. The original maintainer recommends checking out **Bubbletea** as an alternative.

**Capabilities:** Provides accessible prompts with ANSI escape sequence support for Windows and POSIX terminals. Enables input validation, transformations, and structured question definitions.

### Prompter
**Key Features:**
- "Easy to use" interface
- "Care non-interactive (not a tty) environment" with sensible defaults
- Pure Go implementation (no CGO dependencies)
- Cross-compilation friendly

**Methods:** `Prompt()`, `Choose()`, `Password()`, `YN()`, `YesNo()`

### Promptui
**Capabilities:** Two main modes—single-line input (`Prompt`) with optional live validation and masked input, plus list selection (`Select`) with pagination, search, and custom templates.

**Integration:** Easily combines with Cobra and Urfave/CLI frameworks.

## Important Consideration: Interactive Detection

When piping input to CLI applications, prompts automatically read that data. Use `term.IsTerminal()` to detect interactive environments and conditionally implement prompts versus flag-based input alternatives.
