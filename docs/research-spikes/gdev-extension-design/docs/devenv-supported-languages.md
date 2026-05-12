# devenv.sh Supported Languages
- **Source**: https://devenv.sh/supported-languages/
- **Retrieved**: 2026-05-12
- **Note**: This page is an index/navigation listing. Detailed per-language docs are on separate pages.

## Languages Listed (60+)

Ansible, C, Clojure, C++, Crystal, Cue, Dart, Deno, Dotnet, Elixir, Elm, Erlang, Fortran, Gawk, Gleam, Go, Hare, Haskell, Helm, Idris, Java, Javascript, Jsonnet, Julia, Kotlin, Lean4, Lobster, Lua, Nim, Nix, Ocaml, Odin, OpenTofu, Pascal, Perl, PHP, Pkl, PureScript, Python, R, Racket, Raku, Robotframework, Ruby, Rust, Scala, Shell, Solidity, StandardML, Swift, Terraform, Texlive, TypeScript, Typst, Unison, V, Vala, and Zig.

## Example Provided

The only configuration example shown demonstrates:

```nix
languages.python.enable = true;
languages.python.version = "3.11.3";
languages.rust.enable = true;
```

With a note referencing the options reference page for additional rust channel settings.

## Common Pattern

Each language follows the pattern:
- `languages.<name>.enable = true` to activate
- Language-specific options for version, package, toolchain configuration
