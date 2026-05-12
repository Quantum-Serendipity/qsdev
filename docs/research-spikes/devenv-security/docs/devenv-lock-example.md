<!-- Source: https://raw.githubusercontent.com/NixOS/20th-nix/main/devenv.lock -->
<!-- Retrieved: 2026-05-12 -->

# devenv.lock File Example

The devenv.lock file uses JSON format and follows the Nix flake lock file schema (version 7). It contains a dependency graph with pinned versions for reproducibility.

## Structure

```json
{
  "nodes": {
    "devenv": {
      "locked": {
        "dir": "src/modules",
        "lastModified": 1678720350,
        "narHash": "sha256-2WUISGDqlshoLgLh2BCqWn0jAcHOnROYtEpgemmxXaQ=",
        "owner": "cachix",
        "repo": "devenv",
        "rev": "b97ca4bd581f0e2fb620f301cc417791f655f851",
        "type": "github"
      },
      "original": {
        "dir": "src/modules",
        "owner": "cachix",
        "repo": "devenv",
        "type": "github"
      }
    },
    "nixpkgs": {
      "locked": {
        "lastModified": 1678724065,
        "narHash": "sha256-MjeRjunqfGTBGU401nxIjs7PC9PZZ1FBCZp/bRB3C2M=",
        "owner": "NixOS",
        "repo": "nixpkgs",
        "rev": "b8afc8489dc96f29f69bec50fdc51e27883f89c1",
        "type": "github"
      },
      "original": {
        "owner": "NixOS",
        "ref": "nixpkgs-unstable",
        "repo": "nixpkgs",
        "type": "github"
      }
    },
    "pre-commit-hooks": {
      "inputs": {
        "flake-compat": "flake-compat",
        "flake-utils": "flake-utils",
        "gitignore": "gitignore",
        "nixpkgs": ["nixpkgs"],
        "nixpkgs-stable": "nixpkgs-stable"
      },
      "locked": {
        "lastModified": 1678376203,
        "narHash": "sha256-3tyYGyC8h7fBwncLZy5nCUjTJPrHbmNwp47LlNLOHSM=",
        "owner": "cachix",
        "repo": "pre-commit-hooks.nix",
        "rev": "1a20b9708962096ec2481eeb2ddca29ed747770a",
        "type": "github"
      },
      "original": {
        "owner": "cachix",
        "repo": "pre-commit-hooks.nix",
        "type": "github"
      }
    },
    "root": {
      "inputs": {
        "devenv": "devenv",
        "nixpkgs": "nixpkgs",
        "pre-commit-hooks": "pre-commit-hooks"
      }
    }
  },
  "root": "root",
  "version": 7
}
```

## Key Fields Per Node

- **locked.lastModified**: Unix timestamp of when the input was last modified
- **locked.narHash**: SHA-256 hash of the NAR (Nix ARchive) serialization of the input, used for integrity verification
- **locked.owner/repo/rev**: The exact GitHub owner, repository, and commit revision
- **locked.type**: Source type (github, gitlab, git, etc.)
- **original**: The user-specified reference before resolution (may include branch refs like "nixpkgs-unstable")
- **inputs**: For nodes with dependencies, maps input names to other node names (or arrays for "follows" relationships)
- **version**: Lock file schema version (currently 7)
