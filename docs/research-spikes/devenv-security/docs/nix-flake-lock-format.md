# Nix Flake Lock File Structure and Integrity
- **Source**: https://nix.dev/manual/nix/2.24/command-ref/new-cli/nix3-flake
- **Retrieved**: 2026-05-12

## Lock File Format

The flake.lock file is a JSON graph structure that maps unlocked input specifications in flake.nix to their locked versions. "Each node in the graph (except the root node) maps the (usually) unlocked input specifications in flake.nix to locked input specifications."

## Fields Per Input Node

Each locked input entry contains:

- **locked**: The resolved input specification with these key fields:
  - type: The flake type (github, git, tarball, etc.)
  - rev: Git/Mercurial commit hash
  - narHash: "The SHA-256 (in SRI format) of the Nix Archive serialisation of the flake's source tree"
  - lastModified: Timestamp as integer (seconds since epoch)
  - Type-specific attributes (owner, repo, etc.)

- **original**: The unresolved specification from flake.nix

- **flake**: Boolean indicating whether it's a flake or non-flake dependency

- **inputs**: Mapping of this node's dependencies to other node labels

## Integrity Guarantees

The narHash field provides integrity verification. It allows "store paths to be computed" and enables "flake inputs to be substituted from a binary cache" by ensuring expected content matches.

## Missing or Outdated Lock Files

When no lock file exists, "Nix will automatically generate and use a lock file called flake.lock". However, "Lock files transitively lock direct as well as indirect dependencies. That is, if a lock file exists and is up to date, Nix will not look at the lock files of dependencies."

## Flake.nix to Flake.lock Mapping

Input specifications in flake.nix are matched to lock file nodes, where the inputs field in each node "maps input names (e.g. nixpkgs) to node labels (e.g. n2)". The root node's label is specified by the root attribute in the lock file.
