# Python Package Installation Attacks

- **Source**: https://www.veracode.com/blog/python-package-installation-attacks/
- **Retrieved**: 2026-05-12

## Attack Vector: setup.py Execution

Source distributions (sdists) with `setup.py` files execute arbitrary code during installation. Attackers can embed malicious code in `setup.py` that runs when developers execute `pip install`, `pip download`, or even during package building—without any obvious warning signs.

The fundamental issue stems from a decade-old problem: pip must build packages from source to extract metadata needed for dependency resolution. This creates an inescapable window where "arbitrary code execution was gained and it didn't require any trickery beyond including that code in the `setup.py` file."

## Legacy Backwards Compatibility Problem

Even with modern alternatives like PEP 517/518 and `pyproject.toml`, the specification permits exceptions for older metadata formats. An attacker needs only "a single source distribution, anywhere in the dependency graph" meeting these exceptions to trigger the vulnerable behavior.

## pyproject.toml Limitations

While TOML files themselves cannot execute code, the generated `PKG-INFO` file within source distributions references `Requires-Dist` entries. Threat actors can compromise transitive dependencies through expired domain takeovers or account compromises, introducing malicious packages as downstream dependencies.

## Wheel Files: A False Security

Built distributions (wheels) don't execute code during installation, but their transitive dependencies may include source distributions. This "transitive dependency problem" means even secure wheel installation can fail if any dependency requires building from source.

## Available Protections

pip's secure installs guide recommends disallowing source distributions entirely—though this proves impractical for large projects. Additionally, sandboxing solutions like the open-source Birdcage restrict filesystem and network operations, effectively neutralizing malicious installation-time code.
