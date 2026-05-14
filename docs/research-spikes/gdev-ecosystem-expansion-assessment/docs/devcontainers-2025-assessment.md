# DevContainers in 2025: A Personal Take
- **Source**: https://ivanlee.me/devcontainers-in-2025-a-personal-take/
- **Retrieved**: 2026-05-14

## Author's Assessment

Ivan Lee advocates for development containers primarily for personal and open-source work, not as a universal solution. His motivation: mobility across devices and operating systems (Windows, Mac, iPad, cloud) without reinstalling tools.

## Pros

- Reproducible environments that travel between machines
- Eliminates repetitive tool installation (Python, Go versions, plugins)
- Solves the "disk image with pre-installed tools" problem
- Enables seamless context-switching for busy developers

## Cons & Limitations

- **Enterprise adoption remains limited**: "I've tried pitching development containers at different jobs and never been particularly successful"
- Organizations prioritize production firefighting over infrastructure improvements
- Technical debt (Kubernetes upgrade delays, extended support costs) takes precedence
- Not deployed at his current job despite his platform engineering role

## IDE Configuration Strategy

Uses `devcontainer.json` to handle tool integration: "solves the 'How do I get VSCode to find `ruff` installed in my virtual environment?'" problem. Employs `post_install.sh` script for setup tasks like pre-commit hook installation and git configuration.

## Alternative Platforms

- **Gitpod**: Highlighted for API-driven environment management
- **DevPod**: Praised for Kubernetes cluster compatibility
