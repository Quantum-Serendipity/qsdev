<!-- Source: https://flake.parts/options/git-hooks-nix.html -->
<!-- Retrieved: 2026-05-12 -->

# Custom Hook Definition Schema for git-hooks.nix

## Available Attributes

| Attribute | Type | Purpose |
|-----------|------|---------|
| `enable` | boolean | Activates the hook |
| `name` | string | Display name shown during execution |
| `description` | string | Metadata describing the hook's purpose |
| `entry` | string | Command executable to run (required) |
| `files` | string | Regex pattern matching files to process |
| `types` | list of strings | File type filters to apply |
| `types_or` | list of strings | Alternative file type matches |
| `exclude_types` | list of strings | File types to exclude |
| `excludes` | list of strings | Patterns for files to skip |
| `args` | list of strings | Additional command-line parameters |
| `pass_filenames` | boolean | Whether to pass matched files as arguments |
| `stages` | list of strings | When the hook executes |
| `always_run` | boolean | Execute even without matching files |
| `fail_fast` | boolean | Stop execution if hook fails |
| `require_serial` | boolean | Run sequentially instead of in parallel |
| `verbose` | boolean | Print output regardless of success |
| `before`/`after` | list of strings | Hook execution ordering |
| `package` | package | Optional provider package |
| `extraPackages` | list of packages | Additional dependencies |
| `raw` | attribute set | Low-level configuration override |

## Valid Stage Values

`"commit-msg"`, `"post-checkout"`, `"post-commit"`, `"post-merge"`, `"post-rewrite"`, `"pre-commit"`, `"pre-merge-commit"`, `"pre-push"`, `"pre-rebase"`, `"prepare-commit-msg"`, `"manual"`, `"commit"`, `"push"`, `"merge-commit"`

Default: `["pre-commit"]`

## Custom Hook Example

```nix
hooks.my-tool = {
  enable = true;
  name = "my-tool";
  description = "Run MyTool on all files in the project";
  files = "\\.mtl$";
  entry = "${pkgs.my-tool}/bin/mytoolctl";
};
```
