<!-- Source: https://tjayrush.medium.com/evaluating-golang-cli-packages-2ae34bb79787 -->
<!-- Retrieved: 2026-05-12 -->

# Go CLI Packages Comparison

## Overview
Thomas Jay Rush evaluates three Go libraries for building modern command-line interfaces: `survey`, `promptui`, and `go-prompt`. The author ultimately selected `go-prompt` for projects requiring REPL-style interaction and wizard interfaces.

## Package Summaries

### Survey
**Repository:** github.com/AlecAivazis/survey

Survey offers an accessible API with integrated validation, multi-select functionality, and predetermined input formats. The library excels at "quick and user-friendly CLIs" with extensive documentation. However, the GitHub repository has been archived, eliminating it from serious consideration despite its technical merits.

**Strengths:** Multi-select capabilities, built-in validation, simplicity
**Weaknesses:** No tab completion, limited REPL support, archived repository

### Promptui
**Repository:** github.com/manifoldco/promptui

Promptui specializes in aesthetically refined CLIs with customizable templates and spinners for extended operations. While it provides good documentation and community involvement, it lacks native multi-select and tab completion features.

**Strengths:** Polished visual templates, customizable prompts, spinners for tasks
**Weaknesses:** No multi-select, missing tab completion, requires custom REPL coding

### Go-Prompt
**Repository:** github.com/c-bata/go-prompt

Go-prompt delivers sophisticated REPL capabilities including tab completion, intelligent suggestions, and enhanced rendering control. Selected for projects needing multi-step wizards and interactive input features.

**Strengths:** Tab completion, dynamic suggestions, native REPL support, advanced interactivity
**Weaknesses:** No built-in validation, lacks multi-select, complex setup for simple applications

## Key Selection Criteria

The author prioritized three features:
1. **Tab Completion** — Essential for user experience
2. **REPL Support** — Required for wizard-style interfaces
3. **Custom Rendering** — Needed for specialized UI requirements
