<!-- Source: https://devenv.sh/integrations/github-actions/ -->
<!-- Retrieved: 2026-05-12 -->

# Using devenv with GitHub Actions

## Prerequisites Setup
1. Repository checkout using `actions/checkout@v5`
2. Nix installation via `cachix/install-nix-action@v31`
3. Cache configuration with `cachix/cachix-action@v16`
4. devenv installation through `nix profile add nixpkgs#devenv`

```yaml
steps:
- uses: actions/checkout@v5
- uses: cachix/install-nix-action@v31
- uses: cachix/cachix-action@v16
  with:
    name: devenv
- name: Install devenv.sh
  run: nix profile add nixpkgs#devenv
```

## Running Commands

### Single Command
```yaml
- name: Run a single command
  run: devenv shell hello
```

### Multiple Commands
```yaml
- name: Run multiple commands
  shell: devenv shell bash -- -e {0}
  run: |
    hello
    say-bye
```

## Testing
```yaml
- name: Build the devenv shell and run git hooks
  run: devenv test
```

## Nix Flakes Integration
```yaml
- name: Run command in flake shell
  shell: bash -c "nix develop --impure -c bash -- {0}"
  run: devenv test
```

## Complete Multi-Platform Workflow
```yaml
name: "Test"
on:
  pull_request:
  push:
jobs:
  tests:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
    - uses: actions/checkout@v5
    - uses: cachix/install-nix-action@v31
    - uses: cachix/cachix-action@v16
      with:
        name: devenv
    - name: Install devenv.sh
      run: nix profile add nixpkgs#devenv
    - name: Build shell and run hooks
      run: devenv test
    - name: Run single command
      run: devenv shell hello
    - name: Run multiple commands
      shell: devenv shell bash -- -e {0}
      run: |
        hello
        say-bye
```
