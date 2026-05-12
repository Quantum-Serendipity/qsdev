# Garnix Blog: Stop Trusting Nix Caches
- **Source**: https://garnix.io/blog/stop-trusting-nix-caches/
- **Retrieved**: 2026-05-12

## The Core Problem

The article warns that using external Nix caches introduces severe security vulnerabilities. When projects recommend adding their caches to your system configuration, "external caches give people with access to the cache an easy path to replacing most of your executables with malicious ones, allowing potential remote code execution and privilege escalation."

## How Cache Poisoning Works

The attack exploits how Nix resolves packages. When a cache is configured globally, "anyone with access to the cache can push malicious versions of that software, so that you end up installing (and using) it." Attackers don't need to target specific users -- they can upload plausible package candidates and wait for systems to retrieve them.

The threat escalates because "packages invoked with sudo (such as nix itself, due to nix-daemon and nixos-rebuild)" become vectors for privilege escalation.

## The Access Problem

Most projects store cache credentials in CI systems like GitHub Actions. Critically, "everyone with write access to the repo has read access to the secrets," expanding the pool of potential attackers far beyond core maintainers. This multiplies the attack surface considerably.

## Recommended Solutions

The author advocates for restricting signing authority to build infrastructure providers rather than project maintainers. Systems like Garnix and Hydra limit who can sign artifacts, reducing trust vectors. Emerging technologies like GitHub's artifact attestation offer promise for improving verification.

## Immediate Actions

Users should audit their `/etc/nix/nix.conf`, `~/.config/nix/nix.conf`, and `configuration.nix` files, removing caches lacking strong access controls. Project maintainers should migrate toward systems with restricted signing authority or remove cache recommendations entirely.
