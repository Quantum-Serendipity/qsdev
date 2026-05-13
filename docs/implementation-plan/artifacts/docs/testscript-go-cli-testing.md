# testscript: Go CLI Testing DSL
> Source: https://bitfieldconsulting.com/posts/cli-testing
> Retrieved: 2026-05-12

## Overview

testscript is a DSL for testing CLI tools in Go, derived from the testing infrastructure used for the Go tool itself. Part of the go-internal repository.

## How It Works

Tests binaries as actual executables rather than function calls.

1. Extract program logic into testable function:
```go
func main() {
    hello.Main()
}
```

2. Register with TestMain:
```go
func TestMain(m *testing.M) {
    testscript.Main(m, map[string]func(){
        "hello": hello.Main,
    })
}
```

## Script Syntax

```
# Basic execution
exec hello
stdout 'hello world\n'

# Exit status
! exec program        # expects non-zero exit

# Output matching
stdout 'pattern'      # regex match on stdout
stderr 'pattern'      # regex match on stderr
! stdout .            # assert empty stdout

# File operations
cmp stdout golden.txt # compare output to golden file
exists filename       # verify file exists
grep 'pattern' file   # search files

# Golden files (embedded)
exec myprog
cmp stdout output.txt

-- output.txt --
expected content here
```

## Key Advantages

- Test coverage integration (go test -cover works)
- Environment isolation (controlled $PATH)
- Rich assertions built-in
- No separate binary compilation needed

## Limitations

- Uses specific assertion syntax, not arbitrary shell commands
- Work directories deleted after tests (use -testwork to preserve)
- Platform-specific behavior may require separate scripts
