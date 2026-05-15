<!-- Source: https://raw.githubusercontent.com/tree-sitter/tree-sitter-bash/master/src/node-types.json -->
<!-- Retrieved: 2026-05-15 -->

# Bash AST Node Types for Redirections (tree-sitter-bash)

## file_redirect
```
fields:
  - descriptor (optional): file_descriptor
  - destination (optional, multiple): _primary_expression | concatenation

children: none
```

File write targets appear in the `destination` field, typically containing variable expansions or concatenated strings representing filenames.

## herestring_redirect
```
fields:
  - descriptor (optional): file_descriptor

children (required, single):
  - _primary_expression | concatenation
```

## heredoc_redirect
```
fields:
  - argument (optional, multiple): _primary_expression | concatenation
  - descriptor (optional): file_descriptor
  - operator (optional): && | ||
  - redirect (optional, multiple): file_redirect | herestring_redirect
  - right (optional): _statement

children (required, multiple):
  - heredoc_body | heredoc_end | heredoc_start | pipeline
```

## redirected_statement
```
fields:
  - body (optional): _statement
  - redirect (optional, multiple): file_redirect | heredoc_redirect | 
                                    herestring_redirect

children (optional, single): herestring_redirect
```

## pipeline
```
fields: none

children (required, multiple): _statement
```

## command
```
fields:
  - name (required): command_name
  - argument (optional, multiple): $ | == | =~ | _primary_expression | 
                                    concatenation | regex
  - redirect (optional, multiple): file_redirect | herestring_redirect

children (optional, multiple): subshell | variable_assignment
```

## Key Insight

To extract file write targets from bash commands:
1. Walk the AST looking for `file_redirect` nodes
2. Check the redirect operator (>, >>, etc.)
3. Extract the `destination` field value
4. Also check `command` nodes for tools like `tee`, `dd of=`, `sed -i`, `cp` where the write target is an argument, not a redirect
