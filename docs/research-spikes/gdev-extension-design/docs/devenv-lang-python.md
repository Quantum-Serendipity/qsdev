# devenv.sh Python Language Configuration
- **Source**: https://devenv.sh/supported-languages/python/
- **Retrieved**: 2026-05-12

## Core Configuration Options

### languages.python.enable
**Type:** boolean, **Default:** `false`

### languages.python.package
**Type:** package, **Default:** `pkgs.python3`

### languages.python.version
**Type:** null or string, **Default:** `null`, **Example:** `"3.11"` or `"3.11.2"`
Requires adding nixpkgs-python as an input in devenv.yaml when specified.

### languages.python.directory
**Type:** string, **Default:** `config.devenv.root`, **Example:** `"./backend"`

## Virtual Environment Configuration

### languages.python.venv.enable
**Type:** boolean, **Default:** `false`
Activates automatic virtual environment creation stored in `$DEVENV_STATE/venv`.

### languages.python.venv.requirements
**Type:** null or strings concatenated with "\n" or absolute path, **Default:** `null`

### languages.python.venv.quiet
**Type:** boolean, **Default:** `false`

## Package Manager: Poetry

- **languages.python.poetry.enable**: boolean, default `false`
- **languages.python.poetry.package**: package, default `pkgs.poetry`
- **languages.python.poetry.activate.enable**: boolean, default `false` — auto-activates venv
- **languages.python.poetry.install.enable**: boolean, default `false` — runs `poetry install` automatically
- **languages.python.poetry.install.installRootPackage**: boolean, default `false`
- **languages.python.poetry.install.allExtras**: boolean, default `false`
- **languages.python.poetry.install.allGroups**: boolean, default `false`
- **languages.python.poetry.install.extras**: list of string, default `[]`
- **languages.python.poetry.install.groups**: list of string, default `[]`
- **languages.python.poetry.install.ignoredGroups**: list of string, default `[]`
- **languages.python.poetry.install.onlyGroups**: list of string, default `[]`
- **languages.python.poetry.install.onlyInstallRootPackage**: boolean, default `false`
- **languages.python.poetry.install.compile**: boolean, default `false`
- **languages.python.poetry.install.quiet**: boolean, default `false`
- **languages.python.poetry.install.verbosity**: one of "no", "little", "more", "debug", default `"no"`

## Package Manager: uv

- **languages.python.uv.enable**: boolean, default `false`
- **languages.python.uv.package**: package, default `pkgs.uv`
- **languages.python.uv.sync.enable**: boolean, default `false` — runs `uv sync` automatically
- **languages.python.uv.sync.packages**: list of string, default `[]`
- **languages.python.uv.sync.allPackages**: boolean, default `false`
- **languages.python.uv.sync.extras**: list of string, default `[]`
- **languages.python.uv.sync.allExtras**: boolean, default `false`
- **languages.python.uv.sync.groups**: list of string, default `[]`
- **languages.python.uv.sync.allGroups**: boolean, default `false`
- **languages.python.uv.sync.arguments**: list of string, default `[]`

## Language Server Configuration

- **languages.python.lsp.enable**: boolean, default `true`
- **languages.python.lsp.package**: package, default `pkgs.pyright`

## Native Library & Compilation Support

- **languages.python.libraries**: list of absolute path, default `["${config.devenv.dotfile}/profile"]`
- **languages.python.manylinux.enable**: boolean, default `false`
- **languages.python.patches.buildEnv.enable**: boolean, default `true`

## Advanced Features

- **languages.python.import**: function — imports Python projects using uv2nix for reproducible Nix-based builds

## Key Notes

- Poetry and uv are mutually exclusive
- Python version support requires `nixpkgs-python` input
- Virtual environment automatically rebuilds when Python interpreter changes
