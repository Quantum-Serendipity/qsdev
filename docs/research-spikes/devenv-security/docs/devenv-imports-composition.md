# devenv Imports and Composition
- **Source**: https://devenv.sh/composing-using-imports/
- **Retrieved**: 2026-05-12

## How Imports Work

devenv supports composition through an `imports` field in `devenv.yaml`. The system allows referencing configurations from:

- **Local paths** (relative and absolute): `./frontend`, `./backend`
- **Input sources**: `devenv/examples/supported-languages`

When entering a subdirectory like `frontend/`, the environment activates based on that folder's `devenv.nix` file. At the project root, configurations combine -- for instance, running `devenv up` starts processes from both frontend and backend definitions.

## Current Limitations on Remote Imports

The documentation explicitly states: "Remote inputs are not yet supported for devenv.yaml imports." Currently, only local file paths work for composition in `devenv.yaml`. However, the `devenv.yaml` configuration itself can reference remote inputs separately through the `inputs` section.

## Security Considerations

The provided content does not address security implications of importing external sources. While the documentation mentions `devenv/examples/supported-languages` as an importable reference, no security warnings, verification mechanisms, or trust models are discussed.
