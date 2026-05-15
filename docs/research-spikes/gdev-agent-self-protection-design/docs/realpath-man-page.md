<!-- Source: https://man7.org/linux/man-pages/man1/realpath.1.html -->
<!-- Retrieved: 2026-05-15 -->

# realpath(1) Man Page

## Core Functionality
The `realpath` command "print[s] the resolved path" and displays "the resolved absolute file name."

## Key Flags

**Canonicalization Modes:**

- `-E, --canonicalize`: "all but the last component must exist (default)"
- `-e, --canonicalize-existing`: "all components of the path must exist"
- `-m, --canonicalize-missing`: "no path components need exist or be a directory"

**Symlink Resolution:**

- `-L, --logical`: "resolve '..' components before symlinks"
- `-P, --physical`: "resolve symlinks as encountered (default)"

**Additional Options:**

- `-s, --strip, --no-symlinks`: "don't expand symlinks"
- `--relative-to=DIR`: "print the resolved path relative to DIR"
- `--relative-base=DIR`: "print absolute paths unless paths below DIR"
- `-z, --zero`: "end each output line with NUL, not newline"
- `-q, --quiet`: "suppress most error messages"

## Important Distinctions

The three canonicalization modes differ in their flexibility:
- `-m` (missing): nothing needs to exist — resolves lexically for non-existent components
- default (`-E`): all but the final component must exist — resolves symlinks for existing parents
- `-e` (existing): everything must exist — fails if any component is missing

The `-P` (physical, default) mode resolves symlinks as encountered. The `-L` (logical) mode processes `..` components before resolving symlinks, which changes behavior when symlinks contain relative parent references.

## Security-Relevant Behavior

- Default mode (`-P`) resolves symlinks, which is the desired behavior for security canonicalization
- `-m` mode still resolves symlinks for existing path components, only falls back to lexical resolution for non-existent portions
- `-s` (strip/no-symlinks) does NOT resolve symlinks — never use for security purposes
- On NixOS, `realpath` is provided by GNU coreutils in the base system
