# The Art of README
- **Source**: https://github.com/hackergrrl/art-of-readme
- **Retrieved**: 2026-05-15

## Philosophy

The essay promotes **user-centric documentation** prioritizing accessibility over comprehensiveness. Its core philosophy holds that: "Your documentation is complete when someone can use your module without ever having to look at its code." This principle shifts responsibility from users (who must dig through code) to creators (who must communicate clearly).

The author argues READMEs represent an ethical obligation to respect readers' time. Rather than "selling" modules, creators should "let them evaluate what your creation does as objectively as possible, and decide whether it meets their needs or not."

## Structural Advice — Cognitive Funneling

The essay recommends organizing READMEs through **cognitive funneling** — arranging information from broadest to most specific details:

1. **Name** — Self-explanatory titles signal module purpose
2. **One-liner** — Brief description establishing context
3. **Usage** — Practical examples demonstrating functionality
4. **API** — Detailed function signatures and parameters
5. **Installation** — Setup instructions (even standard `npm install`)
6. **License** — Compatibility information (suggested higher placement)

Additional structural recommendations include Background sections, aggressive linking, type information, and example code files maintainers can run.

## Purpose of README

READMEs serve as "your one-stop shop" and typically represent "a module consumer's first — and maybe only — look into your creation." This document carries outsized importance because it:

- Enables quick evaluation of module fitness
- Prevents unnecessary source-code archaeology
- Provides the documented interface separate from implementation
- Determines whether developers continue investigating or move elsewhere

## Patterns Recommended

- Predictable, consistent formatting across modules
- Progressive detail increase matching reader interest
- Clear API formatting expressing parameter optionality and types
- Inline essential content (avoiding external dependencies)
- REPL sessions demonstrating stateless functions

## Anti-Patterns Criticized

- Vague or misleading module names creating confusion
- Lengthy READMEs lacking brevity discipline
- Excessive badges providing limited value
- Missing documentation forcing code inspection
- Unclear function signatures suggesting complexity requiring refactoring

## Historical Context

The essay traces README nomenclature to "at least the 1970s and the PDP-10," possibly referencing informative notes on punchcard stacks marked "READ ME!" All-caps formatting persisted partly because "UNIX systems would sort capitals before lower case letters, conveniently putting the README before the rest of the directory's content."

The author draws heavily from Perl's CPAN community, calling them "monks" with "wisdom to share," positioning Perl as Node's "spiritual grandparent" — both high-level scripting languages fueling internet infrastructure with extensive module ecosystems.
